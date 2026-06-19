// Magicbox SPA Client Engine

const API_BASE = '/api/v1';

// Global state
let state = {
    user: null,
    apps: [],
    csrfToken: '',
    currentTab: 'users',
    users: [],
    registries: [],
    logs: ''
};

// Initial boot sequence
document.addEventListener('DOMContentLoaded', async () => {
    initEventListeners();
    await boot();
});

// Main orchestrator to load views
async function boot() {
    showGlobalSpinner(true);
    try {
        // 1. Fetch CSRF token first
        await refreshCSRFToken();

        // 2. Check login state
        const me = await fetchMe();
        if (me) {
            state.user = me;
            showDashboard();
        } else {
            // Check if setup is needed
            const needsSetup = await checkNeedsSetup();
            if (needsSetup) {
                showView('setup');
            } else {
                showView('login');
            }
        }
    } catch (err) {
        console.error("Boot error:", err);
        showView('login');
    } finally {
        showGlobalSpinner(false);
    }
}

// ---------------------------------------------------------------------------
// View Controller & Navigation
// ---------------------------------------------------------------------------

function showView(viewId) {
    document.querySelectorAll('.view').forEach(v => v.classList.add('hidden'));
    const target = document.getElementById(`view-${viewId}`);
    if (target) {
        target.classList.remove('hidden');
    }
}

function showDashboard() {
    showView('dashboard');
    document.getElementById('user-badge-name').textContent = state.user.username;
    
    // Toggle Admin Console button based on user role
    const adminBtn = document.getElementById('btn-admin-console');
    if (state.user.is_admin) {
        adminBtn.classList.remove('hidden');
    } else {
        adminBtn.classList.add('hidden');
    }

    loadAppsList();
}

function showAdminConsole() {
    showView('admin');
    switchAdminTab(state.currentTab);
}

function switchAdminTab(tabName) {
    state.currentTab = tabName;
    document.querySelectorAll('.sidebar-item').forEach(btn => {
        if (btn.getAttribute('data-tab') === tabName) {
            btn.classList.add('active');
        } else {
            btn.classList.remove('active');
        }
    });

    document.querySelectorAll('.admin-tab-content').forEach(content => {
        content.classList.remove('active');
    });
    document.getElementById(`tab-${tabName}`).classList.add('active');

    if (tabName === 'users') loadUsers();
    if (tabName === 'registries') loadRegistries();
    if (tabName === 'logs') loadLogs();
}

function showGlobalSpinner(show) {
    const app = document.getElementById('app');
    if (show) {
        app.classList.add('loading');
    } else {
        app.classList.remove('loading');
    }
}

// ---------------------------------------------------------------------------
// API Client (With CSRF & Cookies)
// ---------------------------------------------------------------------------

async function apiRequest(method, path, body = null, headers = {}) {
    const opts = {
        method: method,
        headers: {
            'Content-Type': 'application/json',
            ...headers
        }
    };

    if (state.csrfToken && method !== 'GET') {
        opts.headers['X-CSRF-Token'] = state.csrfToken;
    }

    if (body) {
        opts.body = typeof body === 'string' ? body : JSON.stringify(body);
    }

    const response = await fetch(`${API_BASE}${path}`, opts);
    
    if (response.status === 401) {
        // Auto session logout on unauthorized API response
        if (state.user) {
            state.user = null;
            showView('login');
        }
        return { status: 401, data: null };
    }

    let data = null;
    const contentType = response.headers.get('content-type');
    if (contentType && contentType.includes('application/json')) {
        data = await response.json();
    }

    return { status: response.status, data: data };
}

async function refreshCSRFToken() {
    // CSRF token endpoint is outside standard /api prefix in handler
    const response = await fetch('/api/v1/auth/csrf');
    if (response.ok) {
        const body = await response.json();
        state.csrfToken = body.token;
    }
}

async function checkNeedsSetup() {
    // Send an empty setup request
    // If setup exists, server returns 403 before parsing body
    // If setup does not exist, server parses body and fails on invalid input (400)
    const { status } = await apiRequest('POST', '/setup', {});
    return status === 400;
}

async function fetchMe() {
    const { status, data } = await apiRequest('GET', '/me');
    return status === 200 ? data : null;
}

// ---------------------------------------------------------------------------
// Event Listeners Initialization
// ---------------------------------------------------------------------------

