import React, { useState } from 'react';

export default function ContactsTab({ 
    contacts = [], 
    contactRequests = [],
    invitationInfo = null, 
    onAddContact, 
    onDeleteContact, 
    onAcceptContactRequest,
    onRejectContactRequest,
    error, 
    loading 
}) {
    const [contactName, setContactName] = useState('');
    const [contactAddr, setContactAddr] = useState('');
    const [contactFormError, setContactFormError] = useState('');
    const [copySuccess, setCopySuccess] = useState(false);

    const handleContactSubmit = async (e) => {
        e.preventDefault();
        setContactFormError('');
        if (!contactName.trim() || !contactAddr.trim()) {
            setContactFormError('Both Name and Invitation String are required.');
            return;
        }

        const success = await onAddContact({ displayName: contactName, multiaddr: contactAddr });
        if (success) {
            setContactName('');
            setContactAddr('');
        }
    };

    const handleCopyLink = () => {
        const link = invitationInfo?.invite_link || '';
        if (!link) return;
        navigator.clipboard.writeText(link).then(() => {
            setCopySuccess(true);
            setTimeout(() => setCopySuccess(false), 2000);
        });
    };

    const incomingRequests = contactRequests.filter(r => r.direction === 'incoming');
    const outgoingRequests = contactRequests.filter(r => r.direction === 'outgoing');

    return (
        <div>
            <div style={{ borderBottom: '1px solid var(--border-color)', paddingBottom: '16px', marginBottom: '32px' }}>
                <h1 style={{ fontSize: '1.75rem', fontWeight: 600, color: 'var(--text-primary)' }}>Contacts</h1>
                <p style={{ color: 'var(--text-muted)', fontSize: '0.9rem', marginTop: '6px' }}>Manage secure connections and copy your invite code.</p>
            </div>

            {/* Sharing Link Section */}
            <div style={{ marginBottom: '40px' }}>
                <h3 style={{ fontSize: '1.2rem', fontWeight: 600, color: 'var(--text-primary)', marginBottom: '8px' }}>Sharing Link</h3>
                <p style={{ color: 'var(--text-muted)', fontSize: '0.85rem', marginBottom: '16px' }}>Share this link with friends so they can add you as a contact.</p>
                <div style={{ display: 'flex', gap: '8px', maxWidth: '800px' }}>
                    <input 
                        type="text" 
                        readOnly 
                        value={invitationInfo?.invite_link || 'Generating sharing address...'} 
                        style={{ flex: 1, padding: '10px 12px', border: '1px solid var(--border-color)', borderRadius: '6px', background: 'var(--bg-secondary)', color: 'var(--text-primary)', fontFamily: 'monospace', fontSize: '0.85rem' }} 
                    />
                    <button className="btn btn-secondary" onClick={handleCopyLink} disabled={!invitationInfo?.invite_link} style={{ padding: '0 16px', fontSize: '0.9rem', height: '42px' }}>
                        {copySuccess ? 'Copied! ✓' : '📋 Copy'}
                    </button>
                </div>
            </div>

            {/* Add Contact Form Section */}
            <div style={{ marginBottom: '40px' }}>
                <h3 style={{ fontSize: '1.2rem', fontWeight: 600, color: 'var(--text-primary)', marginBottom: '16px' }}>New Contact</h3>
                <form onSubmit={handleContactSubmit}>
                    <div style={{ display: 'flex', gap: '20px', alignItems: 'flex-start', flexWrap: 'wrap' }}>
                        <div className="form-group" style={{ flex: '1 1 250px', marginBottom: '16px' }}>
                            <label style={{ fontWeight: 500, fontSize: '0.85rem', color: 'var(--text-muted)', marginBottom: '6px', display: 'block' }}>Display Name</label>
                            <input 
                                type="text" 
                                placeholder="e.g. Omar Aflak" 
                                value={contactName}
                                onChange={(e) => setContactName(e.target.value)}
                                style={{ width: '100%', padding: '10px 12px', border: '1px solid var(--border-color)', borderRadius: '6px', background: 'var(--bg-secondary)', color: 'var(--text-primary)' }}
                            />
                        </div>
                        <div className="form-group" style={{ flex: '2 1 450px', marginBottom: '16px' }}>
                            <label style={{ fontWeight: 500, fontSize: '0.85rem', color: 'var(--text-muted)', marginBottom: '6px', display: 'block' }}>Invitation String</label>
                            <input 
                                type="text"
                                placeholder="magicbox://invite/QmbQGs..." 
                                value={contactAddr}
                                onChange={(e) => setContactAddr(e.target.value)}
                                style={{ width: '100%', padding: '10px 12px', border: '1px solid var(--border-color)', borderRadius: '6px', background: 'var(--bg-secondary)', color: 'var(--text-primary)', fontFamily: 'monospace', fontSize: '0.85rem' }}
                            />
                        </div>
                        <div style={{ flex: '0 0 auto', alignSelf: 'flex-end', marginBottom: '16px' }}>
                            <button type="submit" className="btn btn-primary" disabled={loading} style={{ padding: '10px 24px', height: '42px', fontSize: '0.9rem', fontWeight: 500 }}>
                                {loading ? 'Sending Request...' : 'Send Request'}
                            </button>
                        </div>
                    </div>

                    {(contactFormError || error) && (
                        <div style={{ color: 'var(--accent-error)', fontSize: '0.85rem', marginTop: '-8px', marginBottom: '16px', fontWeight: 500 }}>
                            {contactFormError || error}
                        </div>
                    )}
                </form>
            </div>

            {/* Pending Requests Section */}
            {(incomingRequests.length > 0 || outgoingRequests.length > 0) && (
                <div style={{ marginBottom: '40px' }}>
                    <h3 style={{ fontSize: '1.2rem', fontWeight: 600, color: 'var(--text-primary)', marginBottom: '16px' }}>Pending Requests</h3>
                    
                    {incomingRequests.length > 0 && (
                        <div style={{ marginBottom: '24px' }}>
                            <h4 style={{ fontSize: '0.95rem', fontWeight: 600, color: 'var(--text-muted)', marginBottom: '12px' }}>Incoming (Action Required)</h4>
                            <div className="card" style={{ padding: '0' }}>
                                {incomingRequests.map((req, idx) => (
                                    <div key={req.id} style={{ 
                                        display: 'flex', 
                                        justifyContent: 'space-between', 
                                        alignItems: 'center', 
                                        padding: '16px 20px', 
                                        borderBottom: idx === incomingRequests.length - 1 ? 'none' : '1px solid var(--border-color)' 
                                    }}>
                                        <div>
                                            <div style={{ fontWeight: 600, color: 'var(--text-primary)' }}>{req.display_name}</div>
                                            <div style={{ fontSize: '0.8rem', color: 'var(--text-muted)', marginTop: '4px', fontFamily: 'monospace', wordBreak: 'break-all' }}>
                                                {req.multiaddr}
                                            </div>
                                        </div>
                                        <div style={{ display: 'flex', gap: '8px', flexShrink: 0 }}>
                                            <button className="btn btn-primary btn-sm" onClick={() => onAcceptContactRequest(req.id)} style={{ padding: '6px 12px', fontSize: '0.85rem' }}>
                                                Accept
                                            </button>
                                            <button className="btn btn-secondary btn-sm" onClick={() => onRejectContactRequest(req.id)} style={{ padding: '6px 12px', fontSize: '0.85rem' }}>
                                                Reject
                                            </button>
                                        </div>
                                    </div>
                                ))}
                            </div>
                        </div>
                    )}

                    {outgoingRequests.length > 0 && (
                        <div>
                            <h4 style={{ fontSize: '0.95rem', fontWeight: 600, color: 'var(--text-muted)', marginBottom: '12px' }}>Sent Requests</h4>
                            <div className="card" style={{ padding: '0' }}>
                                {outgoingRequests.map((req, idx) => (
                                    <div key={req.id} style={{ 
                                        display: 'flex', 
                                        justifyContent: 'space-between', 
                                        alignItems: 'center', 
                                        padding: '16px 20px', 
                                        borderBottom: idx === outgoingRequests.length - 1 ? 'none' : '1px solid var(--border-color)' 
                                    }}>
                                        <div>
                                            <div style={{ fontWeight: 600, color: 'var(--text-primary)' }}>{req.display_name}</div>
                                            <div style={{ fontSize: '0.8rem', color: 'var(--text-muted)', marginTop: '4px', fontFamily: 'monospace', wordBreak: 'break-all' }}>
                                                {req.multiaddr}
                                            </div>
                                        </div>
                                        <button className="btn btn-secondary btn-sm" onClick={() => onRejectContactRequest(req.id)} style={{ padding: '6px 12px', fontSize: '0.85rem', flexShrink: 0 }}>
                                            Cancel
                                        </button>
                                    </div>
                                ))}
                            </div>
                        </div>
                    )}
                </div>
            )}

            {/* Contacts Directory List Section */}
            <div style={{ marginBottom: '24px' }}>
                <h3 style={{ fontSize: '1.2rem', fontWeight: 600, color: 'var(--text-primary)', marginBottom: '16px' }}>Contacts</h3>
                {contacts.length === 0 ? (
                    <div style={{ color: 'var(--text-muted)', fontSize: '0.9rem', fontStyle: 'italic', padding: '24px 0', borderTop: '1px solid var(--border-color)' }}>
                        No contacts added yet. Use the invite link or send a contact request to start sharing!
                    </div>
                ) : (
                    <div className="table-container card">
                        <table style={{ width: '100%' }}>
                            <thead>
                                <tr>
                                    <th style={{ width: '200px' }}>Name</th>
                                    <th>Address / Peer ID</th>
                                    <th className="text-right" style={{ width: '120px' }}>Actions</th>
                                </tr>
                            </thead>
                            <tbody>
                                {contacts.map(c => (
                                    <tr key={c.id}>
                                        <td><strong>{c.display_name}</strong></td>
                                        <td style={{ wordBreak: 'break-all' }}>
                                            <span style={{ fontSize: '0.85rem', fontFamily: 'monospace', color: 'var(--text-primary)' }}>
                                                {c.multiaddr}
                                            </span>
                                        </td>
                                        <td className="text-right">
                                            <button className="btn btn-danger btn-sm" onClick={() => onDeleteContact(c.id)}>
                                                Delete
                                            </button>
                                        </td>
                                    </tr>
                                ))}
                            </tbody>
                        </table>
                    </div>
                )}
            </div>
        </div>
    );
}
