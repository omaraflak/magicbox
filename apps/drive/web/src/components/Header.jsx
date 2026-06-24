import { useState } from 'react';

export default function Header({ username, searchQuery, onSearchChange, activeTransfersCount }) {
  const [focused, setFocused] = useState(false);

  return (
    <header className="header">
      <style>{`
        @keyframes spin {
          from { transform: rotate(0deg); }
          to { transform: rotate(360deg); }
        }
      `}</style>
      <div className="header-left">
        <div className="header-logo">
          <span className="header-logo-icon">✦</span>
          <h1 className="header-title">
            <span className="gradient-text">Magic Drive</span>
          </h1>
        </div>
      </div>

      <div className={`header-search ${focused ? 'focused' : ''}`}>
        <span className="search-icon">🔍</span>
        <input
          type="text"
          placeholder="Search files..."
          value={searchQuery}
          onChange={(e) => onSearchChange(e.target.value)}
          onFocus={() => setFocused(true)}
          onBlur={() => setFocused(false)}
        />
        {searchQuery && (
          <button
            className="search-clear"
            onClick={() => onSearchChange('')}
            aria-label="Clear search"
          >
            ✕
          </button>
        )}
      </div>

      <div className="header-right" style={{ display: 'flex', alignItems: 'center', gap: '16px' }}>
        {activeTransfersCount > 0 && (
          <div className="sync-status" style={{ display: 'flex', alignItems: 'center', gap: '6px', fontSize: '0.8rem', color: 'var(--primary-color)', background: 'rgba(52, 152, 219, 0.05)', padding: '4px 10px', borderRadius: '12px', border: '1px solid rgba(52, 152, 219, 0.2)' }}>
            <span style={{ display: 'inline-block', animation: 'spin 2s linear infinite' }}>🔄</span>
            <span>Syncing {activeTransfersCount} file{activeTransfersCount === 1 ? '' : 's'}...</span>
          </div>
        )}
        {username && (
          <div className="user-badge">
            <div className="user-avatar">
              {username.charAt(0).toUpperCase()}
            </div>
            <span className="user-id">{username}</span>
          </div>
        )}
      </div>
    </header>
  );
}
