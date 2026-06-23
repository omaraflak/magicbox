import React, { useState } from 'react';

export default function ContactsTab({ 
    contacts = [], 
    invitationInfo = null, 
    onAddContact, 
    onDeleteContact, 
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
                                {loading ? 'Saving...' : 'Add Contact'}
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

            {/* Contacts Directory List Section */}
            <div style={{ marginBottom: '24px' }}>
                <h3 style={{ fontSize: '1.2rem', fontWeight: 600, color: 'var(--text-primary)', marginBottom: '16px' }}>Contacts</h3>
                {contacts.length === 0 ? (
                    <div style={{ color: 'var(--text-muted)', fontSize: '0.9rem', fontStyle: 'italic', padding: '24px 0', borderTop: '1px solid var(--border-color)' }}>
                        No contacts added yet. Add a contact above to start sharing files and communicating!
                    </div>
                ) : (
                    <div className="table-container card">
                        <table style={{ width: '100%' }}>
                            <thead>
                                <tr>
                                    <th style={{ width: '200px' }}>Name</th>
                                    <th>Invitation Link / Peer ID</th>
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
