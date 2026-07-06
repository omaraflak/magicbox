// Centralized route definitions for the Magicbox SPA.
// Import these constants everywhere instead of using magic strings.

export const ROUTES = {
    DASHBOARD: '/',
    CONSENT: '/consent',
    SETTINGS: '/settings',
    SETTINGS_SECURITY: '/settings/security',
    SETTINGS_CONTACTS: '/settings/contacts',
    SETTINGS_ADMIN: '/settings/admin',
    SETTINGS_ADMIN_USERS: '/settings/admin/users',
    SETTINGS_ADMIN_REGISTRIES: '/settings/admin/registries',
    SETTINGS_ADMIN_LOGS: '/settings/admin/logs',
    SETTINGS_ADMIN_UPGRADE: '/settings/admin/upgrade',
    SETTINGS_ADMIN_KEYS: '/settings/admin/keys',
};

const ADMIN_SUBTABS = ['users', 'registries', 'logs', 'keys', 'upgrade'];

/**
 * Derive the app view, settings section, and admin sub-tab from a URL pathname.
 * @param {string} pathname - e.g. "/settings/admin/logs"
 * @returns {{ view: string, section: string, tab: string }}
 */
export function viewFromPath(pathname) {
    if (pathname === '/consent') {
        return { view: 'consent', section: 'profile', tab: 'users' };
    }
    if (pathname.startsWith('/app/')) {
        const segments = pathname.substring(5).split('/');
        const slug = segments[0];
        const appPath = segments.slice(1).join('/');
        return { view: 'app', section: 'profile', tab: 'users', appSlug: slug, appPath: appPath };
    }
    if (pathname === '/settings') {
        return { view: 'settings', section: 'profile', tab: 'users' };
    }
    if (pathname === '/settings/security') {
        return { view: 'settings', section: 'security', tab: 'users' };
    }
    if (pathname === '/settings/contacts') {
        return { view: 'settings', section: 'contacts', tab: 'users' };
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
 * @param {string} view - 'dashboard', 'settings', or 'app'
 * @param {string} section - 'profile', 'security', 'contacts', or 'admin'
 * @param {string} tab - 'users', 'registries', or 'logs'
 * @param {string} appSlug - route slug of the app if view is 'app'
 * @param {string} appPath - sub-path within the app if view is 'app'
 * @returns {string}
 */
export function pathFromView(view, section = 'profile', tab = 'users', appSlug = '', appPath = '') {
    if (view === 'app') {
        return `/app/${appSlug}/${appPath || ''}`;
    }
    if (view === 'settings') {
        if (section === 'security') {
            return ROUTES.SETTINGS_SECURITY;
        }
        if (section === 'contacts') {
            return ROUTES.SETTINGS_CONTACTS;
        }
        if (section === 'admin') {
            return tab === 'users' ? ROUTES.SETTINGS_ADMIN : `/settings/admin/${tab}`;
        }
        return ROUTES.SETTINGS;
    }
    return ROUTES.DASHBOARD;
}
