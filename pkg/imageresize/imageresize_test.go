package imageresize_test

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"testing"

	"erdinhrmwn/bangunin/pkg/imageresize"
)

func encodeJPEG(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for x := 0; x < w; x++ {
		for y := 0; y < h; y++ {
			img.Set(x, y, color.RGBA{R: 200, G: 100, B: 50, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, nil); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestResize_DownscalesOversizedImage(t *testing.T) {
	src := encodeJPEG(t, 2000, 1000)

	data, ct, err := imageresize.Resize(bytes.NewReader(src))
	if err != nil {
		t.Fatal(err)
	}
	if ct != "image/jpeg" {
		t.Fatalf("content type = %s, want image/jpeg", ct)
	}

	out, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	b := out.Bounds()
	if b.Dx() != imageresize.MaxDimension || b.Dy() != 800 {
		t.Fatalf("size = %dx%d, want %dx800", b.Dx(), b.Dy(), imageresize.MaxDimension)
	}
}

func TestResize_LeavesSmallImageUnchanged(t *testing.T) {
	src := encodeJPEG(t, 100, 100)

	data, _, err := imageresize.Resize(bytes.NewReader(src))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(data, src) {
		t.Fatal("expected unchanged bytes for image within bounds")
	}
}
