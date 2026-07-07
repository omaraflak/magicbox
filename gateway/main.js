import { createLibp2p } from 'https://esm.sh/libp2p';
import { webSockets } from 'https://esm.sh/@libp2p/websockets';
import { noise } from 'https://esm.sh/@chainsafe/libp2p-noise';
import { yamux } from 'https://esm.sh/@chainsafe/libp2p-yamux';
import { circuitRelayTransport } from 'https://esm.sh/@libp2p/circuit-relay-v2';
import { identify } from 'https://esm.sh/@libp2p/identify';
import { multiaddr } from 'https://esm.sh/@multiformats/multiaddr';
import { peerIdFromString } from 'https://esm.sh/@libp2p/peer-id';
import { webRTC } from 'https://esm.sh/@libp2p/webrtc';

// Register Service Worker
if ('serviceWorker' in navigator) {
  navigator.serviceWorker.register('sw.js')
    .then(reg => console.log('Service Worker registered', reg))
    .catch(err => console.error('Service Worker registration failed', err));
}

const form = document.getElementById('setup-form');
const statusBox = document.getElementById('status-message');
const setupView = document.getElementById('setup-view');
const tunnelFrame = document.getElementById('tunnel-frame');

// Load saved values
const savedCode = localStorage.getItem('p2p_connection_code') || '';
document.getElementById('connection-code').value = savedCode;
document.getElementById('relay-url').value = localStorage.getItem('p2p_relay_multiaddr') || '';
document.getElementById('peer-id').value = localStorage.getItem('p2p_home_peer_id') || '';

// Decode Connection Code helper
function decodeConnectionCode(codeStr) {
  try {
    const rawJson = atob(codeStr.trim());
    const parsed = JSON.parse(rawJson);
    if (parsed.r && parsed.p && parsed.c) {
      return {
        relay: parsed.r,
        peer: parsed.p,
        otp: parsed.c
      };
    }
  } catch (e) {
    // Not a valid base64 json pairing code
  }
  return null;
}

// Auto-fill manual fields on pasting/typing connection code
document.getElementById('connection-code').addEventListener('input', (e) => {
  const code = e.target.value.trim();
  const decoded = decodeConnectionCode(code);
  if (decoded) {
    document.getElementById('relay-url').value = decoded.relay;
    document.getElementById('peer-id').value = decoded.peer;
    document.getElementById('pairing-code').value = decoded.otp;
  }
});

// Parse initial code if present
if (savedCode) {
  const decoded = decodeConnectionCode(savedCode);
  if (decoded) {
    document.getElementById('relay-url').value = decoded.relay;
    document.getElementById('peer-id').value = decoded.peer;
    document.getElementById('pairing-code').value = decoded.otp;
  }
}

let p2pNode = null;
let homePeerId = '';
let relayMultiaddr = '';
let pairingToken = localStorage.getItem('p2p_pairing_token') || '';
let connectedPeerId = null; // Stores the PeerId object after initial connection

// Manual cookie jar — browsers can't store Set-Cookie from SW-constructed responses
const cookieJar = new Map();

