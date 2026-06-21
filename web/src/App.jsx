import React, { useState, useEffect } from 'react';
import Navbar from './components/Navbar';
import SetupView from './components/SetupView';
import LoginView from './components/LoginView';
import DashboardView from './components/DashboardView';
import AdminConsoleView from './components/AdminConsoleView';

// Modals
import InstallAppModal from './components/InstallAppModal';
import CreateUserModal from './components/CreateUserModal';
import AddRegistryModal from './components/AddRegistryModal';
import Modal from './components/Modal';

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
    const [view, setView] = useState('login'); // 'setup', 'login', 'dashboard', 'admin'
    const [adminTab, setAdminTab] = useState('users'); // 'users', 'registries', 'logs'

    // Core Business Data
    const [apps, setApps] = useState([]);
    const [users, setUsers] = useState([]);
    const [registries, setRegistries] = useState([]);
    const [logs, setLogs] = useState([]);
    const [logLevel, setLogLevel] = useState('');

    // Loadings
    const [actionLoading, setActionLoading] = useState(false);
    const [actionError, setActionError] = useState('');
    const [rebuildingAppId, setRebuildingAppId] = useState(null);

    // Modal Visibilities
    const [installModalOpen, setInstallModalOpen] = useState(false);
    const [createUserModalOpen, setCreateUserModalOpen] = useState(false);
    const [addRegistryModalOpen, setAddRegistryModalOpen] = useState(false);

    // Dynamic Confirmation & Alert Modal State
    const [confirmModal, setConfirmModal] = useState({
        isOpen: false,
        title: 'Confirm Action',
        message: '',
        onConfirm: null,
        onCancel: null,
        isAlert: false,
    });

    const showConfirm = (message, onConfirm, title = 'Confirm Action') => {
        setConfirmModal({
            isOpen: true,
            title,
            message,
            onConfirm: () => {
                setConfirmModal(prev => ({ ...prev, isOpen: false }));
                onConfirm();
            },
            onCancel: () => setConfirmModal(prev => ({ ...prev, isOpen: false })),
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
    const navigate = (newView, tab = null) => {
        const t = tab || adminTab || 'users';
        if (newView === 'admin') setAdminTab(t);
        const path = pathFromView(newView, t);
        window.history.pushState({ view: newView, tab: t }, '', path);
        setView(newView);
    };

    // Handle browser back/forward.
    useEffect(() => {
        const onPopState = (e) => {
            if (e.state?.view) {
                setView(e.state.view);
                if (e.state.tab) setAdminTab(e.state.tab);
            } else {
                const { view: v, tab } = viewFromPath(window.location.pathname);
                setView(v);
                setAdminTab(tab);
            }
        };
        window.addEventListener('popstate', onPopState);
        return () => window.removeEventListener('popstate', onPopState);
    }, []);

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
                    const { view: v, tab } = viewFromPath(window.location.pathname);
                    setView(v);
                    setAdminTab(tab);
                    // Replace state so back/forward works from initial load.
                    window.history.replaceState({ view: v, tab }, '', window.location.pathname);
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
        if (status === 200 && data) {
            setApps(data);
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
        if (status === 200 && data) setUsers(data);
    };

    const loadRegistries = async () => {
        const { status, data } = await callAPI('GET', '/admin/registries');
        if (status === 200 && data) setRegistries(data);
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

    useEffect(() => {
        if (view === 'admin') {
            loadUsers();
            loadRegistries();
            loadLogs();
        }
    }, [view, csrfToken]);

    // 4. Action Handlers

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

    // App Control: Start
    const handleStartApp = async (id) => {
        const { status, data } = await callAPI('POST', `/apps/${id}/start`);
        if (status === 200) {
            loadApps();
        } else {
            showAlert(data?.error || "Failed to start application", "Error");
        }
    };

    // App Control: Stop
    const handleStopApp = async (id) => {
        const { status, data } = await callAPI('POST', `/apps/${id}/stop`);
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
                showAlert("Application refreshed successfully.", "Success");
                loadApps();
            } else {
                showAlert(data?.error || "Failed to refresh application", "Error");
            }
        }, "Refresh Application");
    };

    // App Control: Uninstall
    const handleUninstallApp = async (id) => {
        showConfirm("Uninstalling will completely destroy this app container and erase its private configuration. Shared user data folder will remain untouched on the host filesystem. Proceed?", async () => {
            const { status, data } = await callAPI('DELETE', `/apps/${id}`);
            if (status === 200) {
                loadApps();
            } else {
                showAlert(data?.error || "Uninstall failed", "Error");
            }
        }, "Uninstall Application");
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

    // Filter Logs
    const handleLogLevelChange = (level) => {
        setLogLevel(level);
        loadLogs(level);
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
            {(view === 'dashboard' || view === 'admin') && (
                <Navbar 
                    title={view === 'admin' ? "Magicbox Admin Console" : "Magicbox OS"}
                    user={user}
                    onLogout={handleLogout}
                    adminView={view === 'admin'}
                    onToggleView={(v) => navigate(v)}
                />
            )}

            {/* Views */}
            <div className="container" style={{ maxWidth: '1200px', margin: '0 auto', padding: '0 20px' }}>
                {view === 'setup' && (
                    <SetupView onSubmit={handleSetup} error={actionError} loading={actionLoading} />
                )}

                {view === 'login' && (
                    <LoginView onSubmit={handleLogin} error={actionError} loading={actionLoading} />
                )}

                {view === 'dashboard' && (
                    <DashboardView 
                        apps={apps}
                        user={user}
                        onStartApp={handleStartApp}
                        onStopApp={handleStopApp}
                        onUninstallApp={handleUninstallApp}
                        onRotateToken={handleRotateToken}
                        onRebuildApp={handleRebuildApp}
                        rebuildingAppId={rebuildingAppId}
                        onOpenInstallModal={() => {
                            setActionError('');
                            setInstallModalOpen(true);
                        }}
                    />
                )}

                {view === 'admin' && (
                    <AdminConsoleView 
                        users={users}
                        currentUser={user}
                        onDeleteUser={handleDeleteUser}
                        onOpenCreateUserModal={() => {
                            setActionError('');
                            setCreateUserModalOpen(true);
                        }}
                        activeTab={adminTab}
                        onTabChange={(tab) => navigate('admin', tab)}
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
        </div>
    );
}
