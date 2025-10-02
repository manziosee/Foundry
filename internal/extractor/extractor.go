package extractor

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type Extractor struct {
	maxFileSize int64
	maxFiles    int
}

func New() *Extractor {
	return &Extractor{
		maxFileSize: 100 * 1024 * 1024, // 100MB per file
		maxFiles:    10000,              // Max files per archive
	}
}

func (e *Extractor) Extract(archivePath, destDir string) error {
	file, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer file.Close()

	var reader io.Reader = file
	if strings.HasSuffix(archivePath, ".gz") || strings.HasSuffix(archivePath, ".tgz") {
		gzReader, err := gzip.NewReader(file)
		if err != nil {
			return err
		}
		defer gzReader.Close()
		reader = gzReader
	}

	tarReader := tar.NewReader(reader)
	fileCount := 0

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		fileCount++
		if fileCount > e.maxFiles {
			return fmt.Errorf("too many files in archive (max %d)", e.maxFiles)
		}

		if err := e.extractFile(tarReader, header, destDir); err != nil {
			return err
		}
	}

	return nil
}

func (e *Extractor) extractFile(tarReader *tar.Reader, header *tar.Header, destDir string) error {
	// Security checks
	if err := e.validatePath(header.Name, destDir); err != nil {
		return err
	}

	if header.Size > e.maxFileSize {
		return fmt.Errorf("file %s too large: %d bytes (max %d)", header.Name, header.Size, e.maxFileSize)
	}

	target := filepath.Join(destDir, header.Name)

	switch header.Typeflag {
	case tar.TypeDir:
		return os.MkdirAll(target, 0755)

	case tar.TypeReg:
		return e.extractRegularFile(tarReader, target, header)

	case tar.TypeSymlink:
		return e.extractSymlink(header, target, destDir)

	case tar.TypeLink:
		return e.extractHardlink(header, target, destDir)

	default:
		// Skip unsupported file types
		return nil
	}
}

func (e *Extractor) validatePath(name, destDir string) error {
	// Skip current directory entries
	if name == "." || name == "./" {
		return nil
	}

	// Prevent path traversal
	if strings.Contains(name, "..") {
		return fmt.Errorf("path traversal attempt: %s", name)
	}

	// Prevent absolute paths
	if filepath.IsAbs(name) {
		return fmt.Errorf("absolute path not allowed: %s", name)
	}

	// Clean and check final path is within destination
	target := filepath.Clean(filepath.Join(destDir, name))
	destClean := filepath.Clean(destDir)
	if !strings.HasPrefix(target, destClean) {
		return fmt.Errorf("path outside destination: %s", name)
	}

	return nil
}

func (e *Extractor) extractRegularFile(tarReader *tar.Reader, target string, header *tar.Header) error {
	if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
		return err
	}

	file, err := os.Create(target)
	if err != nil {
		return err
	}
	defer file.Close()

	// Limit copy to prevent zip bombs
	limited := io.LimitReader(tarReader, e.maxFileSize)
	if _, err := io.Copy(file, limited); err != nil {
		return err
	}

	// Set file permissions (but limit them)
	mode := header.FileInfo().Mode() & 0777
	if mode&0111 != 0 {
		mode = 0755 // Executable
	} else {
		mode = 0644 // Regular file
	}
	return os.Chmod(target, mode)
}

func (e *Extractor) extractSymlink(header *tar.Header, target, destDir string) error {
	// Validate symlink target
	linkTarget := header.Linkname
	if filepath.IsAbs(linkTarget) {
		return fmt.Errorf("absolute symlink not allowed: %s -> %s", header.Name, linkTarget)
	}

	// Resolve symlink and check it's within destDir
	resolved := filepath.Join(filepath.Dir(target), linkTarget)
	if !strings.HasPrefix(resolved, destDir) {
		return fmt.Errorf("symlink outside destination: %s -> %s", header.Name, linkTarget)
	}

	if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
		return err
	}

	return os.Symlink(linkTarget, target)
}

func (e *Extractor) extractHardlink(header *tar.Header, target, destDir string) error {
	linkTarget := filepath.Join(destDir, header.Linkname)
	
	// Validate hardlink target is within destDir
	if !strings.HasPrefix(linkTarget, destDir) {
		return fmt.Errorf("hardlink outside destination: %s -> %s", header.Name, header.Linkname)
	}

	if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
		return err
	}

	return os.Link(linkTarget, target)
}