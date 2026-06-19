import React, { useState } from 'react';

export default function LoginView({ onSubmit, error, loading }) {
    const [username, setUsername] = useState('');
    const [password, setPassword] = useState('');

    const handleSubmit = (e) => {
        e.preventDefault();
        onSubmit({ username, password });
    };

    return (
        <div className="card auth-card animate-fade-in" style={{ margin: '80px auto' }}>
            <div className="auth-header">
                <div className="logo-icon">🔒</div>
                <h1>Magicbox OS</h1>
                <p>Log in to access your personal cloud portal.</p>
            </div>
            <form onSubmit={handleSubmit} className="auth-form">
                <div className="form-group">
                    <label htmlFor="login-username">Username</label>
                    <input 
                        type="text" 
                        id="login-username" 
                        required 
                        autoComplete="username" 
                        placeholder="Username"
                        value={username}
                        onChange={(e) => setUsername(e.target.value)}
                    />
                </div>
                <div className="form-group">
                    <label htmlFor="login-password">Password</label>
                    <input 
                        type="password" 
                        id="login-password" 
                        required 
                        autoComplete="current-password" 
                        placeholder="••••••••"
                        value={password}
                        onChange={(e) => setPassword(e.target.value)}
                    />
                </div>
                {error && <div className="error-box">{error}</div>}
                <button type="submit" className="btn btn-primary btn-block" disabled={loading}>
                    <span>{loading ? 'Signing In...' : 'Sign In'}</span>
                </button>
            </form>
        </div>
    );
}
