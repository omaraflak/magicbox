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
  let url = `${API_BASE}/files/download?volume=${encodeURIComponent(volume)}&path=${encodeURIComponent(path)}&file=${encodeURIComponent(filename)}`;
  if (volIndex !== null) {
    url += `&vol_index=${volIndex}`;
  }
  return url;
}

export async function getDownloadPlan(volume, path, filename) {
  const res = await fetch(
    `${API_BASE}/files/download-plan?volume=${encodeURIComponent(volume)}&path=${encodeURIComponent(path)}&file=${encodeURIComponent(filename)}`
  );
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

export async function shareFile(volume, path, filename, contactID) {
  const url = `${API_BASE}/files/share?volume=${encodeURIComponent(volume)}&path=${encodeURIComponent(path)}&file=${encodeURIComponent(filename)}&contact_id=${encodeURIComponent(contactID)}`;
  const res = await fetch(url, { method: 'POST' });
  if (!res.ok) {
    const errorData = await res.json().catch(() => ({}));
    throw new Error(errorData.error || `Failed to share file: ${res.statusText}`);
  }
  return res.json();
}
