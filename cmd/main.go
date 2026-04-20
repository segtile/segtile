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
	"path/filepath"

	"gioui.org/app"
	"gioui.org/io/event"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"github.com/struffel/annotation-tool/superpixel"
	"github.com/struffel/annotation-tool/superpixel/slic"
)

type AppState struct {
	img_raw     *image.NRGBA // Original image
	img_preview *image.NRGBA // Image with the annotations and overlays

	superpixels superpixel.SpAlgorithm

	selectionIndex []bool

	window          *app.Window
	canvasTag       bool
	currentlyAdding bool
}

func main() {
	go func() {

		// Initialize app window
		appWindow := new(app.Window)
		appWindow.Option(
			app.Title("Annotation Tool"),
			app.Size(unit.Dp(1280), unit.Dp(900)),
		)

		// Create the application and load the image
		appState := &AppState{}
		appState.window = appWindow
		appState.img_raw, _ = loadImage(os.Args[1])

		// Perform the slic segmentation
		appState.superpixels = &slic.SpSlic{}
		appState.superpixels.SetDefaultParameters()
		appState.superpixels.SetImage(appState.img_raw)
		appState.superpixels.Run()

		// Create a separate copy so img_raw is never modified
		b := appState.img_raw.Bounds()
		dst := image.NewNRGBA(b)
		draw.Draw(dst, b, appState.img_raw, b.Min, draw.Src)
		appState.img_preview = dst
		result := appState.superpixels.GetResult()
		//appState.img_preview = result.GetSuperPixelBorders(1, color.NRGBA{R: 255, G: 0, B: 0, A: 255})

		// Initialize selection index
		appState.selectionIndex = make([]bool, result.SpCount)

		// Run main event loop
		if err := appState.run(appWindow); err != nil {
			log.Fatal(err)
		}
		os.Exit(0)
	}()
	app.Main()
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

			layoutContext := app.NewContext(&ops, e)

			appState.layout(layoutContext)

			appState.handlePointer(layoutContext)

			e.Frame(&ops)
		}
	}
}

func (appState *AppState) layout(layoutContext layout.Context) {
	// This is the size of the window
	size := layoutContext.Constraints.Max

	bgClip := clip.Rect{Max: size}.Push(layoutContext.Ops)

	clickArea := clip.Rect{Max: size}.Push(layoutContext.Ops)
	event.Op(layoutContext.Ops, &appState.canvasTag)

	event.Op(layoutContext.Ops, true)

	paint.ColorOp{Color: color.NRGBA{R: 35, G: 35, B: 35, A: 255}}.Add(layoutContext.Ops)
	paint.PaintOp{}.Add(layoutContext.Ops)
	clickArea.Pop()
	bgClip.Pop()

	if appState.img_preview != nil {
		paint.NewImageOp(appState.img_preview).Add(layoutContext.Ops)
		paint.PaintOp{}.Add(layoutContext.Ops)

	}
}

func (appState *AppState) handlePointer(layoutContext layout.Context) {

	// Iterate over all pointer events that occurred since the last frame
	for {

		ev, ok := layoutContext.Event(
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

		// Get the position of the pointer event
		ix := pe.Position.X
		iy := pe.Position.Y

		// Check if the position is within the bounds of the image
		if ix < 0 || iy < 0 || int(ix) >= appState.img_raw.Bounds().Dx() || int(iy) >= appState.img_raw.Bounds().Dy() {
			fmt.Printf("Clicked outside image bounds at (%d, %d)\n", int(ix), int(iy))
			continue
		}

		// Sample the current superpixel label at the pointer position
		label := appState.superpixels.GetResult().LabelMap[int(ix)][int(iy)]
		fmt.Printf("Pointer event at (%d, %d), superpixel id: %d\n", int(ix), int(iy), label)

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
		result := appState.superpixels.GetResult()
		for x := 0; x < appState.img_raw.Bounds().Dx(); x++ {
			for y := 0; y < appState.img_raw.Bounds().Dy(); y++ {

				if result.LabelMap[x][y] == label {
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

func loadImage(path string) (*image.NRGBA, error) {
	f, err := os.Open(filepath.Clean(path))
	if err != nil {
		return nil, err
	}
	defer f.Close()
	src, _, err := image.Decode(f)
	if err != nil {
		return nil, err
	}
	return toNRGBA(src), nil
}

func toNRGBA(src image.Image) *image.NRGBA {
	if n, ok := src.(*image.NRGBA); ok {
		return n
	}
	b := src.Bounds()
	dst := image.NewNRGBA(b)
	draw.Draw(dst, b, src, b.Min, draw.Src)
	return dst
}
