package main

import (
	"bytes"
	"image"
	"image/png"
	"strings"
	"testing"

	gometadata "github.com/FlavioCFOliveira/GoMetadata"
	"github.com/FlavioCFOliveira/GoMetadata/exif"
	"github.com/FlavioCFOliveira/GoMetadata/xmp"
)

func TestEncodeTIFFWithExifWritesMetadata(t *testing.T) {
	encoded, err := EncodeTIFFWithExif(image.NewRGBA(image.Rect(0, 0, 1, 1)))
	if err != nil {
		t.Fatalf("EncodeTIFFWithExif() error = %v", err)
	}
	if _, err := png.Decode(bytes.NewReader(encoded)); err != nil {
		t.Fatalf("encoded PNG is invalid: %v", err)
	}

	metadata, err := gometadata.Read(bytes.NewReader(encoded))
	if err != nil {
		t.Fatalf("read encoded PNG metadata: %v", err)
	}
	if metadata.EXIF == nil || metadata.EXIF.IFD0 == nil {
		t.Fatal("encoded PNG has no EXIF IFD0")
	}
	if metadata.XMP == nil {
		t.Fatal("encoded PNG has no XMP metadata")
	}

	username := getTestUsername(t)
	assertEXIFValue := func(tag exif.TagID, want string) {
		t.Helper()
		entry := metadata.EXIF.IFD0.Get(tag)
		if entry == nil {
			t.Errorf("EXIF tag %#x is missing", tag)
			return
		}
		if got := strings.TrimRight(entry.String(), "\x00"); got != want {
			t.Errorf("EXIF tag %#x = %q, want %q", tag, got, want)
		}
	}
	assertEXIFValue(exif.TagImageDescription, annotationDescription)
	assertEXIFValue(exif.TagArtist, username)
	assertEXIFValue(exif.TagSoftware, annotationSoftware)

	if got := metadata.XMP.Get(xmp.NSdc, "description"); got != annotationDescription {
		t.Errorf("XMP description = %q, want %q", got, annotationDescription)
	}
	if got := metadata.XMP.Get(xmp.NSdc, "creator"); got != username {
		t.Errorf("XMP creator = %q, want %q", got, username)
	}
	if got := metadata.XMP.Get(xmp.NSxmp, "CreatorTool"); got != annotationSoftware {
		t.Errorf("XMP CreatorTool = %q, want %q", got, annotationSoftware)
	}
}

func getTestUsername(t *testing.T) string {
	t.Helper()
	username, err := getUsername()
	if err != nil {
		t.Fatalf("getUsername() error = %v", err)
	}
	return username
}
