// Package evaluation implements the optional "evaluation mode" used to measure
// annotation speed against existing ground truth masks. In evaluation mode the
// final masks in the project output directory are treated as ground truth (GT),
// every annotation starts from an empty mask, nothing is ever saved, and the
// time per image is recorded in memory.
package evaluation

import (
	"fmt"
	"image"
	"image/color"
	"sync"
	"time"

	"gioui.org/app"
	"gioui.org/io/system"
	"gioui.org/op/paint"
	"github.com/struffel/annotation-tool/pkg/superpixel_map"
)

// Mode describes which annotation tooling is allowed for an image.
type Mode string

const (
	// ModeSuperpixel allows the regular superpixel-based annotation workflow.
	ModeSuperpixel Mode = "superpixel"
	// ModeManual locks the image to the manual-fix brush.
	ModeManual Mode = "manual"
)

// ModeForIndex returns the enforced annotation mode for the image at the given
// dataset index. Every second image (odd indices) is manual-only so that
// superpixel-assisted and manual annotations interleave, which cancels out
// bias from the annotator getting faster as the experiment progresses.
func ModeForIndex(index int) Mode {
	if index%2 == 1 {
		return ModeManual
	}
	return ModeSuperpixel
}

// Result is the recorded outcome of one image visit.
type Result struct {
	ImagePath string
	Index     int
	Mode      Mode
	Duration  time.Duration
	Dice      float64
	Passed    bool
}

// Snapshot is a consistent, thread-safe copy of everything the overlay window
// needs to render one frame.
type Snapshot struct {
	HasImage  bool
	Base      paint.ImageOp
	Overlay   paint.ImageOp
	ImageSize image.Point
	Dice      float64
	Threshold float64
	Mode      Mode
	Start     time.Time
}

// State holds all evaluation-mode state. A nil *State on the AppState means
// evaluation mode is inactive.
type State struct {
	Threshold float64

	// Window is the secondary ground-truth overlay window, owned by the UI layer.
	Window *app.Window

	// Current image state, only accessed from the main application loop.
	gtMask    *superpixel_map.Mask
	dice      float64
	index     int
	imagePath string
	mode      Mode
	start     time.Time
	imageOpen bool

	results []Result

	// Shared with the overlay window goroutine.
	mu           sync.Mutex
	baseOp       paint.ImageOp
	overlayRed   paint.ImageOp
	overlayGreen paint.ImageOp
	sharedSize   image.Point
	sharedDice   float64
	sharedMode   Mode
	sharedStart  time.Time
	sharedHasImg bool
}

// NewState creates the evaluation state with the given Dice threshold.
func NewState(threshold float64) *State {
	return &State{Threshold: threshold}
}

// AttachWindow registers the overlay window.
func (s *State) AttachWindow(w *app.Window) {
	s.mu.Lock()
	s.Window = w
	s.mu.Unlock()
}

// DetachWindow unregisters the overlay window, e.g. after the user closed it.
func (s *State) DetachWindow(w *app.Window) {
	s.mu.Lock()
	if s.Window == w {
		s.Window = nil
	}
	s.mu.Unlock()
}

// CloseWindow asks the overlay window to close, if it is still open.
func (s *State) CloseWindow() {
	s.mu.Lock()
	w := s.Window
	s.mu.Unlock()
	if w != nil {
		w.Perform(system.ActionClose)
	}
}

// CurrentMode returns the annotation mode enforced for the current image.
func (s *State) CurrentMode() Mode {
	return s.mode
}

// CurrentDice returns the most recently computed Dice score of the current image.
func (s *State) CurrentDice() float64 {
	return s.dice
}

// AboveThreshold reports whether the current Dice score meets the threshold.
func (s *State) AboveThreshold() bool {
	return s.dice >= s.Threshold
}

// BeginImage starts the evaluation of a newly loaded image: it stores the GT
// mask, builds the tinted overlay images, resets the Dice score and starts the
// annotation timer. A nil gtMask is tolerated (Dice stays 0), the caller is
// expected to have surfaced the error already.
func (s *State) BeginImage(index int, imagePath string, baseImage image.Image, gtMask *superpixel_map.Mask) {
	if s.imageOpen {
		s.EndImage()
	}

	s.gtMask = gtMask
	s.dice = 0
	s.index = index
	s.imagePath = imagePath
	s.mode = ModeForIndex(index)
	s.start = time.Now()
	s.imageOpen = true

	var baseOp, red, green paint.ImageOp
	var size image.Point
	hasImage := baseImage != nil && gtMask != nil
	if hasImage {
		baseOp = paint.NewImageOp(baseImage)
		baseOp.Filter = paint.FilterNearest
		redImg, greenImg := buildTintedOverlays(gtMask)
		red = paint.NewImageOp(redImg)
		green = paint.NewImageOp(greenImg)
		size = baseImage.Bounds().Size()
	}

	s.mu.Lock()
	s.baseOp = baseOp
	s.overlayRed = red
	s.overlayGreen = green
	s.sharedSize = size
	s.sharedDice = s.dice
	s.sharedMode = s.mode
	s.sharedStart = s.start
	s.sharedHasImg = hasImage
	s.mu.Unlock()

	fmt.Printf("[eval] started %s (index %d, mode=%s)\n", imagePath, index, s.mode)
}

