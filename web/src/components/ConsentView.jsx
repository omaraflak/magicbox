import React, { useEffect, useState } from 'react';

// Light theme color palette
const colors = {
    bg: '#ffffff',
    textPrimary: '#0f172a',
    textMuted: '#64748b',
    borderColor: '#e2e8f0',
    scopeBg: '#f8fafc',
    scopeBorder: '#e2e8f0',
    accentColor: '#0284c7',
};

const IconBack = () => (
    <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round" style={{ cursor: 'pointer' }}><line x1="19" y1="12" x2="5" y2="12"></line><polyline points="12 19 5 12 12 5"></polyline></svg>
);

export default function ConsentView({ callAPI }) {
    const [request, setRequest] = useState(null);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState('');
    const [actionLoading, setActionLoading] = useState(false);

    // Pager & Decisions State
    const [currentIndex, setCurrentIndex] = useState(0);
    const [decisions, setDecisions] = useState({});

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

    const handleDecision = async (approved) => {
        const currentScope = request.requests[currentIndex].scope;
        const newDecisions = { ...decisions, [currentScope]: approved };
        setDecisions(newDecisions);

        if (currentIndex < request.requests.length - 1) {
            setCurrentIndex(prev => prev + 1);
        } else {
            // Last scope: submit all decisions!
            setActionLoading(true);
            try {
                const approvedList = Object.keys(newDecisions).filter(s => newDecisions[s] === true);
                
                if (approvedList.length > 0) {
                    const { status } = await callAPI('POST', `/apps/permissions/requests/${reqId}/approve`, {
                        approved_scopes: approvedList
                    });
                    if (status === 200) {
                        window.opener?.postMessage({ type: 'consent_decision', request_id: reqId, approved: true }, '*');
                        window.close();
                    } else {
                        setError('Failed to approve permission requests');
                    }
                } else {
                    // All scopes denied
                    const { status } = await callAPI('POST', `/apps/permissions/requests/${reqId}/reject`);
                    if (status === 200) {
                        window.opener?.postMessage({ type: 'consent_decision', request_id: reqId, approved: false }, '*');
                        window.close();
                    } else {
                        setError('Failed to reject permission requests');
                    }
                }
            } catch (e) {
                setError('Connection error');
            } finally {
                setActionLoading(false);
            }
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
                <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center', height: '100%' }}>
                    <div style={spinnerStyle}></div>
                    <p style={{ marginTop: '16px', color: colors.textMuted }}>Loading permission details...</p>
                </div>
            </div>
        );
    }

    if (error) {
        return (
            <div style={containerStyle}>
                <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center', height: '100%', textAlign: 'center' }}>
                    <span style={{ fontSize: '3rem' }}>⚠️</span>
                    <h3 style={{ margin: '16px 0 8px 0', color: colors.textPrimary }}>An Error Occurred</h3>
                    <p style={{ color: colors.textMuted, marginBottom: '24px' }}>{error}</p>
                    <button style={btnSecondaryStyle} onClick={() => window.close()}>Close Window</button>
                </div>
            </div>
        );
    }

    const currentReq = request.requests[currentIndex];
    const total = request.requests.length;

    return (
        <div style={containerStyle}>
            {/* Top segment progress bar */}
            {total > 1 && (
                <div style={{ display: 'flex', gap: '6px', width: '100%', marginBottom: '16px' }}>
                    {request.requests.map((_, idx) => (
                        <div 
                            key={idx} 
                            style={{ 
                                flex: 1, 
                                height: '4px', 
                                borderRadius: '2px', 
                                background: idx === currentIndex ? colors.accentColor : (idx < currentIndex ? '#94a3b8' : '#e2e8f0'),
                                transition: 'background 0.3s ease'
                            }}
                        />
                    ))}
                </div>
            )}

            <div style={{ display: 'flex', alignItems: 'center', gap: '16px', borderBottom: `1px solid ${colors.borderColor}`, paddingBottom: '16px' }}>
                {currentIndex > 0 && (
                    <button 
                        onClick={() => setCurrentIndex(prev => prev - 1)}
                        style={{ background: 'none', border: 'none', padding: 0, color: colors.textMuted, display: 'flex', alignItems: 'center', justifyContent: 'center' }}
                    >
                        <IconBack />
                    </button>
                )}
                <span style={{ fontSize: '2.5rem' }}>🛡️</span>
                <div style={{ textAlign: 'left' }}>
                    <h3 style={{ margin: 0, color: colors.textPrimary, fontWeight: 600 }}>Permission Request</h3>
                    <p style={{ margin: 0, color: colors.textMuted, fontSize: '0.9rem' }}>from {request.app_name}</p>
                </div>
            </div>

            <div style={{ flex: 1, overflowY: 'auto', padding: '24px 0', textAlign: 'left' }}>
                <p style={{ color: colors.textPrimary, marginBottom: '20px', lineHeight: 1.5 }}>
                    <strong>{request.app_name}</strong> is requesting the following permission ({currentIndex + 1} of {total}):
                </p>

                <div style={scopeItemStyle}>
                    <div style={{ display: 'flex', alignItems: 'center', gap: '8px', marginBottom: '8px' }}>
                        <span style={{ color: colors.accentColor }}>●</span>
                        <strong style={{ color: colors.textPrimary, fontSize: '1rem' }}>{getScopeLabel(currentReq.scope)}</strong>
                    </div>
                    {currentReq.reason && (
                        <p style={{ margin: '0 0 0 16px', color: colors.textMuted, fontSize: '0.9rem', fontStyle: 'italic', lineHeight: 1.4 }}>
                            "{currentReq.reason}"
                        </p>
                    )}
                </div>
            </div>

            <div style={{ display: 'flex', gap: '12px', justifyContent: 'flex-end', borderTop: `1px solid ${colors.borderColor}`, paddingTop: '16px' }}>
                <button 
                    disabled={actionLoading}
                    onClick={() => handleDecision(false)}
                    style={btnDangerStyle}
                >
                    Deny
                </button>
                <button 
                    disabled={actionLoading}
                    onClick={() => handleDecision(true)}
                    style={btnPrimaryStyle}
                >
                    {actionLoading && <span style={miniSpinnerStyle}></span>}
                    {currentIndex === total - 1 ? 'Allow & Approve' : 'Allow'}
                </button>
            </div>
        </div>
    );
}

