import React from 'react';
import { IconSun, IconMoon } from './Icons';

export function SettingsPage({ theme, setTheme }) {
  return (
    <div className="settings-area animate-fade-in">
      <div className="settings-header">
        <span className="settings-title">Settings</span>
      </div>
      <div className="settings-body">
        <div className="settings-section">
          <div className="settings-section-title">Appearance</div>
          <div className="settings-row">
            <div className="settings-row-info">
              <span className="settings-row-title">App Theme</span>
              <span className="settings-row-desc">Customize how Magic Chat looks on your device</span>
            </div>
            <div className="theme-picker">
              <button 
                className={`theme-option ${theme === 'light' ? 'active' : ''}`}
                onClick={() => setTheme('light')}
              >
                <IconSun /> Light
              </button>
              <button 
                className={`theme-option ${theme === 'dark' ? 'active' : ''}`}
                onClick={() => setTheme('dark')}
              >
                <IconMoon /> Dark
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
