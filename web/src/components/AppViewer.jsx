import React from 'react';

export default function AppViewer({ app, user, onBack }) {
    const appUrl = app.host 
        ? `${window.location.protocol}//${app.host}/` 
        : `/u/${user?.username}/${app.route_slug}/`;

    return (
        <div className="app-viewer-container" style={{ display: 'flex', flexDirection: 'column', width: '100vw', height: '100vh', background: 'var(--bg-primary)' }}>
            <div className="app-viewer-header" style={{
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'space-between',
                padding: '0 24px',
                height: '60px',
                borderBottom: '1px solid var(--border-color)',
                background: 'rgba(15, 15, 20, 0.7)',
                backdropFilter: 'blur(10px)',
                zIndex: 10
            }}>
                <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
                    <span style={{ fontSize: '1.2rem' }}>📦</span>
                    <span style={{ fontWeight: 600, textTransform: 'capitalize', color: 'var(--text-primary)' }}>{app.name || app.app_id.split('.').pop()}</span>
                    <span style={{ fontSize: '0.8rem', color: 'var(--text-muted)', background: 'rgba(255,255,255,0.05)', padding: '2px 6px', borderRadius: '4px' }}>v{app.version || '1.0.0'}</span>
                </div>
                <button 
                    className="btn btn-secondary btn-sm"
                    onClick={onBack}
                    style={{ display: 'flex', alignItems: 'center', gap: '6px' }}
                >
                    🎛️ Return to Dashboard
                </button>
            </div>
            <iframe 
                src={appUrl} 
                title={app.name || app.app_id}
                style={{ 
                    flex: 1, 
                    width: '100%', 
                    height: 'calc(100vh - 60px)', 
                    border: 'none', 
                    background: '#000' 
                }} 
            />
        </div>
    );
}