function initEventListeners() {
    // Setup Account Form
    document.getElementById('setup-form').addEventListener('submit', handleSetupSubmit);

    // Login Form
    document.getElementById('login-form').addEventListener('submit', handleLoginSubmit);

    // Main Logout Button
    document.getElementById('btn-logout').addEventListener('click', handleLogout);
    document.getElementById('admin-btn-logout').addEventListener('click', handleLogout);

    // Navigation Switches
    document.getElementById('btn-admin-console').addEventListener('click', showAdminConsole);
    document.getElementById('btn-back-dashboard').addEventListener('click', showDashboard);

    // Modal Control Links
    document.getElementById('btn-open-install-modal').addEventListener('click', () => openModal('install'));
    document.getElementById('btn-empty-install').addEventListener('click', () => openModal('install'));
    document.getElementById('btn-close-install').addEventListener('click', () => closeModal('install'));
    document.getElementById('btn-cancel-install').addEventListener('click', () => closeModal('install'));

    document.getElementById('btn-open-create-user-modal').addEventListener('click', () => openModal('create-user'));
    document.getElementById('btn-close-create-user').addEventListener('click', () => closeModal('create-user'));
    document.getElementById('btn-cancel-create-user').addEventListener('click', () => closeModal('create-user'));

    document.getElementById('btn-open-add-registry-modal').addEventListener('click', () => openModal('add-registry'));
    document.getElementById('btn-close-add-registry').addEventListener('click', () => closeModal('add-registry'));
    document.getElementById('btn-cancel-add-registry').addEventListener('click', () => closeModal('add-registry'));

    // Manifest Input Consent Checker
    document.getElementById('manifest-content').addEventListener('input', handleManifestInput);

    // Install app submission
    document.getElementById('install-app-form').addEventListener('submit', handleInstallSubmit);

    // Create user submission
    document.getElementById('create-user-form').addEventListener('submit', handleCreateUserSubmit);

    // Add registry submission
    document.getElementById('add-registry-form').addEventListener('submit', handleAddRegistrySubmit);

    // Admin Sidebar Tabs
    document.querySelectorAll('.sidebar-item').forEach(btn => {
        btn.addEventListener('click', (e) => {
            switchAdminTab(e.target.getAttribute('data-tab'));
        });
    });

    // Refresh logs Button
    document.getElementById('btn-refresh-logs').addEventListener('click', loadLogs);
    document.getElementById('select-log-level').addEventListener('change', loadLogs);

    // Event Delegation for strict CSP compliance (no inline onclicks)
    const appsList = document.getElementById('apps-list');
    if (appsList) {
        appsList.addEventListener('click', (e) => {
            const btn = e.target.closest('.app-action-btn');
            if (!btn) return;
            const action = btn.getAttribute('data-action');
            const id = btn.getAttribute('data-id');
            if (action === 'start') startApp(id);
            if (action === 'stop') stopApp(id);
            if (action === 'uninstall') uninstallApp(id);
            if (action === 'rotate-token') rotateAppToken(id);
        });
    }

    const userTableBody = document.getElementById('table-users-body');
    if (userTableBody) {
        userTableBody.addEventListener('click', (e) => {
            const btn = e.target.closest('.user-delete-btn');
            if (!btn) return;
            const id = btn.getAttribute('data-id');
            deleteUser(id);
        });
    }

    const registryTableBody = document.getElementById('table-registries-body');
    if (registryTableBody) {
        registryTableBody.addEventListener('click', (e) => {
            const btn = e.target.closest('.registry-delete-btn');
            if (!btn) return;
            const id = btn.getAttribute('data-id');
            deleteRegistry(id);
        });
    }
}

function openModal(modalId) {
    document.getElementById(`modal-${modalId}`).classList.remove('hidden');
}

function closeModal(modalId) {
    document.getElementById(`modal-${modalId}`).classList.add('hidden');
    // Clear forms inside modal
    const form = document.querySelector(`#modal-${modalId} form`);
    if (form) form.reset();
    
    // Extra resets for installation modal
    if (modalId === 'install') {
        document.getElementById('permissions-consent-box').classList.add('hidden');
        document.getElementById('consent-scopes-list').innerHTML = '';
        document.getElementById('btn-submit-install').disabled = false;
        document.getElementById('btn-install-text').textContent = 'Install App';
        document.getElementById('install-loader').classList.add('hidden');
        document.getElementById('install-error').classList.add('hidden');
    }
}

// ---------------------------------------------------------------------------
// Account Setup & Authentication Actions
// ---------------------------------------------------------------------------

