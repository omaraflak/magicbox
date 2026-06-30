import React, { useState, useEffect } from 'react';
import { getModels } from '../api';
import ModelCard from './ModelCard';

export default function ParamsEditor({ params, onChange, onSave }) {
  const [models, setModels] = useState([]);
  const [loadingModels, setLoadingModels] = useState(false);
  const [expandedModel, setExpandedModel] = useState(null);

  useEffect(() => {
    setLoadingModels(true);
    getModels()
      .then(res => setModels(Array.isArray(res) ? res : []))
      .catch(() => setModels([]))
      .finally(() => setLoadingModels(false));
  }, []);

  const handleChange = (key, value) => {
    onChange({ ...params, [key]: value });
  };

  const selectedModel = models.find(m => m.name === params.model);

  return (
    <div className="params-editor">
      <div className="param-group">
        <label>Model</label>
        <div className="model-selector-wrapper">
          <select
            className="model-select"
            value={params.model || ''}
            onChange={e => {
              handleChange('model', e.target.value);
              setTimeout(onSave, 0);
            }}
          >
            <option value="">Default (gemini-3.1-flash-lite)</option>
            {loadingModels && <option disabled>Loading models...</option>}
            {models.map(m => (
              <option key={m.name} value={m.name}>
                {m.display_name || m.name}
              </option>
            ))}
          </select>
        </div>
        {selectedModel && (
          <div className="model-details">
            <div className="model-details-toggle" onClick={() => setExpandedModel(expandedModel ? null : selectedModel.name)}>
              <span className="model-details-arrow">{expandedModel ? '▾' : '▸'}</span>
              <span className="help-text">Model details</span>
            </div>
            {expandedModel && <ModelCard model={selectedModel} />}
          </div>
        )}
        <span className="help-text">The AI model to use for this conversation.</span>
      </div>


      <div className="param-group">
        <label>System Prompt</label>
        <textarea 
          value={params.system_prompt || ''} 
          onChange={e => handleChange('system_prompt', e.target.value)}
          onBlur={onSave}
          placeholder="You are a helpful assistant..."
          rows={4}
        />
        <span className="help-text">Instructions to set the behavior of the AI.</span>
      </div>

      <div className="param-group">
        <div className="param-header">
          <label>Temperature</label>
          <span className="param-value">{parseFloat(params.temperature || 1.0).toFixed(1)}</span>
        </div>
        <input 
          type="range" min="0" max="2" step="0.1" 
          value={params.temperature || 1.0} 
          onChange={e => handleChange('temperature', parseFloat(e.target.value))}
          onMouseUp={onSave}
          onTouchEnd={onSave}
        />
        <span className="help-text">Higher values make output more random, lower values more deterministic.</span>
      </div>

      <div className="param-group">
        <div className="param-header">
          <label>Top P</label>
          <span className="param-value">{parseFloat(params.top_p || 1.0).toFixed(2)}</span>
        </div>
        <input 
          type="range" min="0" max="1" step="0.05" 
          value={params.top_p || 1.0} 
          onChange={e => handleChange('top_p', parseFloat(e.target.value))}
          onMouseUp={onSave}
          onTouchEnd={onSave}
        />
        <span className="help-text">Controls diversity via nucleus sampling.</span>
      </div>

      <div className="param-group">
        <div className="param-header">
          <label>Top K</label>
          <span className="param-value">{params.top_k || 40}</span>
        </div>
        <input 
          type="range" min="1" max="40" step="1" 
          value={params.top_k || 40} 
          onChange={e => handleChange('top_k', parseInt(e.target.value))}
          onMouseUp={onSave}
          onTouchEnd={onSave}
        />
        <span className="help-text">Limit vocabulary size for tokens generated.</span>
      </div>
    </div>
  );
}
