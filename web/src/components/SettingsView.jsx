import React from 'react';
import ProfileTab from './ProfileTab';
import SecurityTab from './SecurityTab';
import ContactsTab from './ContactsTab';
import AdminUsersTab from './AdminUsersTab';
import AdminRegistriesTab from './AdminRegistriesTab';
import AdminLogsTab from './AdminLogsTab';
import AdminUpgradeTab from './AdminUpgradeTab';
import AdminKeysTab from './AdminKeysTab';

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
    onRefreshLogs,
    contacts = [],
    invitationInfo = null,
    onAddContact,
    onDeleteContact,
    onUpgradeCore,
    upgradeError,
    upgradeStatus,
    mnemonicData,
    onRecoverKeys,
    recoverError,
    recoverStatus
}) {
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
                <button 
                    className={`sidebar-item ${activeSection === 'contacts' ? 'active' : ''}`}
                    onClick={() => onSectionChange('contacts')}
                >
                    👥 Contacts
                </button>

                {user?.is_admin && (
                    <button 
                        className={`sidebar-item ${activeSection === 'admin' ? 'active' : ''}`}
                        onClick={() => onSectionChange('admin')}
                    >
                        ⚙️ Admin Console
                    </button>
                )}

                <div style={{ marginTop: 'auto', padding: '10px' }}>
                    <button className="btn btn-danger btn-block" onClick={onLogout}>
                        🚪 Logout
                    </button>
                </div>
            </aside>

            <main className="admin-main" style={{ padding: '40px 60px', maxWidth: (activeSection === 'admin' || activeSection === 'contacts') ? '1200px' : '800px', width: '100%' }}>
                {activeSection === 'profile' && (
                    <ProfileTab user={user} />
                )}

                {activeSection === 'security' && (
                    <SecurityTab 
                        onSubmit={onSubmit} 
                        error={error} 
                        loading={loading} 
                    />
                )}

                {activeSection === 'contacts' && (
                    <ContactsTab 
                        contacts={contacts}
                        invitationInfo={invitationInfo}
                        onAddContact={onAddContact}
                        onDeleteContact={onDeleteContact}
                        error={error}
                        loading={loading}
                    />
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
                            <button 
                                className={`btn ${adminTab === 'keys' ? 'btn-primary' : 'btn-secondary'}`}
                                onClick={() => onAdminTabChange('keys')}
                                style={{ padding: '6px 16px', fontSize: '0.85rem' }}
                            >
                                🔑 Encryption Keys
                            </button>
                            <button 
                                className={`btn ${adminTab === 'upgrade' ? 'btn-primary' : 'btn-secondary'}`}
                                onClick={() => onAdminTabChange('upgrade')}
                                style={{ padding: '6px 16px', fontSize: '0.85rem' }}
                            >
                                System Upgrade
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
                        {adminTab === 'upgrade' && (
                            <AdminUpgradeTab 
                                onUpgrade={onUpgradeCore} 
                                error={upgradeError} 
                                status={upgradeStatus}
                            />
                        )}
                        {adminTab === 'keys' && (
                            <AdminKeysTab
                                onRecover={onRecoverKeys}
                                loading={loading}
                                error={recoverError}
                                status={recoverStatus}
                            />
                        )}
                    </div>
                )}
            </main>
        </div>
    );
}
