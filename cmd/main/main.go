package main

import (
	"fmt"
	"image"
	"image/color"
	_ "image/jpeg"
	"log"
	"os"
	"strconv"

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
	"github.com/struffel/annotation-tool/rendering"
	"github.com/struffel/annotation-tool/superpixel"
	"github.com/struffel/annotation-tool/superpixel_factory"
	"github.com/struffel/annotation-tool/util"
)

type AppState struct {

	// Images
	currentImage *image.NRGBA // Original image

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
	currentlyAdding            bool
	theme                      *material.Theme
	superpixelParameterEditors map[superpixel.SpParameter]*widget.Editor

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

type CliArguments struct {
	ImagePath  string
	Algorithm  string
	Parameters map[string]float64
}

func main() {
	go func() {

		// Load CLI arguments
		arguments := parseCommandLineArguments()

		// Initialize app window
		appWindow := new(app.Window)
		appWindow.Option(
			app.Title("Annotation Tool"),
			app.Size(unit.Dp(1280), unit.Dp(900)),
		)

		// Create the application and load the image
		appState := newAppState()
		appState.window = appWindow
		appState.currentImage = util.ToNRGBA(util.LoadImage(arguments.ImagePath))

		// Set up the superpixel algorithm
		impl, err := superpixel_factory.GetSuperpixelImplementationByName(arguments.Algorithm)
		if err != nil {
			log.Fatal(err)
		}
		appState.superpixelImplementation = impl
		appState.superpixelImplementation.SetImage(appState.currentImage)

		// Set up SP parameters
		appState.superpixelImplementation.SetRecommendedParameters()
		for name, value := range arguments.Parameters {
			param, exists := superpixel.ParseSpParameter(name)
			if !exists {
				log.Fatalf("Unknown parameter: %s", name)
			}
			err := appState.superpixelImplementation.SetParameter(param, value)
			if err != nil {
				log.Fatalf("Error setting parameter %s=%v: %v", name, value, err)
			}
			fmt.Printf("Set %s to %v\n", param.String(), value)
		}

		// Run the algorithm to compute the superpixels (also initialises selectionIndex)
		appState.recalculateSuperpixels()

		// Cache the base image as an ImageOp for efficient repeated drawing
		appState.imgBaseOp = paint.NewImageOp(appState.currentImage)
		appState.imgBaseOp.Filter = paint.FilterNearest

		// Initialize the map of parameter editors
		appState.superpixelParameterEditors = make(map[superpixel.SpParameter]*widget.Editor)
		for _, param := range appState.superpixelImplementation.GetSupportedParameters() {
			value, _ := appState.superpixelImplementation.GetParameter(param)
			ed := &widget.Editor{
				SingleLine: true,
				Submit:     true, // pressing Enter fires a SubmitEvent
				Filter:     "0123456789",
			}
			ed.SetText(fmt.Sprintf("%g", value))
			appState.superpixelParameterEditors[param] = ed
		}

		// Run main event loop
		if err := appState.run(appWindow); err != nil {
			log.Fatal(err)
		}
		os.Exit(0)
	}()
	app.Main()
}

func newAppState() *AppState {
	return &AppState{
		fitToWindow:          true,
		canvasTag:            false,
		currentlyAdding:      false,
		theme:                material.NewTheme(),
		selectionStrokeWidth: 0.75,
		hoverStrokeWidth:     0.5,
	}
}

func parseCommandLineArguments() CliArguments {
	if len(os.Args) < 3 {
		fmt.Println("Usage: <image_path (required)> <algorithm (required)> <parameter>=<value> ...")
		os.Exit(1)
	}
	parameters, err := util.ParseParameterList(os.Args[3:])
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	return CliArguments{
		ImagePath:  os.Args[1],
		Algorithm:  os.Args[2],
		Parameters: parameters,
	}
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
				var parameterSliderElements []layout.FlexChild

				for _, param := range appState.superpixelImplementation.GetSupportedParameters() {
					parameterSliderElements = append(parameterSliderElements,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							label := material.Body1(appState.theme, param.String())
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
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx, parameterSliderElements...)
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

			// Draw the image
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

				// Draw selection outline using cached path spec
				stack = op.Affine(transformation).Push(gtx.Ops)
				paint.FillShape(gtx.Ops, color.NRGBA{R: 255, G: 200, B: 0, A: 220}, clip.Stroke{Path: appState.selectionPathSpec, Width: appState.selectionStrokeWidth / appState.viewScale}.Op())
				stack.Pop()

			}

			canvasClip.Pop()
			return layout.Dimensions{Size: canvasSize}
		}),
	)
}

func (appState *AppState) handleEditors(gtx layout.Context) {

	recalculateSuperpixels := false
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
					fmt.Printf("Set %s to %v\n", param.String(), v)
					recalculateSuperpixels = true
				}
			}
		}
	}
	if recalculateSuperpixels {
		appState.recalculateSuperpixels()
	}
}

func (appState *AppState) handlePointer(gtx layout.Context) {

	// Iterate over all pointer events that occurred since the last frame
	for {

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

		// Handle simple move

		if pe.Kind == pointer.Move {
			labelMap := appState.superpixelImplementation.GetResult().LabelMap
			ix := int((pe.Position.X - appState.viewOffsetX) / appState.viewScale)
			iy := int((pe.Position.Y - appState.viewOffsetY) / appState.viewScale)
			appState.hoverPathSpec = rendering.CalculateSelectionOutlineFromIndex(*labelMap, labelMap.Gray16At(ix, iy).Y)
			continue
		}

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

		// From the raw position of the pointer, calculate the position in image space
		ix := (pe.Position.X - appState.viewOffsetX) / appState.viewScale
		iy := (pe.Position.Y - appState.viewOffsetY) / appState.viewScale

		// Check if the position is within the bounds of the image
		if ix < 0 || iy < 0 || int(ix) >= appState.currentImage.Bounds().Dx() || int(iy) >= appState.currentImage.Bounds().Dy() {
			continue
		}

		// Sample the current superpixel label at the pointer position
		label := appState.superpixelImplementation.GetResult().LabelMap.Gray16At(int(ix), int(iy)).Y

		// Decide the desired action based on the pointer event type and the current app state
		if pe.Kind == pointer.Press {
			if pe.Buttons&pointer.ButtonPrimary != 0 {
				appState.currentlyAdding = true
			} else if pe.Buttons&pointer.ButtonSecondary != 0 {
				appState.currentlyAdding = false
			}
			gtx.Execute(key.FocusCmd{Tag: &appState.keyboardTag})
		}

		// Update the selection state of the current SP
		selectionChanged := false
		if appState.currentlyAdding && !appState.selectionIndex[label] {
			appState.selectionIndex[label] = true
			selectionChanged = true
		} else if !appState.currentlyAdding && appState.selectionIndex[label] {
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

func (appState *AppState) handleKeyboard(gtx layout.Context) {
	for {
		ev, ok := gtx.Event(
			key.FocusFilter{Target: &appState.keyboardTag},
			key.Filter{Focus: &appState.keyboardTag, Name: "F"},
		)
		if !ok {
			break
		}
		ke, ok := ev.(key.Event)
		if !ok {
			continue
		}
		if ke.State == key.Press {
			appState.fitToWindow = true
			appState.window.Invalidate()
		}
	}
}

func (appState *AppState) recalculateSuperpixels() {

	fmt.Println("Calculating Superpixels...")

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

}
