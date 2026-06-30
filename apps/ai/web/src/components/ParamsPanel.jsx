import React, { useState, useEffect } from 'react';
import { updateParams, createPreset } from '../api';
import { DEFAULT_PARAMS } from '../utils';
import ParamsEditor from './ParamsEditor';
import PresetModal from './PresetModal';

export default function ParamsPanel({ activeId, initialParams, onParamsChange }) {
  const [params, setParams] = useState({
    ...DEFAULT_PARAMS,
    ...initialParams
  });
  const [showSavePresetModal, setShowSavePresetModal] = useState(false);

  useEffect(() => {
    setParams({
      ...DEFAULT_PARAMS,
      ...initialParams
    });
  }, [initialParams, activeId]);

  const handleSave = (newParams) => {
    const updated = newParams || params;
    if (activeId && activeId !== 'new') {
      updateParams(activeId, updated);
    }
    if (onParamsChange) {
      onParamsChange(updated);
    }
  };

  if (!activeId) return null;

  return (
    <div className="params-panel glass-panel">
      <div className="panel-header">
        <h3 className="panel-title">Settings</h3>
      </div>
      
      <ParamsEditor 
        params={params} 
        onChange={(newParams) => {
          setParams(newParams);
        }}
        onSave={(updated) => handleSave(updated)}
      />

      <button 
        onClick={() => setShowSavePresetModal(true)}
        className="btn-preset-save"
      >
        Save as Preset
      </button>

      <PresetModal 
        isOpen={showSavePresetModal}
        onSave={async (name, description) => {
          setShowSavePresetModal(false);
          const presetParams = { ...params, description };
          await createPreset(name, JSON.stringify(presetParams));
        }}
        onCancel={() => setShowSavePresetModal(false)}
      />
    </div>
  );
}
