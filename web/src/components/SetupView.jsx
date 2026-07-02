import React, { useState } from 'react';

export default function SetupView({ onSubmit, onRecoverSubmit, error, loading }) {
    const [mode, setMode] = useState('new'); // 'new' or 'recover'
    const [username, setUsername] = useState('');
    const [password, setPassword] = useState('');
    const [confirmPassword, setConfirmPassword] = useState('');
    const [mnemonic, setMnemonic] = useState('');
    const [validationError, setValidationError] = useState('');

    const handleSubmit = (e) => {
        e.preventDefault();
        setValidationError('');

        if (password.length < 8) {
            setValidationError('Password must be at least 8 characters long');
            return;
        }
        if (password !== confirmPassword) {
            setValidationError('Passwords do not match');
            return;
        }

        if (mode === 'recover') {
            const formattedMnemonic = mnemonic.trim();
            if (!formattedMnemonic) {
                setValidationError('Mnemonic phrase is required for recovery');
                return;
            }
            if (formattedMnemonic.split(/\s+/).length !== 12) {
                setValidationError('Recovery phrase must be exactly 12 words');
                return;
            }
            onRecoverSubmit({ username, password, mnemonic: formattedMnemonic });
        } else {
            onSubmit({ username, password });
        }
    };

    const activeError = validationError || error;

    return (
        <div className="card auth-card animate-fade-in" style={{ margin: '60px auto', maxWidth: '460px' }}>
            <div className="auth-header" style={{ marginBottom: '24px' }}>
                <div className="logo-icon">✨</div>
                <h1>Welcome to Magicbox</h1>
                <p>Initialize your personal cloud kernel. Create the primary administrator account.</p>
            </div>

            {/* Mode Tab Switcher */}
            <div style={{
                display: 'flex',
                background: 'rgba(255, 255, 255, 0.02)',
                border: '1px solid var(--border-color)',
                borderRadius: 'var(--radius-md)',
                padding: '4px',
                marginBottom: '28px'
            }}>
                <button
                    type="button"
                    onClick={() => { setMode('new'); setValidationError(''); }}
                    style={{
                        flex: 1,
                        background: mode === 'new' ? 'var(--bg-card)' : 'transparent',
                        color: mode === 'new' ? 'var(--text-primary)' : 'var(--text-muted)',
                        border: 'none',
                        padding: '8px 12px',
                        fontSize: '0.85rem',
                        fontWeight: 600,
                        borderRadius: 'calc(var(--radius-md) - 2px)',
                        cursor: 'pointer',
                        transition: 'all 0.2s ease'
                    }}
                >
                    Create New Identity
                </button>
                <button
                    type="button"
                    onClick={() => { setMode('recover'); setValidationError(''); }}
                    style={{
                        flex: 1,
                        background: mode === 'recover' ? 'var(--bg-card)' : 'transparent',
                        color: mode === 'recover' ? 'var(--text-primary)' : 'var(--text-muted)',
                        border: 'none',
                        padding: '8px 12px',
                        fontSize: '0.85rem',
                        fontWeight: 600,
                        borderRadius: 'calc(var(--radius-md) - 2px)',
                        cursor: 'pointer',
                        transition: 'all 0.2s ease'
                    }}
                >
                    Recover Identity
                </button>
            </div>

            <form onSubmit={handleSubmit} className="auth-form">
                <div className="form-group">
                    <label htmlFor="setup-username">Username</label>
                    <input 
                        type="text" 
                        id="setup-username" 
                        placeholder="e.g. admin, alice" 
                        required 
                        autoComplete="username"
                        value={username}
                        onChange={(e) => setUsername(e.target.value.toLowerCase())}
                        disabled={loading}
                    />
                    <span className="field-hint">Alphanumeric and underscores only, 3-32 characters.</span>
                </div>
                
                {mode === 'recover' && (
                    <div className="form-group" style={{ marginBottom: '20px' }}>
                        <label htmlFor="setup-mnemonic">12-Word Recovery Phrase</label>
                        <textarea 
                            id="setup-mnemonic" 
                            placeholder="word1 word2 word3..." 
                            required 
                            rows={3}
                            value={mnemonic}
                            onChange={(e) => setMnemonic(e.target.value)}
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
                        <div style={{
                            padding: '12px 16px',
                            background: 'rgba(239, 68, 68, 0.03)',
                            border: '1px solid rgba(239, 68, 68, 0.2)',
                            borderRadius: 'var(--radius-md)',
                            marginTop: '10px',
                            fontSize: '0.78rem',
                            color: 'var(--accent-error)',
                            lineHeight: 1.5
                        }}>
                            ⚠️ <strong>Warning:</strong> Do not recover your identity here if your previous keys were compromised. A compromised mnemonic will generate compromised keys. If compromised, choose "Create New Identity" instead.
                        </div>
                    </div>
                )}

                <div className="form-group">
                    <label htmlFor="setup-password">Password</label>
                    <input 
                        type="password" 
                        id="setup-password" 
                        placeholder="Minimum 8 characters" 
                        required 
                        autoComplete="new-password"
                        value={password}
                        onChange={(e) => setPassword(e.target.value)}
                        disabled={loading}
                    />
                </div>
                
                <div className="form-group">
                    <label htmlFor="setup-confirm-password">Confirm Password</label>
                    <input 
                        type="password" 
                        id="setup-confirm-password" 
                        placeholder="Confirm your password" 
                        required 
                        autoComplete="new-password"
                        value={confirmPassword}
                        onChange={(e) => setConfirmPassword(e.target.value)}
                        disabled={loading}
                    />
                </div>

                {activeError && <div className="error-box">{activeError}</div>}

                <button type="submit" className="btn btn-primary btn-block" disabled={loading}>
                    <span>
                        {loading 
                            ? (mode === 'recover' ? 'Recovering...' : 'Initializing...') 
                            : (mode === 'recover' ? 'Recover & Initialize OS' : 'Initialize OS')
                        }
                    </span>
                </button>
            </form>
        </div>
    );
}
