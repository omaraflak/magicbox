package main

import (
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// Injected OS variables
var (
	apiToken = os.Getenv("MAGICBOX_API_TOKEN")
	coreURL  = os.Getenv("MAGICBOX_CORE_URL")
	userID   = os.Getenv("MAGICBOX_USER_ID")
	appID    = os.Getenv("MAGICBOX_APP_ID")
)

func main() {
	log.Println("Starting Magic Drive app on port 8080...")

	http.HandleFunc("/", handleIndex)
	http.HandleFunc("/upload", handleUpload)
	http.HandleFunc("/download", handleDownload)
	http.HandleFunc("/delete", handleDelete)
	http.HandleFunc("/internal/magicbox-webhook", handleWebhook)

	log.Fatal(http.ListenAndServe(":8080", nil))
}

func handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	// Read directories
	photos, _ := listFiles("/data/shared/photos")
	docs, _ := listFiles("/data/shared/documents")
	state, _ := listFiles("/data/app_state")

	// Parse JWT info (manually decode JWT payload segment to keep container zero-dependency)
	jwtParsed := "Invalid Token"
	if parts := strings.Split(apiToken, "."); len(parts) == 3 {
		payloadSegment := parts[1]
		// Add padding if missing
		if rem := len(payloadSegment) % 4; rem > 0 {
			payloadSegment += strings.Repeat("=", 4-rem)
		}
		decoded, err := base64.URLEncoding.DecodeString(payloadSegment)
		if err == nil {
			jwtParsed = string(decoded)
		}
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Magic Drive</title>
    <link rel="preconnect" href="https://fonts.googleapis.com">
    <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
    <link href="https://fonts.googleapis.com/css2?family=Outfit:wght@300;400;500;600;700;800&display=swap" rel="stylesheet">
    <style>
        :root {
            --bg-primary: #09090e;
            --bg-card: rgba(255, 255, 255, 0.03);
            --border-color: rgba(255, 255, 255, 0.08);
            --accent-cyan: #06b6d4;
            --accent-violet: #8b5cf6;
            --text-primary: #f3f4f6;
            --text-muted: #9ca3af;
            --font-sans: 'Outfit', sans-serif;
        }
        * { box-sizing: border-box; margin: 0; padding: 0; }
        body {
            background-color: var(--bg-primary);
            color: var(--text-primary);
            font-family: var(--font-sans);
            padding: 40px;
            min-height: 100vh;
        }
        .container {
            max-width: 1200px;
            margin: 0 auto;
            display: flex;
            flex-direction: column;
            gap: 30px;
        }
        header {
            display: flex;
            align-items: center;
            justify-content: space-between;
            border-bottom: 1px solid var(--border-color);
            padding-bottom: 20px;
        }
        h1 {
            font-size: 2rem;
            background: linear-gradient(135deg, var(--accent-violet), var(--accent-cyan));
            -webkit-background-clip: text;
            -webkit-text-fill-color: transparent;
        }
        .meta-card {
            background: var(--bg-card);
            border: 1px solid var(--border-color);
            border-radius: 12px;
            padding: 20px;
            font-size: 0.9rem;
            line-height: 1.6;
        }
        .meta-card h3 { color: var(--accent-cyan); margin-bottom: 10px; }
        .meta-card code {
            font-family: monospace;
            background: rgba(0,0,0,0.3);
            padding: 2px 6px;
            border-radius: 4px;
            display: block;
            white-space: pre-wrap;
            margin-top: 5px;
        }
        .sections {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(350px, 1fr));
            gap: 24px;
        }
        .section {
            background: var(--bg-card);
            border: 1px solid var(--border-color);
            border-radius: 12px;
            padding: 24px;
            display: flex;
            flex-direction: column;
            gap: 20px;
        }
        .section h2 { font-size: 1.3rem; border-bottom: 1px solid var(--border-color); padding-bottom: 10px; }
        .file-list {
            display: flex;
            flex-direction: column;
            gap: 10px;
            min-height: 100px;
        }
        .file-item {
            display: flex;
            align-items: center;
            justify-content: space-between;
            background: rgba(255,255,255,0.02);
            border: 1px solid var(--border-color);
            padding: 10px 14px;
            border-radius: 8px;
            font-size: 0.9rem;
        }
        .file-name { font-weight: 500; }
        .file-actions { display: flex; gap: 10px; }
        .btn {
            padding: 6px 12px;
            font-size: 0.8rem;
            font-weight: 500;
            border-radius: 6px;
            cursor: pointer;
            border: none;
            text-decoration: none;
            transition: all 0.2s;
        }
        .btn-primary { background: var(--accent-cyan); color: #000; }
        .btn-secondary { background: rgba(255,255,255,0.08); color: var(--text-primary); border: 1px solid var(--border-color); }
        .btn-danger { background: rgba(239, 68, 68, 0.2); color: #f87171; border: 1px solid rgba(239, 68, 68, 0.4); }
        .btn:hover { opacity: 0.8; }
        .upload-form {
            display: flex;
            gap: 10px;
            margin-top: 10px;
        }
        .upload-form input[type="file"] {
            flex: 1;
            color: var(--text-muted);
            font-size: 0.85rem;
        }
    </style>
</head>
<body>
    <div class="container">
        <header>
            <div>
                <h1>Magic Drive App</h1>
                <p style="color: var(--text-muted); margin-top: 5px;">Sandbox application demonstrating multi-tenant isolated storage.</p>
            </div>
            <div class="btn btn-secondary">User: %s</div>
        </header>

        <div class="meta-card">
            <h3>Mother App Injected Context Telemetry</h3>
            <div><strong>MAGICBOX_CORE_URL:</strong> <code>%s</code></div>
            <div><strong>MAGICBOX_USER_ID:</strong> <code>%s</code></div>
            <div><strong>MAGICBOX_APP_ID:</strong> <code>%s</code></div>
            <div><strong>Decoded JWT Payload (MAGICBOX_API_TOKEN):</strong> <code>%s</code></div>
        </div>

        <div class="sections">
            <!-- Photos Section -->
            <div class="section">
                <h2>Photos shared volume (/data/shared/photos)</h2>
                <div class="file-list">
                    %s
                </div>
                <form class="upload-form" action="/upload?dir=photos" method="POST" enctype="multipart/form-data">
                    <input type="file" name="file" required>
                    <button class="btn btn-primary" type="submit">Upload</button>
                </form>
            </div>

            <!-- Documents Section -->
            <div class="section">
                <h2>Documents shared volume (/data/shared/documents)</h2>
                <div class="file-list">
                    %s
                </div>
                <form class="upload-form" action="/upload?dir=documents" method="POST" enctype="multipart/form-data">
                    <input type="file" name="file" required>
                    <button class="btn btn-primary" type="submit">Upload</button>
                </form>
            </div>

            <!-- Private Config Section -->
            <div class="section">
                <h2>Private App State (/data/app_state)</h2>
                <p style="font-size: 0.85rem; color: var(--text-muted);">This directory is completely private to this app container and is destroyed on uninstall.</p>
                <div class="file-list">
                    %s
                </div>
                <form class="upload-form" action="/upload?dir=state" method="POST" enctype="multipart/form-data">
                    <input type="file" name="file" required>
                    <button class="btn btn-primary" type="submit">Save File</button>
                </form>
            </div>
        </div>
    </div>
</body>
</html>`,
		escape(userID),
		escape(coreURL),
		escape(userID),
		escape(appID),
		escape(jwtParsed),
		renderFilesHTML("photos", photos),
		renderFilesHTML("documents", docs),
		renderFilesHTML("state", state),
	)
}

func handleUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	dirKey := r.URL.Query().Get("dir")
	uploadDir := ""
	switch dirKey {
	case "photos":
		uploadDir = "/data/shared/photos"
	case "documents":
		uploadDir = "/data/shared/documents"
	case "state":
		uploadDir = "/data/app_state"
	default:
		http.Error(w, "Invalid directory", http.StatusBadRequest)
		return
	}

	// Parse file
	r.ParseMultipartForm(10 << 20) // 10MB max
	file, handler, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Failed to parse file: "+err.Error(), http.StatusBadRequest)
		return
	}
	defer file.Close()

	destPath := filepath.Join(uploadDir, filepath.Base(handler.Filename))
	dest, err := os.OpenFile(destPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		http.Error(w, "Failed to create destination file: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer dest.Close()

	if _, err := io.Copy(dest, file); err != nil {
		http.Error(w, "Failed to write file: "+err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func handleDownload(w http.ResponseWriter, r *http.Request) {
	dirKey := r.URL.Query().Get("dir")
	filename := r.URL.Query().Get("file")
	
	uploadDir := ""
	switch dirKey {
	case "photos":
		uploadDir = "/data/shared/photos"
	case "documents":
		uploadDir = "/data/shared/documents"
	case "state":
		uploadDir = "/data/app_state"
	default:
		http.Error(w, "Invalid directory", http.StatusBadRequest)
		return
	}

	safePath := filepath.Join(uploadDir, filepath.Base(filename))
	
	// Double-check base path constraint
	if !strings.HasPrefix(safePath, uploadDir) {
		http.Error(w, "Access Denied", http.StatusForbidden)
		return
	}

	w.Header().Set("Content-Disposition", "attachment; filename="+filepath.Base(filename))
	http.ServeFile(w, r, safePath)
}

func handleDelete(w http.ResponseWriter, r *http.Request) {
	dirKey := r.URL.Query().Get("dir")
	filename := r.URL.Query().Get("file")

	uploadDir := ""
	switch dirKey {
	case "photos":
		uploadDir = "/data/shared/photos"
	case "documents":
		uploadDir = "/data/shared/documents"
	case "state":
		uploadDir = "/data/app_state"
	default:
		http.Error(w, "Invalid directory", http.StatusBadRequest)
		return
	}

	safePath := filepath.Join(uploadDir, filepath.Base(filename))
	if err := os.Remove(safePath); err != nil {
		http.Error(w, "Failed to delete: "+err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func handleWebhook(w http.ResponseWriter, r *http.Request) {
	// Simple webhook responder logging payloads
	body, _ := io.ReadAll(r.Body)
	log.Printf("Webhook received from app %s (user %s): %s",
		r.Header.Get("X-Magicbox-Source-App"),
		r.Header.Get("X-Magicbox-Source-User"),
		string(body),
	)
	w.WriteHeader(http.StatusOK)
}

// Helpers
func listFiles(dir string) ([]string, error) {
	var files []string
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			files = append(files, entry.Name())
		}
	}
	return files, nil
}

func renderFilesHTML(dirKey string, files []string) string {
	if len(files) == 0 {
		return `<div style="color: var(--text-muted); font-size: 0.85rem; padding: 10px 0;">No files present.</div>`
	}
	var html strings.Builder
	for _, f := range files {
		html.WriteString(fmt.Sprintf(`
			<div class="file-item">
				<span class="file-name">%s</span>
				<div class="file-actions">
					<a class="btn btn-secondary" href="/download?dir=%s&file=%s">Download</a>
					<a class="btn btn-danger" href="/delete?dir=%s&file=%s">✕</a>
				</div>
			</div>
		`, escape(f), dirKey, escape(f), dirKey, escape(f)))
	}
	return html.String()
}

func escape(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	s = strings.ReplaceAll(s, "'", "&#39;")
	return s
}
