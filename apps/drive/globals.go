package main

import (
	"database/sql"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"github.com/magicbox/core/sdk"
)

// Volume mapping: logical name → filesystem path

var volumes = map[string]string{
	"storage": "/data/shared/storage",
	"trash":   "/data/shared/storage/.trash",
}

// Environment variables injected by MagicBox

var (
	env      *sdk.Env
	coreURL  string
	apiToken string
	userID   string
	appID    string
)

const (
	maxMemoryBytes   = 32 << 20  // 32MB in-memory buffer
	maxRequestBytes  = 10 << 30  // 10GB max request body size
	maxZipVolumeSize = 200 << 20 // 200MB max split folder zip volume size
)

var cachedUsername string

var dbConn *sql.DB

var retryBackoff = 2 * time.Second

type FileInfo struct {
	Name         string    `json:"name"`
	DisplayName  string    `json:"display_name"`
	Size         int64     `json:"size"`
	ModifiedAt   time.Time `json:"modified_at"`
	IsDir        bool      `json:"is_dir"`
	OriginalPath string    `json:"original_path,omitempty"`
	DeletedAt    *string   `json:"deleted_at,omitempty"`
	IsAutoSend   bool      `json:"is_auto_send,omitempty"`
}

type ZipVolume struct {
	Index int      `json:"index"`
	Name  string   `json:"name"`
	Files []string `json:"files"`
	Size  int64    `json:"size"`
}

type DownloadPlan struct {
	Volumes []ZipVolume `json:"volumes"`
}

type zipItem struct {
	DiskPath string
	ZipPath  string
	Size     int64
}

type TransferMessage struct {
	Filename string `json:"filename"`
	Content  []byte `json:"content"`
	DestPath string `json:"dest_path,omitempty"`
}

type TransferRecord struct {
	ID          string `json:"id"`
	Filename    string `json:"filename"`
	Path        string `json:"path"`
	ContactID   string `json:"contact_id"`
	ContactName string `json:"contact_name"`
	Status      string `json:"status"`
	SentAt      string `json:"sent_at"`
}

type AutoSendTarget struct {
	ContactID   string `json:"contact_id"`
	ContactName string `json:"contact_name"`
}

type AutoSendFolderInfo struct {
	ID        string           `json:"id"`
	Path      string           `json:"path"`
	Targets   []AutoSendTarget `json:"targets"`
	CreatedAt string           `json:"created_at"`
}

type PasteItem struct {
	Name  string `json:"name"`
	IsDir bool   `json:"is_dir"`
}

type PasteRequest struct {
	Action     string      `json:"action"` // "copy" or "cut"
	SrcVolume  string      `json:"src_volume"`
	SrcPath    string      `json:"src_path"`
	DestVolume string      `json:"dest_volume"`
	DestPath   string      `json:"dest_path"`
	Items      []PasteItem `json:"items"`
}
