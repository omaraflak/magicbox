import React, { useState, useEffect, useRef } from 'react';
import Badge from './Badge';

function AppCard({ app, user, hasUpdate, onStartApp, onStopApp, onUninstallApp, onRotateToken, onRebuildApp, onOpenApp, isRebuilding, isUninstalling, isStarting, isStopping }) {
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

    const appTitle = app.name || app.app_id.split('.').pop();
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
                        <span className="app-version">
                            v{app.version || '1.0.0'}
                            {hasUpdate && (
                                <span 
                                    style={{ 
                                        color: 'var(--accent-warning)', 
                                        marginLeft: '6px', 
                                        fontWeight: 'bold', 
                                        display: 'inline-block'
                                    }} 
                                    title="Update available"
                                >
                                    ●
                                </span>
                            )}
                        </span>
                    </div>
                    <span className="app-slug">{app.app_id}</span>
                </div>
                <Badge type={statusText}>
                    {isTransitioning && (
                        <div className="spinner-sm" style={{ marginRight: '4px', width: '12px', height: '12px', borderWidth: '1.5px' }} />
                    )}
                    {statusText}
                </Badge>
            </div>
            
            <div className="app-actions-row" style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginTop: 'auto', borderTop: '1px solid var(--border-color)', paddingTop: '16px' }}>
                {/* Primary Action */}
                <div className="action-buttons" style={{ display: 'flex', gap: '8px', alignItems: 'center' }}>
                    {app.status === 'running' ? (
                        <button 
                            className={`btn btn-primary btn-sm ${isTransitioning ? 'disabled' : ''}`}
                            onClick={() => onOpenApp(app)}
                            disabled={isTransitioning}
                        >
                            Open App
                        </button>
                    ) : (
                        <button 
                            className="btn btn-primary btn-sm" 
                            onClick={() => onStartApp(app.id)} 
                            disabled={isTransitioning}
                        >
                            Start
                        </button>
                    )}

                    {hasUpdate && (
                        <button 
                            className="btn btn-secondary btn-sm" 
                            onClick={() => onRebuildApp(app.id)}
                            style={{ 
                                borderColor: 'var(--accent-warning)', 
                                color: 'var(--accent-warning)',
                                background: 'rgba(245, 158, 11, 0.05)'
                            }}
                            title="Install update"
                        >
                            ⚡ Update
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
    updates,
    onUpgradeCore,
    onStartApp, 
    onStopApp, 
    onUninstallApp, 
    onRotateToken, 
    onRebuildApp,
    rebuildingAppId,
    uninstallingAppId,
    startingAppId,
    stoppingAppId,
    onOpenInstallModal,
    onOpenApp
}) {
    return (
        <div className="dashboard-layout" style={{ padding: '24px 0' }}>
            <main className="main-content animate-fade-in">
                {updates?.core?.update_available && (
                    <div className="update-banner animate-fade-in" style={{
                        display: 'flex',
                        alignItems: 'center',
                        justifyContent: 'space-between',
                        padding: '16px 24px',
                        background: 'linear-gradient(135deg, rgba(99, 102, 241, 0.15) 0%, rgba(2, 132, 199, 0.15) 100%)',
                        border: '1px solid var(--accent-violet)',
                        borderRadius: 'var(--radius-lg)',
                        marginBottom: '24px',
                    }}>
                        <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
                            <span style={{ fontSize: '1.5rem' }}>✨</span>
                            <div>
                                <h4 style={{ fontWeight: 600, color: 'var(--text-primary)' }}>System Update Available</h4>
                                <p style={{ fontSize: '0.85rem', color: 'var(--text-muted)', marginTop: '2px' }}>
                                    A newer version of Magicbox OS is ready. Upgrade to apply the latest security patches and features.
                                </p>
                            </div>
                        </div>
                        <button 
                            className="btn btn-primary" 
                            onClick={() => onUpgradeCore(updates.core.image)}
                            style={{ padding: '8px 20px', fontSize: '0.85rem' }}
                        >
                            Upgrade System
                        </button>
                    </div>
                )}

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
                        {apps.map(app => {
                            const appUpdate = (updates?.apps || []).find(u => u.app_id === app.app_id);
                            const hasUpdate = appUpdate?.update_available;
                            return (
                                <AppCard 
                                    key={app.id} 
                                    app={app} 
                                    user={user}
                                    hasUpdate={hasUpdate}
                                    onStartApp={onStartApp}
                                    onStopApp={onStopApp}
                                    onUninstallApp={onUninstallApp}
                                    onRotateToken={onRotateToken}
                                    onRebuildApp={onRebuildApp}
                                    isRebuilding={rebuildingAppId === app.id}
                                    isUninstalling={uninstallingAppId === app.id}
                                    isStarting={startingAppId === app.id}
                                    isStopping={stoppingAppId === app.id}
                                    onOpenApp={onOpenApp}
                                />
                            );
                        })}
                    </div>
                )}
            </main>
        </div>
    );
}
