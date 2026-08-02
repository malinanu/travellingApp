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

// ThumbMaxDim is the longer-side cap for the grid/list-tile thumbnail —
// sized for tiles that use it (roughly 80-120px on screen at up to ~3x
// device pixel ratio), well below the ~1600px originals the client already
// caps picks at.
const ThumbMaxDim = 320

// MicroMaxDim is the longer-side cap for the tiny blur-up placeholder —
// small and heavily compressed enough (see microJPEGQuality) to inline as
// base64 directly in an API response with no extra network fetch.
const MicroMaxDim = 24

const microJPEGQuality = 40

// Thumbnails holds the two derivatives GenerateThumbnails produces from one
// decode of the source image.
type Thumbnails struct {
	Thumb []byte // ~320px JPEG, quality 75 — grid/list tiles
	Micro []byte // ~24px JPEG, quality 40 — inline base64 blur placeholder
}

// GenerateThumbnails decodes an image (jpeg/png/webp — the three types the
// upload validator allows, see Resolve) once and re-encodes two smaller
// JPEG derivatives from it, regardless of the source format, so the output
// is always predictable, well-compressed types. Never upscales a source
// already smaller than a given derivative's max dimension.
func GenerateThumbnails(r io.Reader) (*Thumbnails, error) {
	src, _, err := image.Decode(r)
	if err != nil {
		return nil, err
	}
	bounds := src.Bounds()
	if bounds.Dx() <= 0 || bounds.Dy() <= 0 {
		return nil, errors.New("empty image")
	}

	thumb, err := scaleAndEncode(src, bounds, ThumbMaxDim, 75)
	if err != nil {
		return nil, err
	}
	micro, err := scaleAndEncode(src, bounds, MicroMaxDim, microJPEGQuality)
	if err != nil {
		return nil, err
	}
	return &Thumbnails{Thumb: thumb, Micro: micro}, nil
}

// scaleAndEncode resizes src (whose bounds are passed in, since Bounds() is
// only cheap to call once) to fit within maxDim on its longer side and
// re-encodes as JPEG at the given quality.
func scaleAndEncode(src image.Image, bounds image.Rectangle, maxDim int, quality int) ([]byte, error) {
	w, h := bounds.Dx(), bounds.Dy()
	scale := float64(maxDim) / float64(max(w, h))
	if scale > 1 {
		scale = 1
	}
	dstW := max(1, int(float64(w)*scale))
	dstH := max(1, int(float64(h)*scale))

	dst := image.NewRGBA(image.Rect(0, 0, dstW, dstH))
	draw.CatmullRom.Scale(dst, dst.Bounds(), src, bounds, draw.Over, nil)

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, dst, &jpeg.Options{Quality: quality}); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
