// Package imageresize downscales images that exceed a max dimension,
// re-encoding in their original format. Used by the media:process worker job.
package imageresize

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"

	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp" // register webp decoder (no encoder — re-encoded as jpeg)
)

const MaxDimension = 1600

// Resize decodes r, downscales it if either dimension exceeds MaxDimension
// (preserving aspect ratio), and returns the re-encoded bytes plus content
// type. If the image is already within bounds, it returns the original
// bytes unchanged.
func Resize(r io.Reader) (data []byte, contentType string, err error) {
	raw, err := io.ReadAll(r)
	if err != nil {
		return nil, "", fmt.Errorf("imageresize: read: %w", err)
	}

	src, format, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return nil, "", fmt.Errorf("imageresize: decode: %w", err)
	}

	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	if w <= MaxDimension && h <= MaxDimension {
		return raw, contentTypeFor(format), nil
	}

	scale := float64(MaxDimension) / float64(max(w, h))
	nw, nh := int(float64(w)*scale), int(float64(h)*scale)

	dst := image.NewRGBA(image.Rect(0, 0, nw, nh))
	draw.CatmullRom.Scale(dst, dst.Bounds(), src, b, draw.Over, nil)

	var buf bytes.Buffer
	switch format {
	case "png":
		err = png.Encode(&buf, dst)
	default:
		err = jpeg.Encode(&buf, dst, &jpeg.Options{Quality: 85})
		format = "jpeg"
	}
	if err != nil {
		return nil, "", fmt.Errorf("imageresize: encode: %w", err)
	}
	return buf.Bytes(), contentTypeFor(format), nil
}

func contentTypeFor(format string) string {
	switch format {
	case "png":
		return "image/png"
	case "webp":
		return "image/webp"
	default:
		return "image/jpeg"
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
