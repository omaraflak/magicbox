import { useState, useRef, useCallback } from 'react';
import { uploadFiles } from '../utils/api';

export default function DropZone({ volume, path, onUploadComplete, children, uploadRef, onContextMenu }) {
  const [isDragging, setIsDragging] = useState(false);
  const [uploading, setUploading] = useState(false);
  const [progress, setProgress] = useState(0);
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
    setUploading(true);
    setProgress(0);
    try {
      await uploadFiles(volume, path, files, (pct) => setProgress(pct));
      onUploadComplete?.();
    } catch (err) {
      console.error('Upload error:', err);
    } finally {
      setUploading(false);
      setProgress(0);
    }
  }, [volume, path, onUploadComplete]);

  const handleDragEnter = useCallback((e) => {
    e.preventDefault();
    e.stopPropagation();
    dragCounter.current += 1;
    if (dragCounter.current === 1) {
      setIsDragging(true);
    }
  }, []);

  const handleDragLeave = useCallback((e) => {
    e.preventDefault();
    e.stopPropagation();
    dragCounter.current -= 1;
    if (dragCounter.current === 0) {
      setIsDragging(false);
    }
  }, []);

  const handleDragOver = useCallback((e) => {
    e.preventDefault();
    e.stopPropagation();
  }, []);

  const handleDrop = useCallback((e) => {
    e.preventDefault();
    e.stopPropagation();
    dragCounter.current = 0;
    setIsDragging(false);
    const files = e.dataTransfer.files;
    handleUpload(files);
  }, [handleUpload]);

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
          <span className="upload-progress-text">Uploading... {progress}%</span>
        </div>
      )}

      {children}
    </div>
  );
}
