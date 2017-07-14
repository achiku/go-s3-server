package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/endpoints"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	uuid "github.com/satori/go.uuid"
	"github.com/vincent-petithory/dataurl"
)

// Uploader uploader
type Uploader interface {
	Upload(io.Reader, []string) (string, error)
	BasePath() string
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

// S3Client s3 client
// https://docs.aws.amazon.com/sdk-for-go/api/service/s3/
type S3Client struct {
	Bucket   string
	BaseKey  string
	sess     *session.Session
	uploader *s3manager.Uploader
}

// Upload upload file
func (c *S3Client) Upload(in io.Reader, p []string) (string, error) {
	pt := append([]string{c.BaseKey}, p...)
	fpth := fmt.Sprintf("%s", path.Join(pt...))
	result, err := c.uploader.Upload(&s3manager.UploadInput{
		Bucket: &c.Bucket,
		Key:    &fpth,
		Body:   in,
	})
	if err != nil {
		return "", err
	}
	return result.Location, nil
}

// BasePath base path
func (c *S3Client) BasePath() string {
	return "basepath"
}

// NewFSClient new file system client
func NewFSClient() (*FSClient, error) {
	ex, err := os.Executable()
	if err != nil {
		return nil, err
	}
	imgPath := path.Join(path.Dir(ex), "image")
	return &FSClient{
		BaseDir:  imgPath,
		Endpoint: "http://localhost:8080/static",
	}, nil
}

// FSClient file system client
type FSClient struct {
	BaseDir  string
	Endpoint string
}

// Upload upload file
func (c *FSClient) Upload(in io.Reader, p []string) (string, error) {
	pt := append([]string{c.BaseDir}, p...)
	fpth := fmt.Sprintf("%s", path.Join(pt...))
	url := append([]string{c.Endpoint}, p...)
	furl := fmt.Sprintf("%s", path.Join(url...))
	log.Printf("%s", fpth)

	f, err := os.Create(fpth)
	if err != nil {
		return "", err
	}
	defer f.Close()
	if _, err := io.Copy(f, in); err != nil {
		return "", err
	}
	return furl, nil
}

// BasePath base path
func (c *FSClient) BasePath() string {
	return c.BaseDir
}

// App app
type App struct {
	Client Uploader
	Logger *log.Logger
}

// FileUploadRequest file upload
type FileUploadRequest struct {
	Content string `json:"content"`
}

// FileUploadResponse file upload
type FileUploadResponse struct {
	ID         string    `json:"id"`
	URL        string    `json:"url"`
	UploadedAt time.Time `json:"uploadedAt"`
}

func (app *App) uploadFile(w http.ResponseWriter, r *http.Request) {
	uuid := uuid.NewV4()
	decoder := json.NewDecoder(r.Body)
	var req FileUploadRequest
	if err := decoder.Decode(&req); err != nil {
		app.Logger.Printf("%s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	dataURL, err := dataurl.DecodeString(req.Content)
	if err != nil {
		app.Logger.Printf("%s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	log.Printf("%s: %s: %s", dataURL.ContentType(), dataURL.MediaType.Type, dataURL.MediaType.Subtype)
	fname := fmt.Sprintf("%s.%s", uuid.String(), dataURL.MediaType.Subtype)
	url, err := app.Client.Upload(bytes.NewReader(dataURL.Data), []string{fname})
	if err != nil {
		app.Logger.Printf("%s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	app.Logger.Printf("url: %s", url)
	res := FileUploadResponse{
		ID:         uuid.String(),
		URL:        url,
		UploadedAt: time.Now(),
	}
	encoder := json.NewEncoder(w)
	if err := encoder.Encode(res); err != nil {
		app.Logger.Printf("%s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	return
}

func (app *App) showFile(w http.ResponseWriter, r *http.Request) {
	path := fmt.Sprintf(
		"%s/%s", app.Client.BasePath(), strings.TrimLeft(r.URL.Path[1:], "static/"))
	app.Logger.Printf("path=%s", path)
	http.ServeFile(w, r, path)
}

func main() {
	useS3 := flag.Bool("s3", false, "use AWS s3 as backedn")
	bucket := flag.String("bucket", "", "AWS s3 bucket name")
	flag.Parse()

	log.Printf("s3=%t", *useS3)
	log.Printf("bucket=%s", *bucket)

	var (
		fsClient Uploader
		err      error
	)
	if *useS3 {
		if *bucket == "" {
			log.Fatal("bucket is empty. export AWS_S3_BUCKET=<your_bucket>.")
		}
		fsClient, err = NewS3Client(*bucket)
	} else {
		fsClient, err = NewFSClient()
	}
	if err != nil {
		log.Fatal(err)
	}
	app := App{
		Client: fsClient,
		Logger: log.New(os.Stdout, "[app]: ", log.Lshortfile),
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/upload", app.uploadFile)
	mux.HandleFunc("/static/", app.showFile)

	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatal(err)
	}
}
