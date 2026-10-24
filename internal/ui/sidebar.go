package ui

import (
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"time"

	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"gioui.org/x/component"
	"github.com/struffel/annotation-tool/internal/core"
	"github.com/struffel/annotation-tool/internal/ui/app_theme"
	"github.com/struffel/annotation-tool/internal/ui/components"
	"github.com/struffel/annotation-tool/pkg/superpixel"
)

const checkboxSize = unit.Dp(20)
const standardDividerThickness = unit.Dp(2)
const sidebarWidth = 300

// drawSidebarSection draws a collapsible sidebar section header followed by its body when open.
func drawSidebarSection(gtx layout.Context, appState *core.AppState, title string, dividerThickness unit.Dp, open bool, button *widget.Clickable, body func(layout.Context) layout.Dimensions) layout.Dimensions {
	HandleUiButtonClick(appState, gtx, button)

	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return components.DrawSidebarSectionHeader(gtx, button, appState.Theme, title, open, dividerThickness)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if !open {
				return layout.Dimensions{}
			}
			return layout.UniformInset(unit.Dp(8)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return body(gtx)
			})
		}),
	)
}

func drawImagesSection(gtx layout.Context, appState *core.AppState) layout.Dimensions {
	return drawSidebarSection(gtx, appState, "Images", standardDividerThickness, appState.ImagesSection.Open, &appState.ImagesSection.Button, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,

			// Image name and counter
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				if len(appState.ImageFiles) == 0 {
					return layout.Dimensions{}
				}
				counter := fmt.Sprintf("%d / %d",
					appState.ActiveImageIndex+1,
					len(appState.ImageFiles),
				)
				lbl := material.Caption(&appState.Theme.GioTheme, counter)
				lbl.Color = appState.Theme.GioTheme.ContrastFg
				return lbl.Layout(gtx)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				if len(appState.ImageFiles) == 0 {
					return layout.Dimensions{}
				}
				lbl := material.Body2(&appState.Theme.GioTheme, filepath.Base(appState.ImageFiles[appState.ActiveImageIndex]))
				lbl.Color = appState.Theme.GioTheme.ContrastFg
				return lbl.Layout(gtx)
			}),
			layout.Rigid(layout.Spacer{Height: unit.Dp(4)}.Layout),

			// Image navigation buttons
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {

				atFirstImage := appState.ActiveImageIndex == 0
				atLastImage := appState.ActiveImageIndex == len(appState.ImageFiles)-1

				return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						if atFirstImage {
							gtx = gtx.Disabled()
						} else {
							HandleUiButtonClick(appState, gtx, &appState.FirstButton)
						}
						return components.DrawIconButton(gtx, &appState.FirstButton, appState.Theme, app_theme.IconFirst, "First", true)
					}),
					layout.Rigid(layout.Spacer{Width: unit.Dp(4)}.Layout),
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						if atFirstImage {
							gtx = gtx.Disabled()
						} else {
							HandleUiButtonClick(appState, gtx, &appState.PrevUnlabeledButton)
						}
						return components.DrawIconButton(gtx, &appState.PrevUnlabeledButton, appState.Theme, app_theme.IconPrevUnlabeled, "Previous Unlabeled", true)
					}),
					layout.Rigid(layout.Spacer{Width: unit.Dp(4)}.Layout),
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						if atFirstImage {
							gtx = gtx.Disabled()
						} else {
							HandleUiButtonClick(appState, gtx, &appState.PrevButton)
						}
						return components.DrawIconButton(gtx, &appState.PrevButton, appState.Theme, app_theme.IconPrev, "Previous", true)
					}),
					layout.Rigid(layout.Spacer{Width: unit.Dp(4)}.Layout),
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {

						if atLastImage {
							gtx = gtx.Disabled()
						} else {
							HandleUiButtonClick(appState, gtx, &appState.NextButton)
						}
						return components.DrawIconButton(gtx, &appState.NextButton, appState.Theme, app_theme.IconNext, "Next", true)
					}),
					layout.Rigid(layout.Spacer{Width: unit.Dp(4)}.Layout),
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						if atLastImage {
							gtx = gtx.Disabled()
						} else {
							HandleUiButtonClick(appState, gtx, &appState.NextUnlabeledButton)
						}
						return components.DrawIconButton(gtx, &appState.NextUnlabeledButton, appState.Theme, app_theme.IconNextUnlabeled, "Next Unlabeled", true)
					}),
					layout.Rigid(layout.Spacer{Width: unit.Dp(4)}.Layout),
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						if atLastImage {
							gtx = gtx.Disabled()
						} else {
							HandleUiButtonClick(appState, gtx, &appState.LastButton)
						}
						return components.DrawIconButton(gtx, &appState.LastButton, appState.Theme, app_theme.IconLast, "Last", true)
					}),
				)
			}),
		)
	})
}

