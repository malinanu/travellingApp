package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func main() {
	r2AccountID := os.Getenv("R2_ACCOUNT_ID")
	r2AccessKeyID := os.Getenv("R2_ACCESS_KEY_ID")
	r2SecretAccessKey := os.Getenv("R2_SECRET_ACCESS_KEY")
	r2Bucket := os.Getenv("R2_BUCKET")

	endpoint := fmt.Sprintf("https://%s.r2.cloudflarestorage.com", r2AccountID)
	client := s3.New(s3.Options{
		Region:       "auto",
		BaseEndpoint: aws.String(endpoint),
		Credentials:  credentials.NewStaticCredentialsProvider(r2AccessKeyID, r2SecretAccessKey, ""),
	})

	output, err := client.ListObjectsV2(context.Background(), &s3.ListObjectsV2Input{
		Bucket: aws.String(r2Bucket),
		Prefix: aws.String("vendor/ven_sigiriya"),
	})
	if err != nil {
		log.Fatalf("failed to list objects: %v", err)
	}

	for _, obj := range output.Contents {
		fmt.Printf("Found: %s\n", *obj.Key)
	}
}
