import React, { useState } from 'react';

export default function AdminKeysTab({ onRecover, loading, error, status }) {
    const [recoverPhrase, setRecoverPhrase] = useState('');

    const handleRecoverSubmit = (e) => {
        e.preventDefault();
        if (!recoverPhrase.trim()) return;
        onRecover(recoverPhrase.trim());
    };

    return (
        <div className="admin-tab-content active">
            {/* Section 1: Recover From Phrase */}
            <div>
                <div className="tab-header" style={{ marginBottom: '20px' }}>
                    <h3>Recover Keys from Recovery Phrase</h3>
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
