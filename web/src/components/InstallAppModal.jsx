import React, { useState, useEffect } from 'react';
import Modal from './Modal';

function scopeToHumanReadable(scope) {
    if (scope === 'profile:read') return 'Read your basic user profile (username, user ID)';
    
    const parts = scope.split(':');
    if (parts[0] === 'shared' && parts.length === 3) {
        const folderName = parts[1].charAt(0).toUpperCase() + parts[1].slice(1);
        const access = parts[2] === 'rw' ? 'read & write' : 'read-only';
        return `Access your shared "${folderName}" folder (${access})`;
    }
    return scope;
}

export default function InstallAppModal({ isOpen, onClose, onInstall, error, loading }) {
    const [manifestContent, setManifestContent] = useState('');
    const [scopes, setScopes] = useState([]);

    // Clear state on open/close
    useEffect(() => {
        if (!isOpen) {
            setManifestContent('');
            setScopes([]);
        }
    }, [isOpen]);

    const handleContentChange = (e) => {
        const value = e.target.value;
        setManifestContent(value);

        if (!value.trim()) {
            setScopes([]);
            return;
        }

        try {
            const parsed = JSON.parse(value);
            if (parsed.required_scopes && Array.isArray(parsed.required_scopes)) {
                setScopes(parsed.required_scopes);
            } else {
                setScopes([]);
            }
        } catch (err) {
            // Ignored; parsed invalid json is fine while editing
            setScopes([]);
        }
    };

    const handleSubmit = (e) => {
        e.preventDefault();
        onInstall(manifestContent);
    };

    return (
        <Modal isOpen={isOpen} onClose={onClose} title="Install Application">
            <p className="modal-desc" style={{ color: 'var(--text-muted)', marginBottom: '16px' }}>
                Paste the third-party application manifest definition (JSON) to install it.
            </p>
            <form onSubmit={handleSubmit}>
                <div className="form-group">
                    <label htmlFor="manifest-content">App Manifest (JSON)</label>
                    <textarea 
                        id="manifest-content" 
                        rows="12" 
                        required
                        placeholder='{
  "app_id": "com.magicbox.drive",
  "name": "Magic Drive",
  "version": "1.0.0",
  "image": "docker.io/magicbox/drive:latest",
  "entry_port": 8080,
  "route_slug": "drive",
  "required_scopes": ["profile:read"],
  "volume_mounts": []
}'
                        value={manifestContent}
                        onChange={handleContentChange}
                    />
                </div>

                {scopes.length > 0 && (
                    <div className="permissions-consent" style={{ marginTop: '16px', padding: '16px', background: 'rgba(255, 255, 255, 0.03)', borderRadius: '8px' }}>
                        <h4 className="consent-title" style={{ color: 'var(--accent-violet)', marginBottom: '8px', fontSize: '14px', fontWeight: '600' }}>
                            Security Permissions Requested
                        </h4>
                        <p className="consent-desc" style={{ fontSize: '12px', color: 'var(--text-muted)', marginBottom: '12px' }}>
                            This application requested access to the following resources. Click install to confirm.
                        </p>
                        <ul className="consent-list" style={{ listStyle: 'none', paddingLeft: 0 }}>
                            {scopes.map((scope, idx) => (
                                <li className="consent-item" key={idx} style={{ fontSize: '13px', display: 'flex', alignItems: 'center', gap: '8px', margin: '4px 0' }}>
                                    <span className="consent-icon">🛡️</span>
                                    <span>{scopeToHumanReadable(scope)}</span>
                                </li>
                            ))}
                        </ul>
                    </div>
                )}

                {error && <div className="error-box" style={{ marginTop: '12px' }}>{error}</div>}
                
                <div className="modal-actions" style={{ display: 'flex', justifyContent: 'flex-end', gap: '12px', marginTop: '20px' }}>
                    <button type="button" className="btn btn-secondary" onClick={onClose} disabled={loading}>
                        Cancel
                    </button>
                    <button type="submit" className="btn btn-primary" disabled={loading}>
                        <span>{loading ? 'Downloading & Deploying...' : 'Install App'}</span>
                    </button>
                </div>
            </form>
        </Modal>
    );
}
