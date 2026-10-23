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

// drawSidebarSection draws a collapsible sidebar section header followed by its body when open.
func drawSidebarSection(gtx layout.Context, appState *core.AppState, title string, open bool, button *widget.Clickable, body func(layout.Context) layout.Dimensions) layout.Dimensions {
	HandleUiButtonClick(appState, gtx, button)

	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return components.DrawSidebarSectionHeader(gtx, button, appState.Theme, title, open)
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
	return drawSidebarSection(gtx, appState, "Images", appState.ImagesSection.Open, &appState.ImagesSection.Button, func(gtx layout.Context) layout.Dimensions {
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
	return drawSidebarSection(gtx, appState, "Superpixel", appState.SuperpixelSection.Open, &appState.SuperpixelSection.Button, func(gtx layout.Context) layout.Dimensions {
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

								label := material.Body1(&appState.Theme.GioTheme, superpixelTitle)
								label.Color = appState.Theme.GioTheme.Palette.Fg
								return label.Layout(gtx)
							}),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return components.DrawIconButton(gtx, &appState.ChangeSpAlgorithmButton, appState.Theme, app_theme.IconChangeAlgorithm, "Change Superpixel Algorithm", false)
							}),
						)
					}),
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
				title := appState.SuperpixelParameterEditors[i].Title
				editor := &appState.SuperpixelParameterEditors[i].Editor
				checkbox := &appState.SuperpixelParameterEditors[i].OptimizeCheckbox
				children = append(children,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Axis: layout.Horizontal, Alignment: layout.End}.Layout(gtx,
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								checkboxStyle := material.CheckBox(&appState.Theme.GioTheme, checkbox, "")
								checkboxStyle.IconColor = appState.Theme.GioTheme.Palette.Fg
								checkboxStyle.Size = unit.Dp(16)
								return checkboxStyle.Layout(gtx)
							}),
							layout.Flexed(3, func(gtx layout.Context) layout.Dimensions {
								label := material.Body1(&appState.Theme.GioTheme, title)
								label.Color = appState.Theme.GioTheme.Palette.Fg
								return label.Layout(gtx)
							}),

							layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
								editorStyle := material.Editor(&appState.Theme.GioTheme, editor, "value")
								editorStyle.Color = appState.Theme.GioTheme.Palette.Fg
								return editorStyle.Layout(gtx)
							}),
						)
					}),
				)
			}

			// Optimize button
			children = append(children,
				layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
			)
			children = append(children,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Horizontal, Alignment: layout.End}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							HandleUiButtonClick(appState, gtx, &appState.OptimizeButton)
							return components.DrawSubtleButton(gtx, &appState.OptimizeButton, appState.Theme, "Suggest Parameters", 14)
						}),
					)
				}),
			)
		}

		return layout.Flex{Axis: layout.Vertical, Spacing: layout.SpaceEnd}.Layout(gtx, children...)
	})
}

func drawSelectionSection(gtx layout.Context, appState *core.AppState) layout.Dimensions {
	return drawSidebarSection(gtx, appState, "Selection", appState.MiscSection.Open, &appState.MiscSection.Button, func(gtx layout.Context) layout.Dimensions {
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

	return drawSidebarSection(gtx, appState, title, appState.SavedImageInfoSection.Open, &appState.SavedImageInfoSection.Button, func(gtx layout.Context) layout.Dimensions {
		if appState.ActiveImageMetadata == nil {
			return material.Caption(&appState.Theme.GioTheme, "Annotation information will show up here once the annotation is saved to disk.").Layout(gtx)
		}

		annotator := appState.ActiveImageMetadata.Annotator
		if annotator == "" {
			annotator = "Unknown"
		}

		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return material.Caption(&appState.Theme.GioTheme, "Annotator:").Layout(gtx)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return material.Body2(&appState.Theme.GioTheme, annotator).Layout(gtx)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						return material.Body2(&appState.Theme.GioTheme, appState.ActiveImageMetadata.AnnotationTime.Format(time.RFC1123)).Layout(gtx)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						if appState.SelectionMaskDiffersVersionOnDisk {
							HandleUiButtonClick(appState, gtx, &appState.RevertSavedMaskButton)
							return components.DrawIconButton(gtx, &appState.RevertSavedMaskButton, appState.Theme, app_theme.IconAvReplay, "Revert to saved annotation", false)
						}
						return layout.Dimensions{}
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

				// Sort the parameter keys
				var keys []string
				for k := range params {
					keys = append(keys, k)
				}
				sort.Strings(keys)

				return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
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
				)
			}),
		)
	})
}
