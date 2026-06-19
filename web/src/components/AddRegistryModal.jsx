import React, { useState, useEffect } from 'react';
import Modal from './Modal';

export default function AddRegistryModal({ isOpen, onClose, onAddRegistry, error, loading }) {
    const [prefix, setPrefix] = useState('');

    useEffect(() => {
        if (!isOpen) {
            setPrefix('');
        }
    }, [isOpen]);

    const handleSubmit = (e) => {
        e.preventDefault();
        onAddRegistry(prefix);
    };

    return (
        <Modal isOpen={isOpen} onClose={onClose} title="Add Allowed Registry Prefix">
            <form onSubmit={handleSubmit}>
                <div className="form-group">
                    <label htmlFor="registry-prefix">Prefix Pattern</label>
                    <input 
                        type="text" 
                        id="registry-prefix" 
                        placeholder="e.g. ghcr.io/myorg/" 
                        required
                        value={prefix}
                        onChange={(e) => setPrefix(e.target.value)}
                    />
                    <span className="field-hint">Docker images must match this prefix pattern to be allowed.</span>
                </div>
                {error && <div className="error-box" style={{ marginBottom: '12px', marginTop: '12px' }}>{error}</div>}
                <div className="modal-actions" style={{ display: 'flex', justifyContent: 'flex-end', gap: '12px', marginTop: '20px' }}>
                    <button type="button" className="btn btn-secondary" onClick={onClose} disabled={loading}>
                        Cancel
                    </button>
                    <button type="submit" className="btn btn-primary" disabled={loading}>
                        {loading ? 'Adding...' : 'Add Prefix'}
                    </button>
                </div>
            </form>
        </Modal>
    );
}
