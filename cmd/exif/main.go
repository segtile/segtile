package main

import (
	"bytes"
	"image"
	"image/png"
	"log"
	"os"
	"os/user"
	"sort"

	gometadata "github.com/FlavioCFOliveira/GoMetadata"
	"github.com/FlavioCFOliveira/GoMetadata/exif"
	"github.com/FlavioCFOliveira/GoMetadata/xmp"
)

const (
	annotationDescription = "slic;count=200;compactness=10"
	annotationSoftware    = "AnnotationTool 1.0"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	// Create empty image
	img := image.NewRGBA(image.Rect(0, 0, 800, 600))
	pngBytes, err := EncodeTIFFWithExif(img)
	if err != nil {
		return err
	}
	return os.WriteFile("./test.png", pngBytes, 0o644)
}

func getUsername() (string, error) {
	u, err := user.Current()
	if err != nil {
		return "", err
	}
	return u.Name, nil
}

func EncodeTIFFWithExif(img image.Image) ([]byte, error) {
	// Encode the image first; GoMetadata adds metadata without re-encoding pixels.
	var pngBuf bytes.Buffer
	if err := png.Encode(&pngBuf, img); err != nil {
		return nil, err
	}

	username, err := getUsername()
	if err != nil {
		return nil, err
	}

	metadata, err := gometadata.Read(bytes.NewReader(pngBuf.Bytes()))
	if err != nil {
		return nil, err
	}
	metadata.SetCaption(annotationDescription)
	metadata.SetCreator(username)
	metadata.XMP.Set(xmp.NSxmp, "CreatorTool", annotationSoftware)
	setEXIFSoftware(metadata.EXIF.IFD0, annotationSoftware)

	var out bytes.Buffer
	if err := gometadata.Write(bytes.NewReader(pngBuf.Bytes()), &out, metadata); err != nil {
		return nil, err
	}

	return out.Bytes(), nil
}

func setEXIFSoftware(ifd *exif.IFD, software string) {
	value := append([]byte(software), 0)
	entry := exif.IFDEntry{
		Tag:   exif.TagSoftware,
		Type:  exif.TypeASCII,
		Count: uint32(len(value)),
		Value: value,
	}
	if existing := ifd.Get(exif.TagSoftware); existing != nil {
		*existing = entry
	} else {
		ifd.Entries = append(ifd.Entries, entry)
		sort.Slice(ifd.Entries, func(i, j int) bool {
			return ifd.Entries[i].Tag < ifd.Entries[j].Tag
		})
	}
}
