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

	result, err := processImage(original)
	if err != nil {
		_ = p.store.UpdateStatus(ctx, message.ID, models.StatusFailed, err.Error())
		return err
	}

	if err := p.store.SaveProcessed(ctx, message.ID, bytes.NewReader(result.Processed)); err != nil {
		_ = p.store.UpdateStatus(ctx, message.ID, models.StatusFailed, err.Error())
		return err
	}
	if err := p.store.SaveThumbnail(ctx, message.ID, bytes.NewReader(result.Thumbnail)); err != nil {
		_ = p.store.UpdateStatus(ctx, message.ID, models.StatusFailed, err.Error())
		return err
	}

	return p.store.UpdateStatus(ctx, message.ID, models.StatusDone, "")
}

type result struct {
	Processed []byte
	Thumbnail []byte
}

func processImage(reader io.Reader) (result, error) {
	src, _, err := image.Decode(reader)
	if err != nil {
		return result{}, err
	}

	resized := resizeToFit(src, 1200, 1200)
	withWatermark := addWatermark(resized, "ImageProcessor")
	thumbnail := resizeToFill(src, 320, 240)

	processedBytes, err := encodeJPEG(withWatermark, 88)
	if err != nil {
		return result{}, err
	}
	thumbnailBytes, err := encodeJPEG(thumbnail, 82)
	if err != nil {
		return result{}, err
	}
	return result{Processed: processedBytes, Thumbnail: thumbnailBytes}, nil
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

func resizeToFill(src image.Image, width int, height int) image.Image {
	bounds := src.Bounds()
	scale := math.Max(float64(width)/float64(bounds.Dx()), float64(height)/float64(bounds.Dy()))
	scaledWidth := int(float64(bounds.Dx()) * scale)
	scaledHeight := int(float64(bounds.Dy()) * scale)

	scaled := image.NewRGBA(image.Rect(0, 0, scaledWidth, scaledHeight))
	xdraw.CatmullRom.Scale(scaled, scaled.Bounds(), src, bounds, xdraw.Over, nil)

	dst := image.NewRGBA(image.Rect(0, 0, width, height))
	offset := image.Point{X: (scaledWidth - width) / 2, Y: (scaledHeight - height) / 2}
	draw.Draw(dst, dst.Bounds(), scaled, offset, draw.Src)
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

func encodeJPEG(src image.Image, quality int) ([]byte, error) {
	var out bytes.Buffer
	if err := jpeg.Encode(&out, src, &jpeg.Options{Quality: quality}); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}
