package main

import (
	"fmt"
	"image"
	"image/color"
	_ "image/jpeg"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"gioui.org/app"
	"gioui.org/f32"
	"gioui.org/io/event"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"github.com/sqweek/dialog"
	projectconfig "github.com/struffel/annotation-tool/config"
	"github.com/struffel/annotation-tool/rendering"
	"github.com/struffel/annotation-tool/superpixel"
	"github.com/struffel/annotation-tool/superpixel_factory"
	"github.com/struffel/annotation-tool/util"
)

// projectResult carries the outcome of a background project-load dialog.
type projectResult struct {
	cfg       *projectconfig.ProjectConfig
	outputDir string
	err       error
}

type AppState struct {

	// Project
	config     *projectconfig.ProjectConfig
	outputDir  string
	imageFiles []string

	// Images
	currentImage *image.NRGBA // Original image
	currentIndex int

	// Superpixel module
	superpixelImplementation superpixel.SpImplementation

	// Selected superpixels
	selectionIndex []bool

	// Render cache (rebuilt only when data changes, not every frame)
	imgBaseOp         paint.ImageOp
	selectionPathSpec clip.PathSpec
	hoverPathSpec     clip.PathSpec

	// UI state
	window                     *app.Window
	canvasTag                  bool
	theme                      *material.Theme
	superpixelParameterEditors map[superpixel.SpParameter]*widget.Editor
	renderSelectionOutlines    bool
	prevButton                 widget.Clickable
	nextButton                 widget.Clickable
	openProjectButton          widget.Clickable
	newProjectButton           widget.Clickable
	projectLoading             bool
	projectLoadResult          chan projectResult

	// Hover state
	isPanning        bool
	lastPanPosition  f32.Point
	keyboardTag      bool
	initialFocusDone bool

	// View transform
	viewScale, viewOffsetX, viewOffsetY float32
	fitToWindow                         bool

	// Display settings
	selectionStrokeWidth float32
	hoverStrokeWidth     float32
}

func main() {
	configPath := parseConfigPath()
	go func() {
		appWindow := new(app.Window)
		appWindow.Option(
			app.Title("Annotation Tool"),
			app.Size(unit.Dp(1280), unit.Dp(900)),
		)

		appState := newAppState()
		appState.window = appWindow

		// If a config was provided on the CLI, load the project immediately.
		if configPath != "" {
			cfg, err := projectconfig.Load(configPath)
			if err != nil {
				log.Printf("Warning: failed to load config %q: %v", configPath, err)
			} else {
				appState.applyProject(cfg, filepath.Dir(configPath))
			}
		}

		if err := appState.run(appWindow); err != nil {
			log.Fatal(err)
		}
		os.Exit(0)
	}()
	app.Main()
}

// parseConfigPath returns the optional config file path from the CLI.
func parseConfigPath() string {
	if len(os.Args) >= 2 {
		return os.Args[1]
	}
	return ""
}

// showOpenProjectDialogs prompts the user to pick an existing config file.
// Must be called from a background goroutine.
func showOpenProjectDialogs() (*projectconfig.ProjectConfig, string, error) {
	path, err := dialog.File().
		Title("Open Configuration File").
		Filter("YAML config", "yaml", "yml").
		Load()
	if err != nil {
		return nil, "", err // includes dialog.ErrCancelled
	}
	cfg, err := projectconfig.Load(path)
	if err != nil {
		return nil, "", fmt.Errorf("failed to load config %q: %w", path, err)
	}
	return cfg, filepath.Dir(path), nil
}

// showNewProjectDialogs prompts the user to select an input directory and then
// an output directory, then initialises a new project.
// Must be called from a background goroutine.
func showNewProjectDialogs() (*projectconfig.ProjectConfig, string, error) {
	inputDir, err := dialog.Directory().
		Title("Select Input Directory (raw images)").Browse()
	if err != nil {
		return nil, "", err
	}

	outputDir, err := dialog.Directory().
		Title("Select Output Directory (stores masks and config)").Browse()
	if err != nil {
		return nil, "", err
	}

	impl, err := superpixel_factory.GetSuperpixelImplementationByName("slic")
	if err != nil {
		return nil, "", err
	}
	impl.SetRecommendedParameters()
	params := make(map[string]float64)
	for _, p := range impl.GetSupportedParameters() {
		v, _ := impl.GetParameter(p)
		params[p.Key()] = v
	}
	spCfg := projectconfig.SuperpixelConfig{Algorithm: "slic", Parameters: params}

	cfg, err := projectconfig.InitNew(outputDir, inputDir, spCfg)
	if err != nil {
		return nil, "", fmt.Errorf("failed to initialize project: %w", err)
	}
	return cfg, outputDir, nil
}

