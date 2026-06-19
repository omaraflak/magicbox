const API_BASE = '/api/v1';

export async function apiRequest(method, path, body = null, csrfToken = '', onUnauthorized = null) {
    const opts = {
        method: method,
        headers: {
            'Content-Type': 'application/json',
        }
    };

    if (csrfToken && method !== 'GET') {
        opts.headers['X-CSRF-Token'] = csrfToken;
    }

    if (body) {
        opts.body = typeof body === 'string' ? body : JSON.stringify(body);
    }

    const response = await fetch(`${API_BASE}${path}`, opts);
    
    if (response.status === 401) {
        if (onUnauthorized) {
            onUnauthorized();
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

export async function refreshCSRFToken() {
    const response = await fetch('/api/v1/auth/csrf');
    if (response.ok) {
        const body = await response.json();
        return body.token;
    }
    return '';
}

export async function checkNeedsSetup(csrfToken) {
    const { status } = await apiRequest('POST', '/setup', {}, csrfToken);
    return status === 400;
}

export async function fetchMe(csrfToken) {
    const { status, data } = await apiRequest('GET', '/me', null, csrfToken);
    return status === 200 ? data : null;
}