async function handleSetupSubmit(e) {
    e.preventDefault();
    const username = document.getElementById('setup-username').value;
    const password = document.getElementById('setup-password').value;
    const confirm = document.getElementById('setup-confirm-password').value;
    const errBox = document.getElementById('setup-error');
    
    errBox.classList.add('hidden');

    if (password !== confirm) {
        errBox.textContent = "Passwords do not match";
        errBox.classList.remove('hidden');
        return;
    }

    const submitBtn = e.target.querySelector('button[type="submit"]');
    setButtonLoading(submitBtn, true);

    const { status, data } = await apiRequest('POST', '/setup', { username, password });
    
    setButtonLoading(submitBtn, false);

    if (status === 201) {
        const me = await fetchMe();
        state.user = me;
        showDashboard();
    } else {
        errBox.textContent = data?.error || "Failed to initialize kernel setup";
        errBox.classList.remove('hidden');
    }
}

async function handleLoginSubmit(e) {
    e.preventDefault();
    const username = document.getElementById('login-username').value;
    const password = document.getElementById('login-password').value;
    const errBox = document.getElementById('login-error');

    errBox.classList.add('hidden');
    const submitBtn = e.target.querySelector('button[type="submit"]');
    setButtonLoading(submitBtn, true);

    const { status, data } = await apiRequest('POST', '/auth/login', { username, password });
    
    setButtonLoading(submitBtn, false);

    if (status === 200) {
        const me = await fetchMe();
        state.user = me;
        showDashboard();
    } else {
        errBox.textContent = data?.error || "Invalid username or password";
        errBox.classList.remove('hidden');
    }
}

async function handleLogout(e) {
    e.preventDefault();
    showGlobalSpinner(true);
    await apiRequest('POST', '/auth/logout');
    state.user = null;
    await boot();
}

function setButtonLoading(btn, loading) {
    const text = btn.querySelector('span');
    const loader = btn.querySelector('.loader');
    if (loading) {
        btn.disabled = true;
        if (text) text.style.opacity = '0.3';
        if (loader) loader.classList.remove('hidden');
    } else {
        btn.disabled = false;
        if (text) text.style.opacity = '1';
        if (loader) loader.classList.add('hidden');
    }
}

// ---------------------------------------------------------------------------
// Dashboard - Application Management View
// ---------------------------------------------------------------------------

async function loadAppsList() {
    const appGrid = document.getElementById('apps-list');
    const emptyState = document.getElementById('apps-empty-state');
    
    appGrid.innerHTML = '';
    emptyState.classList.add('hidden');

    const { status, data } = await apiRequest('GET', '/apps');
    if (status !== 200 || !data || data.length === 0) {
        emptyState.classList.remove('hidden');
        return;
    }

    state.apps = data;
    
    data.forEach(app => {
        const card = document.createElement('div');
        card.className = 'card app-card animate-fade-in';
        
        let statusClass = 'status-stopped';
        if (app.status === 'running') statusClass = 'status-running';
        if (app.status === 'error') statusClass = 'status-error';
        if (app.status === 'installing') statusClass = 'status-installing';

        card.innerHTML = `
            <div class="app-info-block">
                <div class="app-icon-badge">📦</div>
                <div class="app-meta">
                    <div class="app-name-row">
                        <span class="app-title">${escapeHTML(app.app_id.split('.').pop())}</span>
                        <span class="app-version">v${escapeHTML(app.version || '1.0.0')}</span>
                    </div>
                    <span class="app-slug">${escapeHTML(app.app_id)}</span>
                </div>
                <div class="status-indicator ${statusClass}">${escapeHTML(app.status)}</div>
            </div>
            
            <div class="app-actions-row">
                <div class="action-buttons">
                    ${app.status === 'running' 
                        ? `<button class="btn btn-secondary btn-sm app-action-btn" data-action="stop" data-id="${app.id}">Stop</button>`
                        : `<button class="btn btn-primary btn-sm app-action-btn" data-action="start" data-id="${app.id}" ${app.status === 'installing' ? 'disabled' : ''}>Start</button>`
                    }
                    <button class="btn btn-danger btn-sm app-action-btn" data-action="uninstall" data-id="${app.id}">Uninstall</button>
                </div>
                <div class="action-buttons">
                    ${app.status === 'running' 
                        ? `<a href="${app.host ? `${window.location.protocol}//${app.host}/` : `/u/${escapeHTML(state.user.username)}/${escapeHTML(app.route_slug)}/`}" target="_blank" class="btn btn-secondary btn-sm">Open App</a>` 
                        : ''
                    }
                    <button class="btn btn-secondary btn-sm app-action-btn" data-action="rotate-token" data-id="${app.id}" title="Rotate API Token">🔑</button>
                </div>
            </div>
        `;
        appGrid.appendChild(card);
    });
}

async function startApp(appId) {
    const { status, data } = await apiRequest('POST', `/apps/${appId}/start`);
    if (status === 200) {
        loadAppsList();
    } else {
        alert(data?.error || "Failed to start application");
    }
}

