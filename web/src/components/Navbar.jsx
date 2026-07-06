import React from 'react';

export default function Navbar({ title, user, onNavigate, contactRequests = [] }) {
    const pendingCount = contactRequests.filter(r => r.direction === 'incoming').length;

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
                        style={{ cursor: 'pointer', display: 'inline-flex', alignItems: 'center', gap: '6px', position: 'relative' }}
                    >
                        <span className="user-icon">👤</span> {user.username}
                        {pendingCount > 0 && (
                            <span className="notification-bubble" style={{
                                position: 'absolute',
                                top: '-6px',
                                right: '-8px',
                            }}>
                                {pendingCount}
                            </span>
                        )}
                    </span>
                )}
            </div>
        </nav>
    );
}
