package main

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	_ "image/jpeg"
	"log"
	"math"
	"os"

	"gioui.org/app"
	"gioui.org/f32"
	"gioui.org/io/event"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"github.com/struffel/annotation-tool/superpixel"
	"github.com/struffel/annotation-tool/superpixel_factory"
	"github.com/struffel/annotation-tool/util"
)

type AppState struct {

	// Images
	img_raw     *image.NRGBA // Original image
	img_preview *image.NRGBA // Image with the annotations and overlays

	// Superpixel module
	superpixelImplementation superpixel.SpImplementation

	// Selected superpixels
	selectionIndex []bool

	// UI state
	window                     *app.Window
	canvasTag                  bool
	currentlyAdding            bool
	theme                      *material.Theme
	superpixelParameterSliders map[superpixel.SpParameter]*widget.Float

	// Display scaling
	imageScale   float32
	imageOffsetX float32
	imageOffsetY float32
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
		appState.img_raw = util.ToNRGBA(util.LoadImage(arguments.ImagePath))

		// Set up the superpixel algorithm
		impl, err := superpixel_factory.GetSuperpixelImplementationByName(arguments.Algorithm)
		if err != nil {
			log.Fatal(err)
		}
		appState.superpixelImplementation = impl
		appState.superpixelImplementation.SetImage(appState.img_raw)

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

		// Run the algorithm to compute the superpixels
		err = appState.superpixelImplementation.Run()
		if err != nil {
			log.Fatalf("Error running superpixel algorithm: %v", err)
		}

		// Create a new image into which the preview is rendered
		b := appState.img_raw.Bounds()
		dst := image.NewNRGBA(b)
		draw.Draw(dst, b, appState.img_raw, b.Min, draw.Src)
		appState.img_preview = dst
		result := appState.superpixelImplementation.GetResult()
		//appState.img_preview = result.GetSuperPixelBorders(1, color.NRGBA{R: 255, G: 0, B: 0, A: 255})

		// Initialize selection index
		appState.selectionIndex = make([]bool, result.SpCount)

		// Initialize UI state
		appState.canvasTag = false
		appState.currentlyAdding = false
		appState.theme = material.NewTheme()
		appState.superpixelParameterSliders = make(map[superpixel.SpParameter]*widget.Float)
		for _, param := range appState.superpixelImplementation.GetSupportedParameters() {
			appState.superpixelParameterSliders[param] = &widget.Float{
				Value: 0,
			}
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
	return &AppState{}
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

			appState.layout(gtx)

			appState.handlePointer(gtx)

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
							return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
								layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
									return material.Slider(appState.theme, appState.superpixelParameterSliders[param]).Layout(gtx)
								}),
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									valueLabel := material.Body2(appState.theme, fmt.Sprintf("%.2f", appState.superpixelParameterSliders[param].Value))
									valueLabel.Color = color.NRGBA{R: 200, G: 200, B: 200, A: 255}
									return valueLabel.Layout(gtx)
								}),
							)
						}),
					)
				}
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx, parameterSliderElements...)
			})
		}),
		// Main canvas
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			canvasSize := gtx.Constraints.Max

			// Set up click area for the canvas
			clickArea := clip.Rect{Max: canvasSize}.Push(gtx.Ops)
			event.Op(gtx.Ops, &appState.canvasTag)
			event.Op(gtx.Ops, true)
			clickArea.Pop()

			if appState.img_preview != nil {
				imageWidth := float32(appState.img_preview.Bounds().Dx())
				imageHeight := float32(appState.img_preview.Bounds().Dy())
				canvasWidth := float32(canvasSize.X)
				canvasHeight := float32(canvasSize.Y)

				// Scale the image without stretching
				scaleX := canvasWidth / imageWidth
				scaleY := canvasHeight / imageHeight
				scale := scaleX
				if scaleY < scale {
					scale = scaleY
				}

				// Place image in center of canvas
				offsetX := (canvasWidth - imageWidth*scale) / 2
				offsetY := (canvasHeight - imageHeight*scale) / 2

				// Store the transformation for event handling
				appState.imageScale = scale
				appState.imageOffsetX = offsetX
				appState.imageOffsetY = offsetY

				// Apply the transformation
				transformation := f32.Affine2D{}.Scale(f32.Pt(0, 0), f32.Pt(scale, scale)).Offset(f32.Pt(offsetX, offsetY))
				stack := op.Affine(transformation).Push(gtx.Ops)

				paint.NewImageOp(appState.img_preview).Add(gtx.Ops)
				paint.PaintOp{}.Add(gtx.Ops)

				stack.Pop()
			}

			return layout.Dimensions{Size: canvasSize}
		}),
	)
}

func (appState *AppState) handlePointer(gtx layout.Context) {

	// Iterate over all pointer events that occurred since the last frame
	for {

		ev, ok := gtx.Event(
			pointer.Filter{
				Target: &appState.canvasTag,
				Kinds:  pointer.Press | pointer.Drag,
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

		// From the raw position of the pointer, calculate the position in image space
		ix := (pe.Position.X - appState.imageOffsetX) / appState.imageScale
		iy := (pe.Position.Y - appState.imageOffsetY) / appState.imageScale

		// Check if the position is within the bounds of the image
		if ix < 0 || iy < 0 || int(ix) >= appState.img_raw.Bounds().Dx() || int(iy) >= appState.img_raw.Bounds().Dy() {
			fmt.Printf("Clicked outside image bounds at (%d, %d)\n", int(ix), int(iy))
			continue
		}

		// Sample the current superpixel label at the pointer position
		label := appState.superpixelImplementation.GetResult().LabelMap.Gray16At(int(ix), int(iy)).Y
		//fmt.Printf("Pointer event at (%d, %d), superpixel id: %d\n", int(ix), int(iy), label)

		// Decide the desired action based on the pointer event type and the current app state
		if pe.Kind == pointer.Press {
			// Differentiate between left-click (Button 1) and right-click (Button 3)
			if pe.Buttons == pointer.ButtonPrimary {
				// Left-click: enter selection mode
				appState.currentlyAdding = true
			} else if pe.Buttons == pointer.ButtonSecondary {
				// Right-click: enter deselection mode
				appState.currentlyAdding = false
			}
		}

		// For both drag and press events, we update the selection state of the current SP based on the current mode (selection or deselection)
		if appState.currentlyAdding && !appState.selectionIndex[label] {
			// If we are in selection mode and the SP is not already selected, select it
			appState.selectionIndex[label] = true
		} else if !appState.currentlyAdding && appState.selectionIndex[label] {
			// If we are in deselection mode and the SP is currently selected, deselect it
			appState.selectionIndex[label] = false
		}

		// Iterate over the annotated image and update the color of the selected superpixel
		result := appState.superpixelImplementation.GetResult()
		for x := 0; x < appState.img_raw.Bounds().Dx(); x++ {
			for y := 0; y < appState.img_raw.Bounds().Dy(); y++ {

				if result.LabelMap.Gray16At(x, y).Y == label {
					if appState.selectionIndex[label] {
						// If selected, color the superpixel red
						newColor := appState.img_raw.NRGBAAt(x, y)
						newColor.R = uint8(math.Min(255, float64(newColor.R)+50))

						appState.img_preview.Set(x, y, newColor)
					} else {
						// If deselected, revert to the original color
						appState.img_preview.Set(x, y, appState.img_raw.NRGBAAt(x, y))
					}
				}
			}
		}
		appState.window.Invalidate()
	}
}
