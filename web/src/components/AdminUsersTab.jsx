import React from 'react';

export default function AdminUsersTab({ users, currentUser, onDeleteUser, onOpenCreateModal }) {
    return (
        <div className="admin-tab-content active">
            <div className="tab-header" style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '20px' }}>
                <h3>Manage User Accounts</h3>
                <button className="btn btn-primary" onClick={onOpenCreateModal}>Create User</button>
            </div>
            <div className="table-container card">
                <table>
                    <thead>
                        <tr>
                            <th>Username</th>
                            <th>Role</th>
                            <th>Created At</th>
                            <th className="text-right">Actions</th>
                        </tr>
                    </thead>
                    <tbody>
                        {users.map(u => (
                            <tr key={u.id}>
                                <td><strong>{u.username}</strong></td>
                                <td>
                                    {u.is_admin ? (
                                        <span className="status-indicator status-running">Admin</span>
                                    ) : (
                                        <span className="status-indicator status-stopped">User</span>
                                    )}
                                </td>
                                <td>{new Date(u.created_at).toLocaleString()}</td>
                                <td className="text-right">
                                    <button 
                                        className="btn btn-danger btn-sm" 
                                        onClick={() => onDeleteUser(u.id)}
                                        disabled={currentUser?.id === u.id}
                                    >
                                        Delete
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