func drawSuperpixelSection(gtx layout.Context, appState *core.AppState) layout.Dimensions {
	return drawSidebarSection(gtx, appState, "Superpixel Presegmentation", standardDividerThickness, appState.SuperpixelSection.Open, &appState.SuperpixelSection.Button, func(gtx layout.Context) layout.Dimensions {
		if appState.ActiveSettings == nil {
			return layout.Dimensions{}
		}
		HandleUiButtonClick(appState, gtx, &appState.ChangeSpAlgorithmButton)
		if appState.SpAlgorithmMenuOpen {
			for i := range appState.SpAlgorithmMenuEntries {
				HandleUiButtonClick(appState, gtx, &appState.SpAlgorithmMenuEntries[i].Button)
			}
		}

		var children []layout.FlexChild

		// Algorithm Name and change button
		children = append(children,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Axis: layout.Horizontal, Alignment: layout.End}.Layout(gtx,
							layout.Flexed(3, func(gtx layout.Context) layout.Dimensions {

								superpixelTitle, _, _, err := superpixel.GetAlgorithmInfo(appState.ActiveSettings.Superpixel.SpAlgorithmId)
								if err != nil {
									superpixelTitle = fmt.Sprintf("Untitled (%s)", appState.ActiveSettings.Superpixel.SpAlgorithmId)
								}

								label := material.Body2(&appState.Theme.GioTheme, superpixelTitle)
								label.Color = appState.Theme.GioTheme.Palette.Fg
								label.Font.Weight = font.Bold
								return label.Layout(gtx)
							}),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return components.DrawIconButton(gtx, &appState.ChangeSpAlgorithmButton, appState.Theme, app_theme.IconChangeAlgorithm, "Change Superpixel Algorithm", false)
							}),
						)
					}),

					// Superpixel algorithm menu
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						if !appState.SpAlgorithmMenuOpen {
							return layout.Dimensions{}
						}

						children := make([]layout.FlexChild, 0, len(appState.SpAlgorithmMenuEntries))
						for i := range appState.SpAlgorithmMenuEntries {

							button := &appState.SpAlgorithmMenuEntries[i].Button
							title := appState.SpAlgorithmMenuEntries[i].Title

							children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								HandleUiButtonClick(appState, gtx, button)
								return component.MenuItem(&appState.Theme.GioTheme, button, title).Layout(gtx)
							}))
						}
						return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
					}),
				)
			}),
		)

		if !appState.SpAlgorithmMenuOpen {

			// Spacer
			children = append(children,
				layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
			)

			// Individual parameter editors
			for i := range appState.SuperpixelParameterEditors {

				paramId := appState.SuperpixelParameterEditors[i].Id
				title := appState.SuperpixelParameterEditors[i].Title
				editor := &appState.SuperpixelParameterEditors[i].Editor
				children = append(children,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Axis: layout.Horizontal, Alignment: layout.End}.Layout(gtx,
							layout.Flexed(3, func(gtx layout.Context) layout.Dimensions {
								label := material.Body1(&appState.Theme.GioTheme, title)
								label.Color = appState.Theme.GioTheme.Palette.Fg
								return label.Layout(gtx)
							}),

							layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
								uncommitedChange := (fmt.Sprintf("%g", appState.ActiveSettings.Superpixel.SpAlgorithmParameters[paramId]) != editor.Text())
								return components.DrawEditor(gtx, appState, editor, uncommitedChange)
							}),
						)
					}),
				)
			}

			// Spacer before optimization section
			children = append(children,
				layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
			)

			// Recursively draw a separate sidebar section for optimization settings
			children = append(children,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return drawSidebarSection(gtx, appState, "Suggest Parameters", unit.Dp(0.0), appState.SuperpixelOptimizationSection.Open, &appState.SuperpixelOptimizationSection.Button, func(gtx layout.Context) layout.Dimensions {
						var optimizationChildren []layout.FlexChild

						for i := range appState.SuperpixelParameterEditors {
							title := appState.SuperpixelParameterEditors[i].Title
							checkbox := &appState.SuperpixelParameterEditors[i].OptimizeCheckbox
							optimizationChildren = append(optimizationChildren,
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									checkboxStyle := material.CheckBox(&appState.Theme.GioTheme, checkbox, title)
									checkboxStyle.IconColor = appState.Theme.GioTheme.Palette.Fg
									checkboxStyle.Size = unit.Dp(16)
									return checkboxStyle.Layout(gtx)
								}),
							)
						}
						optimizationChildren = append(optimizationChildren,
							layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								if len(appState.SelectedOptimizeParameters()) == 0 {
									gtx = gtx.Disabled()
								} else {
									HandleUiButtonClick(appState, gtx, &appState.OptimizeButton)
								}
								return components.DrawSubtleButton(gtx, &appState.OptimizeButton, appState.Theme, "Optimize", 12)
							}),
						)
						return layout.Flex{Axis: layout.Vertical}.Layout(gtx, optimizationChildren...)
					})
				}),
			)
		}

		return layout.Flex{Axis: layout.Vertical, Spacing: layout.SpaceEnd}.Layout(gtx, children...)
	})
}

