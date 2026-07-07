import { createLibp2p } from 'https://esm.sh/libp2p';
import { webSockets } from 'https://esm.sh/@libp2p/websockets';
import { noise } from 'https://esm.sh/@chainsafe/libp2p-noise';
import { yamux } from 'https://esm.sh/@chainsafe/libp2p-yamux';
import { circuitRelayClient } from 'https://esm.sh/@libp2p/circuit-relay-v2';
import { multiaddr } from 'https://esm.sh/@multiformats/multiaddr';

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
document.getElementById('relay-url').value = localStorage.getItem('p2p_relay_multiaddr') || '';
document.getElementById('peer-id').value = localStorage.getItem('p2p_home_peer_id') || '';

let p2pNode = null;
let homePeerId = '';
let relayMultiaddr = '';
let pairingToken = localStorage.getItem('p2p_pairing_token') || '';

form.addEventListener('submit', async (e) => {
  e.preventDefault();
  
  relayMultiaddr = document.getElementById('relay-url').value.trim();
  homePeerId = document.getElementById('peer-id').value.trim();
  const pairingCode = document.getElementById('pairing-code').value.trim();

  localStorage.setItem('p2p_relay_multiaddr', relayMultiaddr);
  localStorage.setItem('p2p_home_peer_id', homePeerId);

  showStatus('Connecting to P2P Relay...', 'connecting');

  try {
    // 1. Initialize P2P Node
    p2pNode = await createLibp2p({
      transports: [webSockets()],
      connectionEncryption: [noise()],
      streamMuxers: [yamux()],
      services: {
        relay: circuitRelayClient()
      }
    });
    await p2pNode.start();

    // 2. Connect to Relay
    const relayAddr = multiaddr(relayMultiaddr);
    await p2pNode.dial(relayAddr);

    // 3. Pair/Establish Tunnel Connection
    showStatus('Pairing with Home Magicbox...', 'connecting');
    const target = multiaddr(`${relayMultiaddr}/p2p-circuit/p2p/${homePeerId}`);
    
    const stream = await p2pNode.dialProtocol(target, '/magicbox/tunnel/1.0.0');
    
    // Handshake
    const handshake = {
      otp: pairingToken ? '' : pairingCode,
      token: pairingToken
    };
    await writeJSONToStream(stream, handshake);
    
    const response = await readJSONFromStream(stream);
    stream.close();

    if (response.success) {
      if (response.token) {
        pairingToken = response.token;
        localStorage.setItem('p2p_pairing_token', pairingToken);
      }
      showStatus('P2P Tunnel Established!', 'success');
      
      setTimeout(() => {
        setupView.style.display = 'none';
        tunnelFrame.style.display = 'block';
        tunnelFrame.src = '/tunnel/chat/';
      }, 1000);
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

// Write JSON payload to stream (prefixed with 4-byte length header)
async function writeJSONToStream(stream, obj) {
  const str = JSON.stringify(obj);
  const encoder = new TextEncoder();
  const bytes = encoder.encode(str);
  
  const lengthHeader = new Uint8Array(4);
  new DataView(lengthHeader.buffer).setUint32(0, bytes.length, false);

  const writer = stream.sink;
  await writer.push([lengthHeader, bytes]);
}

// Read JSON payload from stream (prefixed with 4-byte length header)
async function readJSONFromStream(stream) {
  const reader = stream.source;
  
  // Read length header (4 bytes)
  let { value: lengthBytes } = await reader.next();
  if (lengthBytes.length < 4) {
    throw new Error('Invalid handshake length header');
  }
  const length = new DataView(lengthBytes.buffer).getUint32(0, false);

  // Read message body
  let { value: messageBytes } = await reader.next();
  const decoder = new TextDecoder();
  const jsonStr = decoder.decode(messageBytes.subarray(0, length));
  return JSON.parse(jsonStr);
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

async function tunnelRequestOverP2P(request) {
  if (!p2pNode) {
    throw new Error('P2P Node is not running');
  }

  const target = multiaddr(`${relayMultiaddr}/p2p-circuit/p2p/${homePeerId}`);
  const stream = await p2pNode.dialProtocol(target, '/magicbox/tunnel/1.0.0');

  // 1. Send authorization header first
  const authHeader = { token: pairingToken };
  await writeJSONToStream(stream, authHeader);

  // 2. Serialize HTTP request and write it to stream
  const serialized = serializeHTTPRequest(request.method, request.path, request.headers, request.body);
  await stream.sink.push([serialized]);

  // 3. Read raw response from stream
  const responseBytes = await readAllBytes(stream.source);
  stream.close();

  // 4. Parse response
  return parseHTTPResponse(responseBytes);
}

function serializeHTTPRequest(method, path, headers, bodyArray) {
  let req = `${method} ${path} HTTP/1.1\r\n`;
  for (const [key, value] of Object.entries(headers)) {
    req += `${key}: ${value}\r\n`;
  }
  req += '\r\n';
  
  const encoder = new TextEncoder();
  const headerBytes = encoder.encode(req);
  
  if (bodyArray) {
    const bodyBytes = new Uint8Array(bodyArray);
    const combined = new Uint8Array(headerBytes.length + bodyBytes.length);
    combined.set(headerBytes, 0);
    combined.set(bodyBytes, headerBytes.length);
    return combined;
  }
  return headerBytes;
}

async function readAllBytes(source) {
  const chunks = [];
  let totalLength = 0;
  for await (const chunk of source) {
    chunks.push(chunk);
    totalLength += chunk.length;
  }
  const result = new Uint8Array(totalLength);
  let offset = 0;
  for (const chunk of chunks) {
    result.set(chunk, offset);
    offset += chunk.length;
  }
  return result;
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
    headers[key] = value;
  }
  
  return {
    status,
    statusText,
    headers,
    body: Array.from(bodyBytes)
  };
}