func newAppState() *AppState {
	impl, _ := superpixel_factory.GetSuperpixelImplementationByName("slic")
	impl.SetRecommendedParameters()
	editors := make(map[superpixel.SpParameter]*widget.Editor)
	for _, param := range impl.GetSupportedParameters() {
		value, _ := impl.GetParameter(param)
		ed := &widget.Editor{SingleLine: true, Submit: true, Filter: "0123456789"}
		ed.SetText(fmt.Sprintf("%g", value))
		editors[param] = ed
	}
	return &AppState{
		fitToWindow:                true,
		renderSelectionOutlines:    true,
		theme:                      material.NewTheme(),
		selectionStrokeWidth:       0.75,
		hoverStrokeWidth:           0.5,
		superpixelImplementation:   impl,
		superpixelParameterEditors: editors,
		projectLoadResult:          make(chan projectResult, 1),
	}
}

// applyProject configures the AppState for the given project, saves any current mask,
// and loads the first image.
func (appState *AppState) applyProject(cfg *projectconfig.ProjectConfig, outputDir string) {
	if appState.currentImage != nil {
		appState.saveCurrentMask()
		appState.currentImage = nil
	}

	impl, err := superpixel_factory.GetSuperpixelImplementationByName(cfg.Superpixel.Algorithm)
	if err != nil {
		log.Printf("Unknown algorithm %q: %v", cfg.Superpixel.Algorithm, err)
		return
	}
	impl.SetRecommendedParameters()
	for k, v := range cfg.Superpixel.Parameters {
		param, err := superpixel.ParseSpParameter(k)
		if err != nil {
			continue
		}
		impl.SetParameter(param, v)
	}
	appState.superpixelImplementation = impl

	editors := make(map[superpixel.SpParameter]*widget.Editor)
	for _, param := range impl.GetSupportedParameters() {
		value, _ := impl.GetParameter(param)
		ed := &widget.Editor{SingleLine: true, Submit: true, Filter: "0123456789"}
		ed.SetText(fmt.Sprintf("%g", value))
		editors[param] = ed
	}
	appState.superpixelParameterEditors = editors

	imageFiles, err := util.ListFiles(cfg.InputDirectory)
	if err != nil || len(imageFiles) == 0 {
		log.Printf("Could not list input directory %q: %v", cfg.InputDirectory, err)
		return
	}

	appState.config = cfg
	appState.outputDir = outputDir
	appState.imageFiles = imageFiles
	appState.currentIndex = 0
	appState.loadImageAtIndex(0)
}

// triggerOpenProject spawns a background goroutine to open an existing project.
// Results are delivered via projectLoadResult for the event loop to apply.
func (appState *AppState) triggerOpenProject() {
	if appState.projectLoading {
		return
	}
	appState.projectLoading = true
	go func() {
		cfg, outputDir, err := showOpenProjectDialogs()
		appState.projectLoadResult <- projectResult{cfg: cfg, outputDir: outputDir, err: err}
		appState.window.Invalidate()
	}()
}

// triggerNewProject spawns a background goroutine to create a new project.
// Results are delivered via projectLoadResult for the event loop to apply.
func (appState *AppState) triggerNewProject() {
	if appState.projectLoading {
		return
	}
	appState.projectLoading = true
	go func() {
		cfg, outputDir, err := showNewProjectDialogs()
		appState.projectLoadResult <- projectResult{cfg: cfg, outputDir: outputDir, err: err}
		appState.window.Invalidate()
	}()
}

// applyProjectResult processes a result returned by a background dialog goroutine.
func (appState *AppState) applyProjectResult(r projectResult) {
	appState.projectLoading = false
	if r.err == dialog.ErrCancelled || r.cfg == nil {
		return
	}
	if r.err != nil {
		log.Printf("Failed to open project: %v", r.err)
		return
	}
	appState.applyProject(r.cfg, r.outputDir)
}

