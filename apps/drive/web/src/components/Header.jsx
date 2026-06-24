import { useState } from 'react';

export default function Header({ username, searchQuery, onSearchChange, activeTransfersCount }) {
  const [focused, setFocused] = useState(false);

  return (
    <header className="header">
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
