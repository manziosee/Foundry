package main

import (
	"context"
	"database/sql"
	"io/ioutil"
	"path/filepath"

	"imgstore/internal/types"
	"imgstore/internal/cache"
	"imgstore/internal/downloader"
	"imgstore/internal/extractor"
	"imgstore/internal/fsm"
	"imgstore/internal/storage"
)

type Service struct {
	db         *sql.DB
	storage    *storage.OverlayStorage
	downloader *downloader.Downloader
	cache      *cache.BlobCache
	extractor  *extractor.Extractor
}

type ImageInfo struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	BlobKey  string `json:"blob_key"`
	Checksum string `json:"checksum"`
	State    string `json:"state"`
	Created  string `json:"created_at"`
	Updated  string `json:"updated_at"`
}

func NewService(db *sql.DB, root string) *Service {
	return &Service{
		db:         db,
		storage:    storage.NewOverlayStorage(root),
		downloader: downloader.New(),
		cache:      cache.NewBlobCache(db, root),
		extractor:  extractor.New(),
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

func (s *Service) GetImageStatus(name string) (string, error) {
	var state string
	err := s.db.QueryRow("SELECT state FROM images WHERE name=?", name).Scan(&state)
	return state, err
}

func (s *Service) GetAllImages() ([]types.ImageInfo, error) {
	rows, err := s.db.Query("SELECT id, name, blob_key, checksum, state, created_at, updated_at FROM images")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var images []types.ImageInfo
	for rows.Next() {
		var img types.ImageInfo
		if err := rows.Scan(&img.ID, &img.Name, &img.BlobKey, &img.Checksum, &img.State, &img.Created, &img.Updated); err != nil {
			continue
		}
		images = append(images, img)
	}
	return images, nil
}

func (s *Service) RemoveImage(name string) error {
	_, err := s.db.Exec("DELETE FROM images WHERE name=?", name)
	return err
}

func (s *Service) Cleanup() error {
	return s.cache.Cleanup()
}

func (s *Service) RunWorker(ctx context.Context) {
	// Worker implementation would go here
	// For now, just a placeholder
	<-ctx.Done()
}

func initSchema(db *sql.DB) error {
	schema, err := ioutil.ReadFile(filepath.Join("migrations", "001_init.sql"))
	if err != nil {
		return err
	}
	_, err = db.Exec(string(schema))
	return err
}