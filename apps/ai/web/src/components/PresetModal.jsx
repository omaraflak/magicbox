import React, { useState } from 'react';

export default function PresetModal({ isOpen, onSave, onCancel }) {
  const [name, setName] = useState('');
  const [description, setDescription] = useState('');

  if (!isOpen) return null;

  const handleSubmit = (e) => {
    e.preventDefault();
    if (name.trim()) {
      onSave(name, description);
      setName('');
      setDescription('');
    }
  };

  return (
    <div className="modal-overlay">
      <div className="modal-content">
        <h3>Save as Preset</h3>
        <form onSubmit={handleSubmit} className="settings-form">
          <div className="form-group">
            <label>Preset Name</label>
            <input 
              type="text" 
              value={name} 
              onChange={e => setName(e.target.value)} 
              placeholder="e.g. Creative Writer"
              required 
            />
          </div>
          <div className="form-group">
            <label>Description (optional)</label>
            <input 
              type="text" 
              value={description} 
              onChange={e => setDescription(e.target.value)} 
              placeholder="e.g. High temperature for creative prose"
            />
          </div>
          <div className="modal-actions">
            <button type="button" className="btn-secondary" onClick={onCancel}>
              Cancel
            </button>
            <button type="submit" className="btn-save" disabled={!name.trim()}>
              Save Preset
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
