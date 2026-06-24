let API_BASE = '/api';
if (window.location.pathname.startsWith('/u/')) {
  const segments = window.location.pathname.split('/');
  const appBase = segments.slice(0, 4).join('/');
  API_BASE = `${appBase}/api`;
}

export async function fetchInfo() {
  const res = await fetch(`${API_BASE}/info`);
  if (!res.ok) throw new Error(`Failed to fetch info: ${res.statusText}`);
  return res.json();
}

export async function listFiles(volume, path = '') {
  const res = await fetch(
    `${API_BASE}/files?volume=${encodeURIComponent(volume)}&path=${encodeURIComponent(path)}`
  );
  if (!res.ok) throw new Error(`Failed to list files: ${res.statusText}`);
  return res.json();
}

export async function uploadFiles(volume, path, files, onProgress) {
  return new Promise((resolve, reject) => {
    const formData = new FormData();
    for (const file of files) {
      formData.append('files', file);
    }

    const xhr = new XMLHttpRequest();
    xhr.open(
      'POST', 
      `${API_BASE}/files?volume=${encodeURIComponent(volume)}&path=${encodeURIComponent(path)}`
    );

    xhr.upload.onprogress = (e) => {
      if (e.lengthComputable && onProgress) {
        onProgress(Math.round((e.loaded / e.total) * 100));
      }
    };

    xhr.onload = () => {
      if (xhr.status >= 200 && xhr.status < 300) {
        resolve(JSON.parse(xhr.responseText || '{}'));
      } else {
        reject(new Error(`Upload failed: ${xhr.statusText}`));
      }
    };

    xhr.onerror = () => reject(new Error('Upload failed'));
    xhr.send(formData);
  });
}

export function getFileUrl(volume, path, filename, volIndex = null) {
  let url = `${API_BASE}/files/download?volume=${encodeURIComponent(volume)}&path=${encodeURIComponent(path)}`;
  if (Array.isArray(filename)) {
    filename.forEach(name => {
      url += `&file=${encodeURIComponent(name)}`;
    });
  } else {
    url += `&file=${encodeURIComponent(filename)}`;
  }
  if (volIndex !== null) {
    url += `&vol_index=${volIndex}`;
  }
  return url;
}

export async function getDownloadPlan(volume, path, filename) {
  let url = `${API_BASE}/files/download-plan?volume=${encodeURIComponent(volume)}&path=${encodeURIComponent(path)}`;
  if (Array.isArray(filename)) {
    filename.forEach(name => {
      url += `&file=${encodeURIComponent(name)}`;
    });
  } else {
    url += `&file=${encodeURIComponent(filename)}`;
  }
  const res = await fetch(url);
  if (!res.ok) throw new Error(`Failed to fetch download plan: ${res.statusText}`);
  return res.json();
}

export async function downloadFile(volume, path, filename) {
  const res = await fetch(
    `${API_BASE}/files/download?volume=${encodeURIComponent(volume)}&path=${encodeURIComponent(path)}&file=${encodeURIComponent(filename)}`
  );
  if (!res.ok) throw new Error(`Failed to download file: ${res.statusText}`);
  const blob = await res.blob();
  const url = window.URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url;
  a.download = filename;
  document.body.appendChild(a);
  a.click();
  a.remove();
  window.URL.revokeObjectURL(url);
}

export async function deleteFile(volume, path, filename) {
  const res = await fetch(
    `${API_BASE}/files?volume=${encodeURIComponent(volume)}&path=${encodeURIComponent(path)}&file=${encodeURIComponent(filename)}`,
    { method: 'DELETE' }
  );
  if (!res.ok) throw new Error(`Failed to delete file: ${res.statusText}`);
  return res.json();
}

export async function createFolder(volume, path, name) {
  const res = await fetch(
    `${API_BASE}/folders?volume=${encodeURIComponent(volume)}&path=${encodeURIComponent(path)}`,
    {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({ name }),
    }
  );
  if (!res.ok) throw new Error(`Failed to create folder: ${res.statusText}`);
  return res.json();
}

export async function moveFile(volume, path, filename, destPath, newName = '') {
  let url = `${API_BASE}/files/move?volume=${encodeURIComponent(volume)}&path=${encodeURIComponent(path)}&file=${encodeURIComponent(filename)}&dest_path=${encodeURIComponent(destPath)}`;
  if (newName) {
    url += `&new_name=${encodeURIComponent(newName)}`;
  }
  const res = await fetch(url, { method: 'POST' });
  if (!res.ok) {
    const errorData = await res.json().catch(() => ({}));
    throw new Error(errorData.error || `Failed to move/rename file: ${res.statusText}`);
  }
  return res.json();
}

