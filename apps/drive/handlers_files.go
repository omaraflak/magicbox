package main

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
)

// Volume mapping: logical name → filesystem path

func handleInfo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	username := cachedUsername
	if username == "" {
		var err error
		username, err = getUsernameFromCore()
		if err != nil {
			log.Printf("Warning: failed to fetch username via gRPC: %v", err)
			username = "User (" + userID + ")" // Fallback
		} else {
			cachedUsername = username
		}
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"user_id":  userID,
		"app_id":   appID,
		"username": username,
	})
}

func handleFiles(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		handleListFiles(w, r)
	case http.MethodPost:
		handleUploadFiles(w, r)
	case http.MethodDelete:
		handleDeleteFile(w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func handleListFiles(w http.ResponseWriter, r *http.Request) {
	volumeName := r.URL.Query().Get("volume")
	subPath := r.URL.Query().Get("path")

	dirPath, err := resolvePath(volumeName, subPath)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	entries, err := os.ReadDir(dirPath)
	if err != nil {
		writeJSON(w, http.StatusOK, []FileInfo{})
		return
	}

	if volumeName == "trash" {
		trashRecords, err := getAllTrashRecords()
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to query trash")
			return
		}

		var dbTrash = make(map[string]FileInfo)
		for _, f := range trashRecords {
			dbTrash[f.Name] = FileInfo{
				Name:         f.Name,
				DisplayName:  f.OriginalName,
				OriginalPath: f.OriginalPath,
				DeletedAt:    &f.DeletedAt,
			}
		}

		files := make([]FileInfo, 0, len(entries))
		for _, entry := range entries {
			info, err := entry.Info()
			if err != nil {
				continue
			}
			item, ok := dbTrash[entry.Name()]
			if !ok {
				item = FileInfo{
					Name:        entry.Name(),
					DisplayName: entry.Name(),
				}
			}
			item.Size = info.Size()
			item.ModifiedAt = info.ModTime().UTC()
			item.IsDir = entry.IsDir()

			files = append(files, item)
		}

		writeJSON(w, http.StatusOK, files)
		return
	}

	files := make([]FileInfo, 0, len(entries))
	for _, entry := range entries {
		if entry.Name() == ".trash" || entry.Name() == ".drive.db" {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}

		isAutoSend := false
		if entry.IsDir() && volumeName == "storage" {
			relPath := entry.Name()
			if subPath != "" {
				relPath = subPath + "/" + entry.Name()
			}
			count, _ := countAutoSendFolder(relPath)
			isAutoSend = (count > 0)
		}

		files = append(files, FileInfo{
			Name:        entry.Name(),
			DisplayName: entry.Name(),
			Size:        info.Size(),
			ModifiedAt:  info.ModTime().UTC(),
			IsDir:       entry.IsDir(),
			IsAutoSend:  isAutoSend,
		})
	}

	writeJSON(w, http.StatusOK, files)
}

func handleUploadFiles(w http.ResponseWriter, r *http.Request) {
	volumeName := r.URL.Query().Get("volume")
	subPath := r.URL.Query().Get("path")

	dirPath, err := resolvePath(volumeName, subPath)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Limit request body size to maxRequestBytes (e.g. 10GB)
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBytes)

	mr, err := r.MultipartReader()
	if err != nil {
		writeError(w, http.StatusBadRequest, "failed to open multipart reader: "+err.Error())
		return
	}

	uploaded := 0
	for {
		part, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to read next part: "+err.Error())
			return
		}

		// Skip parts that don't have a filename (i.e. standard text form fields)
		if part.FileName() == "" {
			part.Close()
			continue
		}

		safeName := filepath.Base(part.FileName())
		destPath := filepath.Join(dirPath, safeName)

		dst, err := os.OpenFile(destPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
		if err != nil {
			part.Close()
			writeError(w, http.StatusInternalServerError, "failed to create destination file: "+err.Error())
			return
		}

		// Pipe the multipart part stream directly into the destination file!
		_, err = io.Copy(dst, part)
		part.Close()
		dst.Close()
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to write file: "+err.Error())
			return
		}

		checkAndTriggerAutoSend(volumeName, subPath, safeName)

		uploaded++
	}

	writeJSON(w, http.StatusOK, map[string]int{"uploaded": uploaded})
}

