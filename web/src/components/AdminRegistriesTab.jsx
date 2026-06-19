import React from 'react';

export default function AdminRegistriesTab({ registries, onDeleteRegistry, onOpenAddModal }) {
    return (
        <div className="admin-tab-content active">
            <div className="tab-header" style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '20px' }}>
                <h3>Allowed Registry Prefixes</h3>
                <button className="btn btn-primary" onClick={onOpenAddModal}>Add Registry</button>
            </div>
            <div className="table-container card">
                <table>
                    <thead>
                        <tr>
                            <th>Allowed Prefix Pattern</th>
                            <th>Created At</th>
                            <th className="text-right">Actions</th>
                        </tr>
                    </thead>
                    <tbody>
                        {registries.map(r => (
                            <tr key={r.id}>
                                <td><strong>{r.prefix}</strong></td>
                                <td>{new Date(r.created_at).toLocaleString()}</td>
                                <td className="text-right">
                                    <button 
                                        className="btn btn-danger btn-sm" 
                                        onClick={() => onDeleteRegistry(r.id)}
                                    >
                                        Remove
                                    </button>
                                </td>
                            </tr>
                        ))}
                    </tbody>
                </table>
            </div>
        </div>
    );
}
