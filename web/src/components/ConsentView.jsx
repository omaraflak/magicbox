import React, { useEffect, useState } from 'react';

export default function ConsentView({ callAPI }) {
    const [request, setRequest] = useState(null);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState('');
    const [actionLoading, setActionLoading] = useState(false);

    const reqId = new URLSearchParams(window.location.search).get('req_id');

    const loadRequest = async () => {
        if (!reqId) {
            setError('Missing request ID');
            setLoading(false);
            return;
        }

        try {
            const { status, data } = await callAPI('GET', '/apps/permissions/requests');
            if (status === 200) {
                const reqs = data || [];
                const matched = reqs.find(r => r.id === reqId);
                if (matched) {
                    setRequest(matched);
                    setError('');
                } else {
                    setError('Permission request not found or already processed');
                }
            } else {
                setError('Failed to fetch permission requests');
            }
        } catch (e) {
            setError('Connection error');
        } finally {
            setLoading(false);
        }
    };

    useEffect(() => {
        loadRequest();
        const interval = setInterval(loadRequest, 3000);
        return () => clearInterval(interval);
    }, [reqId]);

    const handleApprove = async () => {
        setActionLoading(true);
        try {
            const { status } = await callAPI('POST', `/apps/permissions/requests/${reqId}/approve`);
            if (status === 200) {
                window.opener?.postMessage({ type: 'consent_decision', request_id: reqId, approved: true }, '*');
                window.close();
            } else {
                setError('Failed to approve request');
            }
        } catch (e) {
            setError('Connection error');
        } finally {
            setActionLoading(false);
        }
    };

    const handleReject = async () => {
        setActionLoading(true);
        try {
            const { status } = await callAPI('POST', `/apps/permissions/requests/${reqId}/reject`);
            if (status === 200) {
                window.opener?.postMessage({ type: 'consent_decision', request_id: reqId, approved: false }, '*');
                window.close();
            } else {
                setError('Failed to reject request');
            }
        } catch (e) {
            setError('Connection error');
        } finally {
            setActionLoading(false);
        }
    };

    const getScopeLabel = (scope) => {
        const parts = scope.split(':');
        if (parts[0] === 'profile') return 'Read your profile';
        if (parts[0] === 'contacts') {
            if (parts[1] === 'write') return 'Send and manage contact requests';
            return 'Read your contacts list';
        }
        if (parts[0] === 'shared' && parts.length === 3) {
            const access = parts[2] === 'rw' ? 'Read and write' : 'Read';
            const name = parts[1].charAt(0).toUpperCase() + parts[1].slice(1);
            return `${access} access to your shared "${name}" folder`;
        }
        return scope;
    };

    if (loading) {
        return (
            <div style={containerStyle}>
                <div style={cardStyle}>
                    <div style={spinnerStyle}></div>
                    <p style={{ marginTop: '16px', color: 'var(--text-muted)' }}>Loading permission details...</p>
                </div>
            </div>
        );
    }

    if (error) {
        return (
            <div style={containerStyle}>
                <div style={cardStyle}>
                    <span style={{ fontSize: '3rem' }}>⚠️</span>
                    <h3 style={{ margin: '16px 0 8px 0', color: 'var(--text-primary)' }}>An Error Occurred</h3>
                    <p style={{ color: 'var(--text-muted)', marginBottom: '24px' }}>{error}</p>
                    <button className="btn btn-secondary" onClick={() => window.close()}>Close Window</button>
                </div>
            </div>
        );
    }

    return (
        <div style={containerStyle}>
            <div style={cardStyle}>
                <div style={{ display: 'flex', alignItems: 'center', gap: '16px', marginBottom: '24px', borderBottom: '1px solid var(--border-color)', paddingBottom: '16px' }}>
                    <span style={{ fontSize: '2.5rem' }}>🛡️</span>
                    <div style={{ textAlign: 'left' }}>
                        <h3 style={{ margin: 0, color: 'var(--text-primary)', fontWeight: 600 }}>Permission Request</h3>
                        <p style={{ margin: 0, color: 'var(--text-muted)', fontSize: '0.9rem' }}>from {request.app_name}</p>
                    </div>
                </div>

                <p style={{ color: 'var(--text-primary)', textAlign: 'left', marginBottom: '20px', lineHeight: 1.5 }}>
                    <strong>{request.app_name}</strong> is requesting the following permissions on your Magicbox:
                </p>

                <div style={{ maxHeight: '250px', overflowY: 'auto', marginBottom: '24px', textAlign: 'left' }}>
                    {request.requests.map((req, idx) => (
                        <div key={idx} style={scopeItemStyle}>
                            <div style={{ display: 'flex', alignItems: 'center', gap: '8px', marginBottom: '6px' }}>
                                <span style={{ color: '#38bdf8' }}>●</span>
                                <strong style={{ color: 'var(--text-primary)', fontSize: '0.95rem' }}>{getScopeLabel(req.scope)}</strong>
                            </div>
                            {req.reason && (
                                <p style={{ margin: '0 0 0 16px', color: 'var(--text-muted)', fontSize: '0.85rem', fontStyle: 'italic' }}>
                                    "{req.reason}"
                                </p>
                            )}
                        </div>
                    ))}
                </div>

                <div style={{ display: 'flex', gap: '12px', justifyContent: 'flex-end', borderTop: '1px solid var(--border-color)', paddingTop: '16px' }}>
                    <button 
                        className="btn btn-danger" 
                        disabled={actionLoading}
                        onClick={handleReject}
                        style={{ padding: '10px 20px' }}
                    >
                        Deny
                    </button>
                    <button 
                        className="btn btn-primary" 
                        disabled={actionLoading}
                        onClick={handleApprove}
                        style={{ padding: '10px 24px', display: 'flex', alignItems: 'center', gap: '8px' }}
                    >
                        {actionLoading && <span style={miniSpinnerStyle}></span>}
                        Allow Access
                    </button>
                </div>
            </div>
        </div>
    );
}

// Styling (Premium dark overlay layout)
const containerStyle = {
    display: 'flex',
    justifyContent: 'center',
    alignItems: 'center',
    width: '100vw',
    height: '100vh',
    background: '#0a0a0f',
    fontFamily: "'Outfit', sans-serif",
    padding: '24px',
    boxSizing: 'border-box'
};

const cardStyle = {
    width: '100%',
    maxWidth: '480px',
    background: 'rgba(15, 15, 20, 0.6)',
    backdropFilter: 'blur(20px)',
    border: '1px solid rgba(255, 255, 255, 0.05)',
    borderRadius: '16px',
    padding: '32px',
    boxShadow: '0 12px 40px rgba(0, 0, 0, 0.5)',
    textAlign: 'center',
    boxSizing: 'border-box'
};

const scopeItemStyle = {
    background: 'rgba(255, 255, 255, 0.02)',
    border: '1px solid rgba(255, 255, 255, 0.05)',
    borderRadius: '8px',
    padding: '12px 16px',
    marginBottom: '12px',
};

const spinnerStyle = {
    width: '40px',
    height: '40px',
    border: '3px solid rgba(56, 189, 248, 0.1)',
    borderTop: '3px solid #38bdf8',
    borderRadius: '50%',
    animation: 'spin 1s linear infinite',
    margin: '0 auto'
};

const miniSpinnerStyle = {
    width: '14px',
    height: '14px',
    border: '2px solid rgba(255,255,255,0.2)',
    borderTop: '2px solid #fff',
    borderRadius: '50%',
    animation: 'spin 1s linear infinite',
    display: 'inline-block'
};
