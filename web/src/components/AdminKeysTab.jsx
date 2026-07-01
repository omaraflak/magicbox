import React, { useState } from 'react';

export default function AdminKeysTab({ activeIndex, onRecover, loading, error, status }) {
    const [recoverPhrase, setRecoverPhrase] = useState('');
    const [recoverIndex, setRecoverIndex] = useState(activeIndex !== undefined ? activeIndex + 1 : 0);

    React.useEffect(() => {
        if (activeIndex !== undefined) {
            setRecoverIndex(activeIndex + 1);
        }
    }, [activeIndex]);

    const handleRecoverSubmit = (e) => {
        e.preventDefault();
        if (!recoverPhrase.trim()) return;
        onRecover(recoverPhrase.trim(), parseInt(recoverIndex, 10));
    };

    return (
        <div className="admin-tab-content active">
            {/* Section 1: Recover From Phrase */}
            <div>
                <div className="tab-header" style={{ marginBottom: '20px' }}>
                    <h3>Recover Keys from Recovery Phrase</h3>
                </div>

                <div style={{
                    padding: '12px 16px',
                    background: 'rgba(255,255,255,0.01)',
                    border: '1px solid var(--border-color)',
                    borderRadius: 'var(--radius-md)',
                    marginBottom: '20px',
                    fontSize: '0.9rem',
                    color: 'var(--text-primary)',
                    display: 'inline-flex',
                    alignItems: 'center',
                    gap: '8px'
                }}>
                    <span>Active Key Index:</span>
                    <strong style={{ fontSize: '1rem', fontFamily: 'monospace', color: 'var(--accent)' }}>{activeIndex}</strong>
                </div>

                <p style={{ color: 'var(--text-muted)', fontSize: '0.85rem', lineHeight: 1.6, marginBottom: '24px' }}>
                    If you have re-installed Magicbox or need to recover your cryptographic identity, enter your 12-word mnemonic phrase below. This will re-derive and overwrite your active signing and encryption keys.
                </p>

                <form onSubmit={handleRecoverSubmit} style={{ maxWidth: '500px' }}>
                    <div className="form-group" style={{ marginBottom: '20px' }}>
                        <label style={{ fontSize: '0.8rem', fontWeight: 600, color: 'var(--text-primary)', display: 'block', marginBottom: '8px' }}>
                            12-Word Recovery Phrase
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
                            ⚠️ Warning: Recovering will overwrite your existing active keys. A container restart is required for changes to take effect.
                        </span>
                    </div>

                    <div className="form-group" style={{ marginBottom: '20px' }}>
                        <label style={{ fontSize: '0.8rem', fontWeight: 600, color: 'var(--text-primary)', display: 'block', marginBottom: '8px' }}>
                            Derivation Index (Number)
                        </label>
                        <input
                            type="number"
                            min="0"
                            value={recoverIndex}
                            onChange={(e) => setRecoverIndex(e.target.value)}
                            style={{
                                width: '100%',
                                padding: '10px 16px',
                                border: '1px solid var(--border-color)',
                                borderRadius: 'var(--radius-md)',
                                background: 'var(--bg-input)',
                                color: 'var(--text-primary)',
                                fontFamily: 'monospace',
                                fontSize: '0.9rem',
                            }}
                            disabled={loading}
                            required
                        />
                        <span style={{ fontSize: '0.75rem', color: 'var(--text-muted)', display: 'block', marginTop: '6px' }}>
                            Deriving at a new index rotates your keys (e.g. active index + 1). Use 0 for initial setup index.
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