form.addEventListener('submit', async (e) => {
  e.preventDefault();
  
  const connectionCodeInput = document.getElementById('connection-code').value.trim();
  const decoded = decodeConnectionCode(connectionCodeInput);

  if (decoded) {
    relayMultiaddr = decoded.relay;
    homePeerId = decoded.peer;
  } else {
    relayMultiaddr = document.getElementById('relay-url').value.trim();
    homePeerId = document.getElementById('peer-id').value.trim();
  }

  const pairingCode = decoded ? decoded.otp : document.getElementById('pairing-code').value.trim();

  if (!relayMultiaddr || !homePeerId) {
    showStatus('Missing configuration parameters. Please enter a Connection Code or fill manual fields.', 'error');
    return;
  }

  localStorage.setItem('p2p_connection_code', connectionCodeInput);
  localStorage.setItem('p2p_relay_multiaddr', relayMultiaddr);
  localStorage.setItem('p2p_home_peer_id', homePeerId);

  showStatus('Connecting to P2P Relay...', 'connecting');

  try {
    // 1. Initialize P2P Node
    p2pNode = await createLibp2p({
      transports: [
        webSockets(),
        webRTC(),
        circuitRelayTransport()
      ],
      connectionEncrypters: [noise()],
      streamMuxers: [yamux()],
      services: {
        identify: identify()
      },
      connectionGater: {
        denyDialMultiaddr: () => false
      }
    });
    await p2pNode.start();
    console.log('P2P Node started:', p2pNode.peerId.toString());

    // Log connection events to see when WebRTC direct connection is established
    p2pNode.addEventListener('connection:open', (event) => {
      const conn = event.detail;
      const isRelayed = conn.remoteAddr.toString().includes('/p2p-circuit');
      console.log(`Connection opened: ${conn.remoteAddr.toString()} [${isRelayed ? 'RELAY' : 'DIRECT'}]`);
    });
    p2pNode.addEventListener('connection:close', (event) => {
      const conn = event.detail;
      console.log(`Connection closed: ${conn.remoteAddr.toString()}`);
    });

    // 2. Connect to Relay
    const relayAddr = multiaddr(relayMultiaddr);
    await p2pNode.dial(relayAddr);
    console.log('Connected to relay');

    // 3. Pair/Establish Tunnel Connection
    showStatus('Pairing with Home Magicbox...', 'connecting');
    const target = multiaddr(`${relayMultiaddr}/p2p-circuit/p2p/${homePeerId}`);
    
    const stream = await p2pNode.dialProtocol(target, '/magicbox/tunnel/1.0.0', {
      runOnTransientConnection: true,
      runOnLimitedConnection: true
    });
    // Store the peer ID for connection reuse
    connectedPeerId = peerIdFromString(homePeerId);
    console.log('Stream opened, sending handshake...');

    // Handshake: write then read using the stream's byte-level API
    const handshake = {
      otp: pairingToken ? '' : pairingCode,
      token: pairingToken
    };

    const response = await sendAndReceiveJSON(stream, handshake);
    await stream.close();

    if (response.success) {
      if (response.token) {
        pairingToken = response.token;
        localStorage.setItem('p2p_pairing_token', pairingToken);
      }
      showStatus('P2P Tunnel Established!', 'success');
      console.log('Handshake response:', response);
      
      // Wait for service worker to be ready before loading tunnel
      await navigator.serviceWorker.ready;
      console.log('Service worker ready, loading tunnel...');

      setupView.style.display = 'none';
      tunnelFrame.style.display = 'block';
      // Load the magicbox root via the tunnel (SW strips /tunnel prefix)
      tunnelFrame.src = '/tunnel/';
    } else {
      showStatus('Pairing failed: ' + (response.error || 'Invalid passcode'), 'error');
      p2pNode.stop();
    }
  } catch (err) {
    console.error(err);
    showStatus('Connection failed: ' + err.message, 'error');
    if (p2pNode) p2pNode.stop();
  }
});

function showStatus(text, type) {
  statusBox.textContent = text;
  statusBox.className = 'status-box status-' + type;
  statusBox.style.display = 'block';
}

// ── Stream I/O helpers ──────────────────────────────────────────────
// libp2p v3 stream API: stream.send(data) for writing,
// for-await-of stream for reading (Symbol.asyncIterator).

/**
 * Build a 4-byte big-endian length-prefixed frame from a JSON object.
 */
function buildJSONFrame(obj) {
  const payload = new TextEncoder().encode(JSON.stringify(obj));
  const frame = new Uint8Array(4 + payload.length);
  new DataView(frame.buffer).setUint32(0, payload.length, false);
  frame.set(payload, 4);
  return frame;
}

/**
 * Send a framed JSON message and read a framed JSON response on a stream.
 * Reads exactly the expected frame (length header + body) without waiting
 * for the stream to close, since the Go side keeps the stream open.
 */
async function sendAndReceiveJSON(stream, obj) {
  const frame = buildJSONFrame(obj);

  // Write the frame
  await stream.send(frame);

  // Read exactly the response frame using the async iterator
  const reader = new StreamByteReader(stream);
  
  // Read 4-byte length header
  const lenBytes = await reader.readExact(4);
  const length = new DataView(lenBytes.buffer, lenBytes.byteOffset).getUint32(0, false);
  
  // Read JSON body
  const bodyBytes = await reader.readExact(length);
  return JSON.parse(new TextDecoder().decode(bodyBytes));
}

/**
 * Open a tunnel stream, perform the full Go handler protocol, and return
 * the raw HTTP response bytes.
 *
 * Go handler protocol per stream:
 *   1. Read handshake (4-byte len + JSON {otp, token})
 *   2. Send handshake response (4-byte len + JSON {success, token, error})
 *   3. Read auth header (4-byte len + JSON {token})
 *   4. Read raw HTTP request
 *   5. Send raw HTTP response
 *   6. Close stream
 */
