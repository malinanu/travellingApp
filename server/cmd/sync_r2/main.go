package main

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/spotto/server/internal/media"
)

type Place struct {
	ID   string
	Data json.RawMessage
}

func main() {
	if len(os.Args) < 2 {
		log.Fatal("Usage: sync_r2 <path-to-json>")
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

	r2AccountID := os.Getenv("R2_ACCOUNT_ID")
	r2AccessKeyID := os.Getenv("R2_ACCESS_KEY_ID")
	r2SecretAccessKey := os.Getenv("R2_SECRET_ACCESS_KEY")
	r2Bucket := os.Getenv("R2_BUCKET")

	r2 := media.NewR2(r2AccountID, r2AccessKeyID, r2SecretAccessKey, r2Bucket)
	ctx := context.Background()
	baseDir := filepath.Dir(jsonPath)

	for i := range spots {
		var pd map[string]interface{}
		if err := json.Unmarshal(spots[i].Data, &pd); err != nil {
			continue
		}

		sourcePhotosInterface, ok := pd["sourcePhotos"].([]interface{})
		if !ok || len(sourcePhotosInterface) == 0 {
			continue
		}

		uploadedPhotosInterface, ok := pd["uploadedPhotos"].([]interface{})
		if !ok || len(uploadedPhotosInterface) == 0 {
			continue
		}

		for j := 0; j < len(uploadedPhotosInterface) && j < len(sourcePhotosInterface); j++ {
			uPhoto := uploadedPhotosInterface[j].(map[string]interface{})
			sourceURL := sourcePhotosInterface[j].(string)

			objectKey := uPhoto["object_key"].(string)
			thumbKey := uPhoto["thumb_key"].(string)

			if strings.HasPrefix(sourceURL, "/") {
				sourceURL = sourceURL[1:]
			}
			path := filepath.Join(baseDir, sourceURL)
			body, err := os.ReadFile(path)
			if err != nil {
				log.Printf("Failed to read file %s: %v", path, err)
				continue
			}

			ext := strings.ToLower(filepath.Ext(path))
			contentType := "image/jpeg"
			if ext == ".png" {
				contentType = "image/png"
			} else if ext == ".webp" {
				contentType = "image/webp"
			}

			thumbs, err := media.GenerateThumbnails(bytes.NewReader(body))
			if err != nil {
				log.Printf("Failed to generate thumbs for %s: %v", path, err)
				continue
			}

			err = r2.Put(ctx, objectKey, contentType, bytes.NewReader(body))
			if err != nil {
				log.Printf("Failed to upload main image %s: %v", objectKey, err)
			}

			err = r2.Put(ctx, thumbKey, "image/jpeg", bytes.NewReader(thumbs.Thumb))
			if err != nil {
				log.Printf("Failed to upload thumb %s: %v", thumbKey, err)
			}
		}
		log.Printf("Processed %d/%d: %s", i+1, len(spots), spots[i].ID)
	}
	log.Printf("Done uploading existing keys to R2!")
}
