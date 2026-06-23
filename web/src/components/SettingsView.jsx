import React, { useState } from 'react';
import AdminUsersTab from './AdminUsersTab';
import AdminRegistriesTab from './AdminRegistriesTab';
import AdminLogsTab from './AdminLogsTab';
import Badge from './Badge';

export default function SettingsView({ 
    user, 
    onSubmit, 
    error, 
    loading, 
    onBack, 
    onLogout,
    activeSection,
    onSectionChange,
    adminTab,
    onAdminTabChange,
    users,
    onDeleteUser,
    onOpenCreateUserModal,
    registries,
    onDeleteRegistry,
    onOpenAddRegistryModal,
    logs,
    logLevel,
    onLogLevelChange,
    onRefreshLogs
}) {
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
                    onClick={() => onSectionChange('profile')}
                >
                    👤 Profile Details
                </button>
                <button 
                    className={`sidebar-item ${activeSection === 'security' ? 'active' : ''}`}
                    onClick={() => onSectionChange('security')}
                >
                    🔒 Password & Security
                </button>

                {user?.is_admin && (
                    <button 
                        className={`sidebar-item ${activeSection === 'admin' ? 'active' : ''}`}
                        onClick={() => onSectionChange('admin')}
                    >
                        ⚙️ Admin Console
                    </button>
                )}

                <div style={{ marginTop: 'auto', display: 'flex', flexDirection: 'column', gap: '8px', padding: '10px' }}>
                    <button className="btn btn-secondary btn-block" onClick={onBack}>
                        ← Back to Console
                    </button>
                    <button className="btn btn-danger btn-block" onClick={onLogout} style={{ marginTop: '4px' }}>
                        🚪 Logout
                    </button>
                </div>
            </aside>

            <main className="admin-main" style={{ padding: '40px 60px', maxWidth: activeSection === 'admin' ? '1200px' : '800px', width: '100%' }}>
                {activeSection === 'profile' && (
                    <div>
                        <div style={{ borderBottom: '1px solid var(--border-color)', paddingBottom: '16px', marginBottom: '32px' }}>
                            <h1 style={{ fontSize: '1.75rem', fontWeight: 600, color: 'var(--text-primary)' }}>Profile Details</h1>
                            <p style={{ color: 'var(--text-muted)', fontSize: '0.9rem', marginTop: '6px' }}>Overview of your account profile info.</p>
                        </div>

                        <div style={{ display: 'grid', gridTemplateColumns: '150px 1fr', gap: '20px 24px', alignItems: 'center', maxWidth: '600px' }}>
                            <span style={{ color: 'var(--text-muted)', fontSize: '0.95rem', fontWeight: 500 }}>User ID</span>
                            <span style={{ fontFamily: 'monospace', color: 'var(--text-primary)', background: 'var(--bg-secondary)', padding: '6px 10px', borderRadius: '4px', border: '1px solid var(--border-color)', fontSize: '0.85rem', justifySelf: 'start' }}>
                                {user?.id}
                            </span>

                            <span style={{ color: 'var(--text-muted)', fontSize: '0.95rem', fontWeight: 500 }}>Username</span>
                            <span style={{ color: 'var(--text-primary)', fontSize: '0.95rem', fontWeight: 600 }}>{user?.username}</span>

                            <span style={{ color: 'var(--text-muted)', fontSize: '0.95rem', fontWeight: 500 }}>Account Type</span>
                            <span>
                                <Badge type={user?.is_admin ? 'admin' : 'secondary'}>
                                    {user?.is_admin ? 'Administrator' : 'Standard User'}
                                </Badge>
                            </span>

                            <span style={{ color: 'var(--text-muted)', fontSize: '0.95rem', fontWeight: 500 }}>Joined Date</span>
                            <span style={{ color: 'var(--text-muted)', fontSize: '0.9rem' }}>
                                {user?.created_at ? new Date(user.created_at).toLocaleDateString(undefined, { dateStyle: 'long' }) : 'N/A'}
                            </span>
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

                {activeSection === 'admin' && (
                    <div>
                        <div style={{ borderBottom: '1px solid var(--border-color)', paddingBottom: '16px', marginBottom: '24px' }}>
                            <h1 style={{ fontSize: '1.75rem', fontWeight: 600, color: 'var(--text-primary)' }}>Admin Console</h1>
                            <p style={{ color: 'var(--text-muted)', fontSize: '0.9rem', marginTop: '6px' }}>Manage users, registries, and kernel logs.</p>
                        </div>

                        {/* Horizontal top sub-tabs */}
                        <div style={{ display: 'flex', gap: '8px', borderBottom: '1px solid var(--border-color)', paddingBottom: '12px', marginBottom: '24px' }}>
                            <button 
                                className={`btn ${adminTab === 'users' ? 'btn-primary' : 'btn-secondary'}`}
                                onClick={() => onAdminTabChange('users')}
                                style={{ padding: '6px 16px', fontSize: '0.85rem' }}
                            >
                                Users
                            </button>
                            <button 
                                className={`btn ${adminTab === 'registries' ? 'btn-primary' : 'btn-secondary'}`}
                                onClick={() => onAdminTabChange('registries')}
                                style={{ padding: '6px 16px', fontSize: '0.85rem' }}
                            >
                                Registries
                            </button>
                            <button 
                                className={`btn ${adminTab === 'logs' ? 'btn-primary' : 'btn-secondary'}`}
                                onClick={() => onAdminTabChange('logs')}
                                style={{ padding: '6px 16px', fontSize: '0.85rem' }}
                            >
                                Kernel Logs
                            </button>
                        </div>

                        {/* Rendering sub-tab components */}
                        {adminTab === 'users' && (
                            <AdminUsersTab 
                                users={users} 
                                currentUser={user} 
                                onDeleteUser={onDeleteUser} 
                                onOpenCreateModal={onOpenCreateUserModal}
                            />
                        )}
                        {adminTab === 'registries' && (
                            <AdminRegistriesTab 
                                registries={registries} 
                                onDeleteRegistry={onDeleteRegistry} 
                                onOpenAddModal={onOpenAddRegistryModal}
                            />
                        )}
                        {adminTab === 'logs' && (
                            <AdminLogsTab 
                                logs={logs} 
                                logLevel={logLevel} 
                                onLevelChange={onLogLevelChange} 
                                onRefresh={onRefreshLogs}
                            />
                        )}
                    </div>
                )}
            </main>
        </div>
    );
}