func handleDeleteFile(w http.ResponseWriter, r *http.Request) {
	volumeName := r.URL.Query().Get("volume")
	subPath := r.URL.Query().Get("path")
	filename := r.URL.Query().Get("file")

	if filename == "" {
		writeError(w, http.StatusBadRequest, "missing 'file' parameter")
		return
	}

	dirPath, err := resolvePath(volumeName, subPath)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	safeName := filepath.Base(filename)
	fullPath := filepath.Join(dirPath, safeName)

	if volumeName == "trash" {
		if err := os.RemoveAll(fullPath); err != nil {
			if os.IsNotExist(err) {
				writeError(w, http.StatusNotFound, fmt.Sprintf("file not found: %q", safeName))
			} else {
				writeError(w, http.StatusInternalServerError, "failed to delete file/folder: "+err.Error())
			}
			return
		}
		deleteTrashRecord(safeName)
		writeJSON(w, http.StatusOK, map[string]string{"deleted": safeName})
		return
	}

	// Soft delete: move to trash volume
	trashDir := volumes["trash"]
	if err := os.MkdirAll(trashDir, 0755); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to initialize trash folder: "+err.Error())
		return
	}

	trashName := safeName + "_" + strconv.FormatInt(time.Now().Unix(), 10)
	destPath := filepath.Join(trashDir, trashName)

	if err := os.Rename(fullPath, destPath); err != nil {
		if os.IsNotExist(err) {
			writeError(w, http.StatusNotFound, fmt.Sprintf("file not found: %q", safeName))
		} else {
			writeError(w, http.StatusInternalServerError, "failed to move file/folder to trash: "+err.Error())
		}
		return
	}

	trashID := uuid.NewString()
	dbErr := insertTrashRecord(trashID, safeName, subPath, trashName, time.Now().Format(time.RFC3339))
	if dbErr != nil {
		log.Printf("Warning: failed to record trash metadata: %v", dbErr)
	}

	writeJSON(w, http.StatusOK, map[string]string{"deleted": safeName, "trash_name": trashName})
}

func handleDownload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	volumeName := r.URL.Query().Get("volume")
	subPath := r.URL.Query().Get("path")
	filenames := r.URL.Query()["file"]
	volIndexStr := r.URL.Query().Get("vol_index")

	if len(filenames) == 0 {
		writeError(w, http.StatusBadRequest, "missing 'file' parameter")
		return
	}

	dirPath, err := resolvePath(volumeName, subPath)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// 1. Single File Download (No zipping)
	if len(filenames) == 1 {
		safeName := filepath.Base(filenames[0])
		fullPath := filepath.Join(dirPath, safeName)
		fi, err := os.Stat(fullPath)
		if err == nil && !fi.IsDir() {
			w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", safeName))
			http.ServeFile(w, r, fullPath)
			return
		}
	}

	// 2. Folder or Multi-item download (zipped on the fly)
	volIndex := 0
	if volIndexStr != "" {
		volIndex, err = strconv.Atoi(volIndexStr)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid 'vol_index' parameter")
			return
		}
	}

	defaultZipName := "archive"
	if len(filenames) == 1 {
		defaultZipName = filepath.Base(filenames[0])
	}

	plan, items, err := generateMultiItemPlan(dirPath, filenames, defaultZipName)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to generate plan: "+err.Error())
		return
	}

	if volIndex < 0 || volIndex >= len(plan.Volumes) {
		writeError(w, http.StatusBadRequest, "vol_index out of bounds")
		return
	}

	targetVolume := plan.Volumes[volIndex]

	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", targetVolume.Name))

	zw := zip.NewWriter(w)
	defer zw.Close()

	for _, zipPath := range targetVolume.Files {
		// Find corresponding zipItem
		var diskPath string
		for _, item := range items {
			if item.ZipPath == zipPath {
				diskPath = item.DiskPath
				break
			}
		}

		if diskPath == "" {
			continue
		}

		src, err := os.Open(diskPath)
		if err != nil {
			log.Printf("Failed to open file %s for zipping: %v", diskPath, err)
			continue
		}

		header := &zip.FileHeader{
			Name:   filepath.ToSlash(zipPath),
			Method: zip.Deflate,
		}
		header.Modified = time.Now()

		writer, err := zw.CreateHeader(header)
		if err != nil {
			src.Close()
			log.Printf("Failed to create zip header for %s: %v", zipPath, err)
			continue
		}

		_, err = io.Copy(writer, src)
		src.Close()
		if err != nil {
			log.Printf("Failed to copy file %s content to zip: %v", diskPath, err)
			continue
		}
	}
}

