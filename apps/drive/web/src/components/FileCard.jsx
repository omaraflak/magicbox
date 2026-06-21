import { useState } from 'react';
import { downloadFile, deleteFile } from '../utils/api';
import { getFileIcon, formatFileSize, formatDate } from '../utils/format';

export default function FileCard({ file, volume, path, onFolderClick, onDelete }) {
  const [confirming, setConfirming] = useState(false);
  const [deleting, setDeleting] = useState(false);

  const handleDownload = async (e) => {
    e.stopPropagation();
    try {
      await downloadFile(volume, path, file.name);
    } catch (err) {
      console.error('Download error:', err);
    }
  };

  const handleDelete = async (e) => {
    e.stopPropagation();
    if (!confirming) {
      setConfirming(true);
      return;
    }
    setDeleting(true);
    try {
      await deleteFile(volume, path, file.name);
      onDelete?.();
    } catch (err) {
      console.error('Delete error:', err);
    } finally {
      setDeleting(false);
      setConfirming(false);
    }
  };

  const handleCancelDelete = (e) => {
    e.stopPropagation();
    setConfirming(false);
  };

  const handleCardClick = () => {
    if (file.is_dir && onFolderClick) {
      onFolderClick();
    }
  };

  return (
    <div 
      className={`file-card ${deleting ? 'deleting' : ''}`}
      onClick={handleCardClick}
      style={{ cursor: file.is_dir ? 'pointer' : 'default' }}
    >
      <div className="file-card-icon">
        {getFileIcon(file.name, file.is_dir)}
      </div>
      <div className="file-card-info">
        <div className="file-card-name" title={file.name}>
          {file.name}
        </div>
        <div className="file-card-meta">
          {!file.is_dir && (
            <span className="file-card-size">{formatFileSize(file.size)}</span>
          )}
          {file.is_dir && <span className="file-card-size">Folder</span>}
          {file.modified_at && (
            <span className="file-card-date">{formatDate(file.modified_at)}</span>
          )}
        </div>
      </div>

      <div className="file-card-actions">
        {confirming ? (
          <div className="file-card-confirm">
            <button
              className="btn btn-danger btn-sm"
              onClick={handleDelete}
              disabled={deleting}
            >
              {deleting ? '...' : 'Confirm'}
            </button>
            <button
              className="btn btn-secondary btn-sm"
              onClick={handleCancelDelete}
            >
              Cancel
            </button>
          </div>
        ) : (
          <>
            {!file.is_dir && (
              <button
                className="btn btn-icon"
                onClick={handleDownload}
                title="Download"
              >
                ⬇
              </button>
            )}
            <button
              className="btn btn-icon btn-icon-danger"
              onClick={handleDelete}
              title="Delete"
            >
              🗑
            </button>
          </>
        )}
      </div>
    </div>
  );
}
