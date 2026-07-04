import React, { useState, useEffect } from 'react';
import Navbar from './components/Navbar';
import SetupView from './components/SetupView';
import LoginView from './components/LoginView';
import DashboardView from './components/DashboardView';
import AdminConsoleView from './components/AdminConsoleView';
import SettingsView from './components/SettingsView';

// Modals
import InstallAppModal from './components/InstallAppModal';
import CreateUserModal from './components/CreateUserModal';
import AddRegistryModal from './components/AddRegistryModal';
import Modal from './components/Modal';
import MnemonicModal from './components/MnemonicModal';
import AdminKeysTab from './components/AdminKeysTab';

import { 
    apiRequest, 
    refreshCSRFToken, 
    checkNeedsSetup, 
    fetchMe 
} from './utils/api';
import { viewFromPath, pathFromView, ROUTES } from './utils/routes';

export default function App() {
    // Session State
    const [booting, setBooting] = useState(true);
    const [csrfToken, setCsrfToken] = useState('');
    const [user, setUser] = useState(null);
    const [view, setView] = useState('login'); // 'setup', 'login', 'dashboard', 'settings'
    const [settingsSection, setSettingsSection] = useState('profile'); // 'profile', 'security', 'admin'
    const [adminTab, setAdminTab] = useState('users'); // 'users', 'registries', 'logs'

    // Core Business Data
    const [apps, setApps] = useState([]);
    const [users, setUsers] = useState([]);
    const [registries, setRegistries] = useState([]);
    const [logs, setLogs] = useState([]);
    const [logLevel, setLogLevel] = useState('');
    const [contacts, setContacts] = useState([]);
    const [contactRequests, setContactRequests] = useState([]);
    const [invitationInfo, setInvitationInfo] = useState(null);

    // Loadings
    const [actionLoading, setActionLoading] = useState(false);
    const [actionError, setActionError] = useState('');
    const [upgradeStatus, setUpgradeStatus] = useState('');
    const [upgradeError, setUpgradeError] = useState('');
    const [mnemonicData, setMnemonicData] = useState(null);
    const [showMnemonicModal, setShowMnemonicModal] = useState(false);
    const [isIdentityResetPending, setIsIdentityResetPending] = useState(false);

    const [rebuildingAppId, setRebuildingAppId] = useState(null);
    const [uninstallingAppId, setUninstallingAppId] = useState(null);
    const [startingAppId, setStartingAppId] = useState(null);
    const [stoppingAppId, setStoppingAppId] = useState(null);

    // Modal Visibilities
    const [installModalOpen, setInstallModalOpen] = useState(false);
    const [createUserModalOpen, setCreateUserModalOpen] = useState(false);
    const [addRegistryModalOpen, setAddRegistryModalOpen] = useState(false);
    const [uninstallAppId, setUninstallAppId] = useState(null);
    const [wipeAppData, setWipeAppData] = useState(false);

    // Dynamic Confirmation & Alert Modal State
    const [confirmModal, setConfirmModal] = useState({
        isOpen: false,
        title: 'Confirm Action',
        message: '',
        onConfirm: null,
        onCancel: null,
        isAlert: false,
    });

  const showConfirm = (message, onConfirm, title = 'Confirm Action', onCancel = null) => {
        setConfirmModal({
            isOpen: true,
            title,
            message,
            onConfirm: () => {
                setConfirmModal(prev => ({ ...prev, isOpen: false }));
                onConfirm();
            },
          onCancel: () => {
            setConfirmModal(prev => ({ ...prev, isOpen: false }));
            if (onCancel) onCancel();
          },
            isAlert: false,
        });
    };

    const showAlert = (message, title = 'Notification') => {
        setConfirmModal({
            isOpen: true,
            title,
            message,
            onConfirm: () => setConfirmModal(prev => ({ ...prev, isOpen: false })),
            onCancel: null,
            isAlert: true,
        });
    };

    // Navigation: sync view state with URL path.
    const navigate = (newView, section = 'profile', tab = null) => {
        const s = section || settingsSection || 'profile';
        const t = tab || adminTab || 'users';
        if (newView === 'settings') {
            setSettingsSection(s);
            if (s === 'admin') setAdminTab(t);
        }
        const path = pathFromView(newView, s, t);
        window.history.pushState({ view: newView, section: s, tab: t }, '', path);
        setView(newView);
    };

    // Handle browser back/forward.
    useEffect(() => {
        const onPopState = (e) => {
            if (e.state?.view) {
                setView(e.state.view);
                if (e.state.section) setSettingsSection(e.state.section);
                if (e.state.tab) setAdminTab(e.state.tab);
            } else {
                const { view: v, section: s, tab: t } = viewFromPath(window.location.pathname);
                setView(v);
                setSettingsSection(s);
                setAdminTab(t);
            }
        };
        window.addEventListener('popstate', onPopState);
        return () => window.removeEventListener('popstate', onPopState);
    }, [settingsSection, adminTab]);

    // Global unauthorized handler passed to apiRequest
    const handleUnauthorized = () => {
        setUser(null);
        setView('login');
        window.history.replaceState(null, '', ROUTES.DASHBOARD);
    };

    // Helper wrapper for API requests
    const callAPI = async (method, path, body = null) => {
        return await apiRequest(method, path, body, csrfToken, handleUnauthorized);
    };

    // 1. Initial Boot Sequence
    useEffect(() => {
        const runBoot = async () => {
            setBooting(true);
            try {
                const token = await refreshCSRFToken();
                setCsrfToken(token);

                // Fetch current session
                const me = await fetchMe(token);
                if (me) {
                    setUser(me);
                    // Restore view from URL path on refresh.
                    const { view: v, section: s, tab: t } = viewFromPath(window.location.pathname);
                    setView(v);
                    setSettingsSection(s);
                    setAdminTab(t);
                    // Replace state so back/forward works from initial load.
                    window.history.replaceState({ view: v, section: s, tab: t }, '', window.location.pathname);
                } else {
                    // Check if initial setup is required
                    const needsSetup = await checkNeedsSetup(token);
                    if (needsSetup) {
                        setView('setup');
                    } else {
                        setView('login');
                    }
                }
            } catch (err) {
                console.error("Boot error:", err);
                setView('login');
            } finally {
                setBooting(false);
            }
        };
        runBoot();
    }, []);

    // 2. Fetch Dashboard Apps List
    const loadApps = async () => {
        if (view !== 'dashboard' && view !== 'admin') return;
        const { status, data } = await callAPI('GET', '/apps');
        if (status === 200) {
            setApps(data || []);
        }
    };

    // Polling app statuses when dashboard is active (to show active status during installation etc.)
    useEffect(() => {
        loadApps();
        const interval = setInterval(loadApps, 5000);
        return () => clearInterval(interval);
    }, [view, csrfToken]);

    // 3. Fetch Admin Data (Runs when switching to admin tabs)
    const loadUsers = async () => {
        const { status, data } = await callAPI('GET', '/admin/users');
        if (status === 200) setUsers(data || []);
    };

    const loadRegistries = async () => {
        const { status, data } = await callAPI('GET', '/admin/registries');
        if (status === 200) setRegistries(data || []);
    };

    const loadLogs = async (level = logLevel) => {
        let path = '/admin/logs?limit=250';
        if (level) {
            path += `&level=${level.toLowerCase()}`;
        }
        const { status, data } = await callAPI('GET', path);
        if (status === 200 && data) {
            setLogs(data);
        } else {
            setLogs(['Failed to load core logs']);
        }
    };

    const loadContacts = async () => {
        const { status, data } = await callAPI('GET', '/contacts');
        if (status === 200) setContacts(data || []);
    };

    const loadContactRequests = async () => {
        const { status, data } = await callAPI('GET', '/contacts/requests');
        if (status === 200) setContactRequests(data || []);
    };

    const loadInvitationInfo = async () => {
        const { status, data } = await callAPI('GET', '/me/invitation');
        if (status === 200) setInvitationInfo(data);
    };

    const loadMnemonic = async () => {
        const { status, data } = await callAPI('GET', '/admin/mnemonic');
        if (status === 200) setMnemonicData(data);
    };

    useEffect(() => {
        if (view === 'settings') {
            if (settingsSection === 'admin') {
                loadUsers();
                loadRegistries();
                loadLogs();
                loadMnemonic();
            } else if (settingsSection === 'contacts') {
                loadContacts();
                loadContactRequests();
                loadInvitationInfo();
            }
        }
    }, [view, settingsSection, csrfToken]);

    // 4. Action Handlers

    // Setup and recover from existing mnemonic
    const handleSetupRecover = async ({ username, password, mnemonic }) => {
        setActionLoading(true);
        setActionError('');
        const { status, data } = await callAPI('POST', '/setup/recover', { username, password, mnemonic });
        setActionLoading(false);
        if (status === 201) {
            // Auto login after setup
            const me = await fetchMe(csrfToken);
            if (me) {
                setUser(me);
                navigate('dashboard');
            }
        } else {
            setActionError(data?.error || "Recovery setup failed");
        }
    };

    // Setup day-zero account
    const handleSetup = async ({ username, password }) => {
        setActionLoading(true);
        setActionError('');
        const { status, data } = await callAPI('POST', '/setup', { username, password });
        setActionLoading(false);
        if (status === 201) {
            // Auto login after setup
            const me = await fetchMe(csrfToken);
            if (me) {
                setUser(me);
                // Load mnemonic for first-boot display
                if (me.is_admin) {
                    const { status: mStatus, data: mData } = await callAPI('GET', '/admin/mnemonic');
                    if (mStatus === 200 && mData && !mData.acknowledged) {
                        setMnemonicData(mData);
                        setShowMnemonicModal(true);
                    }
                }
                navigate('dashboard');
            }
        } else {
            setActionError(data?.error || "Setup initialization failed");
        }
    };

    // Login
    const handleLogin = async ({ username, password }) => {
        setActionLoading(true);
        setActionError('');
        const { status, data } = await callAPI('POST', '/auth/login', { username, password });
        setActionLoading(false);
        if (status === 200) {
            const me = await fetchMe(csrfToken);
            if (me) {
                setUser(me);
                navigate('dashboard');
            }
        } else {
            setActionError("Invalid username or password");
        }
    };

    // Logout
    const handleLogout = async () => {
        await callAPI('POST', '/auth/logout');
        setUser(null);
        setView('login');
        window.history.replaceState(null, '', '/');
    };

    const handleUpdatePassword = async (currentPassword, newPassword) => {
        setActionLoading(true);
        setActionError('');
        const { status, data } = await callAPI('POST', '/me/password', {
            current_password: currentPassword,
            new_password: newPassword,
        });
        setActionLoading(false);
        if (status === 200) {
            showAlert('Password updated successfully!', 'Success');
            navigate('dashboard');
        } else {
            setActionError(data?.error || 'Failed to update password');
        }
    };

    // App Control: Start
    const handleStartApp = async (id) => {
        setStartingAppId(id);
        const { status, data } = await callAPI('POST', `/apps/${id}/start`);
        setStartingAppId(null);
        if (status === 200) {
            loadApps();
        } else {
            showAlert(data?.error || "Failed to start application", "Error");
        }
    };

    // App Control: Stop
    const handleStopApp = async (id) => {
        setStoppingAppId(id);
        const { status, data } = await callAPI('POST', `/apps/${id}/stop`);
        setStoppingAppId(null);
        if (status === 200) {
            loadApps();
        } else {
            showAlert(data?.error || "Failed to stop application", "Error");
        }
    };

    // App Control: Rotate Token
    const handleRotateToken = async (id) => {
        showConfirm("Are you sure you want to rotate this application's API secret token? The app container will need to be restarted to use the new token.", async () => {
            const { status, data } = await callAPI('POST', `/apps/${id}/rotate-token`);
            if (status === 200) {
                showAlert("API Token successfully rotated. Restart the container to apply changes.", "Success");
                loadApps();
            } else {
                showAlert(data?.error || "Failed to rotate token", "Error");
            }
        }, "Rotate API Token");
    };

    // App Control: Refresh / Rebuild
    const handleRebuildApp = async (id) => {
        showConfirm("Are you sure you want to refresh this application? The container will stop, its latest image will be pulled from the registry, and a new container will start.", async () => {
            setRebuildingAppId(id);
            const { status, data } = await callAPI('POST', `/apps/${id}/rebuild`);
            setRebuildingAppId(null);
            if (status === 200) {
                loadApps();
            } else {
                showAlert(data?.error || "Failed to refresh application", "Error");
            }
        }, "Refresh Application");
    };

    // App Control: Uninstall
    const handleUninstallApp = (id) => {
        setWipeAppData(false);
        setUninstallAppId(id);
    };

    const performUninstall = async (id, wipe) => {
        setUninstallingAppId(id);
        const { status, data } = await callAPI('DELETE', `/apps/${id}?wipe=${wipe}`);
        setUninstallingAppId(null);
        if (status === 200) {
            loadApps();
        } else {
            showAlert(data?.error || "Uninstall failed", "Error");
        }
    };

    // App Control: Install
    const handleInstallApp = async (manifestText) => {
        setActionLoading(true);
        setActionError('');
        const { status, data } = await callAPI('POST', '/apps/install', manifestText);
        setActionLoading(false);
        if (status === 201) {
            setInstallModalOpen(false);
            loadApps();
        } else {
            setActionError(data?.error || "Application installation failed");
        }
    };

    // User Control: Create
    const handleCreateUser = async ({ username, password, isAdmin }) => {
        setActionLoading(true);
        setActionError('');
        const { status, data } = await callAPI('POST', '/admin/users', { username, password, is_admin: isAdmin });
        setActionLoading(false);
        if (status === 201) {
            setCreateUserModalOpen(false);
            loadUsers();
        } else {
            setActionError(data?.error || "Failed to create user account");
        }
    };

    // User Control: Delete
    const handleDeleteUser = async (id) => {
        showConfirm("Are you sure you want to delete this user? This will uninstall all their apps and erase all their directories permanently. Proceed?", async () => {
            const { status, data } = await callAPI('DELETE', `/admin/users/${id}`);
            if (status === 200) {
                loadUsers();
            } else {
                showAlert(data?.error || "Failed to delete user", "Error");
            }
        }, "Delete User");
    };

    // Registry Control: Add
    const handleAddRegistry = async (prefix) => {
        setActionLoading(true);
        setActionError('');
        const { status, data } = await callAPI('POST', '/admin/registries', { prefix });
        setActionLoading(false);
        if (status === 201) {
            setAddRegistryModalOpen(false);
            loadRegistries();
        } else {
            setActionError(data?.error || "Failed to add registry prefix");
        }
    };

    // Registry Control: Delete
    const handleDeleteRegistry = async (id) => {
        showConfirm("Are you sure you want to remove this registry prefix? Apps using images from this registry will no longer be installable.", async () => {
            const { status, data } = await callAPI('DELETE', `/admin/registries/${id}`);
            if (status === 200) {
                loadRegistries();
            } else {
                showAlert(data?.error || "Failed to remove registry", "Error");
            }
        }, "Remove Registry");
    };

    // Contacts: Add/Request
    const handleAddContact = async ({ displayName, multiaddr }) => {
        setActionLoading(true);
        setActionError('');
        const { status, data } = await callAPI('POST', '/contacts/request', { display_name: displayName, multiaddr });
        setActionLoading(false);
        if (status === 201) {
            loadContacts();
            loadContactRequests();
            return true;
        } else {
            setActionError(data?.error || "Failed to send contact request");
            return false;
        }
    };

    // Contacts: Delete
    const handleDeleteContact = async (id) => {
        showConfirm("Are you sure you want to delete this contact?", async () => {
            const { status, data } = await callAPI('DELETE', `/contacts/${id}`);
            if (status === 200) {
                loadContacts();
            } else {
                showAlert(data?.error || "Failed to delete contact", "Error");
            }
        }, "Delete Contact");
    };

    // Contacts: Accept request
    const handleAcceptContactRequest = async (id) => {
        setActionLoading(true);
        setActionError('');
        const { status, data } = await callAPI('POST', `/contacts/requests/${id}/accept`);
        setActionLoading(false);
        if (status === 200) {
            loadContacts();
            loadContactRequests();
            return true;
        } else {
            setActionError(data?.error || "Failed to accept contact request");
            return false;
        }
    };

    // Contacts: Reject request
    const handleRejectContactRequest = async (id) => {
        setActionLoading(true);
        setActionError('');
        const { status, data } = await callAPI('POST', `/contacts/requests/${id}/reject`);
        setActionLoading(false);
        if (status === 200) {
            loadContactRequests();
            return true;
        } else {
            setActionError(data?.error || "Failed to reject contact request");
            return false;
        }
    };

    // Filter Logs
    const handleLogLevelChange = (level) => {
        setLogLevel(level);
        loadLogs(level);
    };

    // Mnemonic: Acknowledge
    const handleAcknowledgeMnemonic = async () => {
        setActionLoading(true);
        await callAPI('POST', '/admin/mnemonic/acknowledge');
        setShowMnemonicModal(false);
        setActionLoading(false);
        setMnemonicData(prev => prev ? { ...prev, acknowledged: true } : prev);

        if (isIdentityResetPending) {
            setIsIdentityResetPending(false);
            triggerSystemRestart();
        }
    };

    // Trigger System Restart (Docker Container Reboot)
    const triggerSystemRestart = async () => {
        setUpgradeStatus('reconnecting'); // Show styled "Reconnecting..." spinner overlay
        try {
            await callAPI('POST', '/admin/restart');
        } catch (e) {
            // Ignore connection abort errors since server terminates immediately
        }
        
        // Poll health endpoint every 1.5 seconds until container is back online
        const interval = setInterval(async () => {
            try {
                const res = await fetch('/api/v1/health');
                if (res.status === 200) {
                    clearInterval(interval);
                    window.location.reload();
                }
            } catch (e) {
                // Keep polling during downtime
            }
        }, 1500);
    };

    // Rotate/Recover Encryption Keys
    const handleRotateEncryptionKeys = async (mnemonic) => {
        const { status, data } = await callAPI('POST', '/admin/keys/rotate-encryption', { mnemonic });
        if (status === 200) {
            loadMnemonic();
            setTimeout(triggerSystemRestart, 1500);
            return { success: true, message: 'Encryption keys rotated successfully! Restarting Magicbox to apply changes...' };
        } else {
            return { success: false, error: data?.error || 'Rotation failed' };
        }
    };

    // Reset & Rotate Identity Keys (Danger Zone)
    const handleRotateIdentityKeys = async (mnemonic) => {
        return new Promise((resolve) => {
            showConfirm(
                "⚠️ WARNING: This will completely RESET your cryptographic identity. You will be disconnected from all your contacts, and you must share your new invite link with them. Are you sure you want to proceed?",
                async () => {
                    const { status, data } = await callAPI('POST', '/admin/keys/rotate-identity', { mnemonic });
                    if (status === 200) {
                        if (data?.mnemonic) {
                            // If a new mnemonic was generated, show the modal so they can copy it!
                            setMnemonicData({ mnemonic: data.mnemonic, acknowledged: false });
                            setShowMnemonicModal(true);
                            setIsIdentityResetPending(true); // restart after modal is acknowledged
                        } else {
                            loadMnemonic();
                            setTimeout(triggerSystemRestart, 1500); // restart immediately
                        }
                        resolve({ success: true, message: 'Identity keys reset successfully! Restarting Magicbox to apply changes...' });
                    } else {
                        resolve({ success: false, error: data?.error || 'Reset failed' });
                    }
                },
                "Confirm Identity Reset",
                () => resolve({ success: false, cancelled: true })
            );
        });
    };

    // Core Self-Upgrade
    const handleUpgradeCore = async (image) => {
        showConfirm(`Are you sure you want to upgrade the core system to image "${image}"? This will restart the container.`, async () => {
            setUpgradeStatus('upgrading');
            setUpgradeError('');
            
            const { status, data } = await callAPI('POST', '/admin/upgrade', { image });
            if (status === 200) {
                setUpgradeStatus('reconnecting');
                
                // Poll health endpoint every 2 seconds until it responds
                const interval = setInterval(async () => {
                    try {
                        const res = await fetch('/health');
                        if (res.status === 200) {
                            clearInterval(interval);
                            window.location.reload();
                        }
                    } catch (e) {
                        // ignore network errors while container is offline
                    }
                }, 2000);
            } else {
                setUpgradeError(data?.error || "Upgrade failed");
                setUpgradeStatus('');
            }
        }, "Confirm System Upgrade");
    };

    // Rendering Helper
    if (booting) {
        return (
            <div className="spinner-container">
                <div className="spinner"></div>
                <div className="spinner-text">Booting Magicbox OS...</div>
            </div>
        );
    }

    return (
        <div id="app">
            {/* Top Navigation Navbar */}
            {(view === 'dashboard' || view === 'settings') && (
                <Navbar 
                    title="Magicbox OS"
                    user={user}
                    onNavigate={navigate}
                />
            )}

            {/* Views */}
            {/* Views */}
            <div className="main-viewport">
                {(view === 'setup' || view === 'login') && (
                    <div className="container auth-container">
                        {view === 'setup' && <SetupView onSubmit={handleSetup} onRecoverSubmit={handleSetupRecover} error={actionError} loading={actionLoading} />}
                        {view === 'login' && <LoginView onSubmit={handleLogin} error={actionError} loading={actionLoading} />}
                    </div>
                )}

                {view === 'dashboard' && (
                    <div className="container dashboard-container">
                        <DashboardView 
                            apps={apps}
                            user={user}
                            onStartApp={handleStartApp}
                            onStopApp={handleStopApp}
                            onUninstallApp={handleUninstallApp}
                            onRotateToken={handleRotateToken}
                            onRebuildApp={handleRebuildApp}
                            rebuildingAppId={rebuildingAppId}
                            uninstallingAppId={uninstallingAppId}
                            startingAppId={startingAppId}
                            stoppingAppId={stoppingAppId}
                            onOpenInstallModal={() => {
                                setActionError('');
                                setInstallModalOpen(true);
                            }}
                        />
                    </div>
                )}

                {view === 'settings' && (
                    <SettingsView 
                        user={user}
                        onSubmit={handleUpdatePassword} 
                        error={actionError} 
                        loading={actionLoading}
                        onBack={() => navigate('dashboard')}
                        onLogout={handleLogout}
                        activeSection={settingsSection}
                        onSectionChange={(sec) => navigate('settings', sec)}
                        adminTab={adminTab}
                        onAdminTabChange={(tab) => navigate('settings', 'admin', tab)}
                        users={users}
                        onDeleteUser={handleDeleteUser}
                        onOpenCreateUserModal={() => {
                            setActionError('');
                            setCreateUserModalOpen(true);
                        }}
                        registries={registries}
                        onDeleteRegistry={handleDeleteRegistry}
                        onOpenAddRegistryModal={() => {
                            setActionError('');
                            setAddRegistryModalOpen(true);
                        }}
                        logs={logs}
                        logLevel={logLevel}
                        onLogLevelChange={handleLogLevelChange}
                        onRefreshLogs={() => loadLogs(logLevel)}
                        contacts={contacts}
                        contactRequests={contactRequests}
                        invitationInfo={invitationInfo}
                        onAddContact={handleAddContact}
                        onDeleteContact={handleDeleteContact}
                        onAcceptContactRequest={handleAcceptContactRequest}
                        onRejectContactRequest={handleRejectContactRequest}
                        onUpgradeCore={handleUpgradeCore}
                        upgradeError={upgradeError}
                        upgradeStatus={upgradeStatus}
                        mnemonicData={mnemonicData}
                        onRotateEncryption={handleRotateEncryptionKeys}
                        onRotateIdentity={handleRotateIdentityKeys}
                    />
                )}
            </div>

            {/* Modals */}
            <InstallAppModal 
                isOpen={installModalOpen}
                onClose={() => setInstallModalOpen(false)}
                onInstall={handleInstallApp}
                error={actionError}
                loading={actionLoading}
            />

            <CreateUserModal 
                isOpen={createUserModalOpen}
                onClose={() => setCreateUserModalOpen(false)}
                onCreateUser={handleCreateUser}
                error={actionError}
                loading={actionLoading}
            />

            <AddRegistryModal 
                isOpen={addRegistryModalOpen}
                onClose={() => setAddRegistryModalOpen(false)}
                onAddRegistry={handleAddRegistry}
                error={actionError}
                loading={actionLoading}
            />

            <Modal
                isOpen={uninstallAppId !== null}
                onClose={() => setUninstallAppId(null)}
                title="Uninstall Application"
            >
                <div className="modal-desc" style={{ marginBottom: '16px' }}>
                    <p style={{ fontWeight: 500, marginBottom: '8px' }}>
                        Are you sure you want to uninstall this application?
                    </p>
                    <p style={{ fontSize: '0.85rem', color: 'var(--text-muted)' }}>
                        This will stop and remove the container.
                    </p>
                </div>
                
                <div style={{ 
                    display: 'flex', 
                    alignItems: 'flex-start', 
                    gap: '10px', 
                    background: 'rgba(255,255,255,0.02)',
                    padding: '12px',
                    borderRadius: '6px',
                    border: '1px solid var(--border-color)',
                    marginBottom: '20px'
                }}>
                    <input 
                        type="checkbox" 
                        id="wipe-data-checkbox"
                        checked={wipeAppData}
                        onChange={(e) => setWipeAppData(e.target.checked)}
                        style={{ marginTop: '3px', cursor: 'pointer' }}
                    />
                    <label htmlFor="wipe-data-checkbox" style={{ fontSize: '0.85rem', color: wipeAppData ? 'var(--accent-red)' : 'var(--text-primary)', cursor: 'pointer', userSelect: 'none' }}>
                        <span style={{ fontWeight: 600, display: 'block', marginBottom: '2px' }}>Wipe all app configuration and database data</span>
                        <span style={{ fontSize: '0.75rem', color: 'var(--text-muted)' }}>
                            Wipes `/data/app_state`. This action is irreversible. User shared volumes (like files/photos) are not deleted.
                        </span>
                    </label>
                </div>

                <div style={{ display: 'flex', justifyContent: 'flex-end', gap: '8px' }}>
                    <button className="btn btn-secondary" onClick={() => setUninstallAppId(null)}>Cancel</button>
                    <button 
                        className={`btn ${wipeAppData ? 'btn-danger' : 'btn-primary'}`}
                        onClick={() => {
                            const id = uninstallAppId;
                            setUninstallAppId(null);
                            performUninstall(id, wipeAppData);
                        }}
                    >
                        {wipeAppData ? 'Uninstall & Wipe Data' : 'Uninstall App'}
                    </button>
                </div>
            </Modal>

            <Modal 
                isOpen={confirmModal.isOpen} 
                onClose={confirmModal.onCancel || confirmModal.onConfirm} 
                title={confirmModal.title}
            >
                <div className="modal-desc">{confirmModal.message}</div>
                <div className="modal-actions">
                    {!confirmModal.isAlert && (
                        <button className="btn btn-secondary" onClick={confirmModal.onCancel}>Cancel</button>
                    )}
                    <button className="btn btn-primary" onClick={confirmModal.onConfirm}>OK</button>
                </div>
            </Modal>

            {showMnemonicModal && mnemonicData && (
                <MnemonicModal
                    mnemonic={mnemonicData.mnemonic}
                    onAcknowledge={handleAcknowledgeMnemonic}
                    loading={actionLoading}
                />
            )}
        </div>
    );
}
