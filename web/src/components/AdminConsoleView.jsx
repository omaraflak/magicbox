import React, { useState } from 'react';
import AdminUsersTab from './AdminUsersTab';
import AdminRegistriesTab from './AdminRegistriesTab';
import AdminLogsTab from './AdminLogsTab';

export default function AdminConsoleView({ 
    users, 
    currentUser, 
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
    const [activeTab, setActiveTab] = useState('users');

    return (
        <div className="admin-layout" style={{ display: 'flex', minHeight: 'calc(100vh - 70px)', marginTop: '20px' }}>
            <aside className="admin-sidebar" style={{ width: '240px', borderRight: '1px solid var(--border-color)', paddingRight: '20px' }}>
                <button 
                    className={`sidebar-item ${activeTab === 'users' ? 'active' : ''}`} 
                    onClick={() => setActiveTab('users')}
                >
                    Users
                </button>
                <button 
                    className={`sidebar-item ${activeTab === 'registries' ? 'active' : ''}`} 
                    onClick={() => setActiveTab('registries')}
                >
                    Registries
                </button>
                <button 
                    className={`sidebar-item ${activeTab === 'logs' ? 'active' : ''}`} 
                    onClick={() => setActiveTab('logs')}
                >
                    Kernel Logs
                </button>
            </aside>

            <main className="admin-main animate-fade-in" style={{ flex: 1, paddingLeft: '24px' }}>
                {activeTab === 'users' && (
                    <AdminUsersTab 
                        users={users} 
                        currentUser={currentUser} 
                        onDeleteUser={onDeleteUser} 
                        onOpenCreateModal={onOpenCreateUserModal}
                    />
                )}
                {activeTab === 'registries' && (
                    <AdminRegistriesTab 
                        registries={registries} 
                        onDeleteRegistry={onDeleteRegistry} 
                        onOpenAddModal={onOpenAddRegistryModal}
                    />
                )}
                {activeTab === 'logs' && (
                    <AdminLogsTab 
                        logs={logs} 
                        logLevel={logLevel} 
                        onLevelChange={onLogLevelChange} 
                        onRefresh={onRefreshLogs}
                    />
                )}
            </main>
        </div>
    );
}
