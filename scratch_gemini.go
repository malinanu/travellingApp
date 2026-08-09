package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
)

func main() {
	body := []byte(`{"requests": [{"model": "models/text-embedding-004", "content": {"parts": [{"text": "hello"}]}}]}`)
	req, _ := http.NewRequest("POST", "https://generativelanguage.googleapis.com/v1beta/models/text-embedding-004:batchEmbedContents", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-goog-api-key", "fake-key")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	defer res.Body.Close()
	b, _ := io.ReadAll(res.Body)
	fmt.Printf("Status: %d\nBody: %s\n", res.StatusCode, string(b))
}