async function stopApp(appId) {
    const { status, data } = await apiRequest('POST', `/apps/${appId}/stop`);
    if (status === 200) {
        loadAppsList();
    } else {
        alert(data?.error || "Failed to stop application");
    }
}

async function rotateAppToken(appId) {
    if (!confirm("Are you sure you want to rotate this application's API secret token? The app container will need to be restarted to use the new token.")) {
        return;
    }
    const { status, data } = await apiRequest('POST', `/apps/${appId}/rotate-token`);
    if (status === 200) {
        alert("API Token successfully rotated. Restart the container to apply changes.");
        loadAppsList();
    } else {
        alert(data?.error || "Failed to rotate token");
    }
}

async function uninstallApp(appId) {
    if (!confirm("Uninstalling will completely destroy this app container and erase its private configuration. Shared user data folder will remain untouched on the host filesystem. Proceed?")) {
        return;
    }
    const { status, data } = await apiRequest('DELETE', `/apps/${appId}`);
    if (status === 200) {
        loadAppsList();
    } else {
        alert(data?.error || "Uninstall failed");
    }
}

// ---------------------------------------------------------------------------
// Permissions Consent & Install Flow
// ---------------------------------------------------------------------------

function handleManifestInput(e) {
    const input = e.target.value;
    const consentBox = document.getElementById('permissions-consent-box');
    const scopesList = document.getElementById('consent-scopes-list');
    
    consentBox.classList.add('hidden');
    scopesList.innerHTML = '';

    if (!input.trim()) return;

    try {
        const manifest = JSON.parse(input);
        if (manifest.required_scopes && manifest.required_scopes.length > 0) {
            manifest.required_scopes.forEach(scope => {
                const li = document.createElement('li');
                li.className = 'consent-item';
                li.innerHTML = `<span class="consent-icon">🛡️</span> <span>${escapeHTML(scopeToHumanReadable(scope))}</span>`;
                scopesList.appendChild(li);
            });
            consentBox.classList.remove('hidden');
        }
    } catch (err) {
        // Invalid JSON - wait for full submission to error
    }
}

function scopeToHumanReadable(scope) {
    if (scope === 'profile:read') return 'Read your basic user profile (username, user ID)';
    
    const parts = scope.split(':');
    if (parts[0] === 'shared' && parts.length === 3) {
        const folderName = parts[1].charAt(0).toUpperCase() + parts[1].slice(1);
        const access = parts[2] === 'rw' ? 'read & write' : 'read-only';
        return `Access your shared "${folderName}" folder (${access})`;
    }
    return scope;
}

async function handleInstallSubmit(e) {
    e.preventDefault();
    
    const manifestText = document.getElementById('manifest-content').value;
    const errBox = document.getElementById('install-error');
    const submitBtn = document.getElementById('btn-submit-install');
    const loader = document.getElementById('install-loader');
    const text = document.getElementById('btn-install-text');

    errBox.classList.add('hidden');
    submitBtn.disabled = true;
    loader.classList.remove('hidden');
    text.textContent = 'Downloading Image & Bootstrapping...';

    // Install request sends raw manifest payload to rest endpoint
    const { status, data } = await apiRequest('POST', '/apps/install', manifestText);
    
    submitBtn.disabled = false;
    loader.classList.add('hidden');
    text.textContent = 'Install App';

    if (status === 201) {
        closeModal('install');
        loadAppsList();
    } else {
        errBox.textContent = data?.error || "Application installation failed";
        errBox.classList.remove('hidden');
    }
}

// ---------------------------------------------------------------------------
// Admin Tab Actions
// ---------------------------------------------------------------------------

async function loadUsers() {
    const tbody = document.getElementById('table-users-body');
    tbody.innerHTML = '<tr><td colspan="4">Loading user accounts...</td></tr>';
    
    const { status, data } = await apiRequest('GET', '/admin/users');
    if (status !== 200) return;

    tbody.innerHTML = '';
    data.forEach(user => {
        const tr = document.createElement('tr');
        tr.innerHTML = `
            <td><strong>${escapeHTML(user.username)}</strong></td>
            <td>${user.is_admin ? '<span class="status-indicator status-running">Admin</span>' : '<span class="status-indicator status-stopped">User</span>'}</td>
            <td>${escapeHTML(new Date(user.created_at).toLocaleString())}</td>
            <td class="text-right">
                <button class="btn btn-danger btn-sm user-delete-btn" data-id="${user.id}" ${state.user.id === user.id ? 'disabled' : ''}>Delete</button>
            </td>
        `;
        tbody.appendChild(tr);
    });
}