func (appState *AppState) run(appWindow *app.Window) error {

	var ops op.Ops

	for {
		switch e := appWindow.Event().(type) {
		case app.DestroyEvent:
			return e.Err
		case app.FrameEvent:
			// The list of draw operations
			ops.Reset()

			// Apply any pending project result delivered by a background dialog goroutine.
			select {
			case result := <-appState.projectLoadResult:
				appState.applyProjectResult(result)
			default:
			}

			gtx := app.NewContext(&ops, e)
			appState.handlePointer(gtx)
			appState.handleEditors(gtx)
			appState.handleKeyboard(gtx)
			appState.layout(gtx)

			e.Frame(&ops)
		}
	}
}

func (appState *AppState) layout(gtx layout.Context) {

	// This is the size of the window
	size := gtx.Constraints.Max

	// We set up a clipping area that covers the entire window.
	// This ensures that all drawing operations are confined to the window bounds, preventing any rendering outside the visible area.
	bgClip := clip.Rect{Max: size}.Push(gtx.Ops)

	// Fill the background with a solid color (dark gray)
	paint.ColorOp{Color: color.NRGBA{R: 35, G: 35, B: 35, A: 255}}.Add(gtx.Ops)
	paint.PaintOp{}.Add(gtx.Ops)
	bgClip.Pop()

	const sidebarWidth = 220

	layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
		// Sidebar
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {

			// Set fixed width for the sidebar
			gtx.Constraints.Min.X = sidebarWidth
			gtx.Constraints.Max.X = sidebarWidth

			// Draw sidebar background
			sidebarClip := clip.Rect{Max: gtx.Constraints.Max}.Push(gtx.Ops)
			paint.ColorOp{Color: color.NRGBA{R: 25, G: 25, B: 25, A: 255}}.Add(gtx.Ops)
			paint.PaintOp{}.Add(gtx.Ops)
			sidebarClip.Pop()

			// Layout sliders inside the sidebar
			inset := layout.UniformInset(unit.Dp(8))
			return inset.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				var children []layout.FlexChild

				for _, param := range appState.superpixelImplementation.GetSupportedParameters() {
					children = append(children,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							label := material.Body1(appState.theme, param.Title())
							label.Color = color.NRGBA{R: 200, G: 200, B: 200, A: 255}
							return label.Layout(gtx)
						}),

						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							currentEditor := appState.superpixelParameterEditors[param]

							editorStyle := material.Editor(appState.theme, currentEditor, "value")
							editorStyle.Color = color.NRGBA{R: 200, G: 200, B: 200, A: 255}
							return editorStyle.Layout(gtx)
						}),
					)
				}

				// Spacer between param editors and navigation controls.
				children = append(children, layout.Rigid(layout.Spacer{Height: unit.Dp(12)}.Layout))

				// Image counter label.
				if len(appState.imageFiles) > 0 {
					counter := fmt.Sprintf("%d / %d  —  %s",
						appState.currentIndex+1,
						len(appState.imageFiles),
						filepath.Base(appState.imageFiles[appState.currentIndex]),
					)
					children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						lbl := material.Body2(appState.theme, counter)
						lbl.Color = color.NRGBA{R: 160, G: 160, B: 160, A: 255}
						return lbl.Layout(gtx)
					}))
				}

				// Prev / Next navigation buttons.
				children = append(children, layout.Rigid(layout.Spacer{Height: unit.Dp(4)}.Layout))
				children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							// Consume any pending click before optionally disabling.
							if appState.prevButton.Clicked(gtx) && appState.currentIndex > 0 {
								appState.loadImageAtIndex(appState.currentIndex - 1)
							}
							if appState.currentIndex == 0 {
								gtx = gtx.Disabled()
							}
							return material.Button(appState.theme, &appState.prevButton, "← Prev").Layout(gtx)
						}),
						layout.Rigid(layout.Spacer{Width: unit.Dp(4)}.Layout),
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							if appState.nextButton.Clicked(gtx) && appState.currentIndex < len(appState.imageFiles)-1 {
								appState.loadImageAtIndex(appState.currentIndex + 1)
							}
							if appState.currentIndex >= len(appState.imageFiles)-1 {
								gtx = gtx.Disabled()
							}
							return material.Button(appState.theme, &appState.nextButton, "Next →").Layout(gtx)
						}),
					)
				}))

				return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
			})
		}),
		// Main canvas
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			canvasSize := gtx.Constraints.Max

			// Set clipping area to the canvas size and register it for events
			canvasClip := clip.Rect{Max: canvasSize}.Push(gtx.Ops)

			// Register the canvas area to receive pointer and keyboard events
			event.Op(gtx.Ops, &appState.canvasTag)
			event.Op(gtx.Ops, &appState.keyboardTag)
			key.InputHintOp{Tag: &appState.keyboardTag}.Add(gtx.Ops)

			// Set initial keyboard focus so the F key works without requiring a click first
			if !appState.initialFocusDone {
				gtx.Execute(key.FocusCmd{Tag: &appState.keyboardTag})
				appState.initialFocusDone = true
			}

			if appState.config == nil {
				// No project loaded: show centered New Project / Open Project buttons.
				promptText := "Create a new project or open an existing one."
				if appState.projectLoading {
					promptText = "Loading project…"
				}
				layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Vertical, Alignment: layout.Middle}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							lbl := material.Body1(appState.theme, promptText)
							lbl.Color = color.NRGBA{R: 160, G: 160, B: 160, A: 255}
							return lbl.Layout(gtx)
						}),
						layout.Rigid(layout.Spacer{Height: unit.Dp(16)}.Layout),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							if appState.newProjectButton.Clicked(gtx) {
								appState.triggerNewProject()
							}
							if appState.projectLoading {
								gtx = gtx.Disabled()
							}
							gtx.Constraints.Min.X = gtx.Dp(220)
							return material.Button(appState.theme, &appState.newProjectButton, "New Project").Layout(gtx)
						}),
						layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							if appState.openProjectButton.Clicked(gtx) {
								appState.triggerOpenProject()
							}
							if appState.projectLoading {
								gtx = gtx.Disabled()
							}
							gtx.Constraints.Min.X = gtx.Dp(220)
							return material.Button(appState.theme, &appState.openProjectButton, "Open Project").Layout(gtx)
						}),
					)
				})
			} else {
				// Project loaded: draw the image.
				if appState.currentImage != nil {
					imageWidth := float32(appState.currentImage.Bounds().Dx())
					imageHeight := float32(appState.currentImage.Bounds().Dy())
					canvasWidth := float32(canvasSize.X)
					canvasHeight := float32(canvasSize.Y)

					// Fit image to window on first render or when requested
					if appState.fitToWindow {
						scaleX := canvasWidth / imageWidth
						scaleY := canvasHeight / imageHeight
						appState.viewScale = scaleX
						if scaleY < appState.viewScale {
							appState.viewScale = scaleY
						}
						appState.viewOffsetX = (canvasWidth - imageWidth*appState.viewScale) / 2
						appState.viewOffsetY = (canvasHeight - imageHeight*appState.viewScale) / 2
						appState.fitToWindow = false
					}

					// Build the image-to-canvas affine transform
					transformation := f32.Affine2D{}.Scale(f32.Pt(0, 0), f32.Pt(appState.viewScale, appState.viewScale)).Offset(f32.Pt(appState.viewOffsetX, appState.viewOffsetY))

					// Draw the base image using the cached ImageOp
					stack := op.Affine(transformation).Push(gtx.Ops)
					appState.imgBaseOp.Add(gtx.Ops)
					paint.PaintOp{}.Add(gtx.Ops)
					stack.Pop()

					// Draw hover outline using cached path spec
					stack = op.Affine(transformation).Push(gtx.Ops)
					paint.FillShape(gtx.Ops, color.NRGBA{R: 0, G: 200, B: 255, A: 220}, clip.Stroke{Path: appState.hoverPathSpec, Width: appState.hoverStrokeWidth / appState.viewScale}.Op())
					stack.Pop()

					if appState.renderSelectionOutlines {
						// Draw selection outline using cached path spec
						stack = op.Affine(transformation).Push(gtx.Ops)
						paint.FillShape(gtx.Ops, color.NRGBA{R: 255, G: 200, B: 0, A: 220}, clip.Stroke{Path: appState.selectionPathSpec, Width: appState.selectionStrokeWidth / appState.viewScale}.Op())
						stack.Pop()
					}
				}

				// Small New Project / Open Project buttons overlaid at the top-left corner.
				layout.NW.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.UniformInset(unit.Dp(8)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						if appState.projectLoading {
							gtx = gtx.Disabled()
						}
						return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								if appState.newProjectButton.Clicked(gtx) {
									appState.triggerNewProject()
								}
								return material.Button(appState.theme, &appState.newProjectButton, "New Project").Layout(gtx)
							}),
							layout.Rigid(layout.Spacer{Width: unit.Dp(4)}.Layout),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								if appState.openProjectButton.Clicked(gtx) {
									appState.triggerOpenProject()
								}
								return material.Button(appState.theme, &appState.openProjectButton, "Open Project").Layout(gtx)
							}),
						)
					})
				})
			}

			canvasClip.Pop()
			return layout.Dimensions{Size: canvasSize}
		}),
	)
}

