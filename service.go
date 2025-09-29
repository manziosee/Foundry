package main

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"database/sql"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"imgstore/internal/fsm"
	"imgstore/internal/storage"
)

type Service struct {
	db      *sql.DB
	storage *storage.OverlayStorage
}

func NewService(db *sql.DB, root string) *Service {
	return &Service{
		db:      db,
		storage: storage.NewOverlayStorage(root),
	}
}

func (s *Service) Init() error {
	return s.storage.Init()
}

func (s *Service) EnqueueImage(ctx context.Context, name, blobURL, checksum string) error {
	_, err := s.db.Exec("INSERT OR IGNORE INTO images(name, blob_key, checksum, state) VALUES (?,?,?,?)",
		name, blobURL, checksum, string(fsm.StateNew))
	return err
}

func (s *Service) RunWorker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			s.processNextImage(ctx)
			time.Sleep(2 * time.Second)
		}
	}
}

func (s *Service) processNextImage(ctx context.Context) {
	rows, err := s.db.Query("SELECT id, name, blob_key, checksum, state FROM images WHERE state NOT IN ('ACTIVE', 'FAILED') LIMIT 1")
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var id int
		var name, blobKey, checksum, state string
		if err := rows.Scan(&id, &name, &blobKey, &checksum, &state); err != nil {
			continue
		}

		currentState := fsm.State(state)
		nextState := fsm.NextState(currentState)
		
		if !fsm.CanTransition(currentState, nextState) {
			continue
		}

		if err := s.executeTransition(ctx, id, name, blobKey, checksum, currentState, nextState); err != nil {
			s.setState(id, fsm.StateFailed)
		} else {
			s.setState(id, nextState)
		}
	}
}

func (s *Service) executeTransition(ctx context.Context, id int, name, blobKey, checksum string, from, to fsm.State) error {
	switch to {
	case fsm.StateDownloading:
		return nil // Just mark as downloading
	case fsm.StateDownloaded:
		return s.downloadBlob(ctx, blobKey, checksum)
	case fsm.StateUnpacking:
		return nil // Just mark as unpacking
	case fsm.StateUnpacked:
		return s.unpackBlob(checksum, name)
	case fsm.StateStored:
		return nil // For overlay, no additional storage step needed
	case fsm.StateActivating:
		return nil // Just mark as activating
	case fsm.StateActive:
		return s.storage.CreateSnapshot(name)
	}
	return nil
}

func (s *Service) downloadBlob(ctx context.Context, blobURL, expectedChecksum string) error {
	blobPath := s.storage.GetBlobPath(expectedChecksum)
	
	// Check if already exists
	if _, err := os.Stat(blobPath); err == nil {
		return s.verifyChecksum(blobPath, expectedChecksum)
	}

	// Download
	resp, err := http.Get(blobURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	tmpPath := blobPath + ".tmp"
	file, err := os.Create(tmpPath)
	if err != nil {
		return err
	}
	defer file.Close()

	if _, err := io.Copy(file, resp.Body); err != nil {
		os.Remove(tmpPath)
		return err
	}

	if err := s.verifyChecksum(tmpPath, expectedChecksum); err != nil {
		os.Remove(tmpPath)
		return err
	}

	return os.Rename(tmpPath, blobPath)
}

func (s *Service) verifyChecksum(path, expected string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return err
	}

	actual := fmt.Sprintf("%x", hash.Sum(nil))
	if actual != expected {
		return fmt.Errorf("checksum mismatch: expected %s, got %s", expected, actual)
	}
	return nil
}

func (s *Service) unpackBlob(checksum, imageName string) error {
	blobPath := s.storage.GetBlobPath(checksum)
	imagePath := s.storage.GetImagePath(imageName)

	if err := os.MkdirAll(imagePath, 0755); err != nil {
		return err
	}

	file, err := os.Open(blobPath)
	if err != nil {
		return err
	}
	defer file.Close()

	var reader io.Reader = file
	if strings.HasSuffix(blobPath, ".tar.gz") || strings.HasSuffix(blobPath, ".tgz") {
		gzReader, err := gzip.NewReader(file)
		if err != nil {
			return err
		}
		defer gzReader.Close()
		reader = gzReader
	}

	tarReader := tar.NewReader(reader)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		target := filepath.Join(imagePath, header.Name)
		if !strings.HasPrefix(target, imagePath) {
			continue // Prevent path traversal
		}

		switch header.Typeflag {
		case tar.TypeDir:
			os.MkdirAll(target, os.FileMode(header.Mode))
		case tar.TypeReg:
			os.MkdirAll(filepath.Dir(target), 0755)
			outFile, err := os.Create(target)
			if err != nil {
				return err
			}
			io.Copy(outFile, tarReader)
			outFile.Close()
			os.Chmod(target, os.FileMode(header.Mode))
		}
	}
	return nil
}

func (s *Service) setState(id int, state fsm.State) {
	s.db.Exec("UPDATE images SET state=?, updated_at=datetime('now') WHERE id=?", string(state), id)
}

func (s *Service) GetImageStatus(name string) (string, error) {
	var state string
	err := s.db.QueryRow("SELECT state FROM images WHERE name=?", name).Scan(&state)
	return state, err
}