package core

import (
	"fmt"
	"image"

	"github.com/reiver/go-giotoast"

	"gioui.org/app"
	"gioui.org/f32"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/widget"
	"github.com/struffel/annotation-tool/internal/ui/app_theme"
	"github.com/struffel/annotation-tool/pkg/superpixel"
	"github.com/struffel/annotation-tool/pkg/superpixel_map"

	"github.com/struffel/annotation-tool/pkg/superpixel_optimizer"
)

// AppState holds the entire state of the application.
type AppState struct {

	// Embedded sub-structs
	UiState
	CacheState
	LoadingState

	// Project
	Settings                *ProjectSettings
	ProjectSettingsDefaults *ProjectSettings
	ImageFiles              []string
	ImageFilesIndex         int

	// Bitmaps and Superpixel
	ActiveImage *image.NRGBA
	ActiveSp    superpixel.SpResult

	// Selection
	SelectionMask *superpixel_map.Mask

	// SelectionMaskDirty marks that SelectionMask was modified since the last ReprocessSelection() call.
	SelectionMaskDirty     bool
	ActiveImageHasMetadata bool
	ActiveImageMetadata    ImageMetadata

	// LastPaintedSpLabel is the last superpixel label toggled during the current select/deselect stroke.
	LastPaintedSpLabel int
}

// UiState holds the state of the user interface, including window, theme, buttons, and other UI elements.
type UiState struct {

	// UI state
	Window    *app.Window
	Toasts    giotoast.Queue
	CanvasTag bool

	// Theme
	Theme *app_theme.AppTheme

	// Sidebar sections
	ImagesSection     SidebarSection
	SuperpixelSection SidebarSection
	MiscSection       SidebarSection

	// Superpixel Editors
	SuperpixelParameterEditors []SuperpixelParameterEditor

	// Selection post-processing
	ShrinkGrowAmount      widget.Float
	ShrinkGrowAmountReset widget.Clickable
	SmoothAmount          widget.Float
	SmoothAmountReset     widget.Clickable
	RevertSavedMaskButton widget.Clickable

	// Image navigation buttons
	FirstButton         widget.Clickable
	LastButton          widget.Clickable
	PrevUnlabeledButton widget.Clickable
	PrevButton          widget.Clickable
	NextUnlabeledButton widget.Clickable
	NextButton          widget.Clickable

	// Project buttons
	OpenProjectButton widget.Clickable
	NewProjectButton  widget.Clickable
	OptimizeButton    widget.Clickable
	SaveButton        widget.Clickable

	// Superpixel algorithm controls
	ChangeSpAlgorithmButton widget.Clickable
	SpAlgorithmMenuEntries  []SpAlgorithmMenuEntry

	// Hover state
	IsPanning              bool
	LastPanPosition        f32.Point
	LastHoverImagePosition f32.Point
	InImageBounds          bool
	ManualFixMode          bool
	KeyboardTag            bool
	InitialFocusDone       bool

	// Stroke interpolation
	StrokeAnchor      f32.Point
	StrokeAnchorValid bool

	// Hover outline cache: remembers what the current HoverPathSpec was built from so
	// RefreshHoverPreview can skip the rebuild when nothing relevant changed.
	HoverCached      bool
	HoverCacheLabel  uint32
	HoverCacheManual bool
	HoverCachePos    image.Point
	HoverCacheRadius int

	// Manual Selection Fixes
	ManualFixBrushRadius int

	// Other UI state
	EnableAutoFloodFill       widget.Bool
	UpdateSelectionOnSpChange widget.Bool
	SpAlgorithmMenuOpen       bool
	RenderSelectionOutlines   bool
	SmoothSelectionOutlines   bool

	// View transform
	ViewScale   float32
	ViewOffset  f32.Point
	FitToWindow bool
}

// SidebarSection groups a collapsible sidebar section's toggle button and open state.
type SidebarSection struct {
	Button widget.Clickable
	Open   bool
}

// SpAlgorithmMenuEntry is one entry in the superpixel algorithm selection menu.
type SpAlgorithmMenuEntry struct {
	Button      widget.Clickable
	AlgorithmId string
	Title       string
}

// SuperpixelParameterEditor groups the UI state for one superpixel algorithm parameter.
type SuperpixelParameterEditor struct {
	Editor           widget.Editor
	Id               string
	Title            string
	OptimizeCheckbox widget.Bool
}

// CacheState holds cached rendering data (rebuilt when selection or image transform changes, not every frame)
type CacheState struct {
	ImgBaseOp         paint.ImageOp
	SelectionPathSpec clip.PathSpec
	HoverPathSpec     clip.PathSpec

	// ProcessedSelectionMask is the current selection's mask after Shrink/Grow and Smooth are
	// applied; it is what actually gets saved to disk and is shown by the (processed) yellow outline.
	ProcessedSelectionMask *superpixel_map.Mask
}

// LoadingState holds the state of background loading and optimization operations.
type LoadingState struct {
	// State for loading operations
	CurrentlyLoadingProject bool
	ProjectLoadResult       chan *ProjectSettings
	CurrentlyOptimizing     bool
	OptimizationResult      chan superpixel_optimizer.SpOptimizerResult
}

func NewAppState() *AppState {

	appState := &AppState{
		UiState: UiState{
			RenderSelectionOutlines: true,
			SmoothSelectionOutlines: true,
			Theme:                   app_theme.Theme,
			FitToWindow:             true,
			ImagesSection:           SidebarSection{Open: true},
			SuperpixelSection:       SidebarSection{Open: true},
			MiscSection:             SidebarSection{Open: true},
			ManualFixBrushRadius:    16,
		},
		LastPaintedSpLabel: -1,
		LoadingState: LoadingState{
			ProjectLoadResult:  make(chan *ProjectSettings, 1),
			OptimizationResult: make(chan superpixel_optimizer.SpOptimizerResult, 1),
		},
	}

	// Initialize the superpixel algorithm menu.
	// This only needs to happen once per application lifetime.
	for _, id := range superpixel.ListAlgorithms() {

		title, _, _, err := superpixel.GetAlgorithmInfo(id)
		if err != nil {
			title = fmt.Sprintf("Untitled (%s)", id)
		}
		appState.SpAlgorithmMenuEntries = append(appState.SpAlgorithmMenuEntries, SpAlgorithmMenuEntry{
			AlgorithmId: id,
			Title:       title,
		})
	}

	appState.RebuildEditors()

	return appState
}