func (appState *AppState) handleEditors(gtx layout.Context) {

	recalculate := false
	for param, editor := range appState.superpixelParameterEditors {
		// Drain events and apply the value on Submit
		for {
			ev, ok := editor.Update(gtx)
			if !ok {
				break
			}
			if _, ok := ev.(widget.SubmitEvent); ok {
				if v, err := strconv.ParseFloat(editor.Text(), 64); err == nil {
					appState.superpixelImplementation.SetParameter(param, v)
					fmt.Printf("Set %s to %v\n", param.Title(), v)
					recalculate = true
				}
			}
		}
	}
	if recalculate {
		if appState.currentImage != nil {
			appState.recalculateSuperpixels()
		}

		// Persist updated parameters to config.
		if appState.config != nil {
			for _, p := range appState.superpixelImplementation.GetSupportedParameters() {
				v, _ := appState.superpixelImplementation.GetParameter(p)
				appState.config.Superpixel.Parameters[p.Key()] = v
			}
			if err := projectconfig.Save(appState.config, appState.outputDir); err != nil {
				fmt.Println("Warning: could not save config:", err)
			}
		}
	}
}

func (appState *AppState) handlePointer(gtx layout.Context) {

	// Iterate over all pointer events that occurred since the last frame
	for {

		// Filter for all events that should be handled
		ev, ok := gtx.Event(
			pointer.Filter{
				Target:  &appState.canvasTag,
				Kinds:   pointer.Press | pointer.Release | pointer.Drag | pointer.Scroll | pointer.Move,
				ScrollY: pointer.ScrollRange{Min: -100, Max: 100},
			},
		)
		if !ok {
			// No more events to process
			break
		}
		pe, ok := ev.(pointer.Event)
		if !ok {
			// Not a pointer event, ignore
			continue
		}

		// No image loaded — skip all image-space pointer logic.
		if appState.currentImage == nil {
			continue
		}

		// Always set the keyboard focus to the canvas when hovering on it, so that key commands work without requiring an extra click
		gtx.Execute(key.FocusCmd{Tag: &appState.keyboardTag})

		// From the raw position of the pointer, calculate the position in image space
		imageX := (pe.Position.X - appState.viewOffsetX) / appState.viewScale
		imageY := (pe.Position.Y - appState.viewOffsetY) / appState.viewScale

		// Check if the position is within the bounds of the image
		inImageBounds := imageX >= 0 && imageY >= 0 && int(imageX) < appState.currentImage.Bounds().Dx() && int(imageY) < appState.currentImage.Bounds().Dy()

		// We start with actions that always fire, even if the cursor is outside the image bounds

		// Handle scroll (zoom)
		if pe.Kind == pointer.Scroll {
			// Use fixed 10% per notch regardless of raw scroll magnitude
			var zoomFactor float32
			if pe.Scroll.Y > 0 {
				zoomFactor = 1 / 1.1
			} else {
				zoomFactor = 1.1
			}
			appState.viewOffsetX = pe.Position.X - (pe.Position.X-appState.viewOffsetX)*zoomFactor
			appState.viewOffsetY = pe.Position.Y - (pe.Position.Y-appState.viewOffsetY)*zoomFactor
			appState.viewScale *= zoomFactor
			appState.fitToWindow = false
			appState.window.Invalidate()
			continue
		}

		// Handle middle mouse panning
		if pe.Kind == pointer.Press && pe.Buttons&pointer.ButtonTertiary != 0 {
			appState.isPanning = true
			appState.lastPanPosition = pe.Position
			continue
		}
		if appState.isPanning {
			if pe.Kind == pointer.Drag {
				appState.viewOffsetX += pe.Position.X - appState.lastPanPosition.X
				appState.viewOffsetY += pe.Position.Y - appState.lastPanPosition.Y
				appState.lastPanPosition = pe.Position
				appState.fitToWindow = false
				appState.window.Invalidate()
			} else if pe.Kind == pointer.Release {
				appState.isPanning = false
			}
			continue
		}

		// Handle simple move and update the hover preview
		if pe.Kind == pointer.Move || pe.Kind == pointer.Drag {
			if inImageBounds {
				labelMap := appState.superpixelImplementation.GetResult().LabelMap
				appState.hoverPathSpec = rendering.CalculateSelectionOutlineFromIndex(*labelMap, labelMap.Gray16At(int(imageX), int(imageY)).Y)
			} else {
				appState.hoverPathSpec = rendering.GetEmptySelectionOutline()
			}
		}

		//  Handle primary/secondary click for de-/selection

		if inImageBounds {
			// Sample the current superpixel label at the pointer position
			labelMap := appState.superpixelImplementation.GetResult().LabelMap
			label := labelMap.Gray16At(int(imageX), int(imageY)).Y

			// Toggle selection based on primary/secondary button and update the selection outline if it changed
			selectionChanged := false
			if pe.Buttons&pointer.ButtonPrimary != 0 && !appState.selectionIndex[label] {
				if pe.Modifiers&key.ModShift != 0 {
					// If shift is held, perform a flood fill selection
					appState.selectionIndex = util.SuperpixelFloodFillSelection(labelMap, int(imageX), int(imageY), appState.selectionIndex)
				} else {
					// Perform a normal selection of a single superpixel
					appState.selectionIndex[label] = true
				}
				selectionChanged = true
			} else if pe.Buttons&pointer.ButtonSecondary != 0 && appState.selectionIndex[label] {
				appState.selectionIndex[label] = false
				selectionChanged = true
			}

			if selectionChanged {
				labelMap := appState.superpixelImplementation.GetResult().LabelMap
				appState.selectionPathSpec = rendering.CalculateSelectionOutlineFromSlice(*labelMap, appState.selectionIndex)
				appState.window.Invalidate()
			}
		}
	}
}

