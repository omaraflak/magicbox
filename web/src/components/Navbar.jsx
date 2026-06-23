import React from 'react';

export default function Navbar({ title, user, onNavigate }) {
    return (
        <nav className="navbar">
            <div className="nav-brand" style={{ cursor: 'pointer' }} onClick={() => onNavigate('dashboard')}>
                <span className="brand-logo">✨</span>
                <span className="brand-name">{title}</span>
            </div>
            <div className="nav-actions">
                {user && (
                    <span 
                        className="user-badge clickable" 
                        onClick={() => onNavigate('settings', 'profile')}
                        style={{ cursor: 'pointer', display: 'inline-flex', alignItems: 'center', gap: '6px' }}
                    >
                        <span className="user-icon">👤</span> {user.username}
                    </span>
                )}
            </div>
        </nav>
    );
}
