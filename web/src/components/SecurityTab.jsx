import React, { useState } from 'react';

export default function SecurityTab({ onSubmit, error, loading }) {
    const [currentPassword, setCurrentPassword] = useState('');
    const [newPassword, setNewPassword] = useState('');
    const [confirmPassword, setConfirmPassword] = useState('');
    const [validationError, setValidationError] = useState('');

    const handlePasswordSubmit = (e) => {
        e.preventDefault();
        setValidationError('');

        if (newPassword !== confirmPassword) {
            setValidationError('New passwords do not match');
            return;
        }

        if (newPassword.length < 8) {
            setValidationError('Password must be at least 8 characters long');
            return;
        }

        onSubmit(currentPassword, newPassword);
        // Clear inputs after submitting
        setCurrentPassword('');
        setNewPassword('');
        setConfirmPassword('');
    };

    return (
        <div>
            <div style={{ borderBottom: '1px solid var(--border-color)', paddingBottom: '16px', marginBottom: '32px' }}>
                <h1 style={{ fontSize: '1.75rem', fontWeight: 600, color: 'var(--text-primary)' }}>Password & Security</h1>
                <p style={{ color: 'var(--text-muted)', fontSize: '0.9rem', marginTop: '6px' }}>Manage and update your password credentials.</p>
            </div>

            <form onSubmit={handlePasswordSubmit} className="auth-form" style={{ maxWidth: '460px' }}>
                <div className="form-group" style={{ marginBottom: '20px' }}>
                    <label htmlFor="settings-current-password" style={{ fontWeight: 500, fontSize: '0.9rem', color: 'var(--text-primary)', marginBottom: '8px', display: 'block' }}>Current Password</label>
                    <input 
                        type="password" 
                        id="settings-current-password" 
                        required 
                        placeholder="••••••••"
                        value={currentPassword}
                        onChange={(e) => setCurrentPassword(e.target.value)}
                        style={{ width: '100%', padding: '10px 12px', border: '1px solid var(--border-color)', borderRadius: '6px', background: 'var(--bg-secondary)', color: 'var(--text-primary)' }}
                    />
                </div>
                <div className="form-group" style={{ marginBottom: '20px' }}>
                    <label htmlFor="settings-new-password" style={{ fontWeight: 500, fontSize: '0.9rem', color: 'var(--text-primary)', marginBottom: '8px', display: 'block' }}>New Password</label>
                    <input 
                        type="password" 
                        id="settings-new-password" 
                        required 
                        placeholder="••••••••"
                        value={newPassword}
                        onChange={(e) => setNewPassword(e.target.value)}
                        style={{ width: '100%', padding: '10px 12px', border: '1px solid var(--border-color)', borderRadius: '6px', background: 'var(--bg-secondary)', color: 'var(--text-primary)' }}
                    />
                </div>
                <div className="form-group" style={{ marginBottom: '20px' }}>
                    <label htmlFor="settings-confirm-password" style={{ fontWeight: 500, fontSize: '0.9rem', color: 'var(--text-primary)', marginBottom: '8px', display: 'block' }}>Confirm New Password</label>
                    <input 
                        type="password" 
                        id="settings-confirm-password" 
                        required 
                        placeholder="••••••••"
                        value={confirmPassword}
                        onChange={(e) => setConfirmPassword(e.target.value)}
                        style={{ width: '100%', padding: '10px 12px', border: '1px solid var(--border-color)', borderRadius: '6px', background: 'var(--bg-secondary)', color: 'var(--text-primary)' }}
                    />
                </div>
                
                {(validationError || error) && (
                    <div className="error-box" style={{ marginBottom: '20px', padding: '12px', borderRadius: '6px', background: 'rgba(239, 68, 68, 0.1)', border: '1px solid rgb(239, 68, 68)', color: 'rgb(239, 68, 68)', fontSize: '0.9rem' }}>
                        {validationError || error}
                    </div>
                )}

                <button type="submit" className="btn btn-primary" style={{ padding: '10px 24px' }} disabled={loading}>
                    <span>{loading ? 'Updating...' : 'Update Password'}</span>
                </button>
            </form>
        </div>
    );
}
