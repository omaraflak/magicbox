import React, { useState } from 'react';

export default function AdminKeysTab({ onRotateEncryption, onRotateIdentity, loading, error, status }) {
    const [encMnemonic, setEncMnemonic] = useState('');
    const [idMnemonic, setIdMnemonic] = useState('');

    const handleRotateEncryptionSubmit = (e) => {
        e.preventDefault();
        if (!encMnemonic.trim()) return;
        onRotateEncryption(encMnemonic.trim());
    };

    const handleRotateIdentitySubmit = (e) => {
        e.preventDefault();
        // Optional mnemonic, if empty it generates a new one.
        if (window.confirm("⚠️ WARNING: This will completely RESET your cryptographic identity (Peer ID). You will be disconnected from all your contacts, and you must share your new invite link with them. Are you sure you want to proceed?")) {
            onRotateIdentity(idMnemonic.trim() || null);
        }
    };

    return (
        <div className="admin-tab-content active" style={{ display: 'flex', flexDirection: 'column', gap: '40px' }}>
            {/* Section 1: Rotate Encryption Keys */}
            <div style={{
                background: 'rgba(255, 255, 255, 0.01)',
                padding: '24px',
                border: '1px solid var(--border-color)',
                borderRadius: 'var(--radius-md)'
            }}>
                <div className="tab-header" style={{ marginBottom: '16px', textAlign: 'left' }}>
                    <h3 style={{ fontSize: '1.1rem', fontWeight: 600, color: 'var(--text-primary)' }}>
                        Rotate Encryption Keys
                    </h3>
                </div>

                <p style={{ color: 'var(--text-muted)', fontSize: '0.85rem', lineHeight: 1.6, marginBottom: '20px' }}>
                    This will rotate the cryptographic keys used to encrypt and decrypt message payloads. 
                    It automatically and silently updates all your contacts with your new encryption key over the P2P network. 
                    <strong> No action is required from your contacts</strong>, and your public Identity remains the same.
                </p>

                <form onSubmit={handleRotateEncryptionSubmit} style={{ maxWidth: '500px' }}>
                    <div className="form-group" style={{ marginBottom: '20px' }}>
                        <label style={{ fontSize: '0.8rem', fontWeight: 600, color: 'var(--text-primary)', display: 'block', marginBottom: '8px' }}>
                            12-Word Recovery Phrase
                        </label>
                        <textarea
                            value={encMnemonic}
                            onChange={(e) => setEncMnemonic(e.target.value)}
                            placeholder="Enter your 12-word mnemonic phrase to authorize key derivation..."
                            rows={2}
                            style={{
                                width: '100%',
                                padding: '12px 16px',
                                border: '1px solid var(--border-color)',
                                borderRadius: 'var(--radius-md)',
                                background: 'var(--bg-input)',
                                color: 'var(--text-primary)',
                                fontFamily: 'monospace',
                                fontSize: '0.9rem',
                                resize: 'none',
                            }}
                            disabled={loading}
                            required
                        />
                    </div>

                    <button
                        type="submit"
                        className="btn btn-primary"
                        disabled={loading || !encMnemonic.trim()}
                        style={{ padding: '10px 24px', fontSize: '0.9rem' }}
                    >
                        {loading ? 'Rotating...' : 'Rotate Encryption Keys'}
                    </button>
                </form>
            </div>

            {/* Section 2: Danger Zone - Reset Identity */}
            <div style={{
                background: 'rgba(239, 68, 68, 0.02)',
                padding: '24px',
                border: '1px solid rgba(239, 68, 68, 0.2)',
                borderRadius: 'var(--radius-md)',
                textAlign: 'left'
            }}>
                <div className="tab-header" style={{ marginBottom: '16px', display: 'flex', alignItems: 'center', gap: '8px', justifyContent: 'flex-start', textAlign: 'left' }}>
                    <span style={{ fontSize: '1.2rem' }}>⚠️</span>
                    <h3 style={{ fontSize: '1.1rem', fontWeight: 600, color: '#ef4444', margin: 0, textAlign: 'left' }}>
                        Danger Zone: Reset P2P Identity
                    </h3>
                </div>

                <p style={{ color: 'var(--text-muted)', fontSize: '0.85rem', lineHeight: 1.6, marginBottom: '20px' }}>
                    This is an emergency recovery action. It completely resets your cryptographic P2P Identity (Peer ID) and generates a brand new encryption key, resetting both key indices back to 0. 
                    <br /><br />
                    <span style={{ color: '#ef4444', fontWeight: 600 }}>Important:</span> You will be completely disconnected from all your contacts. They will not be able to reach you until you share your new invite link with them. Use this only if your current keys/device have been compromised or you want to start fresh.
                </p>

                <form onSubmit={handleRotateIdentitySubmit} style={{ maxWidth: '500px' }}>
                    <div className="form-group" style={{ marginBottom: '20px' }}>
                        <label style={{ fontSize: '0.8rem', fontWeight: 600, color: 'var(--text-primary)', display: 'block', marginBottom: '8px' }}>
                            Mnemonic Phrase (Optional)
                        </label>
                        <textarea
                            value={idMnemonic}
                            onChange={(e) => setIdMnemonic(e.target.value)}
                            placeholder="Leave empty to generate a brand new random identity mnemonic..."
                            rows={2}
                            style={{
                                width: '100%',
                                padding: '12px 16px',
                                border: '1px solid var(--border-color)',
                                borderRadius: 'var(--radius-md)',
                                background: 'var(--bg-input)',
                                color: 'var(--text-primary)',
                                fontFamily: 'monospace',
                                fontSize: '0.9rem',
                                resize: 'none',
                            }}
                            disabled={loading}
                        />
                    </div>

                    <button
                        type="submit"
                        className="btn btn-danger"
                        disabled={loading}
                        style={{ padding: '10px 24px', fontSize: '0.9rem' }}
                    >
                        {loading ? 'Resetting...' : 'Reset P2P Identity'}
                    </button>
                </form>
            </div>

            {/* Error and Status Feedback */}
            {(error || status) && (
                <div style={{
                    padding: '16px',
                    border: `1px solid ${error ? 'rgba(239, 68, 68, 0.2)' : 'rgba(16, 185, 129, 0.2)'}`,
                    background: error ? 'rgba(239, 68, 68, 0.02)' : 'rgba(16, 185, 129, 0.02)',
                    borderRadius: 'var(--radius-md)',
                    fontSize: '0.9rem',
                    color: error ? 'var(--accent-error)' : 'var(--accent-success)',
                    marginTop: '20px',
                    textAlign: 'left'
                }}>
                    {error ? `❌ ${error}` : `✅ ${status}`}
                </div>
            )}
        </div>
    );
}
