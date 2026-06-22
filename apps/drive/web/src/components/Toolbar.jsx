import React from 'react';

export default function Toolbar({ 
  currentPath, 
  onBreadcrumbClick, 
  fileCount, 
  onUploadClick, 
  onCreateFolderClick, 
  viewMode, 
  onViewModeChange,
  selectedCount,
  onToggleSelectAll
}) {
  return (
    <div className="toolbar">
      <div className="toolbar-left" style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
        <h2 className="toolbar-title" style={{ display: 'flex', alignItems: 'center', gap: '8px', fontSize: '1.25rem', fontWeight: 600, margin: 0 }}>
          <span 
            style={{ cursor: 'pointer', color: currentPath ? 'var(--text-muted)' : 'var(--text-primary)' }} 
            onClick={() => onBreadcrumbClick(-1)}
          >
            My Storage
          </span>
          {currentPath && currentPath.split('/').map((seg, idx) => {
            const segments = currentPath.split('/');
            const isLast = idx === segments.length - 1;
            return (
              <React.Fragment key={idx}>
                <span style={{ color: 'var(--text-muted)', fontWeight: 300 }}>/</span>
                <span 
                  style={{ 
                    cursor: 'pointer', 
                    color: isLast ? 'var(--text-primary)' : 'var(--text-muted)' 
                  }} 
                  onClick={() => onBreadcrumbClick(idx)}
                >
                  {seg}
                </span>
              </React.Fragment>
            );
          })}
        </h2>
        <span className="toolbar-count" style={{ marginLeft: '12px', background: 'rgba(255,255,255,0.05)', padding: '2px 8px', borderRadius: '12px', fontSize: '0.75rem' }}>
          {selectedCount > 0 ? `${selectedCount} of ${fileCount} selected` : `${fileCount} ${fileCount === 1 ? 'file' : 'files'}`}
        </span>
      </div>

      <div className="toolbar-right">
        <div className="toolbar-view-toggle">
          <button
            className={`btn btn-icon ${viewMode === 'grid' ? 'active' : ''}`}
            onClick={() => onViewModeChange('grid')}
            title="Grid view"
          >
            ▦
          </button>
          <button
            className={`btn btn-icon ${viewMode === 'list' ? 'active' : ''}`}
            onClick={() => onViewModeChange('list')}
            title="List view"
          >
            ☰
          </button>
        </div>
        <button className="btn btn-secondary" onClick={onCreateFolderClick} style={{ marginRight: '8px' }}>
          New Folder
        </button>
        <button className="btn btn-primary" onClick={onUploadClick}>
          <span className="btn-icon-text">+</span> Upload
        </button>
      </div>
    </div>
  );
}
