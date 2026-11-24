package store

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/cruciblehq/utils-go/file"
)

// Archive file extension.
const archiveExtension = ".tar.zst"

// Returns the directory path for storing archives of a specific version.
//
// The path follows the pattern: {archiveRoot}/{namespace}/{resource}/{version}
func (s *SQLStore) archiveDirectoryPath(namespace, resource, version string) string {
	return filepath.Join(s.archiveRoot, namespace, resource, version)
}

// Returns the final path for an archive file based on its digest.
//
// The path follows the pattern: {archiveDir}/{digest}.tar.zst
func (s *SQLStore) archiveFinalPath(namespace, resource, version, digest string) string {
	archiveDir := s.archiveDirectoryPath(namespace, resource, version)
	return filepath.Join(archiveDir, fmt.Sprintf("%s%s", digest, archiveExtension))
}

// Stores an archive file to disk and calculates its digest.
//
// Writes the archive data to a temporary file while calculating the SHA-256
// digest, then moves it to the final location named by the digest. Returns
// the digest, final file path, and size in bytes.
func (s *SQLStore) storeArchiveFile(namespace, resource, version string, archiveReader io.Reader) (digest, path string, size int64, err error) {
	archiveDir := s.archiveDirectoryPath(namespace, resource, version)
	if err := os.MkdirAll(archiveDir, file.DefaultDirMode); err != nil {
		return "", "", 0, err
	}

	tmp, err := file.CreateAtomic(archiveDir, file.DefaultFileMode)
	if err != nil {
		return "", "", 0, err
	}
	defer tmp.Abort()

	hasher := sha256.New()
	size, err = io.Copy(io.MultiWriter(tmp, hasher), archiveReader)
	if err != nil {
		return "", "", 0, err
	}
	digest = fmt.Sprintf("sha256:%s", hex.EncodeToString(hasher.Sum(nil)))

	path = s.archiveFinalPath(namespace, resource, version, digest)
	if err := tmp.Commit(path); err != nil {
		return "", "", 0, err
	}
	return digest, path, size, nil
}

// Removes an archive file and then prunes now-empty parent directories.
//
// Parent pruning stops at the configured archive root.
func (s *SQLStore) removeArchiveFileAndParents(path string) error {
	if path == "" {
		return nil
	}

	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}

	return file.RemoveEmptyParents(filepath.Dir(path), s.archiveRoot)
}

// Removes archive files that are not referenced by any stored version.
//
// This is run on startup to recover space from stale files left by older
// server versions or interrupted operations.
func (s *SQLStore) cleanupOrphanArchiveFiles(ctx context.Context) error {
	if err := os.MkdirAll(s.archiveRoot, file.DefaultDirMode); err != nil {
		return err
	}

	used, err := s.listArchivePaths(ctx)
	if err != nil {
		return err
	}

	var dirs []string
	err = filepath.WalkDir(s.archiveRoot, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			dirs = append(dirs, path)
			return nil
		}

		if !strings.HasSuffix(path, archiveExtension) && !strings.HasPrefix(d.Name(), file.TempPrefix) {
			return nil
		}

		if _, ok := used[path]; ok {
			return nil
		}

		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return err
		}
		s.logger.Info("removed orphan archive file", "path", path)
		return nil
	})
	if err != nil {
		return err
	}

	sort.Slice(dirs, func(i, j int) bool { return len(dirs[i]) > len(dirs[j]) })
	for _, dir := range dirs {
		if filepath.Clean(dir) == filepath.Clean(s.archiveRoot) {
			continue
		}
		_ = os.Remove(dir)
	}

	return nil
}
