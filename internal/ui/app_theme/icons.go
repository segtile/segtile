package app_theme

import (
	"gioui.org/widget"
	"golang.org/x/exp/shiny/materialdesign/icons"
)

// golang.org/x/exp/shiny/materialdesign/icons does not provide a "play reversed arrow" icon, so we define it ourselves by mirroring the existing play arrow icon.
var aVPlayReverseArrow = []byte{
	0x89, 0x49, 0x56, 0x47, 0x02, 0x0a, 0x00, 0x50, 0x50, 0xb0, 0xb0, 0xc0, 0x90, 0x64, 0xe9, 0xb8,
	0x20, 0x54, 0x64, 0xe1,
}

var (
	IconFirst           = mustIcon(widget.NewIcon(icons.AVSkipPrevious))
	IconPrevUnlabeled   = mustIcon(widget.NewIcon(icons.AVFastRewind))
	IconPrev            = mustIcon(widget.NewIcon(aVPlayReverseArrow))
	IconNext            = mustIcon(widget.NewIcon(icons.AVPlayArrow))
	IconNextUnlabeled   = mustIcon(widget.NewIcon(icons.AVFastForward))
	IconLast            = mustIcon(widget.NewIcon(icons.AVSkipNext))
	IconNew             = mustIcon(widget.NewIcon(icons.ContentAdd))
	IconOpen            = mustIcon(widget.NewIcon(icons.FileFolderOpen))
	IconOptimize        = mustIcon(widget.NewIcon(icons.ImageTune))
	IconSave            = mustIcon(widget.NewIcon(icons.ContentSave))
	IconChangeAlgorithm = mustIcon(widget.NewIcon(icons.ActionAutorenew))
	IconContentClear    = mustIcon(widget.NewIcon(icons.ContentClear))
	IconAvReplay        = mustIcon(widget.NewIcon(icons.AVReplay))
	IconLogoLight       = mustIcon(widget.NewIcon(iconLogoLightData))
	IconSocialPerson    = mustIcon(widget.NewIcon(icons.SocialPerson))
	IconActionDateRange = mustIcon(widget.NewIcon(icons.ActionDateRange))
)

func mustIcon(icon *widget.Icon, err error) *widget.Icon {
	if err != nil {
		panic(err)
	}
	return icon
}
