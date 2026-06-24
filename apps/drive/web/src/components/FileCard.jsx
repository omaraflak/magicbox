import { getFileIcon, formatFileSize, formatDate } from '../utils/format';
import { getFileUrl } from '../utils/api';

export default function FileCard({ 
  file, 
  volume, 
  path, 
  selected, 
  onClick,
  onDoubleClick,
  onContextMenu,
  onDragStart,
  onDragEnd,
  dropHandlers,
  viewMode 
}) {
  
  const isImage = (filename) => {
    const ext = filename.split('.').pop().toLowerCase();
    return ['jpg', 'jpeg', 'png', 'gif', 'webp', 'svg', 'bmp'].includes(ext);
  };

  // 1. List View Row Layout
  if (viewMode === 'list') {
    return (
      <div 
        className={`file-card list-row ${selected ? 'selected' : ''}`}
        onClick={onClick}
        onDoubleClick={onDoubleClick}
        onContextMenu={onContextMenu}
        draggable
        onDragStart={onDragStart}
        onDragEnd={onDragEnd}
        {...dropHandlers}
        data-name={file.name}
        style={{ cursor: file.is_dir ? 'pointer' : 'default' }}
      >
        <div className="file-card-icon">
          {getFileIcon(file.name, file.is_dir)}
        </div>
        <div className="file-card-info">
          <div className="file-card-name" title={file.display_name || file.name}>
            {file.display_name || file.name}
          </div>
          <div className="file-card-meta">
            {file.original_path && (
              <span className="file-card-path" style={{ color: 'var(--text-muted)', marginRight: '8px' }}>
                Original Path: {file.original_path || '/'}
              </span>
            )}
            {file.deleted_at && (
              <span className="file-card-deleted" style={{ color: 'var(--text-muted)', marginRight: '8px' }}>
                Deleted: {new Date(file.deleted_at).toLocaleDateString()}
              </span>
            )}
            {!file.is_dir && (
              <span className="file-card-size">{formatFileSize(file.size)}</span>
            )}
            {file.is_dir && <span className="file-card-size">Folder</span>}
            {!file.deleted_at && file.modified_at && (
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
        className={`file-card folder-card ${selected ? 'selected' : ''}`}
        onClick={onClick}
        onDoubleClick={onDoubleClick}
        onContextMenu={onContextMenu}
        draggable
        onDragStart={onDragStart}
        onDragEnd={onDragEnd}
        {...dropHandlers}
        data-name={file.name}
        style={{ cursor: 'pointer' }}
      >
        <div className="folder-icon" style={{ fontSize: '1.5rem', marginRight: '8px' }}>📁</div>
        <div className="folder-name" title={file.display_name || file.name} style={{ fontWeight: 500, fontSize: '0.875rem', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap', flex: 1 }}>
          {file.display_name || file.name}
        </div>
      </div>
    );
  }

  // 3. Grid View - File Card Layout (with large square preview)
  return (
    <div 
      className={`file-card grid-file-card ${selected ? 'selected' : ''}`}
      onClick={onClick}
      onDoubleClick={onDoubleClick}
      onContextMenu={onContextMenu}
      draggable
      onDragStart={onDragStart}
      onDragEnd={onDragEnd}
      data-name={file.name}
    >
      {/* Square Preview Box */}
      <div className="file-preview-area">
        {isImage(file.name) ? (
          <img 
            src={getFileUrl(volume, path, file.name)} 
            alt={file.display_name || file.name} 
            className="file-preview-image"
            draggable="false"
          />
        ) : (
          <div className="file-preview-placeholder">
            {getFileIcon(file.name, false)}
          </div>
        )}
      </div>

      {/* Title & Info */}
      <div className="file-card-details">
        <div className="file-card-name" title={file.display_name || file.name}>
          {file.display_name || file.name}
        </div>
        <div className="file-card-meta">
          <span className="file-card-size">{formatFileSize(file.size)}</span>
        </div>
      </div>
    </div>
  );
}
