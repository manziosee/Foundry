package cache

import (
	"database/sql"
	"os"
	"path/filepath"
)

type BlobCache struct {
	db   *sql.DB
	root string
}

func NewBlobCache(db *sql.DB, root string) *BlobCache {
	return &BlobCache{db: db, root: root}
}

func (c *BlobCache) Exists(checksum string) bool {
	path := c.getBlobPath(checksum)
	_, err := os.Stat(path)
	return err == nil
}

func (c *BlobCache) GetPath(checksum string) string {
	return c.getBlobPath(checksum)
}

func (c *BlobCache) getBlobPath(checksum string) string {
	return filepath.Join(c.root, "blobs", checksum+".tar")
}

func (c *BlobCache) MarkUsed(checksum string, imageID int) error {
	_, err := c.db.Exec("INSERT OR IGNORE INTO blobs(image_id, path, checksum) VALUES (?,?,?)",
		imageID, c.getBlobPath(checksum), checksum)
	return err
}

func (c *BlobCache) GetUnusedBlobs() ([]string, error) {
	rows, err := c.db.Query(`
		SELECT DISTINCT b.checksum 
		FROM blobs b 
		LEFT JOIN images i ON b.image_id = i.id 
		WHERE i.id IS NULL OR i.state = 'FAILED'`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var checksums []string
	for rows.Next() {
		var checksum string
		if err := rows.Scan(&checksum); err != nil {
			continue
		}
		checksums = append(checksums, checksum)
	}
	return checksums, nil
}

func (c *BlobCache) Cleanup() error {
	unused, err := c.GetUnusedBlobs()
	if err != nil {
		return err
	}

	for _, checksum := range unused {
		path := c.getBlobPath(checksum)
		os.Remove(path)
		c.db.Exec("DELETE FROM blobs WHERE checksum = ?", checksum)
	}
	return nil
}