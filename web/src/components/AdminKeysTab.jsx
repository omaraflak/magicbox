import React, { useState } from 'react';

export default function AdminKeysTab({ mnemonic, onRecover, loading, error, status }) {
    const [revealed, setRevealed] = useState(false);
    const [copySuccess, setCopySuccess] = useState(false);
    const [recoverPhrase, setRecoverPhrase] = useState('');

    const handleCopy = () => {
        if (!mnemonic) return;
        navigator.clipboard.writeText(mnemonic).then(() => {
            setCopySuccess(true);
            setTimeout(() => setCopySuccess(false), 2000);
        });
    };

    const handleRecoverSubmit = (e) => {
        e.preventDefault();
        if (!recoverPhrase.trim()) return;
        onRecover(recoverPhrase.trim());
    };

    const maskedPhrase = mnemonic
        ? mnemonic.trim().split(/\s+/).map(() => '••••').join(' ')
        : '';

    return (
        <div className="admin-tab-content active">
            {/* Section 1: Current Recovery Phrase */}
            <div style={{ marginBottom: '40px' }}>
                <div className="tab-header" style={{ marginBottom: '20px' }}>
                    <h3>Current Recovery Phrase</h3>
                </div>

                <p style={{ color: 'var(--text-muted)', fontSize: '0.85rem', lineHeight: 1.6, marginBottom: '20px' }}>
                    Your 12-word recovery phrase can be used to restore your encryption keys on a new device or after a reset.
                </p>

                <div style={{
                    padding: '16px',
                    background: mnemonic ? 'rgba(0, 0, 0, 0.02)' : 'rgba(255, 193, 7, 0.05)',
                    border: '1px solid var(--border-color)',
                    borderRadius: 'var(--radius-md)',
                    marginBottom: '16px',
                    maxWidth: '500px',
                }}>
                    <div style={{
                        fontFamily: mnemonic ? 'monospace' : 'inherit',
                        fontSize: '0.9rem',
                        color: mnemonic ? 'var(--text-primary)' : 'var(--text-muted)',
                        lineHeight: 1.8,
                        wordBreak: 'break-word',
                        userSelect: (revealed && mnemonic) ? 'text' : 'none',
                        fontStyle: mnemonic ? 'normal' : 'italic',
                    }}>
                        {mnemonic ? (revealed ? mnemonic : maskedPhrase) : 'ℹ️ Recovery phrase has been cleared from disk for security reasons. Only your derived active private keys are kept in secure storage.'}
                    </div>
                </div>

                <div style={{ display: 'flex', gap: '8px', marginBottom: '12px' }}>
                    <button
                        className={`btn ${revealed && mnemonic ? 'btn-primary' : 'btn-secondary'}`}
                        onClick={() => mnemonic && setRevealed(!revealed)}
                        disabled={!mnemonic}
                        style={{ padding: '6px 16px', fontSize: '0.85rem' }}
                    >
                        {revealed ? '🙈 Hide' : '👁️ Reveal'}
                    </button>
                    <button
                        className="btn btn-secondary"
                        onClick={handleCopy}
                        disabled={!mnemonic}
                        style={{ padding: '6px 16px', fontSize: '0.85rem' }}
                    >
                        {copySuccess ? 'Copied! ✓' : '📋 Copy'}
                    </button>
                </div>

                {mnemonic && (
                    <p style={{
                        fontSize: '0.8rem',
                        color: 'var(--accent-error)',
                        fontWeight: 500,
                    }}>
                        ⚠️ Never share your recovery phrase with anyone.
                    </p>
                )}
            </div>

            {/* Section 2: Recover From Phrase */}
            <div>
                <div className="tab-header" style={{ marginBottom: '20px' }}>
                    <h3>Recover From Phrase</h3>
                </div>

                <p style={{ color: 'var(--text-muted)', fontSize: '0.85rem', lineHeight: 1.6, marginBottom: '24px' }}>
                    Paste a 12-word mnemonic phrase to restore encryption keys from a backup.
                </p>

                <form onSubmit={handleRecoverSubmit} style={{ maxWidth: '500px' }}>
                    <div className="form-group" style={{ marginBottom: '20px' }}>
                        <label style={{ fontSize: '0.8rem', fontWeight: 600, color: 'var(--text-primary)', display: 'block', marginBottom: '8px' }}>
                            Recovery Phrase
                        </label>
                        <textarea
                            value={recoverPhrase}
                            onChange={(e) => setRecoverPhrase(e.target.value)}
                            placeholder="word1 word2 word3 word4 word5 word6 word7 word8 word9 word10 word11 word12"
                            rows={3}
                            style={{
                                width: '100%',
                                padding: '12px 16px',
                                border: '1px solid var(--border-color)',
                                borderRadius: 'var(--radius-md)',
                                background: 'var(--bg-input)',
                                color: 'var(--text-primary)',
                                fontFamily: 'monospace',
                                fontSize: '0.9rem',
                                resize: 'vertical',
                            }}
                            disabled={loading}
                        />
                        <span style={{ fontSize: '0.75rem', color: 'var(--text-muted)', display: 'block', marginTop: '6px' }}>
                            ⚠️ This will overwrite your current encryption keys. A restart is required for changes to take effect.
                        </span>
                    </div>

                    {error && (
                        <div style={{ color: 'var(--accent-error)', fontSize: '0.85rem', marginBottom: '16px' }}>
                            ❌ {error}
                        </div>
                    )}

                    {status && (
                        <div style={{ color: 'var(--accent-success)', fontSize: '0.85rem', marginBottom: '16px' }}>
                            ✅ {status}
                        </div>
                    )}

                    <button
                        type="submit"
                        className="btn btn-primary"
                        disabled={loading || !recoverPhrase.trim()}
                        style={{ padding: '10px 24px', fontSize: '0.9rem' }}
                    >
                        {loading ? 'Recovering...' : 'Recover Keys'}
                    </button>
                </form>
            </div>
        </div>
    );
}
