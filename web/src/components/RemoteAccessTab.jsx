import React, { useState } from 'react';

export default function RemoteAccessTab() {
    const [pairingData, setPairingData] = useState(null);
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState('');
    const [copiedField, setCopiedField] = useState(''); // '', 'relay', 'peer', 'otp'

    const handleGeneratePairingCode = async () => {
        setLoading(true);
        setError('');
        try {
            const res = await fetch('/api/v1/pairing/generate', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                }
            });
            if (res.ok) {
                const data = await res.json();
                setPairingData(data);
            } else {
                const text = await res.text();
                setError(text || 'Failed to generate pairing code');
            }
        } catch (err) {
            setError('Error contacting server: ' + err.message);
        } finally {
            setLoading(false);
        }
    };

    const handleCopy = (text, field) => {
        navigator.clipboard.writeText(text);
        setCopiedField(field);
        setTimeout(() => setCopiedField(''), 2000);
    };

    return (
        <div style={{ maxWidth: '600px' }}>
            <div style={{ borderBottom: '1px solid var(--border-color)', paddingBottom: '16px', marginBottom: '24px' }}>
                <h1 style={{ fontSize: '1.75rem', fontWeight: 600, color: 'var(--text-primary)' }}>Remote Access</h1>
                <p style={{ color: 'var(--text-muted)', fontSize: '0.9rem', marginTop: '6px' }}>
                    Connect to your Magicbox remotely from outside your home using a secure, private peer-to-peer web tunnel.
                </p>
            </div>

            <div style={{ marginTop: '24px' }}>
                {error && (
                    <div style={{ padding: '12px 16px', backgroundColor: 'rgba(239, 68, 68, 0.1)', border: '1px solid var(--danger-color)', color: 'var(--danger-color)', borderRadius: '6px', fontSize: '0.85rem', marginBottom: '16px' }}>
                        ⚠️ {error}
                    </div>
                )}

                {!pairingData ? (
                    <button 
                        className="btn btn-primary" 
                        onClick={handleGeneratePairingCode}
                        disabled={loading}
                    >
                        {loading ? 'Generating...' : '🔑 Generate Pairing Code'}
                    </button>
                ) : (
                    <div>
                        <p style={{ fontSize: '0.9rem', color: 'var(--text-muted)', marginBottom: '20px' }}>
                            Copy the Connection Code below and paste it into your P2P Gateway to establish the secure tunnel.
                        </p>

                        <div style={{ marginBottom: '24px' }}>
                            <label style={{ display: 'block', fontSize: '11px', fontWeight: 600, textTransform: 'uppercase', color: 'var(--text-muted)', marginBottom: '6px' }}>
                                Connection Code
                            </label>
                            <div style={{ display: 'flex', gap: '8px', marginBottom: '12px' }}>
                                <textarea 
                                    readOnly 
                                    value={btoa(JSON.stringify({
                                        r: pairingData.relay_multiaddr,
                                        p: pairingData.peer_id,
                                        c: pairingData.pairing_code
                                    }))}
                                    style={{ flex: 1, padding: '10px 14px', borderRadius: '6px', border: '1px solid var(--border-color)', backgroundColor: 'var(--bg-primary)', color: 'var(--text-primary)', fontFamily: 'monospace', fontSize: '0.85rem', minHeight: '80px', resize: 'none', wordBreak: 'break-all' }}
                                />
                                <button 
                                    className="btn btn-secondary"
                                    onClick={() => handleCopy(btoa(JSON.stringify({
                                        r: pairingData.relay_multiaddr,
                                        p: pairingData.peer_id,
                                        c: pairingData.pairing_code
                                    })), 'code')}
                                    style={{ padding: '10px 16px', minWidth: '90px' }}
                                >
                                    {copiedField === 'code' ? '✓ Copied' : 'Copy'}
                                </button>
                            </div>
                            <div style={{ display: 'flex', alignItems: 'center', gap: '8px', fontSize: '0.8rem', color: 'var(--text-muted)' }}>
                                <span>⏱️ Valid for 5 minutes</span>
                            </div>
                        </div>

                        <button 
                            className="btn btn-secondary" 
                            onClick={() => setPairingData(null)}
                        >
                            Done
                        </button>
                    </div>
                )}
            </div>
        </div>
    );
}
