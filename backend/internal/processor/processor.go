package processor

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/draw"
	_ "image/gif"
	"image/jpeg"
	_ "image/png"
	"io"
	"math"

	"imageprocessor/backend/internal/models"
	"imageprocessor/backend/internal/storage"

	xdraw "golang.org/x/image/draw"
)

type Processor struct {
	store storage.Store
}

func New(store storage.Store) *Processor {
	return &Processor{store: store}
}

func (p *Processor) Handle(ctx context.Context, message models.ProcessMessage) error {
	if err := p.store.UpdateStatus(ctx, message.ID, models.StatusProcessing, ""); err != nil {
		return err
	}

	original, err := p.store.OpenOriginal(ctx, message.ID)
	if err != nil {
		_ = p.store.UpdateStatus(ctx, message.ID, models.StatusFailed, err.Error())
		return err
	}
	defer original.Close()

	processed, err := processImage(original)
	if err != nil {
		_ = p.store.UpdateStatus(ctx, message.ID, models.StatusFailed, err.Error())
		return err
	}

	if err := p.store.SaveProcessed(ctx, message.ID, bytes.NewReader(processed)); err != nil {
		_ = p.store.UpdateStatus(ctx, message.ID, models.StatusFailed, err.Error())
		return err
	}

	return p.store.UpdateStatus(ctx, message.ID, models.StatusDone, "")
}

func processImage(reader io.Reader) ([]byte, error) {
	src, _, err := image.Decode(reader)
	if err != nil {
		return nil, err
	}

	resized := resizeToFit(src, 1200, 1200)
	withWatermark := addWatermark(resized, "ImageProcessor")

	var out bytes.Buffer
	if err := jpeg.Encode(&out, withWatermark, &jpeg.Options{Quality: 88}); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

func resizeToFit(src image.Image, maxWidth int, maxHeight int) image.Image {
	bounds := src.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()
	scale := math.Min(float64(maxWidth)/float64(width), float64(maxHeight)/float64(height))
	if scale >= 1 {
		scale = 1
	}

	dstWidth := int(float64(width) * scale)
	dstHeight := int(float64(height) * scale)
	dst := image.NewRGBA(image.Rect(0, 0, dstWidth, dstHeight))
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), src, bounds, xdraw.Over, nil)
	return dst
}

func addWatermark(src image.Image, text string) image.Image {
	dst := image.NewRGBA(src.Bounds())
	draw.Draw(dst, dst.Bounds(), src, src.Bounds().Min, draw.Src)

	bounds := dst.Bounds()
	rect := image.Rect(bounds.Max.X-230, bounds.Max.Y-54, bounds.Max.X-18, bounds.Max.Y-18)
	draw.Draw(dst, rect, &image.Uniform{C: color.RGBA{A: 120}}, image.Point{}, draw.Over)

	// A compact block watermark keeps the implementation dependency-light.
	for i, r := range text {
		x := rect.Min.X + 10 + i*10
		if x+6 >= rect.Max.X {
			break
		}
		shade := uint8(210 + int(r)%45)
		draw.Draw(dst, image.Rect(x, rect.Min.Y+12, x+6, rect.Max.Y-12), &image.Uniform{C: color.RGBA{R: shade, G: shade, B: shade, A: 220}}, image.Point{}, draw.Over)
	}
	return dst
}
