package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spotto/server/internal/media"
)

type PhotoResult struct {
	Role        string `json:"role"`
	ObjectKey   string `json:"object_key"`
	URL         string `json:"url"`
	ThumbKey    string `json:"thumb_key"`
	MicroThumb  string `json:"micro_thumb"`
	Width       int    `json:"width"`
	Height      int    `json:"height"`
	Size        int    `json:"size"`
	ContentType string `json:"content_type"`
}

type Place struct {
	ID   string          `json:"id"`
	Name string          `json:"name"`
	Data json.RawMessage `json:"data"`
}

func vendorIDFor(placeID string) string {
	id := "ven_" + placeID
	if len(id) > 50 {
		id = id[:50]
	}
	return id
}

func randomID(prefix string) string {
	b := make([]byte, 6)
	_, _ = rand.Read(b)
	return prefix + "_" + hex.EncodeToString(b)
}

func main() {
	if len(os.Args) < 2 {
		log.Fatal("Usage: local_r2_uploader <path-to-json>")
	}
	jsonPath := os.Args[1]

	data, err := os.ReadFile(jsonPath)
	if err != nil {
		log.Fatalf("failed to read json: %v", err)
	}

	var spots []Place
	if err := json.Unmarshal(data, &spots); err != nil {
		log.Fatalf("failed to parse json: %v", err)
	}

	ctx := context.Background()

	// Hardcode or load env
	r2AccountID := os.Getenv("R2_ACCOUNT_ID")
	r2AccessKeyID := os.Getenv("R2_ACCESS_KEY_ID")
	r2SecretAccessKey := os.Getenv("R2_SECRET_ACCESS_KEY")
	r2Bucket := os.Getenv("R2_BUCKET")

	if r2AccountID == "" {
		log.Fatal("Missing R2 environment variables")
	}

	r2 := media.NewR2(r2AccountID, r2AccessKeyID, r2SecretAccessKey, r2Bucket)
	client := &http.Client{Timeout: 20 * time.Second}

	for i := range spots {
		var pd map[string]interface{}
		if err := json.Unmarshal(spots[i].Data, &pd); err != nil {
			continue
		}

		sourcePhotosInterface, ok := pd["sourcePhotos"].([]interface{})
		if !ok || len(sourcePhotosInterface) == 0 {
			continue
		}

		var sourcePhotos []string
		for _, sp := range sourcePhotosInterface {
			if s, ok := sp.(string); ok {
				sourcePhotos = append(sourcePhotos, s)
			}
		}

		vendorID := vendorIDFor(spots[i].ID)
		var uploadedPhotos []PhotoResult

		n := len(sourcePhotos)
		if n > 5 {
			n = 5
		}

		for j := 0; j < n; j++ {
			role := "gallery"
			if j == 0 {
				role = "cover"
			}
			res, uploadErr := uploadOnePhoto(ctx, client, r2, vendorID, role, sourcePhotos[j], jsonPath)
			if uploadErr != nil {
				log.Printf("photo %d for %s: %v", j, vendorID, uploadErr)
				continue
			}
			uploadedPhotos = append(uploadedPhotos, res)
		}

		pd["uploadedPhotos"] = uploadedPhotos

		newData, _ := json.Marshal(pd)
		spots[i].Data = newData
		log.Printf("Processed %d/%d: %s (uploaded %d photos)", i+1, len(spots), spots[i].ID, len(uploadedPhotos))
	}

	outPath := strings.Replace(jsonPath, "_formatted.json", "_r2_ready.json", 1)
	outBytes, _ := json.MarshalIndent(spots, "", "  ")
	os.WriteFile(outPath, outBytes, 0644)
	log.Printf("Done! Wrote results to %s", outPath)
}

func uploadOnePhoto(ctx context.Context, client *http.Client, r2 *media.R2, vendorID, role, sourceURL, jsonPath string) (PhotoResult, error) {
	baseDir := filepath.Dir(jsonPath)
	if strings.HasPrefix(sourceURL, "/") {
		sourceURL = sourceURL[1:]
	}
	path := filepath.Join(baseDir, sourceURL)
	body, err := os.ReadFile(path)
	if err != nil {
		return PhotoResult{}, err
	}

	ext := strings.ToLower(filepath.Ext(path))
	contentType := "image/jpeg"
	if ext == ".png" {
		contentType = "image/png"
	} else if ext == ".webp" {
		contentType = "image/webp"
	}

	_, kind, maxSize, ok := media.Resolve(contentType)
	if !ok || int64(len(body)) > maxSize {
		return PhotoResult{}, fmt.Errorf("rejected (type=%s size=%d)", contentType, len(body))
	}
	_ = kind

	thumbs, err := media.GenerateThumbnails(bytes.NewReader(body))
	if err != nil {
		return PhotoResult{}, fmt.Errorf("thumbnails: %w", err)
	}

	srcImg, _, err := image.DecodeConfig(bytes.NewReader(body))
	width, height := 0, 0
	if err == nil {
		width, height = srcImg.Width, srcImg.Height
	}

	key := fmt.Sprintf("vendor/%s/%s%s", vendorID, randomID("m"), ext)
	if err := putToR2(ctx, client, r2, key, contentType, body); err != nil {
		return PhotoResult{}, fmt.Errorf("upload to R2: %w", err)
	}

	thumbKey := fmt.Sprintf("vendor/%s/%s_thumb.jpg", vendorID, randomID("m"))
	if err := putToR2(ctx, client, r2, thumbKey, "image/jpeg", thumbs.Thumb); err != nil {
		return PhotoResult{}, fmt.Errorf("upload thumb to R2: %w", err)
	}

	microBase64 := "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(thumbs.Micro)
	url := r2.PublicURL(key)

	return PhotoResult{
		Role:        role,
		ObjectKey:   key,
		URL:         url,
		ThumbKey:    thumbKey,
		MicroThumb:  microBase64,
		Width:       width,
		Height:      height,
		Size:        len(body),
		ContentType: contentType,
	}, nil
}

func putToR2(ctx context.Context, client *http.Client, r2 *media.R2, key, contentType string, body []byte) error {
	putURL, err := r2.PresignPut(ctx, key, contentType, 2*time.Minute)
	if err != nil {
		return fmt.Errorf("presign: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, putURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", contentType)
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("unexpected status %d", resp.StatusCode)
	}
	return nil
}
