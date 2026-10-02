package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
)

func main() {
	apiKey := os.Getenv("GEMINI_API_KEY")
	body := []byte(`{"model":"gemini-2.5-flash","messages":[{"role":"user","content":"Hi"}]}`)
	req, _ := http.NewRequest("POST", "https://generativelanguage.googleapis.com/v1beta/openai/chat/completions", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Println("Err:", err)
		return
	}
	defer res.Body.Close()
	fmt.Println("Status:", res.StatusCode)
	b, _ := io.ReadAll(res.Body)
	fmt.Println("Body:", string(b))
}