func (appState *AppState) handleKeyboard(gtx layout.Context) {
	for {
		ev, ok := gtx.Event(
			key.FocusFilter{Target: &appState.keyboardTag},
			key.Filter{Focus: &appState.keyboardTag, Name: "F"},
			key.Filter{Focus: &appState.keyboardTag, Name: "C"},
			key.Filter{Focus: &appState.keyboardTag, Name: "D"},
			key.Filter{Focus: &appState.keyboardTag, Name: key.NameTab, Optional: key.ModShift},
		)
		if !ok {
			break
		}
		ke, ok := ev.(key.Event)
		if !ok {
			continue
		}

		// [F]it to window
		if ke.State == key.Press && ke.Name == "F" {
			appState.fitToWindow = true
			appState.window.Invalidate()
		}

		// [C]lear selection
		if ke.State == key.Press && ke.Name == "C" {
			for i := range appState.selectionIndex {
				appState.selectionIndex[i] = false
			}
			appState.selectionPathSpec = rendering.GetEmptySelectionOutline()
			appState.window.Invalidate()
		}

		// [D]isable preview (while held)
		if ke.Name == "D" {
			if ke.State == key.Press {
				appState.renderSelectionOutlines = false
				appState.window.Invalidate()
			} else if ke.State == key.Release {
				appState.renderSelectionOutlines = true
				appState.window.Invalidate()
			}
		}

		// Tab → next image, Shift+Tab → previous image
		if ke.State == key.Press && ke.Name == key.NameTab {
			if ke.Modifiers.Contain(key.ModShift) {
				if appState.currentIndex > 0 {
					appState.loadImageAtIndex(appState.currentIndex - 1)
				}
			} else {
				if appState.currentIndex < len(appState.imageFiles)-1 {
					appState.loadImageAtIndex(appState.currentIndex + 1)
				}
			}
		}
	}
}