func generateMultiItemPlan(baseDir string, filenames []string, defaultZipName string) (DownloadPlan, []zipItem, error) {
	var items []zipItem
	for _, name := range filenames {
		safeName := filepath.Base(name)
		fullPath := filepath.Join(baseDir, safeName)
		fi, err := os.Stat(fullPath)
		if err != nil {
			return DownloadPlan{}, nil, err
		}
		if fi.IsDir() {
			nestedFiles, sizes, err := walkDirectory(fullPath)
			if err != nil {
				return DownloadPlan{}, nil, err
			}
			for _, rel := range nestedFiles {
				items = append(items, zipItem{
					DiskPath: filepath.Join(fullPath, rel),
					ZipPath:  filepath.Join(safeName, rel),
					Size:     sizes[rel],
				})
			}
		} else {
			items = append(items, zipItem{
				DiskPath: fullPath,
				ZipPath:  safeName,
				Size:     fi.Size(),
			})
		}
	}

	// Sort items by ZipPath for deterministic volume contents
	sort.Slice(items, func(i, j int) bool {
		return items[i].ZipPath < items[j].ZipPath
	})

	plan := DownloadPlan{Volumes: []ZipVolume{}}
	if len(items) == 0 {
		return plan, items, nil
	}

	var currentVol ZipVolume
	currentVol.Index = 0
	currentVol.Name = fmt.Sprintf("%s.part%d.zip", defaultZipName, currentVol.Index+1)
	currentVol.Files = []string{}
	currentVol.Size = 0

	for _, item := range items {
		if currentVol.Size+item.Size > maxZipVolumeSize && len(currentVol.Files) > 0 {
			plan.Volumes = append(plan.Volumes, currentVol)
			currentVol.Index++
			currentVol.Name = fmt.Sprintf("%s.part%d.zip", defaultZipName, currentVol.Index+1)
			currentVol.Files = []string{}
			currentVol.Size = 0
		}
		currentVol.Files = append(currentVol.Files, item.ZipPath)
		currentVol.Size += item.Size
	}
	plan.Volumes = append(plan.Volumes, currentVol)

	if len(plan.Volumes) == 1 {
		plan.Volumes[0].Name = defaultZipName + ".zip"
	}

	return plan, items, nil
}

func handleDownloadPlan(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	volumeName := r.URL.Query().Get("volume")
	subPath := r.URL.Query().Get("path")
	filenames := r.URL.Query()["file"]

	if len(filenames) == 0 {
		writeError(w, http.StatusBadRequest, "missing 'file' parameter")
		return
	}

	dirPath, err := resolvePath(volumeName, subPath)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Single item file download plan (direct download, no zip)
	if len(filenames) == 1 {
		safeName := filepath.Base(filenames[0])
		fullPath := filepath.Join(dirPath, safeName)
		fi, err := os.Stat(fullPath)
		if err != nil {
			if os.IsNotExist(err) {
				writeError(w, http.StatusNotFound, "file not found")
			} else {
				writeError(w, http.StatusInternalServerError, err.Error())
			}
			return
		}
		if !fi.IsDir() {
			writeJSON(w, http.StatusOK, DownloadPlan{
				Volumes: []ZipVolume{
					{
						Index: 0,
						Name:  safeName,
						Files: []string{safeName},
						Size:  fi.Size(),
					},
				},
			})
			return
		}
	}

	// Multiple items or single folder (needs zip plan)
	defaultZipName := "archive"
	if len(filenames) == 1 {
		defaultZipName = filepath.Base(filenames[0])
	}

	plan, _, err := generateMultiItemPlan(dirPath, filenames, defaultZipName)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to generate download plan: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, plan)
}

func handleFolders(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	volumeName := r.URL.Query().Get("volume")
	subPath := r.URL.Query().Get("path")

	var body struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if body.Name == "" {
		writeError(w, http.StatusBadRequest, "missing 'name' property")
		return
	}

	if strings.Contains(body.Name, "/") || strings.Contains(body.Name, "\\") || body.Name == "." || body.Name == ".." {
		writeError(w, http.StatusBadRequest, "invalid folder name")
		return
	}

	dirPath, err := resolvePath(volumeName, subPath)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	targetPath := filepath.Join(dirPath, body.Name)
	if err := os.MkdirAll(targetPath, 0755); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create folder: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"created": body.Name})
}