func drawSelectionSection(gtx layout.Context, appState *core.AppState) layout.Dimensions {
	return drawSidebarSection(gtx, appState, "Selection Tuning", standardDividerThickness, appState.MiscSection.Open, &appState.MiscSection.Button, func(gtx layout.Context) layout.Dimensions {
		children := []layout.FlexChild{
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				checkbox := material.CheckBox(&appState.Theme.GioTheme, &appState.EnableAutoFloodFill, "Auto Flood Fill")
				checkbox.Size = checkboxSize
				return checkbox.Layout(gtx)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				checkbox := material.CheckBox(&appState.Theme.GioTheme, &appState.UpdateSelectionOnSpChange, "Snap to SP borders")
				checkbox.Size = checkboxSize
				return checkbox.Layout(gtx)
			}),
			layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return material.Caption(&appState.Theme.GioTheme, "Shrink / Grow").Layout(gtx)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						return material.Slider(&appState.Theme.GioTheme, &appState.ShrinkGrowAmount).Layout(gtx)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						HandleUiButtonClick(appState, gtx, &appState.ShrinkGrowAmountReset)
						return components.DrawIconButton(gtx, &appState.ShrinkGrowAmountReset, appState.Theme, app_theme.IconAvReplay, "Reset Shrink/Grow", false)
					}),
				)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return material.Caption(&appState.Theme.GioTheme, "Smooth").Layout(gtx)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {

				return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						return material.Slider(&appState.Theme.GioTheme, &appState.SmoothAmount).Layout(gtx)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						HandleUiButtonClick(appState, gtx, &appState.SmoothAmountReset)
						return components.DrawIconButton(gtx, &appState.SmoothAmountReset, appState.Theme, app_theme.IconAvReplay, "Reset Smooth", false)
					}),
				)
			}),
		}

		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			children...,
		)
	})
}

