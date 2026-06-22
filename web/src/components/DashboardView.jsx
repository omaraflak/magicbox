import React, { useState, useEffect, useRef } from 'react';

function AppCard({ app, user, onStartApp, onStopApp, onUninstallApp, onRotateToken, onRebuildApp, isRebuilding, isUninstalling, isStarting, isStopping }) {
    const [menuOpen, setMenuOpen] = useState(false);
    const menuRef = useRef(null);

    const isInstalling = app.status === 'installing';
    const isTransitioning = isInstalling || isRebuilding || isUninstalling || isStarting || isStopping;

    let statusText = app.status;
    let statusClass = `status-${app.status}`;
    
    if (isRebuilding) {
        statusText = 'updating';
        statusClass = 'status-updating';
    } else if (isUninstalling) {
        statusText = 'uninstalling';
        statusClass = 'status-uninstalling';
    } else if (isStarting) {
        statusText = 'starting';
        statusClass = 'status-starting';
    } else if (isStopping) {
        statusText = 'stopping';
        statusClass = 'status-stopping';
    }

    // Hide dropdown on outside click
    useEffect(() => {
        if (!menuOpen) return;
        const handleOutsideClick = (e) => {
            if (menuRef.current && !menuRef.current.contains(e.target)) {
                setMenuOpen(false);
            }
        };
        document.addEventListener('mousedown', handleOutsideClick);
        return () => document.removeEventListener('mousedown', handleOutsideClick);
    }, [menuOpen]);

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
                <div className={`status-indicator ${statusClass}`}>
                    {isTransitioning && (
                        <div className="spinner-sm" style={{ marginRight: '4px', width: '12px', height: '12px', borderWidth: '1.5px' }} />
                    )}
                    {statusText}
                </div>
            </div>
            
            <div className="app-actions-row" style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginTop: 'auto', borderTop: '1px solid var(--border-color)', paddingTop: '16px' }}>
                {/* Primary Action */}
                <div className="action-buttons">
                    {app.status === 'running' ? (
                        <a 
                            href={appUrl} 
                            target="_blank" 
                            rel="noreferrer" 
                            className={`btn btn-primary btn-sm ${isTransitioning ? 'disabled' : ''}`}
                            style={isTransitioning ? { pointerEvents: 'none', opacity: 0.5 } : {}}
                        >
                            Open App
                        </a>
                    ) : (
                        <button 
                            className="btn btn-primary btn-sm" 
                            onClick={() => onStartApp(app.id)} 
                            disabled={isTransitioning}
                        >
                            Start
                        </button>
                    )}
                </div>

                {/* Dropdown Options Menu */}
                <div className="card-menu-container" ref={menuRef}>
                    <button 
                        className="btn btn-secondary btn-sm" 
                        onClick={() => setMenuOpen(!menuOpen)}
                        disabled={isTransitioning}
                    >
                        Options ▾
                    </button>
                    
                    {menuOpen && (
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
                    )}
                </div>
            </div>
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
    startingAppId,
    stoppingAppId,
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
                                isStarting={startingAppId === app.id}
                                isStopping={stoppingAppId === app.id}
                            />
                        ))}
                    </div>
                )}
            </main>
        </div>
    );
}