// Styling (Premium light theme full-screen layout)
const containerStyle = {
    display: 'flex',
    flexDirection: 'column',
    width: '100vw',
    height: '100vh',
    background: colors.bg,
    fontFamily: "'Outfit', sans-serif",
    padding: '24px',
    boxSizing: 'border-box',
    justifyContent: 'space-between'
};

const scopeItemStyle = {
    background: colors.scopeBg,
    border: `1px solid ${colors.scopeBorder}`,
    borderRadius: '8px',
    padding: '16px 20px',
    marginBottom: '12px',
};

const spinnerStyle = {
    width: '40px',
    height: '40px',
    border: '3px solid rgba(2, 132, 199, 0.1)',
    borderTop: `3px solid ${colors.accentColor}`,
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

// Explicit light mode button style overrides to ensure premium look independent of global index.css
const btnBaseStyle = {
    fontFamily: "'Outfit', sans-serif",
    fontSize: '0.95rem',
    fontWeight: 500,
    borderRadius: '6px',
    cursor: 'pointer',
    transition: 'all 0.2s ease',
    border: 'none',
    outline: 'none',
};

const btnPrimaryStyle = {
    ...btnBaseStyle,
    background: colors.accentColor,
    color: '#ffffff',
    padding: '10px 24px',
    display: 'flex',
    alignItems: 'center',
    gap: '8px',
};

const btnDangerStyle = {
    ...btnBaseStyle,
    background: '#ef4444',
    color: '#ffffff',
    padding: '10px 20px',
};

const btnSecondaryStyle = {
    ...btnBaseStyle,
    background: '#e2e8f0',
    color: '#334155',
    padding: '10px 20px',
};
