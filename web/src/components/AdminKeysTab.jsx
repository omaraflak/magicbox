import React, { useState } from 'react';

export default function AdminKeysTab({ onRotateEncryption, onRotateIdentity, onResetIdentity }) {
    const [encMnemonic, setEncMnemonic] = useState('');
    const [idMnemonic, setIdMnemonic] = useState('');
    const [resetMnemonic, setResetMnemonic] = useState('');

    // Local state for Section 1 (Rotate Encryption Keys)
    const [encLoading, setEncLoading] = useState(false);
    const [encError, setEncError] = useState('');
    const [encStatus, setEncStatus] = useState('');

    // Local state for Section 2 (Rotate Identity Keys - Hygiene)
    const [idLoading, setIdLoading] = useState(false);
    const [idError, setIdError] = useState('');
    const [idStatus, setIdStatus] = useState('');

    // Local state for Section 3 (Reset P2P Identity - Emergency)
    const [resetLoading, setResetLoading] = useState(false);
    const [resetError, setResetError] = useState('');
    const [resetStatus, setResetStatus] = useState('');

    const handleRotateEncryptionSubmit = async (e) => {
        e.preventDefault();
        if (!encMnemonic.trim()) return;

        setEncError('');
        setEncStatus('');
        setIdError('');
        setIdStatus('');
        setResetError('');
        setResetStatus('');

        setEncLoading(true);
        try {
            const result = await onRotateEncryption(encMnemonic.trim());
            if (result.success) {
                setEncStatus(result.message);
                setEncMnemonic('');
            } else {
                setEncError(result.error);
            }
        } catch (err) {
            setEncError(err.message || 'Operation failed');
        } finally {
            setEncLoading(false);
        }
    };

    const handleRotateIdentitySubmit = async (e) => {
        e.preventDefault();
        if (!idMnemonic.trim()) return;

        setIdError('');
        setIdStatus('');
        setEncError('');
        setEncStatus('');
        setResetError('');
        setResetStatus('');

        setIdLoading(true);
        try {
            const result = await onRotateIdentity(idMnemonic.trim());
            if (result.cancelled) return;
            if (result.success) {
                setIdStatus(result.message);
                setIdMnemonic('');
            } else {
                setIdError(result.error);
            }
        } catch (err) {
            setIdError(err.message || 'Operation failed');
        } finally {
            setIdLoading(false);
        }
    };

    const handleResetIdentitySubmit = async (e) => {
        e.preventDefault();

        setResetError('');
        setResetStatus('');
        setEncError('');
        setEncStatus('');
        setIdError('');
        setIdStatus('');

        setResetLoading(true);
        try {
            const result = await onResetIdentity(resetMnemonic.trim() || null);
            if (result.cancelled) return;
            if (result.success) {
                setResetStatus(result.message);
                setResetMnemonic('');
            } else {
                setResetError(result.error);
            }
        } catch (err) {
            setResetError(err.message || 'Operation failed');
        } finally {
            setResetLoading(false);
        }
    };

    const isAnyLoading = encLoading || idLoading || resetLoading;

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
                            disabled={isAnyLoading}
                            required
                        />
                    </div>

                    {(encError || encStatus) && (
                        <div style={{
                            padding: '12px 16px',
                            border: `1px solid ${encError ? 'rgba(239, 68, 68, 0.2)' : 'rgba(16, 185, 129, 0.2)'}`,
                            background: encError ? 'rgba(239, 68, 68, 0.02)' : 'rgba(16, 185, 129, 0.02)',
                            borderRadius: 'var(--radius-md)',
                            fontSize: '0.85rem',
                            color: encError ? 'var(--accent-error)' : 'var(--accent-success)',
                            marginBottom: '20px',
                            textAlign: 'left',
                            display: 'flex',
                            alignItems: 'center',
                            gap: '10px'
                        }}>
                            <span>{encError ? `❌ ${encError}` : `✅ ${encStatus}`}</span>
                        </div>
                    )}

                    <button
                        type="submit"
                        className="btn btn-primary"
                        disabled={isAnyLoading || !encMnemonic.trim()}
                        style={{ padding: '10px 24px', fontSize: '0.9rem' }}
                    >
                        {encLoading ? 'Rotating...' : 'Rotate Encryption Keys'}
                    </button>
                </form>
            </div>

            {/* Section 2: Rotate Identity Keys (Hygiene) */}
            <div style={{
                background: 'rgba(255, 255, 255, 0.01)',
                padding: '24px',
                border: '1px solid var(--border-color)',
                borderRadius: 'var(--radius-md)'
            }}>
                <div className="tab-header" style={{ marginBottom: '16px', textAlign: 'left' }}>
                    <h3 style={{ fontSize: '1.1rem', fontWeight: 600, color: 'var(--text-primary)' }}>
                        Rotate Identity Keys (Hygiene)
                    </h3>
                </div>

                <p style={{ color: 'var(--text-muted)', fontSize: '0.85rem', lineHeight: 1.6, marginBottom: '20px' }}>
                    This will rotate your public P2P Identity key. It automatically signs a succession certificate using your old identity key and queues it to all your contacts so they can update your address without losing connection. 
                    <strong> No manual action is required from your contacts</strong>, and your secure channels remain intact.
                </p>

                <form onSubmit={handleRotateIdentitySubmit} style={{ maxWidth: '500px' }}>
                    <div className="form-group" style={{ marginBottom: '20px' }}>
                        <label style={{ fontSize: '0.8rem', fontWeight: 600, color: 'var(--text-primary)', display: 'block', marginBottom: '8px' }}>
                            12-Word Recovery Phrase
                        </label>
                        <textarea
                            value={idMnemonic}
                            onChange={(e) => setIdMnemonic(e.target.value)}
                            placeholder="Enter your 12-word mnemonic phrase to authorize identity rotation..."
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
                            disabled={isAnyLoading}
                            required
                        />
                    </div>

                    {(idError || idStatus) && (
                        <div style={{
                            padding: '12px 16px',
                            border: `1px solid ${idError ? 'rgba(239, 68, 68, 0.2)' : 'rgba(16, 185, 129, 0.2)'}`,
                            background: idError ? 'rgba(239, 68, 68, 0.02)' : 'rgba(16, 185, 129, 0.02)',
                            borderRadius: 'var(--radius-md)',
                            fontSize: '0.85rem',
                            color: idError ? 'var(--accent-error)' : 'var(--accent-success)',
                            marginBottom: '20px',
                            textAlign: 'left',
                            display: 'flex',
                            alignItems: 'center',
                            gap: '10px'
                        }}>
                            <span>{idError ? `❌ ${idError}` : `✅ ${idStatus}`}</span>
                        </div>
                    )}

                    <button
                        type="submit"
                        className="btn btn-primary"
                        disabled={isAnyLoading || !idMnemonic.trim()}
                        style={{ padding: '10px 24px', fontSize: '0.9rem' }}
                    >
                        {idLoading ? 'Rotating...' : 'Rotate Identity Keys'}
                    </button>
                </form>
            </div>

            {/* Section 3: Danger Zone - Reset Identity */}
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
                        Danger Zone: Reset P2P Identity (Compromised)
                    </h3>
                </div>

                <p style={{ color: 'var(--text-muted)', fontSize: '0.85rem', lineHeight: 1.6, marginBottom: '20px' }}>
                    This is an emergency recovery action. It completely resets your cryptographic P2P Identity and generates a brand new encryption key.
                    <br /><br />
                    <span style={{ color: '#ef4444', fontWeight: 600 }}>Important:</span> You will be completely disconnected from all your contacts. They will not be able to reach you until you share your new invite link with them. Use this only if your current keys/device have been compromised or you want to start fresh.
                </p>

                <form onSubmit={handleResetIdentitySubmit} style={{ maxWidth: '500px' }}>
                    <div className="form-group" style={{ marginBottom: '20px' }}>
                        <label style={{ fontSize: '0.8rem', fontWeight: 600, color: 'var(--text-primary)', display: 'block', marginBottom: '8px' }}>
                            Mnemonic Phrase (Optional)
                        </label>
                        <textarea
                            value={resetMnemonic}
                            onChange={(e) => setResetMnemonic(e.target.value)}
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
                            disabled={isAnyLoading}
                        />
                    </div>

                    {(resetError || resetStatus) && (
                        <div style={{
                            padding: '12px 16px',
                            border: `1px solid ${resetError ? 'rgba(239, 68, 68, 0.2)' : 'rgba(16, 185, 129, 0.2)'}`,
                            background: resetError ? 'rgba(239, 68, 68, 0.02)' : 'rgba(16, 185, 129, 0.02)',
                            borderRadius: 'var(--radius-md)',
                            fontSize: '0.85rem',
                            color: resetError ? 'var(--accent-error)' : 'var(--accent-success)',
                            marginBottom: '20px',
                            textAlign: 'left',
                            display: 'flex',
                            alignItems: 'center',
                            gap: '10px'
                        }}>
                            <span>{resetError ? `❌ ${resetError}` : `✅ ${resetStatus}`}</span>
                        </div>
                    )}

                    <button
                        type="submit"
                        className="btn btn-danger"
                        disabled={isAnyLoading}
                        style={{ padding: '10px 24px', fontSize: '0.9rem' }}
                    >
                        {resetLoading ? 'Resetting...' : 'Reset P2P Identity'}
                    </button>
                </form>
            </div>

        </div>
    );
}
