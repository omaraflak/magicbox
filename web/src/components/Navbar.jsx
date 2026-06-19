import React from 'react';

export default function Navbar({ title, user, onLogout, adminView, onToggleView }) {
    return (
        <nav className="navbar">
            <div className="nav-brand">
                <span className="brand-logo">{adminView ? '⚙️' : '✨'}</span>
                <span className="brand-name">{title}</span>
            </div>
            <div className="nav-actions">
                {user?.is_admin && (
                    adminView ? (
                        <button className="btn btn-secondary nav-btn" onClick={() => onToggleView('dashboard')}>
                            Back to Dashboard
                        </button>
                    ) : (
                        <button className="btn btn-secondary nav-btn" onClick={() => onToggleView('admin')}>
                            Admin Console
                        </button>
                    )
                )}
                {user && (
                    <>
                        <span className="user-badge">{user.username}</span>
                        <button className="btn btn-icon-only" onClick={onLogout} title="Log Out">
                            <span className="logout-icon">↩</span>
                        </button>
                    </>
                )}
            </div>
        </nav>
    );
}
