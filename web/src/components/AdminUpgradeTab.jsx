import React, { useState } from 'react';

export default function AdminUpgradeTab({ onUpgrade, error, loading }) {
    const [image, setImage] = useState('magicbox/core:latest');

    const handleSubmit = (e) => {
        e.preventDefault();
        onUpgrade(image);
    };

    return (
        <div style={{ maxWidth: '600px', width: '100%', margin: '0 auto' }}>
            <div className="card" style={{ padding: '32px' }}>
                <h2 style={{ fontSize: '1.25rem', fontWeight: 600, color: 'var(--text-primary)', marginBottom: '12px' }}>
                    🚀 Core System Upgrade
                </h2>
                <p style={{ color: 'var(--text-muted)', fontSize: '0.85rem', lineHeight: 1.5, marginBottom: '24px' }}>
                    Upgrading the core system pulls the requested Core container image, renames the running container, 
                    spawns a replacement instance containing the upgraded kernel, and cleanly deletes the old instance. 
                    <strong> Note:</strong> You will be disconnected for a few seconds.
                </p>

                <form onSubmit={handleSubmit}>
                    <div className="form-group" style={{ marginBottom: '20px' }}>
                        <label style={{ fontSize: '0.8rem', fontWeight: 600, color: 'var(--text-primary)', display: 'block', marginBottom: '8px' }}>
                            Docker Image Reference
                        </label>
                        <input 
                            type="text" 
                            className="form-control" 
                            value={image}
                            onChange={(e) => setImage(e.target.value)}
                            placeholder="magicbox/core:latest"
                            required
                            disabled={loading}
                        />
                        <span style={{ fontSize: '0.75rem', color: 'var(--text-muted)', display: 'block', marginTop: '6px' }}>
                            e.g. <code>magicbox/core:latest</code> or <code>magicbox/core:v1.2.0</code>
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
                        disabled={loading}
                        style={{ width: '100%', padding: '10px 0', fontSize: '0.9rem' }}
                    >
                        {loading ? 'Initiating Upgrade...' : 'Start Self-Upgrade'}
                    </button>
                </form>
            </div>
        </div>
    );
}
