package downloader

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

type Downloader struct {
	client     *http.Client
	maxRetries int
}

type ProgressCallback func(downloaded, total int64)

func New() *Downloader {
	return &Downloader{
		client: &http.Client{
			Timeout: 30 * time.Minute,
		},
		maxRetries: 3,
	}
}

func (d *Downloader) Download(ctx context.Context, url, destPath, expectedChecksum string, progress ProgressCallback) error {
	var lastErr error
	
	for attempt := 0; attempt <= d.maxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(time.Duration(attempt) * time.Second):
			}
		}
		
		if err := d.downloadAttempt(ctx, url, destPath, expectedChecksum, progress); err != nil {
			lastErr = err
			continue
		}
		return nil
	}
	
	return fmt.Errorf("download failed after %d attempts: %v", d.maxRetries+1, lastErr)
}

func (d *Downloader) downloadAttempt(ctx context.Context, url, destPath, expectedChecksum string, progress ProgressCallback) error {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}
	
	resp, err := d.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}
	
	tmpPath := destPath + ".tmp"
	file, err := os.Create(tmpPath)
	if err != nil {
		return err
	}
	defer file.Close()
	
	hash := sha256.New()
	writer := io.MultiWriter(file, hash)
	
	var downloaded int64
	total := resp.ContentLength
	
	buf := make([]byte, 32*1024)
	for {
		select {
		case <-ctx.Done():
			os.Remove(tmpPath)
			return ctx.Err()
		default:
		}
		
		n, err := resp.Body.Read(buf)
		if n > 0 {
			if _, writeErr := writer.Write(buf[:n]); writeErr != nil {
				os.Remove(tmpPath)
				return writeErr
			}
			downloaded += int64(n)
			if progress != nil {
				progress(downloaded, total)
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			os.Remove(tmpPath)
			return err
		}
	}
	
	actualChecksum := fmt.Sprintf("%x", hash.Sum(nil))
	if actualChecksum != expectedChecksum {
		os.Remove(tmpPath)
		return fmt.Errorf("checksum mismatch: expected %s, got %s", expectedChecksum, actualChecksum)
	}
	
	return os.Rename(tmpPath, destPath)
}