async function tunnelStreamRequest(stream, token, requestBytes) {
  const reader = new StreamByteReader(stream);

  // 1. Send handshake with token
  await stream.send(buildJSONFrame({ otp: '', token: token }));

  // 2. Read handshake response
  const respLenBytes = await reader.readExact(4);
  const respLen = new DataView(respLenBytes.buffer, respLenBytes.byteOffset).getUint32(0, false);
  const respBody = await reader.readExact(respLen);
  const handshakeResp = JSON.parse(new TextDecoder().decode(respBody));
  if (!handshakeResp.success) {
    throw new Error('Tunnel auth failed: ' + (handshakeResp.error || 'unauthorized'));
  }

  // 3. Send auth header
  await stream.send(buildJSONFrame({ token: token }));

  // 4. Send raw HTTP request
  await stream.send(requestBytes);

  // 5. Read raw HTTP response (until stream closes)
  const responseBytes = await reader.readAll();
  return responseBytes;
}

/**
 * A byte reader that reads exact byte counts from an async iterable stream.
 * Handles the case where chunks from the stream may be smaller or larger
 * than the requested read size.
 */
class StreamByteReader {
  constructor(stream) {
    this.iterator = stream[Symbol.asyncIterator]();
    this.buffer = new Uint8Array(0);
  }

  async readExact(n) {
    const parts = [];
    let remaining = n;

    // Use leftover bytes from previous reads
    if (this.buffer.length > 0) {
      if (this.buffer.length >= n) {
        const result = this.buffer.subarray(0, n);
        this.buffer = this.buffer.subarray(n);
        return result;
      }
      parts.push(this.buffer);
      remaining -= this.buffer.length;
      this.buffer = new Uint8Array(0);
    }

    // Read from stream until we have enough bytes
    while (remaining > 0) {
      const { value, done } = await this.iterator.next();
      if (done) throw new Error(`Stream ended with ${remaining} bytes still needed`);
      
      const bytes = value.subarray ? value.subarray() : new Uint8Array(value);
      if (bytes.length <= remaining) {
        parts.push(bytes);
        remaining -= bytes.length;
      } else {
        parts.push(bytes.subarray(0, remaining));
        this.buffer = bytes.subarray(remaining);
        remaining = 0;
      }
    }

    // Concatenate parts
    if (parts.length === 1) return parts[0];
    const result = new Uint8Array(n);
    let offset = 0;
    for (const p of parts) {
      result.set(p, offset);
      offset += p.length;
    }
    return result;
  }

  /** Read all remaining bytes until stream closes. */
  async readAll() {
    const chunks = [];
    let total = 0;

    if (this.buffer.length > 0) {
      chunks.push(this.buffer);
      total += this.buffer.length;
      this.buffer = new Uint8Array(0);
    }

    while (true) {
      const { value, done } = await this.iterator.next();
      if (done) break;
      const bytes = value.subarray ? value.subarray() : new Uint8Array(value);
      chunks.push(bytes);
      total += bytes.length;
    }

    const result = new Uint8Array(total);
    let offset = 0;
    for (const c of chunks) {
      result.set(c, offset);
      offset += c.length;
    }
    return result;
  }
}

/**
 * Collect all bytes from an async iterable stream (reads until close).
 */
async function collectBytes(stream) {
  const chunks = [];
  let total = 0;
  for await (const chunk of stream) {
    // chunk may be Uint8Array or Uint8ArrayList; get raw bytes
    const bytes = chunk.subarray ? chunk.subarray() : new Uint8Array(chunk);
    chunks.push(bytes);
    total += bytes.length;
  }
  const result = new Uint8Array(total);
  let offset = 0;
  for (const c of chunks) {
    result.set(c, offset);
    offset += c.length;
  }
  return result;
}

// Listen to service worker fetch requests
navigator.serviceWorker.addEventListener('message', async (event) => {
  if (event.data && event.data.type === 'TUNNEL_REQUEST') {
    const request = event.data.payload;
    try {
      const response = await tunnelRequestOverP2P(request);
      event.source.postMessage({
        type: 'TUNNEL_RESPONSE',
        id: request.id,
        payload: response
      });
    } catch (err) {
      console.error('Failed to tunnel request', err);
      event.source.postMessage({
        type: 'TUNNEL_RESPONSE',
        id: request.id,
        payload: {
          status: 502,
          statusText: 'Bad Gateway',
          headers: { 'Content-Type': 'text/plain' },
          body: Array.from(new TextEncoder().encode('P2P Tunnel Error: ' + err.message))
        }
      });
    }
  }
});

// Serial queue for tunnel requests — relay connections can't handle
// many concurrent stream opens, so we serialize them.
let tunnelQueue = Promise.resolve();

async function tunnelRequestOverP2P(request) {
  // Queue requests serially to avoid overwhelming the relay
  const result = tunnelQueue.then(() => _doTunnelRequest(request));
  tunnelQueue = result.catch(() => {}); // prevent queue from breaking on errors
  return result;
}