func (appState *AppState) recalculateSuperpixels() {

	fmt.Println("Calculating superpixels...")
	start := time.Now().UnixMicro()

	// Snapshot the current selection as a mask before the labels change
	oldResult := appState.superpixelImplementation.GetResult()
	var oldSelectionMask *image.Gray16
	if oldResult.SpCount > 0 && len(appState.selectionIndex) > 0 {
		oldSelectionMask = oldResult.SelectionToMask(appState.selectionIndex)
	}

	// Re-run the superpixel algorithm
	err := appState.superpixelImplementation.Run()
	if err != nil {
		log.Fatalf("Error running superpixel algorithm: %v", err)
	}

	// Restore the selection by matching old mask pixels to new superpixels,
	// or reset to an empty selection when there was no previous result.
	result := appState.superpixelImplementation.GetResult()
	if oldSelectionMask != nil {
		appState.selectionIndex = result.SelectFromMask(oldSelectionMask, 0.5)
	} else {
		appState.selectionIndex = make([]bool, result.SpCount)
	}
	//appState.updateSelectionOutline()
	labelMap := appState.superpixelImplementation.GetResult().LabelMap
	appState.selectionPathSpec = rendering.CalculateSelectionOutlineFromSlice(*labelMap, appState.selectionIndex)

	end := time.Now().UnixMicro()
	fmt.Printf("Finished calculating superpixels in %.1f milliseconds\n", float64(end-start)/1000)

}

