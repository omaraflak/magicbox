function showCustomAlert(message) {
  const overlay = document.createElement('div');
  overlay.style.position = 'fixed';
  overlay.style.top = '0';
  overlay.style.left = '0';
  overlay.style.right = '0';
  overlay.style.bottom = '0';
  overlay.style.background = 'rgba(0, 0, 0, 0.7)';
  overlay.style.backdropFilter = 'blur(8px)';
  overlay.style.display = 'flex';
  overlay.style.justifyContent = 'center';
  overlay.style.alignItems = 'center';
  overlay.style.zIndex = '100000';

  const card = document.createElement('div');
  card.style.background = '#1e293b';
  card.style.border = '1px solid #334155';
  card.style.borderRadius = '12px';
  card.style.padding = '24px';
  card.style.maxWidth = '400px';
  card.style.width = '90%';
  card.style.textAlign = 'center';
  card.style.boxShadow = '0 10px 25px -5px rgba(0, 0, 0, 0.3)';
  card.style.color = '#f8fafc';
  card.style.fontFamily = 'system-ui, -apple-system, sans-serif';

  const title = document.createElement('h3');
  title.innerText = '⚠️ Attention Required';
  title.style.fontSize = '1.1rem';
  title.style.fontWeight = '600';
  title.style.marginBottom = '12px';
  title.style.color = '#ef4444';

  const text = document.createElement('p');
  text.innerText = message;
  text.style.fontSize = '0.85rem';
  text.style.color = '#94a3b8';
  text.style.marginBottom = '24px';
  text.style.lineHeight = '1.5';

  const btn = document.createElement('button');
  btn.innerText = 'Dismiss';
  btn.style.background = '#334155';
  btn.style.color = '#f8fafc';
  btn.style.border = 'none';
  btn.style.borderRadius = '6px';
  btn.style.padding = '8px 20px';
  btn.style.fontSize = '0.85rem';
  btn.style.cursor = 'pointer';
  btn.style.fontWeight = '500';
  btn.style.transition = 'background-color 0.2s';
  btn.onmouseover = () => btn.style.background = '#475569';
  btn.onmouseout = () => btn.style.background = '#334155';
  btn.onclick = () => {
    document.body.removeChild(overlay);
  };

  card.appendChild(title);
  card.appendChild(text);
  card.appendChild(btn);
  overlay.appendChild(card);
  document.body.appendChild(overlay);
}

export let API_BASE = '/api';
if (window.location.pathname.startsWith('/u/')) {
  const segments = window.location.pathname.split('/');
  const appBase = segments.slice(0, 4).join('/');
  API_BASE = `${appBase}/api`;
}

async function apiFetch(url, options = {}) {
  const res = await fetch(url, options);
  if (res.status === 403) {
    const clone = res.clone();
    try {
      const err = await clone.json();
      if (err.error === 'consent_required' && err.request_id) {
        return new Promise((resolve, reject) => {
          const popup = window.open(`/consent?req_id=${err.request_id}`, 'ConsentRequired', 'width=500,height=600');
          if (!popup) {
            showCustomAlert('Consent popup was blocked. Please allow popups for this site.');
            reject(new Error('Consent popup blocked'));
            return;
          }

          const listener = async (event) => {
            if (event.data?.type === 'consent_decision' && event.data?.request_id === err.request_id) {
              window.removeEventListener('message', listener);
              if (event.data?.approved) {
                try {
                  const retryRes = await apiFetch(url, options);
                  resolve(retryRes);
                } catch (retryErr) {
                  reject(retryErr);
                }
              } else {
                reject(new Error('Consent denied'));
              }
            }
          };
          window.addEventListener('message', listener);
        });
      }
    } catch (e) {
      // Fall through to returning response
    }
  }
  return res;
}

export const getSettings = () => apiFetch(`${API_BASE}/settings`).then(r => r.json());
export const saveSettings = (apiKey) => apiFetch(`${API_BASE}/settings`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ api_key: apiKey })
}).then(r => r.json().catch(() => ({})));

export const getConversations = (limit = 20, offset = 0) => apiFetch(`${API_BASE}/conversations?limit=${limit}&offset=${offset}`).then(r => r.json());
export const getConversationForks = (id) => apiFetch(`${API_BASE}/conversations/${id}/forks`).then(r => r.json());
export const getConversationTreeContext = (id) => apiFetch(`${API_BASE}/conversations/${id}/tree-context`).then(r => r.json());
export const createConversation = (title, params) => apiFetch(`${API_BASE}/conversations`, {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({ title: title || '', params: params || '' })
}).then(r => r.json());
export const getConversation = (id) => apiFetch(`${API_BASE}/conversations/${id}`).then(r => r.json());

export const updateParams = (id, params) => {
  const paramsStr = typeof params === 'string' ? params : JSON.stringify(params);
  return apiFetch(`${API_BASE}/conversations/${id}/params`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ params: paramsStr })
  }).then(r => r.json().catch(() => ({})));
};

export const updateTitle = (id, title) => apiFetch(`${API_BASE}/conversations/${id}/title`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ title })
}).then(r => r.json().catch(() => ({})));

export const getPresets = () => apiFetch(`${API_BASE}/presets`).then(r => r.json());
export const createPreset = (name, params) => apiFetch(`${API_BASE}/presets`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ name, params })
}).then(r => r.json());
export const deletePreset = (id) => apiFetch(`${API_BASE}/presets/${id}`, { method: 'DELETE' }).then(r => r.json());
export const updatePreset = (id, name, params) => apiFetch(`${API_BASE}/presets/${id}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ name, params })
}).then(r => r.json());
export const deleteConversation = (id) => apiFetch(`${API_BASE}/conversations/${id}`, { method: 'DELETE' }).then(r => r.json().catch(() => ({})));
export const forkConversation = (id, messageId) => apiFetch(`${API_BASE}/conversations/${id}/fork`, {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({ message_id: messageId })
}).then(r => r.json());
export const getModels = () => apiFetch(`${API_BASE}/models`).then(r => r.json());
