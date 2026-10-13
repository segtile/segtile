package main

import (
	"bytes"
	"image"
	"image/png"
	"os"
	"os/user"

	"github.com/dsoprea/go-exif/v3"
	exifcommon "github.com/dsoprea/go-exif/v3/common"
	pngstructure "github.com/dsoprea/go-png-image-structure/v2"
)

func main() {
	// Create empty image
	img := image.NewRGBA(image.Rect(0, 0, 800, 600))
	// Encode the image to PNG with EXIF data
	pngBytes, err := EncodeTIFFWithExif(img)
	if err != nil {
		panic(err)
	}
	// Write the png bytes to a file
	f, _ := os.Create("./test.png")
	defer f.Close()
	f.Write(pngBytes)

}

func getUsername() (string, error) {
	u, err := user.Current()
	if err != nil {
		return "", err
	}
	return u.Name, nil
}

func EncodeTIFFWithExif(img image.Image) ([]byte, error) {
	// 1. Encode the in-memory image to TIFF bytes first.
	var pngBuf bytes.Buffer
	if err := png.Encode(&pngBuf, img); err != nil {
		return nil, err
	}

	// 2. Build the EXIF IFD with go-exif.
	im, err := exifcommon.NewIfdMappingWithStandard()
	if err != nil {
		return nil, err
	}
	ti := exif.NewTagIndex()

	ib := exif.NewIfdBuilder(
		im,
		ti,
		exifcommon.IfdStandardIfdIdentity,
		exifcommon.EncodeDefaultByteOrder, // binary.BigEndian
	)

	// Add whatever tags you need, by standard name:
	if err := ib.AddStandardWithName("ImageDescription", "slic;count=200;compactness=10"); err != nil {
		return nil, err
	}
	if err := ib.AddStandardWithName("Software", "AnnotationTool 1.0"); err != nil {
		return nil, err
	}

	username, err := getUsername()
	if err != nil {
		return nil, err
	}
	if err := ib.AddStandardWithName("Artist", username); err != nil {
		return nil, err
	}

	// Numeric/rational tags, GPS, etc. work the same way, e.g.:
	// ib.AddStandardWithName("ImageWidth", []uint32{uint32(img.Bounds().Dx())})

	// 3. Parse the in-memory PNG bytes into a ChunkSlice.
	pmp := pngstructure.NewPngMediaParser()
	intfc, err := pmp.ParseBytes(pngBuf.Bytes())
	if err != nil {
		return nil, err
	}
	cs := intfc.(*pngstructure.ChunkSlice)

	// 4. Inject/replace the eXIf chunk with our IFD.
	if err := cs.SetExif(ib); err != nil {
		return nil, err
	}

	// 5. Write the final PNG (with EXIF) back out to bytes.
	var out bytes.Buffer
	if err := cs.WriteTo(&out); err != nil {
		return nil, err
	}

	return out.Bytes(), nil
}
