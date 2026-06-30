import React, { useState, useEffect } from 'react';
import { getModels } from '../api';
import ModelCard from './ModelCard';

let cachedModels = null;

export default function ModelPicker({ value, onChange }) {
  const [models, setModels] = useState(cachedModels || []);
  const [loading, setLoading] = useState(!cachedModels);
  const [expanded, setExpanded] = useState(false);
  const [detailModel, setDetailModel] = useState(null);

  useEffect(() => {
    const autoSelect = (list) => {
      if (list.length === 0) return;
      const hasValueInList = list.some(m => m.name === value);
      if (!hasValueInList) {
        const defaultModel = list.find(m => m.name.endsWith('gemini-3.1-flash-lite')) || list[0];
        if (defaultModel) {
          onChange(defaultModel.name);
        }
      }
    };

    if (cachedModels) {
      autoSelect(cachedModels);
      return;
    }

    setLoading(true);
    getModels()
      .then(res => {
        const list = Array.isArray(res) ? res : [];
        cachedModels = list;
        setModels(list);
        autoSelect(list);
      })
      .catch(() => setModels([]))
      .finally(() => setLoading(false));
  }, [value]);

  const selectedModel = models.find(m => m.name === value);

  const handleSelect = (name) => {
    onChange(name);
    setExpanded(false);
    setDetailModel(null);
  };

  const toggleDetail = (e, modelName) => {
    e.stopPropagation();
    setDetailModel(detailModel === modelName ? null : modelName);
  };

  const formatTokens = (n) => {
    if (!n) return '—';
    if (n >= 1000000) return `${(n / 1000000).toFixed(1)}M`;
    if (n >= 1000) return `${Math.round(n / 1000)}K`;
    return n.toLocaleString();
  };

  const slug = (name) => name?.replace(/^models\//, '') || name;

  return (
    <div className="model-picker">
      <button
        type="button"
        className={`model-picker-trigger ${expanded ? 'open' : ''}`}
        onClick={() => setExpanded(!expanded)}
      >
        <div className="model-picker-trigger-content">
          {loading ? (
            <div className="model-picker-loading">
              <span className="spinner" />
              <span>Loading models…</span>
            </div>
          ) : selectedModel ? (
            <>
              <span className="model-picker-selected-name">{slug(selectedModel.name)}</span>
              <span className="model-picker-selected-meta">
                {formatTokens(selectedModel.input_token_limit)} in · {formatTokens(selectedModel.output_token_limit)} out
              </span>
            </>
          ) : (
            <span className="model-picker-placeholder">Select a model…</span>
          )}
        </div>
        <span className="model-picker-chevron">{expanded ? '▴' : '▾'}</span>
      </button>

      {expanded && (
        <div className="model-picker-dropdown">

          {models.map(m => (
            <div key={m.name} className="model-picker-option-wrapper">
              <button
                type="button"
                className={`model-picker-option ${value === m.name ? 'selected' : ''}`}
                onClick={() => handleSelect(m.name)}
              >
                <div className="model-option-header">
                  <span className="model-option-name">{slug(m.name)}</span>
                  <span className="model-option-tokens">
                    {formatTokens(m.input_token_limit)} in · {formatTokens(m.output_token_limit)} out
                  </span>
                </div>
                {m.description && (
                  <p className="model-option-description">{m.description}</p>
                )}
                <div className="model-option-footer">
                  <button
                    type="button"
                    className="model-option-detail-btn"
                    onClick={(e) => toggleDetail(e, m.name)}
                  >
                    {detailModel === m.name ? 'Hide details' : 'Details'}
                  </button>
                </div>
              </button>
              {detailModel === m.name && <ModelCard model={m} />}
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
