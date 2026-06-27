import React, { useState, useEffect } from 'react';
import { getSettings, saveSettings, getPresets, deletePreset, updatePreset } from '../api';
import ParamsEditor from './ParamsEditor';
import ConfirmationModal from './ConfirmationModal';

export default function SettingsPage({ activeTab, onTabChange, onClose }) {
  const [apiKey, setApiKey] = useState('');
  const [saving, setSaving] = useState(false);
  const [saved, setSaved] = useState(false);
  const [presets, setPresets] = useState([]);
  const [editingPresetId, setEditingPresetId] = useState(null);
  
  // Deletion modal state
  const [showDeleteModal, setShowDeleteModal] = useState(false);
  const [presetToDelete, setPresetToDelete] = useState(null);

  useEffect(() => {
    getSettings().then(res => {
      if (res.api_key) setApiKey(res.api_key);
    });
    loadPresets();
  }, []);

  const loadPresets = () => {
    getPresets().then(res => setPresets(res || []));
  };

  const handleDeletePresetClick = (id) => {
    setPresetToDelete(id);
    setShowDeleteModal(true);
  };

  const confirmDeletePreset = async () => {
    if (presetToDelete) {
      await deletePreset(presetToDelete);
      setPresetToDelete(null);
      setShowDeleteModal(false);
      loadPresets();
    }
  };

  const handleUpdatePresetField = async (id, field, value, currentParams) => {
    const preset = presets.find(p => p.id === id);
    if (!preset) return;
    
    let updatedName = preset.name;
    let updatedParams = { ...currentParams };
    
    if (field === 'name') {
      updatedName = value;
    } else if (field === 'description') {
      updatedParams.description = value;
    }
    
    // Update local state first
    setPresets(prev => prev.map(p => p.id === id ? { ...p, name: updatedName, params: JSON.stringify(updatedParams) } : p));
    
    await updatePreset(id, updatedName, JSON.stringify(updatedParams));
  };

  const handlePresetParamChange = (presetId, newParams) => {
    setPresets(prev => prev.map(p => p.id === presetId ? { ...p, params: JSON.stringify(newParams) } : p));
  };

  const handleSavePresetParams = async (preset) => {
    await updatePreset(preset.id, preset.name, preset.params);
  };

  const handleSave = async (e) => {
    e.preventDefault();
    setSaving(true);
    await saveSettings(apiKey);
    setSaving(false);
    setSaved(true);
    setTimeout(() => setSaved(false), 2000);
  };

  return (
    <div className="settings-page">
      <div className="settings-header">
        <h2>Settings</h2>
      </div>

      <div className="tab-navigation">
        <button 
          className={`tab-btn ${activeTab === 'apikey' ? 'active' : ''}`}
          onClick={() => onTabChange('apikey')}
        >
          API Key
        </button>
        <button 
          className={`tab-btn ${activeTab === 'presets' ? 'active' : ''}`}
          onClick={() => onTabChange('presets')}
        >
          Presets
        </button>
      </div>

      <div className="settings-content">
        {activeTab === 'apikey' && (
          <form onSubmit={handleSave} className="settings-form glass">
            <div className="form-group">
              <label>Gemini API Key</label>
              <p className="help-text">Your API key is stored securely in your local database. It is never sent to any server other than Google's Gemini API.</p>
              <input 
                type="password" 
                value={apiKey} 
                onChange={e => setApiKey(e.target.value)} 
                placeholder="AIzaSy..." 
                required 
              />
            </div>
            <button type="submit" disabled={saving || !apiKey.trim()} className="btn-save">
              {saving ? 'Saving...' : saved ? 'Saved!' : 'Save Settings'}
            </button>
          </form>
        )}

        {activeTab === 'presets' && (
          <div className="presets-tab">
            <div className="preset-list">
              {presets.map(p => {
                let presetParams = {};
                try { presetParams = JSON.parse(p.params); } catch(e) {}
                const isEditing = editingPresetId === p.id;
                
                return (
                  <div key={p.id} className="preset-item-wrapper glass">
                    <div className="preset-item-header" onClick={() => setEditingPresetId(isEditing ? null : p.id)}>
                      <div>
                        <h4 style={{ margin: 0 }}>{p.name}</h4>
                        <p className="help-text" style={{ margin: '4px 0 0 0' }}>{presetParams.description || 'No description'}</p>
                      </div>
                      <div className="preset-item-actions" onClick={e => e.stopPropagation()}>
                        <button className="btn-secondary" onClick={() => setEditingPresetId(isEditing ? null : p.id)}>
                          {isEditing ? 'Hide Settings' : 'Edit Settings'}
                        </button>
                        <button className="btn-danger" onClick={() => handleDeletePresetClick(p.id)}>
                          Delete
                        </button>
                      </div>
                    </div>
                    
                    {isEditing && (
                      <div className="preset-item-body">
                        <div className="form-group">
                          <label>Preset Name</label>
                          <input 
                            type="text" 
                            value={p.name} 
                            onChange={e => handleUpdatePresetField(p.id, 'name', e.target.value, presetParams)}
                          />
                        </div>
                        <div className="form-group">
                          <label>Description</label>
                          <input 
                            type="text" 
                            value={presetParams.description || ''} 
                            onChange={e => handleUpdatePresetField(p.id, 'description', e.target.value, presetParams)}
                          />
                        </div>
                        <ParamsEditor 
                          params={presetParams}
                          onChange={(newParams) => handlePresetParamChange(p.id, newParams)}
                          onSave={() => handleSavePresetParams(p)}
                        />
                      </div>
                    )}
                  </div>
                );
              })}
              {presets.length === 0 && <p className="text-secondary">No presets saved yet.</p>}
            </div>
          </div>
        )}
      </div>

      <ConfirmationModal 
        isOpen={showDeleteModal}
        title="Delete Preset"
        message="Are you sure you want to delete this preset? This action cannot be undone."
        onConfirm={confirmDeletePreset}
        onCancel={() => {
          setPresetToDelete(null);
          setShowDeleteModal(false);
        }}
      />
    </div>
  );
}
