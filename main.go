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
	slic "github.com/kurige/SLIC"
)

type AppState struct {
	img_raw        *image.NRGBA
	img_annotated  image.Image
	slic           slic.SLIC
	selectionIndex []bool
	window         *app.Window

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
		appState.slic = *slic.MakeSlic(appState.img_raw, 5, appState.img_raw.Rect.Size().X*appState.img_raw.Rect.Size().Y/1000)
		appState.slic.Run(10)

		// Create a separate copy so img_raw is never modified
		b := appState.img_raw.Bounds()
		dst := image.NewNRGBA(b)
		draw.Draw(dst, b, appState.img_raw, b.Min, draw.Src)
		appState.img_annotated = dst

		// Initialize selection index
		appState.selectionIndex = make([]bool, len(appState.slic.Labels))

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

	if appState.img_annotated != nil {
		paint.NewImageOp(appState.img_annotated).Add(layoutContext.Ops)
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
		labelIndex := int(iy)*appState.img_raw.Bounds().Dx() + int(ix)
		label := appState.slic.Labels[labelIndex]
		fmt.Printf("Pointer event at (%d, %d), superpixel id: %d\n", int(ix), int(iy), label)

		// Decide the desired action based on the pointer event type and the current app state
		if pe.Kind == pointer.Press {
			// This is a new press. If the current SP is selected, we enter deselection mode and stay there until the press is lifted.
			// If the current SP is not selected, we enter selection mode and stay there until the press is lifted.
			appState.currentlyAdding = !appState.selectionIndex[label]
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
		for i := 0; i < len(appState.slic.Labels); i++ {
			if appState.slic.Labels[i] == label {
				x := i % appState.img_raw.Bounds().Dx()
				y := i / appState.img_raw.Bounds().Dx()
				if appState.selectionIndex[label] {
					// If selected, color the superpixel red
					newColor := appState.img_raw.NRGBAAt(x, y)
					newColor.R = uint8(math.Min(255, float64(newColor.R)+50))

					appState.img_annotated.(*image.NRGBA).Set(x, y, newColor)
				} else {
					// If deselected, revert to the original color
					appState.img_annotated.(*image.NRGBA).Set(x, y, appState.img_raw.NRGBAAt(x, y))
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
