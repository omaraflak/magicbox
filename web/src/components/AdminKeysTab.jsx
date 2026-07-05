import React, { useState, useEffect } from 'react';

export default function AdminKeysTab({ onRotateKeys, onResetIdentity, onUnlock, onGetStatus }) {
    const [unlocked, setUnlocked] = useState(false);
    const [unlockMnemonic, setUnlockMnemonic] = useState('');
    const [unlockLoading, setUnlockLoading] = useState(false);
    const [unlockError, setUnlockError] = useState('');
    const [unlockSuccess, setUnlockSuccess] = useState('');

    // Local state for key rotation
    const [rotateLoading, setRotateLoading] = useState(false);
    const [rotateError, setRotateError] = useState('');
    const [rotateStatus, setRotateStatus] = useState('');
    const [rotateEncryption, setRotateEncryption] = useState(true);
    const [rotateIdentity, setRotateIdentity] = useState(false);

    // Local state for Section 3 (Reset P2P Identity - Emergency)
    const [resetLoading, setResetLoading] = useState(false);
    const [resetError, setResetError] = useState('');
    const [resetStatus, setResetStatus] = useState('');

    // Fetch status on load
    useEffect(() => {
        const checkStatus = async () => {
            try {
                const status = await onGetStatus();
                if (status) {
                    setUnlocked(status.unlocked);
                }
            } catch (err) {
                console.error("Failed to fetch system status:", err);
            }
        };
        checkStatus();
    }, [onGetStatus]);

    const handleUnlockSubmit = async (e) => {
        e.preventDefault();
        if (!unlockMnemonic.trim()) return;

        setUnlockError('');
        setUnlockSuccess('');
        setUnlockLoading(true);

        try {
            const result = await onUnlock(unlockMnemonic.trim());
            if (result.success) {
                setUnlocked(true);
                setUnlockSuccess('System unlocked successfully.');
                setUnlockMnemonic('');
            } else {
                setUnlockError(result.error || 'Failed to unlock system');
            }
        } catch (err) {
            setUnlockError(err.message || 'Operation failed');
        } finally {
            setUnlockLoading(false);
        }
    };

    const handleRotateSubmit = async (e) => {
        e.preventDefault();
        if (!rotateEncryption && !rotateIdentity) {
            setRotateError('At least one key type must be selected for rotation.');
            return;
        }

        setRotateError('');
        setRotateStatus('');
        setResetError('');
        setResetStatus('');

        setRotateLoading(true);
        try {
            const result = await onRotateKeys(rotateEncryption, rotateIdentity);
            if (result.cancelled) return;
            if (result.success) {
                setRotateStatus(result.message);
            } else {
                setRotateError(result.error);
            }
        } catch (err) {
            setRotateError(err.message || 'Operation failed');
        } finally {
            setRotateLoading(false);
        }
    };

    const handleResetIdentitySubmit = async (e) => {
        e.preventDefault();

        setResetError('');
        setResetStatus('');
        setRotateError('');
        setRotateStatus('');

        setResetLoading(true);
        try {
            const result = await onResetIdentity(unlocked);
            if (result.cancelled) return;
            if (result.success) {
                setResetStatus(result.message);
            } else {
                setResetError(result.error);
            }
        } catch (err) {
            setResetError(err.message || 'Operation failed');
        } finally {
            setResetLoading(false);
        }
    };

    const isAnyLoading = rotateLoading || resetLoading || unlockLoading;

    return (
        <div className="admin-tab-content active" style={{ display: 'flex', flexDirection: 'column', gap: '40px' }}>

            {/* System Unlock Section */}
            <div style={{
                background: 'rgba(255, 255, 255, 0.01)',
                padding: '24px',
                border: '1px solid var(--border-color)',
                borderRadius: 'var(--radius-md)'
            }}>
                <div className="tab-header" style={{ marginBottom: '16px', display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                    <h3 style={{ fontSize: '1.1rem', fontWeight: 600, color: 'var(--text-primary)', margin: 0 }}>
                        System Lock Status
                    </h3>
                    {unlocked ? (
                        <span style={{
                            padding: '4px 12px',
                            background: 'rgba(16, 185, 129, 0.1)',
                            border: '1px solid rgba(16, 185, 129, 0.3)',
                            borderRadius: '12px',
                            fontSize: '0.8rem',
                            fontWeight: 600,
                            color: 'var(--accent-success)'
                        }}>
                            Status: Unlocked
                        </span>
                    ) : (
                        <span style={{
                            padding: '4px 12px',
                            background: 'rgba(239, 68, 68, 0.1)',
                            border: '1px solid rgba(239, 68, 68, 0.3)',
                            borderRadius: '12px',
                            fontSize: '0.8rem',
                            fontWeight: 600,
                            color: 'var(--accent-error)'
                        }}>
                            Status: Locked
                        </span>
                    )}
                </div>

                {unlocked ? (
                    <p style={{ color: 'var(--text-muted)', fontSize: '0.85rem', lineHeight: 1.6, margin: 0 }}>
                        The system is currently unlocked. Cryptographic key rotation functions are enabled.
                    </p>
                ) : (
                    <>
                        <p style={{ color: 'var(--text-muted)', fontSize: '0.85rem', lineHeight: 1.6, marginBottom: '20px' }}>
                            The system is currently locked. To enable key rotations, you must authorize by inputting the master mnemonic phrase.
                        </p>
                        <form onSubmit={handleUnlockSubmit} style={{ maxWidth: '500px' }}>
                            <div className="form-group" style={{ marginBottom: '20px' }}>
                                <label style={{ fontSize: '0.8rem', fontWeight: 600, color: 'var(--text-primary)', display: 'block', marginBottom: '8px' }}>
                                    12-Word Recovery Phrase
                                </label>
                                <textarea
                                    value={unlockMnemonic}
                                    onChange={(e) => setUnlockMnemonic(e.target.value)}
                                    placeholder="Enter your 12-word mnemonic phrase to unlock key rotation..."
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

                            {(unlockError || unlockSuccess) && (
                                <div style={{
                                    padding: '12px 16px',
                                    border: `1px solid ${unlockError ? 'rgba(239, 68, 68, 0.2)' : 'rgba(16, 185, 129, 0.2)'}`,
                                    background: unlockError ? 'rgba(239, 68, 68, 0.02)' : 'rgba(16, 185, 129, 0.02)',
                                    borderRadius: 'var(--radius-md)',
                                    fontSize: '0.85rem',
                                    color: unlockError ? 'var(--accent-error)' : 'var(--accent-success)',
                                    marginBottom: '20px',
                                    textAlign: 'left',
                                    display: 'flex',
                                    alignItems: 'center',
                                    gap: '10px'
                                }}>
                                    <span>{unlockError ? `❌ ${unlockError}` : `✅ ${unlockSuccess}`}</span>
                                </div>
                            )}

                            <button
                                type="submit"
                                className="btn btn-primary"
                                disabled={isAnyLoading || !unlockMnemonic.trim()}
                                style={{ padding: '10px 24px', fontSize: '0.9rem' }}
                            >
                                {unlockLoading ? 'Unlocking...' : 'Unlock System'}
                            </button>
                        </form>
                    </>
                )}
            </div>

            {/* Rotate Keys Section */}
            <div style={{
                background: 'rgba(255, 255, 255, 0.01)',
                padding: '24px',
                border: '1px solid var(--border-color)',
                borderRadius: 'var(--radius-md)'
            }}>
                <div className="tab-header" style={{ marginBottom: '16px', textAlign: 'left' }}>
                    <h3 style={{ fontSize: '1.1rem', fontWeight: 600, color: 'var(--text-primary)' }}>
                        Rotate Keys
                    </h3>
                </div>

                <p style={{ color: 'var(--text-muted)', fontSize: '0.85rem', lineHeight: 1.6, marginBottom: '20px' }}>
                    Rotate your cryptographic encryption and/or P2P identity keys. Selecting <strong>Encryption</strong> will rotate the payload encryption keys and update your contacts silently. Selecting <strong>Identity</strong> will rotate your public P2P Identity key and queue succession certificates for your contacts to trust the update.
                </p>

                <form onSubmit={handleRotateSubmit} style={{ maxWidth: '500px' }}>
                    <div style={{ display: 'flex', flexDirection: 'column', gap: '12px', marginBottom: '20px' }}>
                        <label style={{ display: 'flex', alignItems: 'center', gap: '8px', cursor: 'pointer', fontSize: '0.9rem', color: 'var(--text-primary)' }}>
                            <input
                                type="checkbox"
                                checked={rotateEncryption}
                                onChange={(e) => setRotateEncryption(e.target.checked)}
                                disabled={isAnyLoading}
                                style={{ width: '16px', height: '16px', cursor: 'pointer' }}
                            />
                            Rotate Encryption Keys
                        </label>
                        <label style={{ display: 'flex', alignItems: 'center', gap: '8px', cursor: 'pointer', fontSize: '0.9rem', color: 'var(--text-primary)' }}>
                            <input
                                type="checkbox"
                                checked={rotateIdentity}
                                onChange={(e) => setRotateIdentity(e.target.checked)}
                                disabled={isAnyLoading}
                                style={{ width: '16px', height: '16px', cursor: 'pointer' }}
                            />
                            Rotate Identity Keys
                        </label>
                    </div>

                    {(rotateError || rotateStatus) && (
                        <div style={{
                            padding: '12px 16px',
                            border: `1px solid ${rotateError ? 'rgba(239, 68, 68, 0.2)' : 'rgba(16, 185, 129, 0.2)'}`,
                            background: rotateError ? 'rgba(239, 68, 68, 0.02)' : 'rgba(16, 185, 129, 0.02)',
                            borderRadius: 'var(--radius-md)',
                            fontSize: '0.85rem',
                            color: rotateError ? 'var(--accent-error)' : 'var(--accent-success)',
                            marginBottom: '20px',
                            textAlign: 'left',
                            display: 'flex',
                            alignItems: 'center',
                            gap: '10px'
                        }}>
                            <span>{rotateError ? `❌ ${rotateError}` : `✅ ${rotateStatus}`}</span>
                        </div>
                    )}

                    {!unlocked && (
                        <p style={{ color: 'var(--accent-error)', fontSize: '0.85rem', marginBottom: '16px' }}>
                            Unlock system to enable key rotation.
                        </p>
                    )}

                    <button
                        type="submit"
                        className="btn btn-primary"
                        disabled={isAnyLoading || !unlocked || (!rotateEncryption && !rotateIdentity)}
                        style={{ padding: '10px 24px', fontSize: '0.9rem' }}
                    >
                        {rotateLoading ? (
                            <span style={{ display: 'flex', alignItems: 'center', gap: '8px', justifyContent: 'center' }}>
                                Rotating...
                                <div className="spinner-sm" style={{ width: '12px', height: '12px', borderWidth: '1.5px' }} />
                            </span>
                        ) : (
                            'Rotate Selected Keys'
                        )}
                    </button>
                </form>
            </div>

            {/* Danger Zone - Reset Identity */}
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
                        Mnemonic Compromised: Reset Identity
                    </h3>
                </div>

                <p style={{ color: 'var(--text-muted)', fontSize: '0.85rem', lineHeight: 1.6, marginBottom: '20px' }}>
                    Use this option only if you suspect your 12-word recovery mnemonic phrase was compromised, or you want to start fresh with a brand new mnemonic and identity.
                </p>

                <form onSubmit={handleResetIdentitySubmit} style={{ maxWidth: '500px' }}>
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
                        {resetLoading ? (
                            <span style={{ display: 'flex', alignItems: 'center', gap: '8px', justifyContent: 'center' }}>
                                Resetting...
                                <div className="spinner-sm" style={{ width: '12px', height: '12px', borderWidth: '1.5px' }} />
                            </span>
                        ) : (
                            'Reset Identity'
                        )}
                    </button>
                </form>
            </div>

        </div>
    );
}

