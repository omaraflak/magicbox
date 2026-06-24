export default function Sidebar({ activeVolume, onVolumeChange }) {
  const menuItems = [
    { id: 'storage', name: 'My Storage', icon: '💾' },
    { id: 'trash', name: 'Trash', icon: '🗑️' },
    { id: 'shares', name: 'History', icon: '🕒' },
  ];

  return (
    <aside className="sidebar">
      <nav className="sidebar-nav">
        <div className="sidebar-section-label">Magic Drive</div>
        <ul className="sidebar-list">
          {menuItems.map((item) => (
            <li key={item.id}>
              <button
                className={`sidebar-item ${activeVolume === item.id ? 'active' : ''}`}
                onClick={() => onVolumeChange(item.id)}
              >
                <span className="sidebar-item-icon">{item.icon}</span>
                <span className="sidebar-item-name">{item.name}</span>
              </button>
            </li>
          ))}
        </ul>
      </nav>
    </aside>
  );
}
