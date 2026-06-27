package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

// Volume mapping: logical name → filesystem path

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

// resolvePath returns the clean, absolute host path safely ensuring no directory traversal.

func resolvePath(volumeName, subPath string) (string, error) {
	basePath, ok := volumes[volumeName]
	if !ok {
		return "", fmt.Errorf("invalid volume: %q", volumeName)
	}

	cleanSub := filepath.Clean(subPath)
	if cleanSub == "." || cleanSub == "/" || cleanSub == "" {
		return basePath, nil
	}

	// Block relative directory traversal and absolute escape routes.
	if strings.HasPrefix(cleanSub, "/") || strings.HasPrefix(cleanSub, "..") || strings.Contains(cleanSub, "../") {
		return "", fmt.Errorf("invalid subpath: %q", subPath)
	}

	fullPath := filepath.Join(basePath, cleanSub)

	if !strings.HasPrefix(fullPath, basePath) {
		return "", fmt.Errorf("access denied")
	}

	return fullPath, nil
}

// --- gRPC Client helper ---

func walkDirectory(fullPath string) ([]string, map[string]int64, error) {
	files := []string{}
	sizes := make(map[string]int64)

	err := filepath.Walk(fullPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			relPath, err := filepath.Rel(fullPath, path)
			if err != nil {
				return err
			}
			files = append(files, relPath)
			sizes[relPath] = info.Size()
		}
		return nil
	})
	if err != nil {
		return nil, nil, err
	}

	sort.Strings(files)
	return files, sizes, nil
}

func resolveUniqueFilename(dirPath, filename string) string {
	candidatePath := filepath.Join(dirPath, filename)
	if _, err := os.Stat(candidatePath); os.IsNotExist(err) {
		return filename
	}

	ext := filepath.Ext(filename)
	base := filename[:len(filename)-len(ext)]

	for i := 1; ; i++ {
		candidate := fmt.Sprintf("%s (%d)%s", base, i, ext)
		if _, err := os.Stat(filepath.Join(dirPath, candidate)); os.IsNotExist(err) {
			return candidate
		}
	}
}

func copyFile(src, dest string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err = io.Copy(out, in); err != nil {
		return err
	}

	// Copy file permissions
	si, err := os.Stat(src)
	if err == nil {
		os.Chmod(dest, si.Mode())
	}
	return nil
}

func copyDir(src, dest string) error {
	si, err := os.Stat(src)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(dest, si.Mode()); err != nil {
		return err
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		destPath := filepath.Join(dest, entry.Name())

		if entry.IsDir() {
			if err := copyDir(srcPath, destPath); err != nil {
				return err
			}
		} else {
			if err := copyFile(srcPath, destPath); err != nil {
				return err
			}
		}
	}
	return nil
}
