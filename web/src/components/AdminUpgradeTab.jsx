import React, { useState } from 'react';

export default function AdminUpgradeTab({ onUpgrade, error, status }) {
    const [image, setImage] = useState('docker.io/magicbox/core:latest');

    const handleSubmit = (e) => {
        e.preventDefault();
        onUpgrade(image);
    };

    const isPending = status !== '';

    return (
        <div className="admin-tab-content active">
            <div className="tab-header" style={{ marginBottom: '20px' }}>
                <h3>Core System Upgrade</h3>
            </div>
            
            <p style={{ color: 'var(--text-muted)', fontSize: '0.85rem', lineHeight: 1.6, marginBottom: '24px' }}>
                Upgrading the core system pulls the requested Core container image, renames the running container, 
                spawns a replacement instance containing the upgraded kernel, and cleanly deletes the old instance. 
                <strong> Note:</strong> You will be disconnected for a few seconds.
            </p>

            <form onSubmit={handleSubmit} style={{ maxWidth: '500px' }}>
                <div className="form-group" style={{ marginBottom: '20px' }}>
                    <label style={{ fontSize: '0.8rem', fontWeight: 600, color: 'var(--text-primary)', display: 'block', marginBottom: '8px' }}>
                        Docker Image Reference
                    </label>
                    <input 
                        type="text" 
                        className="form-control" 
                        value={image}
                        onChange={(e) => setImage(e.target.value)}
                        placeholder="docker.io/magicbox/core:latest"
                        required
                        disabled={isPending}
                    />
                    <span style={{ fontSize: '0.75rem', color: 'var(--text-muted)', display: 'block', marginTop: '6px' }}>
                        e.g. <code>docker.io/magicbox/core:latest</code> or <code>docker.io/magicbox/core:v1.2.0</code>
                    </span>
                </div>

                {error && (
                    <div style={{ color: 'var(--danger-color)', fontSize: '0.85rem', marginBottom: '16px' }}>
                        ❌ {error}
                    </div>
                )}

                <button 
                    type="submit" 
                    className="btn btn-primary" 
                    disabled={isPending}
                    style={{ padding: '10px 24px', fontSize: '0.9rem' }}
                >
                    Start Self-Upgrade
                </button>

                {isPending && (
                    <div style={{ display: 'flex', alignItems: 'center', gap: '12px', marginTop: '20px' }}>
                        <div className="spinner-sm"></div>
                        <span style={{ fontSize: '0.85rem', color: 'var(--text-secondary)' }}>
                            {status === 'upgrading' 
                                ? 'Downloading image & recreating core container...' 
                                : 'Re-connecting to Magicbox Core... this may take up to 10 seconds.'}
                        </span>
                    </div>
                )}
            </form>
        </div>
    );
}
