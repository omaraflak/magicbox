import React, { useState, useEffect } from 'react';
import Modal from './Modal';

export default function CreateUserModal({ isOpen, onClose, onCreateUser, error, loading }) {
    const [username, setUsername] = useState('');
    const [password, setPassword] = useState('');
    const [isAdmin, setIsAdmin] = useState(false);

    useEffect(() => {
        if (!isOpen) {
            setUsername('');
            setPassword('');
            setIsAdmin(false);
        }
    }, [isOpen]);

    const handleSubmit = (e) => {
        e.preventDefault();
        onCreateUser({ username, password, isAdmin });
    };

    return (
        <Modal isOpen={isOpen} onClose={onClose} title="Create New User Account">
            <form onSubmit={handleSubmit}>
                <div className="form-group">
                    <label htmlFor="create-username">Username</label>
                    <input 
                        type="text" 
                        id="create-username" 
                        placeholder="e.g. bob" 
                        required 
                        autoComplete="username"
                        value={username}
                        onChange={(e) => setUsername(e.target.value.toLowerCase())}
                    />
                </div>
                <div className="form-group">
                    <label htmlFor="create-password">Password</label>
                    <input 
                        type="password" 
                        id="create-password" 
                        placeholder="Minimum 8 characters" 
                        required 
                        autoComplete="new-password"
                        value={password}
                        onChange={(e) => setPassword(e.target.value)}
                    />
                </div>
                <div className="form-group check-group" style={{ display: 'flex', alignItems: 'center', gap: '8px', margin: '16px 0' }}>
                    <input 
                        type="checkbox" 
                        id="create-is-admin"
                        checked={isAdmin}
                        onChange={(e) => setIsAdmin(e.target.checked)}
                    />
                    <label htmlFor="create-is-admin" style={{ margin: 0, cursor: 'pointer' }}>
              Grant administrative access
                    </label>
                </div>
                {error && <div className="error-box" style={{ marginBottom: '12px' }}>{error}</div>}
                <div className="modal-actions" style={{ display: 'flex', justifyContent: 'flex-end', gap: '12px', marginTop: '20px' }}>
                    <button type="button" className="btn btn-secondary" onClick={onClose} disabled={loading}>
                        Cancel
                    </button>
                    <button type="submit" className="btn btn-primary" disabled={loading}>
                        {loading ? 'Creating...' : 'Create User'}
                    </button>
                </div>
            </form>
        </Modal>
    );
}
