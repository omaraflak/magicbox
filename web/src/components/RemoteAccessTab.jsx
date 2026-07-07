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

            <div className="card" style={{ padding: '24px', backgroundColor: 'var(--bg-secondary)', borderRadius: '8px', border: '1px solid var(--border-color)' }}>
                <h3 style={{ fontSize: '1.1rem', fontWeight: 600, marginBottom: '12px' }}>P2P Pairing Setup</h3>
                <p style={{ fontSize: '0.9rem', lineHeight: '1.5', color: 'var(--text-muted)', marginBottom: '20px' }}>
                    Generate a temporary pairing code (One-Time Passcode) to link your mobile browser to this Magicbox. This pairing code will be valid for 5 minutes.
                </p>

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
                        <div style={{ marginBottom: '20px' }}>
                            <label style={{ display: 'block', fontSize: '11px', fontWeight: 600, textTransform: 'uppercase', color: 'var(--text-muted)', marginBottom: '6px' }}>
                                P2P Relay Multiaddress
                            </label>
                            <div style={{ display: 'flex', gap: '8px' }}>
                                <input 
                                    type="text" 
                                    readOnly 
                                    value={pairingData.relay_multiaddr}
                                    style={{ flex: 1, padding: '10px 14px', borderRadius: '6px', border: '1px solid var(--border-color)', backgroundColor: 'var(--bg-primary)', color: 'var(--text-primary)', fontFamily: 'monospace', fontSize: '0.85rem' }}
                                />
                                <button 
                                    className="btn btn-secondary"
                                    onClick={() => handleCopy(pairingData.relay_multiaddr, 'relay')}
                                    style={{ padding: '10px 16px', minWidth: '90px' }}
                                >
                                    {copiedField === 'relay' ? '✓ Copied' : 'Copy'}
                                </button>
                            </div>
                        </div>

                        <div style={{ marginBottom: '20px' }}>
                            <label style={{ display: 'block', fontSize: '11px', fontWeight: 600, textTransform: 'uppercase', color: 'var(--text-muted)', marginBottom: '6px' }}>
                                Home Peer ID (Server ID)
                            </label>
                            <div style={{ display: 'flex', gap: '8px' }}>
                                <input 
                                    type="text" 
                                    readOnly 
                                    value={pairingData.peer_id}
                                    style={{ flex: 1, padding: '10px 14px', borderRadius: '6px', border: '1px solid var(--border-color)', backgroundColor: 'var(--bg-primary)', color: 'var(--text-primary)', fontFamily: 'monospace', fontSize: '0.85rem' }}
                                />
                                <button 
                                    className="btn btn-secondary"
                                    onClick={() => handleCopy(pairingData.peer_id, 'peer')}
                                    style={{ padding: '10px 16px', minWidth: '90px' }}
                                >
                                    {copiedField === 'peer' ? '✓ Copied' : 'Copy'}
                                </button>
                            </div>
                        </div>

                        <div style={{ marginBottom: '24px' }}>
                            <label style={{ display: 'block', fontSize: '11px', fontWeight: 600, textTransform: 'uppercase', color: 'var(--text-muted)', marginBottom: '6px' }}>
                                Pairing Code (OTP)
                            </label>
                            <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
                                <div style={{ fontSize: '2rem', fontWeight: 700, letterSpacing: '4px', color: 'var(--accent-color)', padding: '10px 20px', backgroundColor: 'var(--bg-primary)', borderRadius: '8px', border: '1px solid var(--border-color)', fontFamily: 'monospace' }}>
                                    {pairingData.pairing_code}
                                </div>
                                <button 
                                    className="btn btn-secondary"
                                    onClick={() => handleCopy(pairingData.pairing_code, 'otp')}
                                    style={{ padding: '10px 16px', minWidth: '110px' }}
                                >
                                    {copiedField === 'otp' ? '✓ Copied' : 'Copy Code'}
                                </button>
                                <span style={{ fontSize: '0.8rem', color: 'var(--text-muted)' }}>
                                    ⏱️ Valid for 5 minutes
                                </span>
                            </div>
                        </div>

                        <button 
                            className="btn btn-secondary" 
                            onClick={() => setPairingData(null)}
                        >
                            Reset / Done
                        </button>
                    </div>
                )}
            </div>
        </div>
    );
}
