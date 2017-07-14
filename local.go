package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"path"
)

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
func (c *FSClient) Upload(in io.ReadSeeker, p []string) (string, error) {
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