// loadImageAtIndex saves the current mask (if any), loads the image at index i,
// runs the superpixel algorithm, and restores any previously saved mask.
func (appState *AppState) loadImageAtIndex(i int) {
	if appState.currentImage != nil {
		appState.saveCurrentMask()
	}

	appState.currentImage = util.ToNRGBA(util.LoadImage(appState.imageFiles[i]))
	appState.superpixelImplementation.SetImage(appState.currentImage)
	appState.recalculateSuperpixels()

	// Restore selection from an existing mask file, if present.
	maskPath := appState.maskPathForIndex(i)
	if _, err := os.Stat(maskPath); err == nil {
		existingMask := util.ToGray16(util.LoadImage(maskPath))
		result := appState.superpixelImplementation.GetResult()
		appState.selectionIndex = result.SelectFromMask(existingMask, 0.5)
		appState.selectionPathSpec = rendering.CalculateSelectionOutlineFromSlice(
			*result.LabelMap,
			appState.selectionIndex,
		)
	}

	// Rebuild image paint op.
	appState.imgBaseOp = paint.NewImageOp(appState.currentImage)
	appState.imgBaseOp.Filter = paint.FilterNearest

	// Reset view and hover.
	appState.fitToWindow = true
	appState.hoverPathSpec = rendering.GetEmptySelectionOutline()

	// Sync editor text to current parameter values.
	for _, param := range appState.superpixelImplementation.GetSupportedParameters() {
		if ed, ok := appState.superpixelParameterEditors[param]; ok {
			value, _ := appState.superpixelImplementation.GetParameter(param)
			ed.SetText(fmt.Sprintf("%g", value))
		}
	}

	appState.currentIndex = i
	if appState.window != nil {
		appState.window.Invalidate()
	}
}

// saveCurrentMask writes the current selection as a Gray16 PNG to the output directory.
func (appState *AppState) saveCurrentMask() {
	result := appState.superpixelImplementation.GetResult()
	if result.SpCount == 0 {
		return
	}
	mask := result.SelectionToMask(appState.selectionIndex)
	outPath := appState.maskPathForIndex(appState.currentIndex)
	util.SaveImage(outPath, mask)
}

// maskPathForIndex returns the output PNG path for image at index i.
// The base name of the source file is kept but the extension is replaced with .png.
func (appState *AppState) maskPathForIndex(i int) string {
	base := filepath.Base(appState.imageFiles[i])
	ext := filepath.Ext(base)
	name := base[:len(base)-len(ext)] + ".png"
	return filepath.Join(appState.outputDir, name)
}
