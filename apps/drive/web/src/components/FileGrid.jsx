import FileCard from './FileCard';

export default function FileGrid({ 
  files, 
  volume, 
  path, 
  searchQuery, 
  viewMode, 
  onFolderClick, 
  onDelete 
}) {
  const filtered = (files || [])
    .filter((f) => {
      if (!searchQuery) return true;
      return f.name.toLowerCase().includes(searchQuery.toLowerCase());
    })
    .sort((a, b) => {
      // Directories first
      if (a.is_dir && !b.is_dir) return -1;
      if (!a.is_dir && b.is_dir) return 1;
      // Then alphabetical
      return a.name.localeCompare(b.name);
    });

  if (filtered.length === 0 && searchQuery) {
    return (
      <div className="empty-search">
        <span className="empty-search-icon">🔍</span>
        <p>No files matching &ldquo;{searchQuery}&rdquo;</p>
      </div>
    );
  }

  return (
    <div className={`file-grid ${viewMode === 'list' ? 'list' : ''}`}>
      {filtered.map((file, index) => (
        <div
          key={file.name}
          className="file-grid-item"
          style={{ animationDelay: `${index * 0.03}s` }}
        >
          <FileCard 
            file={file} 
            volume={volume} 
            path={path} 
            onFolderClick={file.is_dir ? () => onFolderClick(file.name) : null} 
            onDelete={onDelete} 
            viewMode={viewMode}
          />
        </div>
      ))}
    </div>
  );
}
