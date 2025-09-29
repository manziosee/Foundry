package storage

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

type OverlayStorage struct {
	root string
}

func NewOverlayStorage(root string) *OverlayStorage {
	return &OverlayStorage{root: root}
}

func (o *OverlayStorage) Init() error {
	dirs := []string{"blobs", "images", "overlays", "active"}
	for _, dir := range dirs {
		if err := os.MkdirAll(filepath.Join(o.root, dir), 0755); err != nil {
			return err
		}
	}
	return nil
}

func (o *OverlayStorage) CreateSnapshot(imageName string) error {
	overlayDir := filepath.Join(o.root, "overlays", imageName)
	upperDir := filepath.Join(overlayDir, "upper")
	workDir := filepath.Join(overlayDir, "work")
	activeDir := filepath.Join(o.root, "active", imageName)
	lowerDir := filepath.Join(o.root, "images", imageName, "rootfs")

	// Create directories
	for _, dir := range []string{upperDir, workDir, activeDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	// Mount overlay
	opts := fmt.Sprintf("lowerdir=%s,upperdir=%s,workdir=%s", lowerDir, upperDir, workDir)
	cmd := exec.Command("mount", "-t", "overlay", "overlay", "-o", opts, activeDir)
	return cmd.Run()
}

func (o *OverlayStorage) RemoveSnapshot(imageName string) error {
	activeDir := filepath.Join(o.root, "active", imageName)
	cmd := exec.Command("umount", activeDir)
	return cmd.Run()
}

func (o *OverlayStorage) GetImagePath(imageName string) string {
	return filepath.Join(o.root, "images", imageName, "rootfs")
}

func (o *OverlayStorage) GetBlobPath(checksum string) string {
	return filepath.Join(o.root, "blobs", checksum+".tar")
}