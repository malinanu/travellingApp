// Package media handles image/video/audio/document uploads to Cloudflare R2 via
// its S3-compatible API. Clients never stream bytes through our server: we
// hand out a short-lived presigned PUT URL, the client uploads straight to
// R2, then calls back to confirm — see internal/http/media.go.
package media

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awshttp "github.com/aws/aws-sdk-go-v2/aws/transport/http"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// Kinds and per-kind limits. Video is generous because R2 stores originals
// (there is no transcoding step) and short promo clips can be large; audio
// more so, since a full audiobook is one file.
const (
	KindImage = "image"
	KindVideo = "video"
	KindDoc   = "doc"
	KindAudio = "audio"

	maxImage = 8 << 20   // 8 MB
	maxVideo = 100 << 20 // 100 MB
	maxDoc   = 15 << 20  // 15 MB
	maxAudio = 200 << 20 // 200 MB
)

// contentTypes maps an allowed MIME type to (file extension, kind).
var contentTypes = map[string]struct {
	ext  string
	kind string
}{
	"image/jpeg":      {".jpg", KindImage},
	"image/png":       {".png", KindImage},
	"image/webp":      {".webp", KindImage},
	"video/mp4":       {".mp4", KindVideo},
	"video/webm":      {".webm", KindVideo},
	"video/quicktime": {".mov", KindVideo},
	"application/pdf": {".pdf", KindDoc},
	"audio/mpeg":      {".mp3", KindAudio},
	"audio/mp3":       {".mp3", KindAudio},
	"audio/mp4":       {".m4a", KindAudio},
	"audio/x-m4a":     {".m4a", KindAudio},
	"audio/aac":       {".aac", KindAudio},
}

// Resolve validates a content type and returns its extension, kind, and the
// max byte size allowed for that kind. ok is false for a disallowed type.
func Resolve(contentType string) (ext, kind string, maxSize int64, ok bool) {
	m, present := contentTypes[strings.ToLower(strings.TrimSpace(contentType))]
	if !present {
		return "", "", 0, false
	}
	switch m.kind {
	case KindVideo:
		maxSize = maxVideo
	case KindDoc:
		maxSize = maxDoc
	case KindAudio:
		maxSize = maxAudio
	default:
		maxSize = maxImage
	}
	return m.ext, m.kind, maxSize, true
}

// Uploader is the surface the HTTP handlers depend on, so tests can stub R2.
type Uploader interface {
	PresignPut(ctx context.Context, key, contentType string, ttl time.Duration) (string, error)
	Head(ctx context.Context, key string) (size int64, contentType string, err error)
	Get(ctx context.Context, key string) (body io.ReadCloser, contentType string, err error)
	// GetRange reads one byte range of an object ("bytes=0-1023", as in an
	// HTTP Range header), or all of it when byteRange is empty — so audio
	// and video players can seek. It returns ErrInvalidRange when the range
	// lies outside the object.
	GetRange(ctx context.Context, key, byteRange string) (Object, error)
	// Put is a server-initiated write, unlike PresignPut (which only ever
	// hands a client a URL to upload to directly) — used for generated
	// derivatives the server creates itself, like a resized thumbnail,
	// where there's no client upload step to presign.
	Put(ctx context.Context, key, contentType string, body io.Reader) error
	Delete(ctx context.Context, key string) error
	PublicURL(key string) string
}

// Object is a read of a stored object (or one range of it). The caller must
// Close Body.
type Object struct {
	Body        io.ReadCloser
	ContentType string
	// Length is the byte count of Body, or -1 when unknown.
	Length int64
	// ContentRange is set ("bytes 0-1023/4096") when only a range was read.
	ContentRange string
}

// ErrInvalidRange means a requested byte range starts past the object's end.
var ErrInvalidRange = errors.New("requested range not satisfiable")

// R2 is the Cloudflare R2 implementation of Uploader.
type R2 struct {
	client  *s3.Client
	presign *s3.PresignClient
	bucket  string
}

// NewR2 builds an R2 client from static credentials. accountID selects the
// R2 endpoint. The bucket is kept private — objects are never served
// straight from R2; see PublicURL and internal/http/media.go's
// handleMediaRaw, which proxies reads through our own API instead.
func NewR2(accountID, accessKey, secretKey, bucket string) *R2 {
	endpoint := fmt.Sprintf("https://%s.r2.cloudflarestorage.com", accountID)
	client := s3.New(s3.Options{
		Region:       "auto",
		BaseEndpoint: aws.String(endpoint),
		Credentials:  credentials.NewStaticCredentialsProvider(accessKey, secretKey, ""),
	})
	return &R2{
		client:  client,
		presign: s3.NewPresignClient(client),
		bucket:  bucket,
	}
}

func (r *R2) PresignPut(ctx context.Context, key, contentType string, ttl time.Duration) (string, error) {
	req, err := r.presign.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(r.bucket),
		Key:         aws.String(key),
		ContentType: aws.String(contentType),
	}, s3.WithPresignExpires(ttl))
	if err != nil {
		return "", err
	}
	return req.URL, nil
}

func (r *R2) Head(ctx context.Context, key string) (int64, string, error) {
	out, err := r.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(r.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return 0, "", err
	}
	var size int64
	if out.ContentLength != nil {
		size = *out.ContentLength
	}
	ct := ""
	if out.ContentType != nil {
		ct = *out.ContentType
	}
	return size, ct, nil
}

// Get streams an object's body straight from R2 — the caller must Close it.
// Used by handleMediaRaw to proxy a read without ever handing the client
// direct (or signed) R2 access.
func (r *R2) Get(ctx context.Context, key string) (io.ReadCloser, string, error) {
	out, err := r.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(r.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, "", err
	}
	ct := ""
	if out.ContentType != nil {
		ct = *out.ContentType
	}
	return out.Body, ct, nil
}

func (r *R2) GetRange(ctx context.Context, key, byteRange string) (Object, error) {
	in := &s3.GetObjectInput{Bucket: aws.String(r.bucket), Key: aws.String(key)}
	if byteRange != "" {
		in.Range = aws.String(byteRange)
	}
	out, err := r.client.GetObject(ctx, in)
	if err != nil {
		var re *awshttp.ResponseError
		if errors.As(err, &re) && re.HTTPStatusCode() == 416 {
			return Object{}, ErrInvalidRange
		}
		return Object{}, err
	}
	obj := Object{Body: out.Body, Length: -1}
	if out.ContentType != nil {
		obj.ContentType = *out.ContentType
	}
	if out.ContentLength != nil {
		obj.Length = *out.ContentLength
	}
	if out.ContentRange != nil {
		obj.ContentRange = *out.ContentRange
	}
	return obj, nil
}

// Put uploads body directly (server-side) — see Uploader.Put's doc comment
// for how this differs from the client-facing PresignPut flow.
func (r *R2) Put(ctx context.Context, key, contentType string, body io.Reader) error {
	_, err := r.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(r.bucket),
		Key:         aws.String(key),
		ContentType: aws.String(contentType),
		Body:        body,
	})
	return err
}

func (r *R2) Delete(ctx context.Context, key string) error {
	_, err := r.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(r.bucket),
		Key:    aws.String(key),
	})
	return err
}

// PublicURL returns the app-relative proxy path for media objects.
func (r *R2) PublicURL(key string) string {
	return "/media/raw/" + key
}
