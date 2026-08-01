package media

import (
	"bytes"
	"errors"
	"image"
	"image/jpeg"
	_ "image/png" // register PNG decoding for image.Decode
	"io"

	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp" // register WebP decoding for image.Decode
)

// ThumbMaxDim is the longer-side cap for a generated thumbnail — sized for
// the grid/list tiles that use it (roughly 80-120px on screen at up to ~3x
// device pixel ratio), well below the ~1600px originals the client already
// caps picks at.
const ThumbMaxDim = 320

// GenerateThumbnail decodes an image (jpeg/png/webp — the three types the
// upload validator allows, see Resolve) and re-encodes a smaller JPEG
// derivative regardless of the source format, so the output is always one
// predictable, well-compressed type. Never upscales a source already
// smaller than ThumbMaxDim.
func GenerateThumbnail(r io.Reader) ([]byte, error) {
	src, _, err := image.Decode(r)
	if err != nil {
		return nil, err
	}
	bounds := src.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	if w <= 0 || h <= 0 {
		return nil, errors.New("empty image")
	}

	scale := float64(ThumbMaxDim) / float64(max(w, h))
	if scale > 1 {
		scale = 1
	}
	dstW := max(1, int(float64(w)*scale))
	dstH := max(1, int(float64(h)*scale))

	dst := image.NewRGBA(image.Rect(0, 0, dstW, dstH))
	draw.CatmullRom.Scale(dst, dst.Bounds(), src, bounds, draw.Over, nil)

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, dst, &jpeg.Options{Quality: 75}); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
