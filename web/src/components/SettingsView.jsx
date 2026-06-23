import React, { useState } from 'react';

export default function SettingsView({ user, onSubmit, error, loading, onBack }) {
    const [activeSection, setActiveSection] = useState('security'); // 'profile', 'security'
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
        <div className="admin-layout animate-fade-in">
            <aside className="admin-sidebar" style={{ paddingTop: '32px' }}>
                <div style={{ padding: '0 20px', marginBottom: '24px' }}>
                    <h2 style={{ fontSize: '1.2rem', fontWeight: 600, color: 'var(--text-primary)' }}>Settings</h2>
                    <p style={{ fontSize: '0.8rem', color: 'var(--text-muted)', marginTop: '4px' }}>Manage account preferences</p>
                </div>
                
                <button 
                    className={`sidebar-item ${activeSection === 'profile' ? 'active' : ''}`}
                    onClick={() => setActiveSection('profile')}
                >
                    👤 Profile Details
                </button>
                <button 
                    className={`sidebar-item ${activeSection === 'security' ? 'active' : ''}`}
                    onClick={() => setActiveSection('security')}
                >
                    🔒 Password & Security
                </button>

                <div style={{ marginTop: 'auto', padding: '10px' }}>
                    <button className="btn btn-secondary btn-block" onClick={onBack}>
                        ← Back to Console
                    </button>
                </div>
            </aside>

            <main className="admin-main" style={{ padding: '40px 60px', maxWidth: '800px' }}>
                {activeSection === 'profile' && (
                    <div>
                        <div style={{ borderBottom: '1px solid var(--border-color)', paddingBottom: '16px', marginBottom: '32px' }}>
                            <h1 style={{ fontSize: '1.75rem', fontWeight: 600, color: 'var(--text-primary)' }}>Profile Details</h1>
                            <p style={{ color: 'var(--text-muted)', fontSize: '0.9rem', marginTop: '6px' }}>Overview of your account profile info.</p>
                        </div>

                        <div className="card" style={{ padding: '24px', background: 'var(--bg-secondary)', border: '1px solid var(--border-color)', borderRadius: 'var(--radius-lg)' }}>
                            <div style={{ display: 'grid', gridTemplateColumns: '150px 1fr', gap: '16px 24px', alignItems: 'center' }}>
                                <span style={{ color: 'var(--text-muted)', fontSize: '0.9rem', fontWeight: 500 }}>User ID:</span>
                                <span style={{ fontFamily: 'monospace', color: 'var(--text-primary)', background: 'var(--bg-primary)', padding: '6px 10px', borderRadius: '4px', border: '1px solid var(--border-color)', fontSize: '0.85rem' }}>
                                    {user?.id}
                                </span>

                                <span style={{ color: 'var(--text-muted)', fontSize: '0.9rem', fontWeight: 500 }}>Username:</span>
                                <span style={{ color: 'var(--text-primary)', fontSize: '0.95rem', fontWeight: 600 }}>{user?.username}</span>

                                <span style={{ color: 'var(--text-muted)', fontSize: '0.9rem', fontWeight: 500 }}>Account Type:</span>
                                <span>
                                    <span style={{ 
                                        display: 'inline-block', 
                                        padding: '4px 8px', 
                                        borderRadius: '12px', 
                                        fontSize: '0.75rem', 
                                        fontWeight: 600,
                                        background: user?.is_admin ? 'rgba(6, 182, 212, 0.1)' : 'rgba(255, 255, 255, 0.05)',
                                        color: user?.is_admin ? 'var(--accent-cyan)' : 'var(--text-muted)',
                                        border: '1px solid ' + (user?.is_admin ? 'var(--accent-cyan)' : 'var(--border-color)')
                                    }}>
                                        {user?.is_admin ? 'Administrator' : 'Standard User'}
                                    </span>
                                </span>

                                <span style={{ color: 'var(--text-muted)', fontSize: '0.9rem', fontWeight: 500 }}>Joined Date:</span>
                                <span style={{ color: 'var(--text-muted)', fontSize: '0.9rem' }}>
                                    {user?.created_at ? new Date(user.created_at).toLocaleDateString(undefined, { dateStyle: 'long' }) : 'N/A'}
                                </span>
                            </div>
                        </div>
                    </div>
                )}

                {activeSection === 'security' && (
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
                            <div className="form-group" style={{ marginBottom: '24px' }}>
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
                )}
            </main>
        </div>
    );
}