async function handleCreateUserSubmit(e) {
    e.preventDefault();
    const username = document.getElementById('create-username').value;
    const password = document.getElementById('create-password').value;
    const isAdmin = document.getElementById('create-is-admin').checked;
    const errBox = document.getElementById('create-user-error');

    errBox.classList.add('hidden');

    const { status, data } = await apiRequest('POST', '/admin/users', { 
        username: username, 
        password: password, 
        is_admin: isAdmin 
    });

    if (status === 201) {
        closeModal('create-user');
        loadUsers();
    } else {
        errBox.textContent = data?.error || "Failed to create user account";
        errBox.classList.remove('hidden');
    }
}

async function deleteUser(id) {
    if (!confirm("Are you sure you want to delete this user? All applications associated with this user will be uninstalled, and their directories will be erased!")) {
        return;
    }
    const { status, data } = await apiRequest('DELETE', `/admin/users/${id}`);
    if (status === 200) {
        loadUsers();
    } else {
        alert(data?.error || "Failed to delete user");
    }
}

async function loadRegistries() {
    const tbody = document.getElementById('table-registries-body');
    tbody.innerHTML = '<tr><td colspan="3">Loading allowed registries...</td></tr>';
    
    const { status, data } = await apiRequest('GET', '/admin/registries');
    if (status !== 200) return;

    tbody.innerHTML = '';
    data.forEach(reg => {
        const tr = document.createElement('tr');
        tr.innerHTML = `
            <td><code style="font-size: 0.95rem; color: var(--accent-cyan)">${escapeHTML(reg.prefix)}</code></td>
            <td>${escapeHTML(new Date(reg.created_at).toLocaleString())}</td>
            <td class="text-right">
                <button class="btn btn-danger btn-sm registry-delete-btn" data-id="${reg.id}">Remove</button>
            </td>
        `;
        tbody.appendChild(tr);
    });
}

async function handleAddRegistrySubmit(e) {
    e.preventDefault();
    const prefix = document.getElementById('registry-prefix').value;
    const errBox = document.getElementById('add-registry-error');

    errBox.classList.add('hidden');

    const { status, data } = await apiRequest('POST', '/admin/registries', { prefix });

    if (status === 201) {
        closeModal('add-registry');
        loadRegistries();
    } else {
        errBox.textContent = data?.error || "Failed to add registry prefix";
        errBox.classList.remove('hidden');
    }
}

async function deleteRegistry(id) {
    if (!confirm("Are you sure you want to remove this registry prefix? Apps using images from this registry will no longer be installable.")) {
        return;
    }
    const { status, data } = await apiRequest('DELETE', `/admin/registries/${id}`);
    if (status === 200) {
        loadRegistries();
    } else {
        alert(data?.error || "Failed to remove registry");
    }
}

async function loadLogs() {
    const pre = document.getElementById('logs-output');
    pre.textContent = 'Loading system logs...';

    const level = document.getElementById('select-log-level').value;
    let url = '/admin/logs?limit=250';
    if (level) {
        url += `&level=${level.toLowerCase()}`;
    }

    const { status, data } = await apiRequest('GET', url);
    if (status !== 200) {
        pre.textContent = 'Failed to load core logs';
        return;
    }

    if (!data || data.length === 0) {
        pre.textContent = 'No logs matching current criteria';
        return;
    }

    // Format JSON log entries for readability
    pre.textContent = data.map(line => {
        try {
            const entry = JSON.parse(line);
            // Re-order timestamp and message for cleaner output
            const { ts, level, msg, ...rest } = entry;
            const timeStr = new Date(ts).toLocaleTimeString();
            const restStr = Object.keys(rest).length > 0 ? `  |  ${JSON.stringify(rest)}` : '';
            return `[${timeStr}] [${level}] ${msg}${restStr}`;
        } catch (e) {
            return line; // Fallback to raw line
        }
    }).join('\n');
}

// ---------------------------------------------------------------------------
// HTML Escaper helper utility
// ---------------------------------------------------------------------------

function escapeHTML(str) {
    if (!str) return '';
    return str.replace(/[&<>'"]/g, 
        tag => ({
            '&': '&amp;',
            '<': '&lt;',
            '>': '&gt;',
            "'": '&#39;',
            '"': '&quot;'
        }[tag] || tag)
    );
}

// Global exposes for inline onclick definitions in templated HTML
window.startApp = startApp;
window.stopApp = stopApp;
window.uninstallApp = uninstallApp;
window.deleteUser = deleteUser;
window.deleteRegistry = deleteRegistry;
window.rotateAppToken = rotateAppToken;