func drawSelectionInfoSection(gtx layout.Context, appState *core.AppState) layout.Dimensions {
	title := "Annotation Info"

	if appState.SelectionMaskDiffersVersionOnDisk {
		title += "*"
	}

	return drawSidebarSection(gtx, appState, title, standardDividerThickness, appState.SavedImageInfoSection.Open, &appState.SavedImageInfoSection.Button, func(gtx layout.Context) layout.Dimensions {
		if appState.ActiveImageMetadata == nil {
			return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					return material.Caption(&appState.Theme.GioTheme, "Annotation has not been saved yet.").Layout(gtx)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					HandleUiButtonClick(appState, gtx, &appState.SaveButton)
					return components.DrawIconButton(gtx, &appState.SaveButton, appState.Theme, app_theme.IconSave, "Save annotation", false)
				}),
			)
		}

		annotator := appState.ActiveImageMetadata.Annotator
		if annotator == "" {
			annotator = "Unknown"
		}

		annotationTime := "Unknown"
		if !appState.ActiveImageMetadata.AnnotationTime.IsZero() {
			annotationTime = appState.ActiveImageMetadata.AnnotationTime.Format(time.RFC1123)
		}

		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return components.DrawInfoRow(gtx, appState, "Last saved by:", annotator, app_theme.IconSocialPerson)
							}),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return layout.Spacer{Height: unit.Dp(4)}.Layout(gtx)
							}),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return components.DrawInfoRow(gtx, appState, "Last saved at:", annotationTime, app_theme.IconActionDateRange)
							}),
						)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								HandleUiButtonClick(appState, gtx, &appState.SaveButton)
								return components.DrawIconButton(gtx, &appState.SaveButton, appState.Theme, app_theme.IconSave, "Save annotation", false)
							}),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								if !appState.SelectionMaskDiffersVersionOnDisk {
									return layout.Dimensions{}
								}
								HandleUiButtonClick(appState, gtx, &appState.RevertSavedMaskButton)
								return components.DrawIconButton(gtx, &appState.RevertSavedMaskButton, appState.Theme, app_theme.IconAvReplay, "Revert to saved annotation", false)
							}),
						)
					}),
				)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Spacer{Height: unit.Dp(4)}.Layout(gtx)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return components.DrawDivider(gtx, appState.Theme, unit.Dp(0.5))
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Spacer{Height: unit.Dp(4)}.Layout(gtx)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return material.Caption(&appState.Theme.GioTheme, "Presegmentation:").Layout(gtx)
							}),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								label := material.Body2(&appState.Theme.GioTheme, appState.ActiveImageMetadata.Superpixel.SpAlgorithmId)
								label.Font.Weight = font.Bold
								return label.Layout(gtx)
							}),
							// Render the parameters line by line
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								params := appState.ActiveImageMetadata.Superpixel.SpAlgorithmParameters

								var keys []string
								for k := range params {
									keys = append(keys, k)
								}
								sort.Strings(keys)

								var lines []layout.FlexChild
								for _, k := range keys {
									key := k
									value := params[k]
									lines = append(lines, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
										return material.Body2(&appState.Theme.GioTheme, key+": "+strconv.FormatFloat(value, 'f', 3, 64)).Layout(gtx)
									}))
								}
								return layout.Flex{Axis: layout.Vertical}.Layout(gtx, lines...)
							}),
						)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						if appState.HasDifferentSavedSuperpixelSettings() {
							HandleUiButtonClick(appState, gtx, &appState.UseSavedSpSettingsButton)
							return components.DrawIconButton(gtx, &appState.UseSavedSpSettingsButton, appState.Theme, app_theme.IconAvReplay, "Use saved presegmentation settings", false)
						}
						return layout.Dimensions{}
					}),
				)
			}),
		)
	})
}
