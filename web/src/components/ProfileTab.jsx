import React from 'react';
import Badge from './Badge';

export default function ProfileTab({ user }) {
    return (
        <div>
            <div style={{ borderBottom: '1px solid var(--border-color)', paddingBottom: '16px', marginBottom: '32px' }}>
                <h1 style={{ fontSize: '1.75rem', fontWeight: 600, color: 'var(--text-primary)' }}>Profile Details</h1>
                <p style={{ color: 'var(--text-muted)', fontSize: '0.9rem', marginTop: '6px' }}>Overview of your account profile info.</p>
            </div>

            <div style={{ display: 'grid', gridTemplateColumns: '150px 1fr', gap: '20px 24px', alignItems: 'center', maxWidth: '600px' }}>
                <span style={{ color: 'var(--text-muted)', fontSize: '0.95rem', fontWeight: 500 }}>User ID</span>
                <span style={{ fontFamily: 'monospace', color: 'var(--text-primary)', background: 'var(--bg-secondary)', padding: '6px 10px', borderRadius: '4px', border: '1px solid var(--border-color)', fontSize: '0.85rem', justifySelf: 'start' }}>
                    {user?.id}
                </span>

                <span style={{ color: 'var(--text-muted)', fontSize: '0.95rem', fontWeight: 500 }}>Username</span>
                <span style={{ color: 'var(--text-primary)', fontSize: '0.95rem', fontWeight: 600 }}>{user?.username}</span>

                <span style={{ color: 'var(--text-muted)', fontSize: '0.95rem', fontWeight: 500 }}>Account Type</span>
                <span>
                    <Badge type={user?.is_admin ? 'admin' : 'secondary'}>
                        {user?.is_admin ? 'Administrator' : 'Standard User'}
                    </Badge>
                </span>

                <span style={{ color: 'var(--text-muted)', fontSize: '0.95rem', fontWeight: 500 }}>Joined Date</span>
                <span style={{ color: 'var(--text-muted)', fontSize: '0.9rem' }}>
                    {user?.created_at ? new Date(user.created_at).toLocaleDateString(undefined, { dateStyle: 'long' }) : 'N/A'}
                </span>
            </div>
        </div>
    );
}
