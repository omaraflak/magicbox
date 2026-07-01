import React, { useState } from 'react';

export default function MnemonicModal({ mnemonic, onAcknowledge, loading }) {
    const [copySuccess, setCopySuccess] = useState(false);

    const words = mnemonic ? mnemonic.trim().split(/\s+/) : [];

    const handleCopy = () => {
        if (!mnemonic) return;
        navigator.clipboard.writeText(mnemonic).then(() => {
            setCopySuccess(true);
            setTimeout(() => setCopySuccess(false), 2000);
        });
    };

    return (
        <div style={{
            position: 'fixed',
            top: 0,
            left: 0,
            width: '100%',
            height: '100%',
            background: 'rgba(0, 0, 0, 0.7)',
            backdropFilter: 'blur(6px)',
            zIndex: 200,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
        }}>
            <div className="card animate-scale-up" style={{
                width: '100%',
                maxWidth: '560px',
                padding: '40px',
                display: 'flex',
                flexDirection: 'column',
                gap: '24px',
            }}>
                {/* Header */}
                <div style={{ textAlign: 'center' }}>
                    <div style={{ fontSize: '2.5rem', marginBottom: '12px' }}>🔐</div>
                    <h2 style={{
                        fontSize: '1.5rem',
                        fontWeight: 700,
                        color: 'var(--text-primary)',
                        marginBottom: '8px',
                    }}>Save Your Recovery Phrase</h2>
                    <p style={{
                        fontSize: '0.9rem',
                        color: 'var(--text-muted)',
                        lineHeight: 1.6,
                        maxWidth: '440px',
                        margin: '0 auto',
                    }}>
                        This 12-word phrase is the only way to recover your encryption keys if you need to reset your device. Write it down and store it in a safe place.
                    </p>
                </div>

                {/* Word Grid */}
                <div style={{
                    display: 'grid',
                    gridTemplateColumns: 'repeat(3, 1fr)',
                    gap: '10px',
                    padding: '20px',
                    background: 'rgba(0, 0, 0, 0.02)',
                    border: '1px solid var(--border-color)',
                    borderRadius: 'var(--radius-lg)',
                }}>
                    {words.map((word, i) => (
                        <div key={i} style={{
                            display: 'flex',
                            alignItems: 'center',
                            gap: '8px',
                            padding: '10px 12px',
                            background: 'var(--bg-card)',
                            border: '1px solid var(--border-color)',
                            borderRadius: 'var(--radius-md)',
                        }}>
                            <span style={{
                                fontSize: '0.75rem',
                                fontWeight: 600,
                                color: 'var(--text-muted)',
                                minWidth: '18px',
                            }}>{i + 1}.</span>
                            <span style={{
                                fontSize: '0.95rem',
                                fontWeight: 600,
                                color: 'var(--text-primary)',
                                fontFamily: 'monospace',
                            }}>{word}</span>
                        </div>
                    ))}
                </div>

                {/* Actions */}
                <div style={{
                    display: 'flex',
                    flexDirection: 'column',
                    gap: '10px',
                    alignItems: 'stretch',
                }}>
                    <button
                        className="btn btn-secondary"
                        onClick={handleCopy}
                        style={{ padding: '10px 20px', fontSize: '0.9rem' }}
                    >
                        {copySuccess ? '✓ Copied to Clipboard' : '📋 Copy to Clipboard'}
                    </button>
                    <button
                        className="btn btn-primary"
                        onClick={onAcknowledge}
                        disabled={loading}
                        style={{ padding: '12px 20px', fontSize: '0.95rem', fontWeight: 600 }}
                    >
                        {loading ? 'Saving...' : '✅ I\'ve Saved My Recovery Phrase'}
                    </button>
                </div>

                {/* Warning */}
                <p style={{
                    textAlign: 'center',
                    fontSize: '0.8rem',
                    color: 'var(--accent-error)',
                    fontWeight: 500,
                }}>
					⚠️ This phrase will be permanently cleared from memory once you click confirm. It is never stored on disk and cannot be retrieved later.
				</p>
            </div>
        </div>
    );
}
