export default function Sidebar({ volumes, activeVolume, onVolumeChange, fileCounts }) {
  return (
    <aside className="sidebar">
      <nav className="sidebar-nav">
        <div className="sidebar-section-label">Volumes</div>
        <ul className="sidebar-list">
          {volumes.map((vol) => (
            <li key={vol.id}>
              <button
                className={`sidebar-item ${activeVolume === vol.id ? 'active' : ''}`}
                onClick={() => onVolumeChange(vol.id)}
              >
                <span className="sidebar-item-icon">{vol.icon}</span>
                <span className="sidebar-item-name">{vol.name}</span>
                {fileCounts[vol.id] !== undefined && (
                  <span className="sidebar-item-count">{fileCounts[vol.id]}</span>
                )}
              </button>
            </li>
          ))}
        </ul>
      </nav>

      <div className="sidebar-footer">
        <div className="sidebar-storage">
          <div className="sidebar-storage-label">Storage</div>
          {volumes.map((vol) => (
            <div key={vol.id} className="sidebar-storage-item">
              <span className="sidebar-storage-icon">{vol.icon}</span>
              <span className="sidebar-storage-name">{vol.name}</span>
              <span className="sidebar-storage-count">
                {fileCounts[vol.id] ?? 0} files
              </span>
            </div>
          ))}
        </div>
      </div>
    </aside>
  );
}
