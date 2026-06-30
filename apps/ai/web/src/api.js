export let API_BASE = '/api';
if (window.location.pathname.startsWith('/u/')) {
  const segments = window.location.pathname.split('/');
  const appBase = segments.slice(0, 4).join('/');
  API_BASE = `${appBase}/api`;
}

export const getSettings = () => fetch(`${API_BASE}/settings`).then(r => r.json());
export const saveSettings = (apiKey) => fetch(`${API_BASE}/settings`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ api_key: apiKey })
}).then(r => r.json().catch(() => ({})));

export const getConversations = (limit = 20, offset = 0) => fetch(`${API_BASE}/conversations?limit=${limit}&offset=${offset}`).then(r => r.json());
export const getConversationForks = (id) => fetch(`${API_BASE}/conversations/${id}/forks`).then(r => r.json());
export const getConversationTreeContext = (id) => fetch(`${API_BASE}/conversations/${id}/tree-context`).then(r => r.json());
export const createConversation = (title, params) => fetch(`${API_BASE}/conversations`, {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({ title: title || '', params: params || '' })
}).then(r => r.json());
export const getConversation = (id) => fetch(`${API_BASE}/conversations/${id}`).then(r => r.json());

export const updateParams = (id, params) => {
  const paramsStr = typeof params === 'string' ? params : JSON.stringify(params);
  return fetch(`${API_BASE}/conversations/${id}/params`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ params: paramsStr })
  }).then(r => r.json().catch(() => ({})));
};

export const updateTitle = (id, title) => fetch(`${API_BASE}/conversations/${id}/title`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ title })
}).then(r => r.json().catch(() => ({})));

export const getPresets = () => fetch(`${API_BASE}/presets`).then(r => r.json());
export const createPreset = (name, params) => fetch(`${API_BASE}/presets`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ name, params })
}).then(r => r.json());
export const deletePreset = (id) => fetch(`${API_BASE}/presets/${id}`, { method: 'DELETE' }).then(r => r.json());
export const updatePreset = (id, name, params) => fetch(`${API_BASE}/presets/${id}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ name, params })
}).then(r => r.json());
export const deleteConversation = (id) => fetch(`${API_BASE}/conversations/${id}`, { method: 'DELETE' }).then(r => r.json().catch(() => ({})));
export const forkConversation = (id, messageId) => fetch(`${API_BASE}/conversations/${id}/fork`, {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({ message_id: messageId })
}).then(r => r.json());
export const getModels = () => fetch(`${API_BASE}/models`).then(r => r.json());

