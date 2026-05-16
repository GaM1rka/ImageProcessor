package processor

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"testing"
)

func TestProcessImageCreatesProcessedImageAndThumbnail(t *testing.T) {
	t.Parallel()

	var input bytes.Buffer
	src := image.NewRGBA(image.Rect(0, 0, 640, 420))
	for y := 0; y < src.Bounds().Dy(); y++ {
		for x := 0; x < src.Bounds().Dx(); x++ {
			src.Set(x, y, color.RGBA{R: uint8(x % 255), G: uint8(y % 255), B: 120, A: 255})
		}
	}
	if err := png.Encode(&input, src); err != nil {
		t.Fatalf("encode input: %v", err)
	}

	result, err := processImage(bytes.NewReader(input.Bytes()))
	if err != nil {
		t.Fatalf("processImage: %v", err)
	}
	if len(result.Processed) == 0 {
		t.Fatal("processed image is empty")
	}
	if len(result.Thumbnail) == 0 {
		t.Fatal("thumbnail is empty")
	}

	processed, err := jpeg.Decode(bytes.NewReader(result.Processed))
	if err != nil {
		t.Fatalf("decode processed image: %v", err)
	}
	if processed.Bounds().Dx() > 1200 || processed.Bounds().Dy() > 1200 {
		t.Fatalf("processed image exceeds max size: %v", processed.Bounds())
	}

	thumbnail, err := jpeg.Decode(bytes.NewReader(result.Thumbnail))
	if err != nil {
		t.Fatalf("decode thumbnail: %v", err)
	}
	if thumbnail.Bounds().Dx() != 320 || thumbnail.Bounds().Dy() != 240 {
		t.Fatalf("unexpected thumbnail size: %v", thumbnail.Bounds())
	}
}
