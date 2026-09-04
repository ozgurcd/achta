package safefile

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const DefaultMaxSize int64 = 8 << 20

// Snapshot captures a bounded regular file for validated replacement.
type Snapshot struct {
	Path    string
	Data    []byte
	Mode    os.FileMode
	Size    int64
	ModTime int64
	Digest  [sha256.Size]byte
	root    string
}

// Read reads one non-linked regular file confined beneath root.
func Read(root, path string, maxSize int64) (Snapshot, error) {
	if maxSize <= 0 {
		return Snapshot{}, errors.New("maximum size must be positive")
	}
	root, path, err := validatePath(root, path)
	if err != nil {
		return Snapshot{}, err
	}
	if err := rejectSecretLike(path); err != nil {
		return Snapshot{}, err
	}
	if err := rejectLinkedComponents(root, path); err != nil {
		return Snapshot{}, err
	}
	info, err := os.Lstat(path)
	if err != nil {
		return Snapshot{}, err
	}
	if !info.Mode().IsRegular() {
		return Snapshot{}, fmt.Errorf("%q is not a regular file", path)
	}
	if info.Size() > maxSize {
		return Snapshot{}, fmt.Errorf("%q exceeds %d-byte limit", path, maxSize)
	}
	f, err := os.Open(path)
	if err != nil {
		return Snapshot{}, err
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, maxSize+1))
	if err != nil {
		return Snapshot{}, err
	}
	if int64(len(data)) > maxSize {
		return Snapshot{}, fmt.Errorf("%q exceeds %d-byte limit", path, maxSize)
	}
	return Snapshot{Path: path, Data: data, Mode: info.Mode().Perm(), Size: info.Size(), ModTime: info.ModTime().UnixNano(), Digest: sha256.Sum256(data), root: root}, nil
}

// Replace atomically replaces a snapshot after rechecking source identity.
func (s Snapshot) Replace(replacement []byte, maxSize int64) error {
	if int64(len(replacement)) > maxSize {
		return fmt.Errorf("replacement exceeds %d-byte limit", maxSize)
	}
	if err := rejectLinkedComponents(s.root, s.Path); err != nil {
		return err
	}
	current, err := Read(s.root, s.Path, maxSize)
	if err != nil {
		return fmt.Errorf("recheck source: %w", err)
	}
	if current.Size != s.Size || current.ModTime != s.ModTime || current.Mode != s.Mode || current.Digest != s.Digest {
		return errors.New("source changed concurrently; refusing replacement")
	}
	dir := filepath.Dir(s.Path)
	tmp, err := os.CreateTemp(dir, ".achta-replace-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	keep := false
	defer func() {
		if !keep {
			_ = os.Remove(tmpPath)
		}
	}()
	if err := tmp.Chmod(s.Mode); err != nil {
		tmp.Close()
		return err
	}
	if _, err := io.Copy(tmp, bytes.NewReader(replacement)); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	final, err := Read(s.root, s.Path, maxSize)
	if err != nil {
		return fmt.Errorf("final source recheck: %w", err)
	}
	if final.Size != s.Size || final.ModTime != s.ModTime || final.Mode != s.Mode || final.Digest != s.Digest {
		return errors.New("source changed concurrently; refusing replacement")
	}
	if err := os.Rename(tmpPath, s.Path); err != nil {
		return err
	}
	keep = true
	if d, err := os.Open(dir); err == nil {
		defer d.Close()
		_ = d.Sync()
	}
	return nil
}

func validatePath(root, path string) (string, string, error) {
	rootAbs, err := filepath.Abs(filepath.Clean(root))
	if err != nil {
		return "", "", err
	}
	rootReal, err := filepath.EvalSymlinks(rootAbs)
	if err != nil {
		return "", "", err
	}
	var rel string
	if filepath.IsAbs(path) {
		path, err = filepath.Abs(filepath.Clean(path))
		if err != nil {
			return "", "", err
		}
		rel, err = filepath.Rel(rootAbs, path)
	} else {
		rel = filepath.Clean(path)
	}
	if err != nil || rel == ".." || filepath.IsAbs(rel) || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", "", fmt.Errorf("path %q escapes root", path)
	}
	return rootReal, filepath.Join(rootReal, rel), nil
}

func rejectLinkedComponents(root, path string) error {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return err
	}
	current := root
	for _, component := range strings.Split(rel, string(filepath.Separator)) {
		current = filepath.Join(current, component)
		info, err := os.Lstat(current)
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("linked path component rejected: %q", current)
		}
	}
	return nil
}

func rejectSecretLike(path string) error {
	base := strings.ToLower(filepath.Base(path))
	if base == ".env" || strings.HasPrefix(base, ".env.") || base == "dev.env" || strings.HasSuffix(base, ".key") || strings.HasSuffix(base, ".pem") || strings.HasSuffix(base, ".p12") || strings.Contains(base, "credential") || strings.Contains(base, "secret") || strings.Contains(base, "token") || strings.Contains(base, "cookie") || strings.Contains(base, "license") {
		return fmt.Errorf("secret-like path rejected: %q", filepath.Base(path))
	}
	return nil
}
