import { useState, useRef, useCallback } from 'react';
import { uploadFiles } from '../utils/api';

export default function DropZone({ volume, path, onUploadComplete, children, uploadRef, onContextMenu }) {
  const [isDragging, setIsDragging] = useState(false);
  const [uploading, setUploading] = useState(false);
  const [progress, setProgress] = useState(0);
  const [activeUploads, setActiveUploads] = useState({ completed: 0, total: 0 });
  const fileInputRef = useRef(null);
  const dragCounter = useRef(0);

  // Expose trigger method via uploadRef
  if (uploadRef) {
    uploadRef.current = {
      trigger: () => fileInputRef.current?.click(),
    };
  }

  const handleUpload = useCallback(async (files) => {
    if (!files || files.length === 0) return;
    const fileArray = Array.from(files);
    const total = fileArray.length;

    setUploading(true);
    setProgress(0);
    setActiveUploads({ completed: 0, total });

    const progressMap = {};
    let completedCount = 0;
    let nextIndex = 0;

    const updateProgress = (filename, percent) => {
      progressMap[filename] = percent;
      const totalProgress = fileArray.reduce((acc, f) => acc + (progressMap[f.name] || 0), 0);
      setProgress(Math.round(totalProgress / total));
    };

    const uploadWorker = async () => {
      while (nextIndex < fileArray.length) {
        const i = nextIndex++;
        const file = fileArray[i];
        try {
          await uploadFiles(volume, path, [file], (pct) => {
            updateProgress(file.name, pct);
          });
          completedCount++;
          setActiveUploads({ completed: completedCount, total });
          // Trigger file list refresh IMMEDIATELY so the user sees files appear one by one!
          onUploadComplete?.();
        } catch (err) {
          console.error(`Failed to upload ${file.name}:`, err);
        }
      }
    };

    const workers = [];
    const concurrency = Math.min(4, total);
    for (let w = 0; w < concurrency; w++) {
      workers.push(uploadWorker());
    }
    await Promise.all(workers);

    setUploading(false);
    setProgress(0);
    setActiveUploads({ completed: 0, total: 0 });
  }, [volume, path, onUploadComplete]);

  const handleDragEnter = useCallback((e) => {
    if (volume !== 'storage') return;
    const isFileDrag = e.dataTransfer.types && e.dataTransfer.types.includes('Files');
    if (!isFileDrag) return;

    e.preventDefault();
    e.stopPropagation();
    dragCounter.current += 1;
    if (dragCounter.current === 1) {
      setIsDragging(true);
    }
  }, [volume]);

  const handleDragLeave = useCallback((e) => {
    if (volume !== 'storage') return;
    const isFileDrag = e.dataTransfer.types && e.dataTransfer.types.includes('Files');
    if (!isFileDrag) return;

    e.preventDefault();
    e.stopPropagation();
    dragCounter.current -= 1;
    if (dragCounter.current === 0) {
      setIsDragging(false);
    }
  }, [volume]);

  const handleDragOver = useCallback((e) => {
    if (volume !== 'storage') return;
    const isFileDrag = e.dataTransfer.types && e.dataTransfer.types.includes('Files');
    if (!isFileDrag) return;

    e.preventDefault();
    e.stopPropagation();
  }, [volume]);

  const handleDrop = useCallback((e) => {
    if (volume !== 'storage') return;
    const isFileDrag = e.dataTransfer.types && e.dataTransfer.types.includes('Files');
    if (!isFileDrag) return;

    e.preventDefault();
    e.stopPropagation();
    dragCounter.current = 0;
    setIsDragging(false);
    const files = e.dataTransfer.files;
    handleUpload(files);
  }, [volume, handleUpload]);

  const handleFileSelect = useCallback((e) => {
    handleUpload(e.target.files);
    e.target.value = '';
  }, [handleUpload]);

  return (
    <div
      className="dropzone"
      onDragEnter={handleDragEnter}
      onDragLeave={handleDragLeave}
      onDragOver={handleDragOver}
      onDrop={handleDrop}
      onContextMenu={onContextMenu}
    >
      <input
        ref={fileInputRef}
        type="file"
        multiple
        className="dropzone-input"
        onChange={handleFileSelect}
      />

      {isDragging && (
        <div className="dropzone-overlay">
          <div className="dropzone-overlay-content">
            <span className="dropzone-overlay-icon">📂</span>
            <span className="dropzone-overlay-text">Drop files here</span>
          </div>
        </div>
      )}

      {uploading && (
        <div className="upload-progress">
          <div className="upload-progress-bar">
            <div
              className="upload-progress-fill"
              style={{ width: `${progress}%` }}
            />
          </div>
          <span className="upload-progress-text">
            Uploading {activeUploads.completed} of {activeUploads.total} files ({progress}%)
          </span>
        </div>
      )}

      {children}
    </div>
  );
}
