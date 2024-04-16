package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"runtime"
	"time"
)

func SystemInformation() (OperatingSystem, Architecture) {
	var sys OperatingSystem
	var arch Architecture
	var err error
	ctx := CreateSentryCtx("SystemInformation")

	switch runtime.GOOS {
	case "windows":
		sys = Windows
	case "linux":
		sys = Linux
	case "darwin":
		sys = Mac
	default:
		err = errors.New("unsupported operating system")
	}
	CrumbCaptureExit(ctx, err, "checking OS: "+runtime.GOOS)

	switch runtime.GOARCH {
	case "amd64":
		arch = x86
	case "arm64":
		arch = Arm64
	default:
		err = errors.New("unsupported system architecture")
	}
	CrumbCaptureExit(ctx, err, "checking Arch: "+runtime.GOARCH)

	return sys, arch
}

func (sys OperatingSystem) JavaExecutable() string {
	if sys == Windows {
		return "javaw.exe"
	}
	return "java"
}

func GetFromURL(url string) (io.ReadCloser, error) {
	// Create the HTTP client
	client := http.Client{
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout: 30 * time.Second,
			}).DialContext,
		},
	}

	// Create the HTTP request
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("User-Agent", fmt.Sprintf("Pinnacle/%s (%s; %s)", version, Sys, Arch))

	// Perform the HTTP request
	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}

	// Check if request was successful
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("received status code: %d", response.StatusCode)
	}

	return response.Body, nil
}

func DownloadFromURL(url string, path string) error {
	ctx := CreateSentryCtx("DownloadFromUrl")
	// Perform the HTTP request
	body, err := GetFromURL(url)
	if err != nil {
		return err
	}
	defer func() {
		if err = body.Close(); err != nil {
			CrumbCaptureExit(ctx, err, "closing request body")
		}
	}()

	// Create or truncate the file
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func() {
		if err = file.Close(); err != nil {
			CrumbCaptureExit(ctx, err, "closing file body")
		}
	}()

	// Copy response body to file
	_, err = io.Copy(file, body)
	return err
}

func FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
