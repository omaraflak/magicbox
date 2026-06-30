import React from 'react';

export default function ModelCard({ model }) {
  if (!model) return null;

  const slug = model.name?.replace(/^models\//, '') || model.name;

  return (
    <div className="model-info-card">
      <div className="model-info-row">
        <span className="model-info-label">Model</span>
        <span className="model-info-value">{slug}</span>
      </div>

      {model.description && (
        <div className="model-info-row">
          <span className="model-info-label">Description</span>
          <span className="model-info-value model-info-description">{model.description}</span>
        </div>
      )}

      <div className="model-info-row">
        <span className="model-info-label">Input Tokens</span>
        <span className="model-info-value">{model.input_token_limit?.toLocaleString() || '—'}</span>
      </div>
      <div className="model-info-row">
        <span className="model-info-label">Output Tokens</span>
        <span className="model-info-value">{model.output_token_limit?.toLocaleString() || '—'}</span>
      </div>
      {model.temperature > 0 && (
        <div className="model-info-row">
          <span className="model-info-label">Default Temperature</span>
          <span className="model-info-value">{model.temperature}</span>
        </div>
      )}
      {model.top_p > 0 && (
        <div className="model-info-row">
          <span className="model-info-label">Default Top P</span>
          <span className="model-info-value">{model.top_p}</span>
        </div>
      )}
      {model.top_k > 0 && (
        <div className="model-info-row">
          <span className="model-info-label">Default Top K</span>
          <span className="model-info-value">{model.top_k}</span>
        </div>
      )}
    </div>
  );
}
