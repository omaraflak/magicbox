// Centralized route definitions for the Magicbox SPA.
// Import these constants everywhere instead of using magic strings.

export const ROUTES = {
    DASHBOARD: '/',
    ADMIN: '/admin',
    ADMIN_USERS: '/admin',          // users is the default admin tab
    ADMIN_REGISTRIES: '/admin/registries',
    ADMIN_LOGS: '/admin/logs',
    SETTINGS: '/settings',
};

// Valid admin tab segments (used to parse URL paths).
const ADMIN_TABS = ['registries', 'logs'];

/**
 * Derive the app view and admin tab from a URL pathname.
 * @param {string} pathname - e.g. "/admin/logs"
 * @returns {{ view: string, tab: string }}
 */
export function viewFromPath(pathname) {
    if (pathname.startsWith('/admin')) {
        const segment = pathname.replace('/admin', '').replace(/^\//, '');
        const tab = ADMIN_TABS.includes(segment) ? segment : 'users';
        return { view: 'admin', tab };
    }
    if (pathname === '/settings') {
        return { view: 'settings', tab: 'users' };
    }
    return { view: 'dashboard', tab: 'users' };
}

/**
 * Build a URL path for a given view and optional admin tab.
 * @param {string} view - 'dashboard', 'admin', or 'settings'
 * @param {string} [tab] - 'users', 'registries', or 'logs'
 * @returns {string}
 */
export function pathFromView(view, tab = 'users') {
    if (view === 'admin') {
        return tab === 'users' ? ROUTES.ADMIN : `/admin/${tab}`;
    }
    if (view === 'settings') {
        return ROUTES.SETTINGS;
    }
    return ROUTES.DASHBOARD;
}
