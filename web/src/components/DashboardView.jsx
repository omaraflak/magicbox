import React, { useState } from 'react';

function AppCard({ app, user, onStartApp, onStopApp, onUninstallApp, onRotateToken, onRebuildApp, isRebuilding, isUninstalling }) {
    const [menuOpen, setMenuOpen] = useState(false);

    let statusClass = 'status-stopped';
    if (app.status === 'running') statusClass = 'status-running';
    if (app.status === 'error') statusClass = 'status-error';
    if (app.status === 'installing') statusClass = 'status-installing';

    const appTitle = app.app_id.split('.').pop();
    const appUrl = app.host 
        ? `${window.location.protocol}//${app.host}/` 
        : `/u/${user?.username}/${app.route_slug}/`;

    return (
        <div className="card app-card animate-fade-in">
            <div className="app-info-block">
                <div className="app-icon-badge">📦</div>
                <div className="app-meta">
                    <div className="app-name-row">
                        <span className="app-title" style={{ textTransform: 'capitalize' }}>{appTitle}</span>
                        <span className="app-version">v{app.version || '1.0.0'}</span>
                    </div>
                    <span className="app-slug">{app.app_id}</span>
                </div>
                <div className={`status-indicator ${statusClass}`}>{app.status}</div>
            </div>
            
            <div className="app-actions-row" style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginTop: 'auto', borderTop: '1px solid var(--border-color)', paddingTop: '16px' }}>
                {/* Primary Action */}
                <div className="action-buttons">
                    {app.status === 'running' ? (
                        <a href={appUrl} target="_blank" rel="noreferrer" className="btn btn-primary btn-sm">
                            Open App
                        </a>
                    ) : (
                        <button 
                            className="btn btn-primary btn-sm" 
                            onClick={() => onStartApp(app.id)} 
                            disabled={app.status === 'installing'}
                        >
                            Start
                        </button>
                    )}
                </div>

                {/* Dropdown Options Menu */}
                <div className="card-menu-container">
                    <button className="btn btn-secondary btn-sm" onClick={() => setMenuOpen(!menuOpen)}>
                        Options ▾
                    </button>
                    
                    {menuOpen && (
                        <>
                            {/* Full-screen invisible overlay to handle clicks outside the menu */}
                            <div 
                                style={{ position: 'fixed', top: 0, left: 0, right: 0, bottom: 0, zIndex: 98, background: 'transparent' }} 
                                onClick={() => setMenuOpen(false)} 
                            />
                            <div className="menu-dropdown" style={{ zIndex: 99 }} onClick={() => setMenuOpen(false)}>
                                {app.status === 'running' && (
                                    <button className="menu-item" onClick={() => onStopApp(app.id)}>
                                        🛑 Stop App
                                    </button>
                                )}
                                <button className="menu-item" onClick={() => onRebuildApp(app.id)} title="Pull latest image and restart container">
                                    ✨ Update App
                                </button>
                                <button className="menu-item" onClick={() => onRotateToken(app.id)}>
                                    🔑 Rotate API Token
                                </button>
                                <button className="menu-item menu-item-danger" onClick={() => onUninstallApp(app.id)}>
                                    🗑️ Uninstall App
                                </button>
                            </div>
                        </>
                    )}
                </div>
            </div>
            
            {(isRebuilding || isUninstalling || app.status === 'installing') && (
                <div style={{
                    position: 'absolute',
                    top: 0,
                    left: 0,
                    right: 0,
                    bottom: 0,
                    background: 'rgba(255, 255, 255, 0.8)',
                    backdropFilter: 'blur(4px)',
                    display: 'flex',
                    flexDirection: 'column',
                    justifyContent: 'center',
                    alignItems: 'center',
                    zIndex: 10,
                    borderRadius: 'var(--radius-lg)',
                    color: 'var(--text-primary)',
                }}>
                    <div className="spinner" style={{ width: '32px', height: '32px', border: '3px solid rgba(0, 0, 0, 0.1)', borderTopColor: isUninstalling ? 'var(--accent-error)' : 'var(--accent-cyan)' }}></div>
                    <span style={{ marginTop: '12px', fontSize: '0.8rem', fontWeight: 500 }}>
                        {isUninstalling ? 'Uninstalling...' : app.status === 'installing' ? 'Installing...' : 'Updating...'}
                    </span>
                </div>
            )}
        </div>
    );
}

export default function DashboardView({ 
    apps, 
    user, 
    onStartApp, 
    onStopApp, 
    onUninstallApp, 
    onRotateToken, 
    onRebuildApp,
    rebuildingAppId,
    uninstallingAppId,
    onOpenInstallModal 
}) {
    return (
        <div className="dashboard-layout" style={{ padding: '24px 0' }}>
            <main className="main-content animate-fade-in">
                <div className="section-header" style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '24px' }}>
                    <h2>Installed Applications</h2>
                    <button className="btn btn-primary" onClick={onOpenInstallModal}>Install App</button>
                </div>

                {apps.length === 0 ? (
                    <div className="empty-state">
                        <div className="empty-icon" style={{ fontSize: '48px', marginBottom: '16px' }}>📦</div>
                        <h3>No applications installed yet</h3>
                        <p style={{ color: 'var(--text-muted)', marginBottom: '20px' }}>
                             Get started by installing an application from a manifest definition.
                        </p>
                        <button className="btn btn-secondary" onClick={onOpenInstallModal}>Install First App</button>
                    </div>
                ) : (
                    <div className="apps-grid">
                        {apps.map(app => (
                            <AppCard 
                                key={app.id}
                                app={app}
                                user={user}
                                onStartApp={onStartApp}
                                onStopApp={onStopApp}
                                onUninstallApp={onUninstallApp}
                                onRotateToken={onRotateToken}
                                onRebuildApp={onRebuildApp}
                                isRebuilding={rebuildingAppId === app.id}
                                isUninstalling={uninstallingAppId === app.id}
                            />
                        ))}
                    </div>
                )}
            </main>
        </div>
    );
}