func handleMoveFile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	volumeName := r.URL.Query().Get("volume")
	subPath := r.URL.Query().Get("path")
	filename := r.URL.Query().Get("file")
	destPath := r.URL.Query().Get("dest_path")
	newName := r.URL.Query().Get("new_name")

	if filename == "" {
		writeError(w, http.StatusBadRequest, "missing 'file' parameter")
		return
	}

	srcDir, err := resolvePath(volumeName, subPath)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	destDir, err := resolvePath(volumeName, destPath)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	safeName := filepath.Base(filename)
	srcFullPath := filepath.Join(srcDir, safeName)

	targetName := safeName
	if newName != "" {
		targetName = filepath.Base(newName)
	}
	destFullPath := filepath.Join(destDir, targetName)

	if _, err := os.Stat(srcFullPath); os.IsNotExist(err) {
		writeError(w, http.StatusNotFound, "source file/folder not found")
		return
	}

	fi, err := os.Stat(destDir)
	if err != nil {
		if os.IsNotExist(err) {
			writeError(w, http.StatusNotFound, "destination directory not found")
		} else {
			writeError(w, http.StatusInternalServerError, err.Error())
		}
		return
	} else if !fi.IsDir() {
		writeError(w, http.StatusBadRequest, "destination is not a directory")
		return
	}

	if strings.HasPrefix(destFullPath, srcFullPath+string(filepath.Separator)) || destFullPath == srcFullPath {
		writeError(w, http.StatusBadRequest, "cannot move a directory into itself")
		return
	}

	if _, err := os.Stat(destFullPath); !os.IsNotExist(err) {
		writeError(w, http.StatusConflict, "a file/folder with the same name already exists in the destination folder")
		return
	}

	if err := os.Rename(srcFullPath, destFullPath); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to move file/folder: "+err.Error())
		return
	}

	if volumeName == "storage" {
		isDir := false
		if fi, err := os.Stat(destFullPath); err == nil {
			isDir = fi.IsDir()
		}

		if isDir {
			filepath.Walk(destFullPath, func(path string, info os.FileInfo, err error) error {
				if err == nil && !info.IsDir() {
					rel, err := filepath.Rel(volumes["storage"], path)
					if err == nil {
						subDir := filepath.Dir(rel)
						if subDir == "." {
							subDir = ""
						}
						checkAndTriggerAutoSend("storage", subDir, info.Name())
					}
				}
				return nil
			})
		} else {
			checkAndTriggerAutoSend(volumeName, destPath, targetName)
		}
	}

	writeJSON(w, http.StatusOK, map[string]string{"moved": safeName})
}

func handlePaste(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req PasteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Action != "copy" && req.Action != "cut" {
		writeError(w, http.StatusBadRequest, "invalid action (must be 'copy' or 'cut')")
		return
	}

	srcBase, srcErr := resolvePath(req.SrcVolume, req.SrcPath)
	destBase, destErr := resolvePath(req.DestVolume, req.DestPath)
	if srcErr != nil || destErr != nil {
		writeError(w, http.StatusBadRequest, "invalid path resolution")
		return
	}

	// Query active auto-send folder paths
	autoSendPaths, err := getAllAutoSendFolderPaths()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to query auto-send folders: "+err.Error())
		return
	}

	// Validate items before any copying/moving begins
	for _, item := range req.Items {
		if !item.IsDir {
			continue
		}

		srcRel := filepath.Join(req.SrcPath, item.Name)
		destRel := req.DestPath

		// Recursion check: cannot copy or move a folder inside itself or its children
		if req.SrcVolume == req.DestVolume {
			if srcRel == destRel || strings.HasPrefix(destRel, srcRel+"/") {
				writeError(w, http.StatusBadRequest, "cannot copy/move a folder into itself or its subfolders")
				return
			}
		}

		// Auto-send nesting check
		srcIsAutoSend := false
		for _, asp := range autoSendPaths {
			if asp == srcRel || strings.HasPrefix(asp, srcRel+"/") {
				srcIsAutoSend = true
				break
			}
		}

		destIsAutoSend := false
		for _, asp := range autoSendPaths {
			if asp == destRel || strings.HasPrefix(destRel, asp+"/") {
				destIsAutoSend = true
				break
			}
		}

		if req.Action == "cut" && srcIsAutoSend && destIsAutoSend {
			writeError(w, http.StatusBadRequest, "cannot move an auto-send folder into another auto-send folder")
			return
		}
	}

	for _, item := range req.Items {
		srcPath := filepath.Join(srcBase, item.Name)

		// Find a unique name in destination directory
		uniqueName := resolveUniqueFilename(destBase, item.Name)
		destPath := filepath.Join(destBase, uniqueName)

		if req.Action == "cut" {
			// Move file or folder
			if err := os.Rename(srcPath, destPath); err != nil {
				writeError(w, http.StatusInternalServerError, "failed to move item: "+err.Error())
				return
			}

			// If it's a directory, update any auto-send folders
			if item.IsDir {
				srcRel := filepath.Join(req.SrcPath, item.Name)
				destRel := filepath.Join(req.DestPath, uniqueName)

				// 1. Update the directory itself
				updateAutoSendFolderPath(srcRel, destRel)

				// 2. Update nested directories that match prefix (recursive folder moves)
				prefix := srcRel + "/%"
				updateAutoSendFolderPrefix(destRel, len(srcRel)+1, prefix)
			}
		} else {
			// Copy file or folder
			if item.IsDir {
				if err := copyDir(srcPath, destPath); err != nil {
					writeError(w, http.StatusInternalServerError, "failed to copy directory: "+err.Error())
					return
				}
			} else {
				if err := copyFile(srcPath, destPath); err != nil {
					writeError(w, http.StatusInternalServerError, "failed to copy file: "+err.Error())
					return
				}
			}
		}
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "success"})
}
