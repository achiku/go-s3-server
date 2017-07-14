package main

import (
	"fmt"
	"io"
	"log"
	"path"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/endpoints"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

// S3Client s3 client
// https://docs.aws.amazon.com/sdk-for-go/api/service/s3/
type S3Client struct {
	Bucket   string
	BaseKey  string
	sess     *session.Session
	uploader *s3manager.Uploader
}

// NewS3Client new s3 client
func NewS3Client(bkt string) (*S3Client, error) {
	sess := session.Must(session.NewSession(&aws.Config{
		Region: aws.String(endpoints.ApNortheast1RegionID),
	}))
	uploader := s3manager.NewUploader(sess)
	return &S3Client{
		Bucket:   bkt,
		BaseKey:  "dev/image",
		sess:     sess,
		uploader: uploader,
	}, nil
}

// Upload upload file
func (c *S3Client) Upload(in io.ReadSeeker, p []string) (string, error) {
	pt := append([]string{c.BaseKey}, p...)
	fpth := fmt.Sprintf("%s", path.Join(pt...))
	result, err := c.uploader.Upload(&s3manager.UploadInput{
		Bucket:      aws.String(c.Bucket),
		Key:         aws.String(fpth),
		ContentType: aws.String("image/jpeg"),
		Body:        in,
	})
	req, _ := c.uploader.S3.GetObjectRequest(&s3.GetObjectInput{
		Bucket: aws.String(c.Bucket),
		Key:    aws.String(fpth),
	})
	str, err := req.Presign(5 * time.Minute)
	if err != nil {
		return "", err
	}
	log.Printf("%s", str)
	log.Printf("%s", result.Location)
	return str, nil
}

// BasePath base path
func (c *S3Client) BasePath() string {
	return "basepath"
}
