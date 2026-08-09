package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
)

func main() {
	key := os.Getenv("GEMINI_API_KEY")
	if key == "" {
		fmt.Println("No key")
		return
	}

	body := []byte(`{"model": "text-embedding-004", "input": ["hello"]}`)
	req, _ := http.NewRequest("POST", "https://generativelanguage.googleapis.com/v1beta/openai/embeddings", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	// Try Bearer
	req.Header.Set("Authorization", "Bearer "+key)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	defer res.Body.Close()
	b, _ := io.ReadAll(res.Body)
	fmt.Printf("Bearer Status: %d\nBody: %s\n", res.StatusCode, string(b))
}
