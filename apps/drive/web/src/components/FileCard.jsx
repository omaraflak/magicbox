import { useState } from 'react';
import { downloadFile, deleteFile, getFileUrl } from '../utils/api';
import { getFileIcon, formatFileSize, formatDate } from '../utils/format';

export default function FileCard({ file, volume, path, onFolderClick, onDelete, viewMode }) {
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

  const isImage = (filename) => {
    const ext = filename.split('.').pop().toLowerCase();
    return ['jpg', 'jpeg', 'png', 'gif', 'webp', 'svg', 'bmp'].includes(ext);
  };

  // 1. List View Row Layout
  if (viewMode === 'list') {
    return (
      <div 
        className={`file-card list-row ${deleting ? 'deleting' : ''}`}
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
              <button className="btn btn-danger btn-sm" onClick={handleDelete} disabled={deleting}>
                {deleting ? '...' : 'Confirm'}
              </button>
              <button className="btn btn-secondary btn-sm" onClick={handleCancelDelete}>
                Cancel
              </button>
            </div>
          ) : (
            <>
              {!file.is_dir && (
                <button className="btn btn-icon" onClick={handleDownload} title="Download">
                  ⬇
                </button>
              )}
              <button className="btn btn-icon btn-icon-danger" onClick={handleDelete} title="Delete">
                🗑
              </button>
            </>
          )}
        </div>
      </div>
    );
  }

  // 2. Grid View - Folder Card Layout
  if (file.is_dir) {
    return (
      <div 
        className={`file-card folder-card ${deleting ? 'deleting' : ''}`}
        onClick={handleCardClick}
        style={{ cursor: 'pointer' }}
      >
        <div className="folder-icon" style={{ fontSize: '1.5rem', marginRight: '4px' }}>📁</div>
        <div className="folder-name" title={file.name} style={{ fontWeight: 500, fontSize: '0.875rem', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap', flex: 1 }}>
          {file.name}
        </div>
        <div className="folder-actions" onClick={e => e.stopPropagation()}>
          {confirming ? (
            <div className="file-card-confirm">
              <button className="btn btn-danger btn-sm" onClick={handleDelete} disabled={deleting}>
                {deleting ? '...' : 'Confirm'}
              </button>
              <button className="btn btn-secondary btn-sm" onClick={handleCancelDelete}>
                Cancel
              </button>
            </div>
          ) : (
            <button className="btn btn-icon btn-icon-danger btn-sm" onClick={handleDelete} style={{ padding: '4px' }}>
              🗑
            </button>
          )}
        </div>
      </div>
    );
  }

  // 3. Grid View - File Card Layout (with large square preview)
  return (
    <div className={`file-card grid-file-card ${deleting ? 'deleting' : ''}`}>
      {/* Square Preview Box */}
      <div className="file-preview-area">
        {isImage(file.name) ? (
          <img 
            src={getFileUrl(volume, path, file.name)} 
            alt={file.name} 
            className="file-preview-image"
          />
        ) : (
          <div className="file-preview-placeholder">
            {getFileIcon(file.name, false)}
          </div>
        )}
      </div>

      {/* Title & Info */}
      <div className="file-card-details">
        <div className="file-card-name" title={file.name}>
          {file.name}
        </div>
        <div className="file-card-meta">
          <span className="file-card-size">{formatFileSize(file.size)}</span>
        </div>
      </div>

      {/* Actions */}
      <div className="file-card-actions">
        {confirming ? (
          <div className="file-card-confirm">
            <button className="btn btn-danger btn-sm" onClick={handleDelete} disabled={deleting}>
              {deleting ? '...' : 'Confirm'}
            </button>
            <button className="btn btn-secondary btn-sm" onClick={handleCancelDelete}>
              Cancel
            </button>
          </div>
        ) : (
          <>
            <button className="btn btn-icon" onClick={handleDownload} title="Download">
              ⬇
            </button>
            <button className="btn btn-icon btn-icon-danger" onClick={handleDelete} title="Delete">
              🗑
            </button>
          </>
        )}
      </div>
    </div>
  );
}
