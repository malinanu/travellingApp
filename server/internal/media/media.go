// Package media handles image/video/document uploads to Cloudflare R2 via
// its S3-compatible API. Clients never stream bytes through our server: we
// hand out a short-lived presigned PUT URL, the client uploads straight to
// R2, then calls back to confirm — see internal/http/media.go.
package media

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// Kinds and per-kind limits. Video is generous because R2 stores originals
// (there is no transcoding step) and short promo clips can be large.
const (
	KindImage = "image"
	KindVideo = "video"
	KindDoc   = "doc"

	maxImage = 8 << 20   // 8 MB
	maxVideo = 100 << 20 // 100 MB
	maxDoc   = 15 << 20  // 15 MB
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
	default:
		maxSize = maxImage
	}
	return m.ext, m.kind, maxSize, true
}

// Uploader is the surface the HTTP handlers depend on, so tests can stub R2.
type Uploader interface {
	PresignPut(ctx context.Context, key, contentType string, ttl time.Duration) (string, error)
	Head(ctx context.Context, key string) (size int64, contentType string, err error)
	Delete(ctx context.Context, key string) error
	PublicURL(key string) string
}

// R2 is the Cloudflare R2 implementation of Uploader.
type R2 struct {
	client     *s3.Client
	presign    *s3.PresignClient
	bucket     string
	publicBase string
}

// NewR2 builds an R2 client from static credentials. accountID selects the
// R2 endpoint; publicBase is the public bucket / custom-domain URL objects
// are served from.
func NewR2(accountID, accessKey, secretKey, bucket, publicBase string) *R2 {
	endpoint := fmt.Sprintf("https://%s.r2.cloudflarestorage.com", accountID)
	client := s3.New(s3.Options{
		Region:       "auto",
		BaseEndpoint: aws.String(endpoint),
		Credentials:  credentials.NewStaticCredentialsProvider(accessKey, secretKey, ""),
	})
	return &R2{
		client:     client,
		presign:    s3.NewPresignClient(client),
		bucket:     bucket,
		publicBase: strings.TrimRight(publicBase, "/"),
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

func (r *R2) Delete(ctx context.Context, key string) error {
	_, err := r.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(r.bucket),
		Key:    aws.String(key),
	})
	return err
}

func (r *R2) PublicURL(key string) string {
	return r.publicBase + "/" + key
}