async function _doTunnelRequest(request) {
  if (!p2pNode) {
    throw new Error('P2P Node is not running');
  }

  console.log('Tunneling request:', request.method, request.path);

  const maxRetries = 3;
  for (let attempt = 1; attempt <= maxRetries; attempt++) {
    try {
      const stream = await p2pNode.dialProtocol(connectedPeerId, '/magicbox/tunnel/1.0.0', {
        runOnTransientConnection: true,
        runOnLimitedConnection: true
      });

      const serialized = serializeHTTPRequest(request.method, request.path, request.headers, request.body);
      const responseBytes = await tunnelStreamRequest(stream, pairingToken, serialized);
      await stream.close();

      const parsed = parseHTTPResponse(responseBytes);
      console.log('Tunnel response:', parsed.status, parsed.statusText);
      return parsed;
    } catch (err) {
      const isRetryable = err.message && (
        err.message.includes('CONNECTION_FAILED') ||
        err.message.includes('stream has been reset') ||
        err.message.includes('connection closed')
      );

      if (isRetryable && attempt < maxRetries) {
        console.warn(`Tunnel attempt ${attempt} failed, reconnecting...`, err.message);
        try {
          // Re-establish relay connection
          const target = multiaddr(`${relayMultiaddr}/p2p-circuit/p2p/${homePeerId}`);
          await p2pNode.dial(target);
        } catch (reconnectErr) {
          console.warn('Reconnect failed:', reconnectErr.message);
        }
        // Small delay before retry
        await new Promise(r => setTimeout(r, 200 * attempt));
        continue;
      }
      throw err;
    }
  }
}

function serializeHTTPRequest(method, path, headers, bodyArray) {
  let req = `${method} ${path} HTTP/1.1\r\n`;
  for (const [key, value] of Object.entries(headers)) {
    // Skip browser-only headers that the Go server doesn't need
    if (key.toLowerCase() === 'cookie') continue; // We'll add our own
    req += `${key}: ${value}\r\n`;
  }
  
  // Inject cookies from our jar
  if (cookieJar.size > 0) {
    const cookieStr = Array.from(cookieJar.entries())
      .map(([name, value]) => `${name}=${value}`)
      .join('; ');
    req += `Cookie: ${cookieStr}\r\n`;
  }
  
  // Add Content-Length if there's a body and the header isn't already present
  if (bodyArray && bodyArray.length > 0) {
    const hasContentLength = Object.keys(headers).some(k => k.toLowerCase() === 'content-length');
    if (!hasContentLength) {
      req += `Content-Length: ${bodyArray.length}\r\n`;
    }
  }
  
  req += '\r\n';
  
  const encoder = new TextEncoder();
  const headerBytes = encoder.encode(req);
  
  if (bodyArray && bodyArray.length > 0) {
    const bodyBytes = new Uint8Array(bodyArray);
    const combined = new Uint8Array(headerBytes.length + bodyBytes.length);
    combined.set(headerBytes, 0);
    combined.set(bodyBytes, headerBytes.length);
    return combined;
  }
  return headerBytes;
}

function parseHTTPResponse(bytes) {
  let headerEnd = -1;
  for (let i = 0; i < bytes.length - 3; i++) {
    if (bytes[i] === 13 && bytes[i+1] === 10 && bytes[i+2] === 13 && bytes[i+3] === 10) {
      headerEnd = i;
      break;
    }
  }
  
  if (headerEnd === -1) {
    throw new Error('Invalid HTTP response wire bytes');
  }
  
  const decoder = new TextDecoder();
  const headerStr = decoder.decode(bytes.subarray(0, headerEnd));
  const bodyBytes = bytes.subarray(headerEnd + 4);
  
  const lines = headerStr.split('\r\n');
  const statusLine = lines[0];
  const statusParts = statusLine.split(' ');
  const status = parseInt(statusParts[1]);
  const statusText = statusParts.slice(2).join(' ');
  
  const headers = {};
  for (let i = 1; i < lines.length; i++) {
    if (!lines[i]) continue;
    const colon = lines[i].indexOf(':');
    if (colon === -1) continue;
    const key = lines[i].substring(0, colon).trim().toLowerCase();
    const value = lines[i].substring(colon + 1).trim();
    
    // Extract cookies into our jar
    if (key === 'set-cookie') {
      const cookieParts = value.split(';')[0]; // Get "name=value" part
      const eqIdx = cookieParts.indexOf('=');
      if (eqIdx > 0) {
        const cookieName = cookieParts.substring(0, eqIdx).trim();
        const cookieValue = cookieParts.substring(eqIdx + 1).trim();
        cookieJar.set(cookieName, cookieValue);
        console.log('Cookie stored:', cookieName);
      }
    }
    
    headers[key] = value;
  }
  
  return {
    status,
    statusText,
    headers,
    body: Array.from(bodyBytes)
  };
}
