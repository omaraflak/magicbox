import React from 'react';

export default function DashboardView({ 
    apps, 
    user, 
    onStartApp, 
    onStopApp, 
    onUninstallApp, 
    onRotateToken, 
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
                        {apps.map(app => {
                            let statusClass = 'status-stopped';
                            if (app.status === 'running') statusClass = 'status-running';
                            if (app.status === 'error') statusClass = 'status-error';
                            if (app.status === 'installing') statusClass = 'status-installing';

                            const appTitle = app.app_id.split('.').pop();
                            const appUrl = app.host 
                                ? `${window.location.protocol}//${app.host}/` 
                                : `/u/${user?.username}/${app.route_slug}/`;

                            return (
                                <div className="card app-card animate-fade-in" key={app.id}>
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
                                    
                                    <div className="app-actions-row">
                                        <div className="action-buttons">
                                            {app.status === 'running' ? (
                                                <button className="btn btn-secondary btn-sm" onClick={() => onStopApp(app.id)}>Stop</button>
                                            ) : (
                                                <button className="btn btn-primary btn-sm" onClick={() => onStartApp(app.id)} disabled={app.status === 'installing'}>
                                                    Start
                                                </button>
                                            )}
                                            <button className="btn btn-danger btn-sm" onClick={() => onUninstallApp(app.id)}>Uninstall</button>
                                        </div>
                                        <div className="action-buttons">
                                            {app.status === 'running' && (
                                                <a href={appUrl} target="_blank" rel="noreferrer" className="btn btn-secondary btn-sm">Open App</a>
                                            )}
                                            <button className="btn btn-secondary btn-sm" onClick={() => onRotateToken(app.id)} title="Rotate API Token">
                                                🔑
                                            </button>
                                        </div>
                                    </div>
                                </div>
                            );
                        })}
                    </div>
                )}
            </main>
        </div>
    );
}
