package app_config

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
	"github.com/struffel/annotation-tool/pkg/superpixel_optimizer"
)

// AppState holds the entire state of the application.
type AppState struct {

	// Embedded sub-structs
	UiState
	CacheState
	LoadingState

	// Project
	Settings        *ProjectSettings
	ImageFiles      []string
	ImageFilesIndex int

	// Bitmaps and Superpixel
	ActiveImage             *image.NRGBA
	ActiveImageExistingMask *image.Gray16
	ActiveSp                superpixel.SpResult

	// Selection
	SelectionMask *image.Gray16

	// Manual Selection Fixes
	ManualFixBrushRadius int

	// SelectionMaskDirty marks that SelectionMask was modified since the last ReprocessSelection() call.
	SelectionMaskDirty bool

	LastSelectedSpLabel   int
	LastDeselectedSpLabel int

	// AwaitingMaskDecision is true when an on-disk mask was found for the current image and the
	// user has not yet chosen to start from scratch or edit it.
	AwaitingMaskDecision bool
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
	ImagesSectionButton     widget.Clickable
	ImagesSectionOpen       bool
	SuperpixelSectionButton widget.Clickable
	SuperpixelSectionOpen   bool
	MiscSectionButton       widget.Clickable
	MiscSectionOpen         bool

	// Superpixel Editors
	SuperpixelParameterEditors      []widget.Editor
	SuperpixelParameterEditorIds    []string
	SuperpixelParameterEditorTitles []string
	OptimizeParameterCheckboxes     []widget.Bool

	// Selection post-processing
	ShrinkGrowAmount      widget.Float
	ShrinkGrowAmountReset widget.Clickable
	SmoothAmount          widget.Float
	SmoothAmountReset     widget.Clickable

	// Existing-mask decision popover
	EditExistingMaskButton widget.Clickable

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
	ChangeSpAlgorithmButton     widget.Clickable
	SpAlgorithmMenuButtons      []widget.Clickable
	SpAlgorithmMenuAlgorithmIds []string
	SpAlgoritmMenuTitles        []string

	// Hover state
	IsPanning              bool
	LastPanPosition        f32.Point
	LastHoverImagePosition f32.Point
	InImageBounds          bool
	ManualFixMode          bool
	KeyboardTag            bool
	InitialFocusDone       bool

	// Hover outline cache: remembers what the current HoverPathSpec was built from so
	// RefreshHoverPreview can skip the rebuild when nothing relevant changed.
	HoverCached      bool
	HoverCacheLabel  uint16
	HoverCacheManual bool
	HoverCacheX      int
	HoverCacheY      int
	HoverCacheRadius int

	// Other UI state
	EnableAutoFloodFill       widget.Bool
	UpdateSelectionOnSpChange widget.Bool
	SpAlgorithmMenuOpen       bool
	RenderSelectionOutlines   bool
	SmoothSelectionOutlines   bool

	// View transform
	ViewScale, ViewOffsetX, ViewOffsetY float32
	FitToWindow                         bool
}

// CacheState holds cached rendering data (rebuilt when selection or image transform changes, not every frame)
type CacheState struct {
	ImgBaseOp            paint.ImageOp
	SelectionPathSpec    clip.PathSpec
	HoverPathSpec        clip.PathSpec
	ExistingMaskPathSpec clip.PathSpec

	// ProcessedSelectionMask is the current selection's mask after Shrink/Grow and Smooth are
	// applied; it is what actually gets saved to disk and is shown by the (processed) yellow outline.
	ProcessedSelectionMask *image.Gray16
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
			ImagesSectionOpen:       true,
			SuperpixelSectionOpen:   true,
			MiscSectionOpen:         true,
		},
		ManualFixBrushRadius:  16,
		LastSelectedSpLabel:   -1,
		LastDeselectedSpLabel: -1,
		LoadingState: LoadingState{
			ProjectLoadResult:  make(chan *ProjectSettings, 1),
			OptimizationResult: make(chan superpixel_optimizer.SpOptimizerResult, 1),
		},
	}

	// Initialize the superpixel algorithm menu.
	// This only needs to happen once per application lifetime.
	for _, id := range superpixel.ListAlgorithms() {

		button := new(widget.Clickable)
		appState.SpAlgorithmMenuButtons = append(appState.SpAlgorithmMenuButtons, *button)
		appState.SpAlgorithmMenuAlgorithmIds = append(appState.SpAlgorithmMenuAlgorithmIds, id)

		title, _, _, err := superpixel.GetAlgorithmInfo(id)
		if err != nil {
			title = fmt.Sprintf("Untitled (%s)", id)
		}
		appState.SpAlgoritmMenuTitles = append(appState.SpAlgoritmMenuTitles, title)
	}

	appState.RebuildEditors()

	return appState
}
