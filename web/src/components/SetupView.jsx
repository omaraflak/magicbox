import React, { useState } from 'react';

export default function SetupView({ onSubmit, error, loading }) {
    const [username, setUsername] = useState('');
    const [password, setPassword] = useState('');
    const [confirmPassword, setConfirmPassword] = useState('');
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

        onSubmit({ username, password });
    };

    const activeError = validationError || error;

    return (
        <div className="card auth-card animate-fade-in" style={{ margin: '80px auto' }}>
            <div className="auth-header">
                <div className="logo-icon">✨</div>
                <h1>Welcome to Magicbox</h1>
                <p>Initialize your personal cloud kernel. Create the primary administrator account.</p>
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
                    />
                    <span className="field-hint">Alphanumeric and underscores only, 3-32 characters.</span>
                </div>
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
                    />
                </div>
                {activeError && <div className="error-box">{activeError}</div>}
                <button type="submit" className="btn btn-primary btn-block" disabled={loading}>
                    <span>{loading ? 'Initializing...' : 'Initialize OS'}</span>
                </button>
            </form>
        </div>
    );
}
