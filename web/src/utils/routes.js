// Centralized route definitions for the Magicbox SPA.
// Import these constants everywhere instead of using magic strings.

export const ROUTES = {
    DASHBOARD: '/',
    SETTINGS: '/settings',
    SETTINGS_SECURITY: '/settings/security',
    SETTINGS_ADMIN: '/settings/admin',
    SETTINGS_ADMIN_USERS: '/settings/admin/users',
    SETTINGS_ADMIN_REGISTRIES: '/settings/admin/registries',
    SETTINGS_ADMIN_LOGS: '/settings/admin/logs',
};

const ADMIN_SUBTABS = ['users', 'registries', 'logs'];

/**
 * Derive the app view, settings section, and admin sub-tab from a URL pathname.
 * @param {string} pathname - e.g. "/settings/admin/logs"
 * @returns {{ view: string, section: string, tab: string }}
 */
export function viewFromPath(pathname) {
    if (pathname === '/settings') {
        return { view: 'settings', section: 'profile', tab: 'users' };
    }
    if (pathname === '/settings/security') {
        return { view: 'settings', section: 'security', tab: 'users' };
    }
    if (pathname.startsWith('/settings/admin')) {
        const segment = pathname.replace('/settings/admin', '').replace(/^\//, '');
        const tab = ADMIN_SUBTABS.includes(segment) ? segment : 'users';
        return { view: 'settings', section: 'admin', tab };
    }
    // Backward compatibility for old admin URLs:
    if (pathname.startsWith('/admin')) {
        const segment = pathname.replace('/admin', '').replace(/^\//, '');
        const tab = ADMIN_SUBTABS.includes(segment) ? segment : 'users';
        return { view: 'settings', section: 'admin', tab };
    }
    return { view: 'dashboard', section: 'profile', tab: 'users' };
}

/**
 * Build a URL path for a given view, settings section, and admin sub-tab.
 * @param {string} view - 'dashboard' or 'settings'
 * @param {string} section - 'profile', 'security', or 'admin'
 * @param {string} tab - 'users', 'registries', or 'logs'
 * @returns {string}
 */
export function pathFromView(view, section = 'profile', tab = 'users') {
    if (view === 'settings') {
        if (section === 'security') {
            return ROUTES.SETTINGS_SECURITY;
        }
        if (section === 'admin') {
            return tab === 'users' ? ROUTES.SETTINGS_ADMIN : `/settings/admin/${tab}`;
        }
        return ROUTES.SETTINGS;
    }
    return ROUTES.DASHBOARD;
}
