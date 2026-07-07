const pendingRequests = new Map();
let requestIdCounter = 0;

self.addEventListener('install', (event) => {
  self.skipWaiting();
});

self.addEventListener('activate', (event) => {
  event.waitUntil(self.clients.claim());
});

self.addEventListener('fetch', (event) => {
  const url = new URL(event.request.url);

  // We only intercept requests that go to the "/tunnel/" path prefix
  if (url.pathname.startsWith('/tunnel/')) {
    event.respondWith(handleTunnelRequest(event.request));
  }
});

async function handleTunnelRequest(request) {
  const clients = await self.clients.matchAll();
  if (clients.length === 0) {
    return new Response('P2P Gateway is offline. Please keep the gateway tab open.', {
      status: 503,
      statusText: 'Service Unavailable',
      headers: { 'Content-Type': 'text/plain' }
    });
  }

  const requestId = ++requestIdCounter;
  const url = new URL(request.url);
  // Strip "/tunnel" prefix to get the path that should go to the local home server
  const relativePath = url.pathname.replace(/^\/tunnel/, '') + url.search;

  // Read request body if present
  let body = null;
  if (request.method !== 'GET' && request.method !== 'HEAD') {
    body = await request.arrayBuffer();
  }

  // Gather headers
  const headers = {};
  for (const [key, value] of request.headers.entries()) {
    headers[key] = value;
  }

  const requestPayload = {
    id: requestId,
    path: relativePath,
    method: request.method,
    headers: headers,
    body: body ? Array.from(new Uint8Array(body)) : null
  };

  // Create promise that resolves when the main client responds with the HTTP response
  const responsePromise = new Promise((resolve) => {
    pendingRequests.set(requestId, resolve);
  });

  // Broadcast the request payload to the active client tabs (which are running the P2P node)
  clients.forEach(client => {
    client.postMessage({
      type: 'TUNNEL_REQUEST',
      payload: requestPayload
    });
  });

  // Wait for the main page to send back the response payload
  const responsePayload = await responsePromise;

  // Re-create the Response object
  const bodyUint8Array = new Uint8Array(responsePayload.body || []);
  const responseHeaders = new Headers();
  for (const [key, value] of Object.entries(responsePayload.headers || {})) {
    responseHeaders.set(key, value);
  }

  return new Response(bodyUint8Array, {
    status: responsePayload.status || 200,
    statusText: responsePayload.statusText || 'OK',
    headers: responseHeaders
  });
}

// Receive response back from the client tab
self.addEventListener('message', (event) => {
  if (event.data && event.data.type === 'TUNNEL_RESPONSE') {
    const { id, payload } = event.data;
    const resolve = pendingRequests.get(id);
    if (resolve) {
      resolve(payload);
      pendingRequests.delete(id);
    }
  }
});