export async function fetchContacts() {
  const res = await fetch(`${API_BASE}/contacts`);
  if (!res.ok) throw new Error(`Failed to fetch contacts: ${res.statusText}`);
  return res.json();
}

export async function sendFile(volume, path, filename, contactID) {
  const url = `${API_BASE}/files/send?volume=${encodeURIComponent(volume)}&path=${encodeURIComponent(path)}&file=${encodeURIComponent(filename)}&contact_id=${encodeURIComponent(contactID)}`;
  const res = await fetch(url, { method: 'POST' });
  if (!res.ok) {
    const errorData = await res.json().catch(() => ({}));
    throw new Error(errorData.error || `Failed to send file: ${res.statusText}`);
  }
  return res.json();
}

export async function restoreTrashFile(filename) {
  const url = `${API_BASE}/trash/restore?file=${encodeURIComponent(filename)}`;
  const res = await fetch(url, { method: 'POST' });
  if (!res.ok) {
    const errorData = await res.json().catch(() => ({}));
    throw new Error(errorData.error || `Failed to restore file: ${res.statusText}`);
  }
  return res.json();
}

export async function emptyTrash() {
  const url = `${API_BASE}/trash/empty`;
  const res = await fetch(url, { method: 'POST' });
  if (!res.ok) {
    const errorData = await res.json().catch(() => ({}));
    throw new Error(errorData.error || `Failed to empty trash: ${res.statusText}`);
  }
  return res.json();
}

export async function fetchTransfers() {
  const res = await fetch(`${API_BASE}/transfers`);
  if (!res.ok) throw new Error(`Failed to fetch sent history: ${res.statusText}`);
  return res.json();
}

export async function fetchFileTransfers(filename, path = '') {
  const res = await fetch(
    `${API_BASE}/transfers/file?file=${encodeURIComponent(filename)}&path=${encodeURIComponent(path)}`
  );
  if (!res.ok) throw new Error(`Failed to fetch file sent history: ${res.statusText}`);
  return res.json();
}

export async function fetchAutoSendFolders() {
  const res = await fetch(`${API_BASE}/auto-send/all`);
  if (!res.ok) throw new Error(`Failed to fetch auto-send folders list: ${res.statusText}`);
  return res.json();
}

export async function fetchAutoSendConfig(path) {
  const res = await fetch(`${API_BASE}/auto-send?path=${encodeURIComponent(path)}`);
  if (!res.ok) throw new Error(`Failed to fetch auto-send folder config: ${res.statusText}`);
  return res.json();
}

export async function saveAutoSendConfig(path, contactIDs) {
  const res = await fetch(`${API_BASE}/auto-send`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ path, contact_ids: contactIDs }),
  });
  if (!res.ok) {
    const errorData = await res.json().catch(() => ({}));
    throw new Error(errorData.error || `Failed to save config: ${res.statusText}`);
  }
  return res.json();
}

export async function disableAutoSend(path) {
  const res = await fetch(`${API_BASE}/auto-send?path=${encodeURIComponent(path)}`, {
    method: 'DELETE',
  });
  if (!res.ok) {
    const errorData = await res.json().catch(() => ({}));
    throw new Error(errorData.error || `Failed to disable auto-send: ${res.statusText}`);
  }
  return res.json();
}

export async function fetchRecentTransfers(limit = 15) {
  const res = await fetch(`${API_BASE}/transfers?limit=${limit}`);
  if (!res.ok) throw new Error(`Failed to fetch recent transfers: ${res.statusText}`);
  return res.json();
}

export async function fetchActiveTransfers() {
  const res = await fetch(`${API_BASE}/transfers/active-list`);
  if (!res.ok) throw new Error(`Failed to fetch active transfers: ${res.statusText}`);
  return res.json();
}

export async function pasteItems(action, srcVolume, srcPath, destVolume, destPath, items) {
  const res = await fetch(`${API_BASE}/files/paste`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      action,
      src_volume: srcVolume,
      src_path: srcPath,
      dest_volume: destVolume,
      dest_path: destPath,
      items
    })
  });
  if (!res.ok) {
    const err = await res.json();
    throw new Error(err.error || 'Failed to paste items');
  }
  return res.json();
}


