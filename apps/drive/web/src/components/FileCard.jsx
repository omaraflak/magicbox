import { getFileIcon, formatFileSize, formatDate } from '../utils/format';
import { getFileUrl } from '../utils/api';

export default function FileCard({ file, volume, path, onFolderClick, onContextMenu, viewMode }) {
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
        className="file-card list-row"
        onClick={handleCardClick}
        onContextMenu={onContextMenu}
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
      </div>
    );
  }

  // 2. Grid View - Folder Card Layout
  if (file.is_dir) {
    return (
      <div 
        className="file-card folder-card"
        onClick={handleCardClick}
        onContextMenu={onContextMenu}
        style={{ cursor: 'pointer' }}
      >
        <div className="folder-icon" style={{ fontSize: '1.5rem', marginRight: '8px' }}>📁</div>
        <div className="folder-name" title={file.name} style={{ fontWeight: 500, fontSize: '0.875rem', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap', flex: 1 }}>
          {file.name}
        </div>
      </div>
    );
  }

  // 3. Grid View - File Card Layout (with large square preview)
  return (
    <div 
      className="file-card grid-file-card"
      onContextMenu={onContextMenu}
    >
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
    </div>
  );
}
