import { useState, useEffect } from 'react';
import { fetchContacts, fetchAutoSendConfig, saveAutoSendConfig, disableAutoSend } from '../utils/api';

export default function AutoSendModal({ folderPath, onClose, onSaveSuccess }) {
  const [contacts, setContacts] = useState([]);
  const [selectedIDs, setSelectedIDs] = useState([]);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [isAlreadyEnabled, setIsAlreadyEnabled] = useState(false);
  const [confirmDisableOpen, setConfirmDisableOpen] = useState(false);
  const [error, setError] = useState('');

  useEffect(() => {
    async function loadData() {
      try {
        const [contactsList, config] = await Promise.all([
          fetchContacts(),
          fetchAutoSendConfig(folderPath)
        ]);
        setContacts(contactsList);
        if (config && config.is_auto_send) {
          setSelectedIDs(config.targets.map(t => t.contact_id));
          setIsAlreadyEnabled(true);
        }
      } catch (err) {
        console.error('Failed to load Auto-Send config:', err);
      } finally {
        setLoading(false);
      }
    }
    loadData();
  }, [folderPath]);

  const handleToggleContact = (id) => {
    setSelectedIDs(prev => 
      prev.includes(id) ? prev.filter(item => item !== id) : [...prev, id]
    );
  };

  const handleSave = async () => {
    if (selectedIDs.length === 0) {
      setError('Please select at least one contact, or click "Disable" to turn off.');
      return;
    }
    setSaving(true);
    setError('');
    try {
      await saveAutoSendConfig(folderPath, selectedIDs);
      onSaveSuccess();
      onClose();
    } catch (err) {
      setError('Failed to save configuration: ' + err.message);
    } finally {
      setSaving(false);
    }
  };

  const handleDisable = () => {
    setConfirmDisableOpen(true);
  };

  const handleConfirmDisable = async () => {
    setSaving(true);
    setError('');
    try {
      await disableAutoSend(folderPath);
      onSaveSuccess();
      onClose();
    } catch (err) {
      setError('Failed to disable: ' + err.message);
    } finally {
      setSaving(false);
      setConfirmDisableOpen(false);
    }
  };

  if (loading) {
    return (
      <div 
        style={{
          position: 'fixed',
          top: 0,
          left: 0,
          right: 0,
          bottom: 0,
          background: 'rgba(0, 0, 0, 0.7)',
          backdropFilter: 'blur(8px)',
          display: 'flex',
          justifyContent: 'center',
          alignItems: 'center',
          zIndex: 9999,
        }}
      >
        <div 
          className="card"
          style={{ 
            background: 'var(--bg-secondary)', 
            border: '1px solid var(--border-color)', 
            borderRadius: 'var(--radius-lg)', 
            padding: '24px', 
            maxWidth: '400px', 
            width: '90%',
            textAlign: 'center',
            boxShadow: 'var(--shadow-premium)',
          }}
        >
          <div className="spinner" style={{ margin: '20px auto' }}></div>
          <p style={{ color: 'var(--text-secondary)', fontSize: '0.9rem' }}>Loading folder settings...</p>
        </div>
      </div>
    );
  }

  if (confirmDisableOpen) {
    return (
      <div 
        style={{
          position: 'fixed',
          top: 0,
          left: 0,
          right: 0,
          bottom: 0,
          background: 'rgba(0, 0, 0, 0.85)',
          backdropFilter: 'blur(10px)',
          display: 'flex',
          justifyContent: 'center',
          alignItems: 'center',
          zIndex: 10000,
        }}
      >
        <div 
          className="card"
          style={{ 
            background: 'var(--bg-secondary)', 
            border: '1px solid var(--border-color)', 
            borderRadius: 'var(--radius-lg)', 
            padding: '24px', 
            maxWidth: '380px', 
            width: '90%',
            textAlign: 'center',
            boxShadow: 'var(--shadow-premium)',
          }}
        >
          <h3 style={{ fontSize: '1.05rem', fontWeight: 600, marginBottom: '12px' }}>Disable Auto-Send?</h3>
          <p style={{ fontSize: '0.85rem', color: 'var(--text-secondary)', marginBottom: '24px', lineHeight: 1.4 }}>
            Are you sure you want to disable Auto-Send on this folder? New files will no longer be forwarded.
          </p>
          <div style={{ display: 'flex', justifyContent: 'center', gap: '8px' }}>
            <button className="btn btn-secondary" onClick={() => setConfirmDisableOpen(false)} style={{ fontSize: '0.85rem', padding: '6px 12px' }}>
              Cancel
            </button>
            <button className="btn btn-danger" onClick={handleConfirmDisable} disabled={saving} style={{ fontSize: '0.85rem', padding: '6px 12px' }}>
              {saving ? 'Disabling...' : 'Yes, Disable'}
            </button>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div 
      onClick={onClose}
      style={{
        position: 'fixed',
        top: 0,
        left: 0,
        right: 0,
        bottom: 0,
        background: 'rgba(0, 0, 0, 0.7)',
        backdropFilter: 'blur(8px)',
        display: 'flex',
        justifyContent: 'center',
        alignItems: 'center',
        zIndex: 9999,
      }}
    >
      <div 
        className="card"
        onClick={(e) => e.stopPropagation()}
        style={{ 
          background: 'var(--bg-secondary)', 
          border: '1px solid var(--border-color)', 
          borderRadius: 'var(--radius-lg)', 
          padding: '24px', 
          maxWidth: '450px', 
          width: '90%',
          boxShadow: 'var(--shadow-premium)',
        }}
      >
        {/* Header */}
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '16px' }}>
          <h3 style={{ fontSize: '1.1rem', fontWeight: 600, margin: 0 }}>📤 Auto-Send Settings</h3>
          <button 
            onClick={onClose} 
            style={{ 
              background: 'none', 
              border: 'none', 
              color: 'var(--text-muted)', 
              cursor: 'pointer', 
              fontSize: '1.25rem',
              padding: '4px'
            }}
          >
            ✕
          </button>
        </div>

        {/* Body */}
        <div style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
          {error && (
            <div style={{ 
              background: 'rgba(231, 76, 60, 0.1)', 
              color: 'var(--danger-color)', 
              padding: '10px 12px', 
              borderRadius: '8px', 
              fontSize: '0.8rem', 
              border: '1px solid rgba(231, 76, 60, 0.2)', 
              lineHeight: 1.4,
              textAlign: 'left'
            }}>
              ⚠️ {error}
            </div>
          )}
          <div style={{ 
            background: 'rgba(255,255,255,0.02)', 
            padding: '12px', 
            borderRadius: '8px', 
            fontSize: '0.8rem', 
            color: 'var(--text-secondary)', 
            border: '1px solid var(--border-color)',
            lineHeight: 1.4
          }}>
            <strong>How Auto-Send works:</strong> Anything placed or uploaded inside <code>/{folderPath || 'root'}</code> will be automatically sent to the contacts below. Deleting files locally will not affect them.
          </div>

          <div>
            <label style={{ display: 'block', fontWeight: 600, fontSize: '0.85rem', marginBottom: '8px', color: 'var(--text-muted)' }}>
              Select Contacts:
            </label>
            {contacts.length === 0 ? (
              <p style={{ color: 'var(--text-muted)', fontSize: '0.85rem' }}>No contacts found. Add contacts in Magicbox OS first.</p>
            ) : (
              <div style={{ maxHeight: '180px', overflowY: 'auto', border: '1px solid var(--border-color)', borderRadius: '8px', padding: '6px' }}>
                {contacts.map(c => {
                  const checked = selectedIDs.includes(c.id);
                  return (
                    <div 
                      key={c.id} 
                      onClick={() => handleToggleContact(c.id)}
                      style={{ 
                        display: 'flex', 
                        alignItems: 'center', 
                        gap: '10px', 
                        padding: '8px 10px', 
                        cursor: 'pointer',
                        borderRadius: '6px',
                        background: checked ? 'rgba(255,255,255,0.03)' : 'transparent',
                        marginBottom: '2px'
                      }}
                    >
                      <input 
                        type="checkbox" 
                        checked={checked}
                        onChange={() => {}} // handled by click
                        style={{ pointerEvents: 'none' }}
                      />
                      <span style={{ fontSize: '0.875rem', color: 'var(--text-primary)' }}>{c.display_name || c.id}</span>
                    </div>
                  );
                })}
              </div>
            )}
          </div>
        </div>

        {/* Footer */}
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginTop: '24px' }}>
          <div>
            {isAlreadyEnabled && (
              <button className="btn btn-danger" onClick={handleDisable} disabled={saving} style={{ fontSize: '0.85rem', padding: '6px 12px' }}>
                Disable
              </button>
            )}
          </div>
          <div style={{ display: 'flex', gap: '8px' }}>
            <button className="btn btn-secondary" onClick={onClose} disabled={saving} style={{ fontSize: '0.85rem', padding: '6px 12px' }}>
              Cancel
            </button>
            <button className="btn btn-primary" onClick={handleSave} disabled={saving} style={{ fontSize: '0.85rem', padding: '6px 12px' }}>
              {saving ? 'Saving...' : 'Save'}
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}