// EndImage stops the timer for the current image and records the result.
// The result is printed immediately so a log exists even if the application
// exits abnormally.
func (s *State) EndImage() {
	if !s.imageOpen {
		return
	}
	s.imageOpen = false

	result := Result{
		ImagePath: s.imagePath,
		Index:     s.index,
		Mode:      s.mode,
		Duration:  time.Since(s.start),
		Dice:      s.dice,
		Passed:    s.dice >= s.Threshold,
	}
	s.results = append(s.results, result)

	status := "incomplete"
	if result.Passed {
		status = "ok"
	}
	fmt.Printf("[eval] %s (index %d) took %.1fs (mode=%s, dice=%.3f, %s)\n",
		result.ImagePath, result.Index, result.Duration.Seconds(), result.Mode, result.Dice, status)
}

// UpdateDice recomputes the Dice score between the user's (processed) selection
// mask and the GT mask. It is called whenever the processed selection changed.
func (s *State) UpdateDice(processed *superpixel_map.Mask) {
	if !s.imageOpen || s.gtMask == nil {
		return
	}
	s.dice = Dice(processed, s.gtMask)

	s.mu.Lock()
	s.sharedDice = s.dice
	s.mu.Unlock()
}

// Snapshot returns a consistent copy of the data needed to render the overlay window.
func (s *State) Snapshot() Snapshot {
	s.mu.Lock()
	defer s.mu.Unlock()

	snap := Snapshot{
		HasImage:  s.sharedHasImg,
		Base:      s.baseOp,
		ImageSize: s.sharedSize,
		Dice:      s.sharedDice,
		Threshold: s.Threshold,
		Mode:      s.sharedMode,
		Start:     s.sharedStart,
	}
	if s.sharedDice >= s.Threshold {
		snap.Overlay = s.overlayGreen
	} else {
		snap.Overlay = s.overlayRed
	}
	return snap
}

// PrintSummary prints all recorded results plus per-mode averages to the
// console: once over only the images that met the Dice threshold and once over
// all images, each including the average Dice score.
func (s *State) PrintSummary() {
	fmt.Println("[eval] === Evaluation summary ===")
	for _, r := range s.results {
		status := "incomplete"
		if r.Passed {
			status = "ok"
		}
		fmt.Printf("[eval] %s (index %d): %.1fs mode=%s dice=%.3f %s\n",
			r.ImagePath, r.Index, r.Duration.Seconds(), r.Mode, r.Dice, status)
	}
	s.printModeStats("images that met the threshold", func(r Result) bool { return r.Passed })
	s.printModeStats("all images", func(r Result) bool { return true })
}

// printModeStats prints mean annotation time and mean Dice score per mode for
// the subset of results selected by include.
func (s *State) printModeStats(label string, include func(Result) bool) {
	type modeStats struct {
		totalTime time.Duration
		totalDice float64
		n         int
	}
	byMode := map[Mode]*modeStats{ModeSuperpixel: {}, ModeManual: {}}
	for _, r := range s.results {
		if !include(r) {
			continue
		}
		stats := byMode[r.Mode]
		stats.totalTime += r.Duration
		stats.totalDice += r.Dice
		stats.n++
	}

	fmt.Printf("[eval] --- %s ---\n", label)
	for _, mode := range []Mode{ModeSuperpixel, ModeManual} {
		stats := byMode[mode]
		if stats.n == 0 {
			continue
		}
		fmt.Printf("[eval] %s: mean time %.1fs, mean dice %.3f (n=%d)\n",
			mode, stats.totalTime.Seconds()/float64(stats.n), stats.totalDice/float64(stats.n), stats.n)
	}
}

// Dice computes the pixel-wise Dice coefficient between two masks:
// 2*|A∩B| / (|A|+|B|). A nil a is treated as an empty mask. Returns 0 if the
// masks differ in size or both are empty.
func Dice(a, b *superpixel_map.Mask) float64 {
	if b == nil {
		return 0
	}
	if a != nil && (a.Width != b.Width || a.Height != b.Height) {
		return 0
	}

	var intersection, aCount, bCount int
	for i := range b.Mask {
		if b.Mask[i] {
			bCount++
			if a != nil && a.Mask[i] {
				intersection++
			}
		}
		if a != nil && a.Mask[i] {
			aCount++
		}
	}

	total := aCount + bCount
	if total == 0 {
		return 0
	}
	return float64(2*intersection) / float64(total)
}

// buildTintedOverlays renders the GT mask as a red and a green overlay image.
// The mask is shown with sharp edges; having to glance at the second window
// already introduces enough uncertainty for the evaluation.
func buildTintedOverlays(gt *superpixel_map.Mask) (red, green *image.NRGBA) {
	alpha := make([]uint8, gt.Width*gt.Height)
	for i, set := range gt.Mask {
		if set {
			alpha[i] = 255
		}
	}
	red = tintImage(alpha, gt.Width, gt.Height, color.NRGBA{R: 255, A: 255})
	green = tintImage(alpha, gt.Width, gt.Height, color.NRGBA{G: 220, A: 255})
	return red, green
}

// tintImage converts alpha values into a solid-color overlay image with
// at most ~50% opacity so the underlying image stays visible.
func tintImage(alpha []uint8, w, h int, c color.NRGBA) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for i, a := range alpha {
		o := i * 4
		img.Pix[o] = c.R
		img.Pix[o+1] = c.G
		img.Pix[o+2] = c.B
		img.Pix[o+3] = a / 2
	}
	return img
}
