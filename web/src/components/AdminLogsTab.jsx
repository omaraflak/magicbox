import React from 'react';

export default function AdminLogsTab({ logs, logLevel, onLevelChange, onRefresh }) {
    const formattedLogs = logs.map(line => {
        try {
            const entry = JSON.parse(line);
            const { ts, level, msg, ...rest } = entry;
            const timeStr = new Date(ts).toLocaleTimeString();
            const restStr = Object.keys(rest).length > 0 ? `  |  ${JSON.stringify(rest)}` : '';
            return `[${timeStr}] [${level}] ${msg}${restStr}`;
        } catch (e) {
            return line; // Fallback to raw line
        }
    }).join('\n');

    return (
        <div className="admin-tab-content active">
            <div className="tab-header" style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '20px' }}>
                <h3>System Core Logs</h3>
                <div className="filter-group" style={{ display: 'flex', gap: '8px', alignItems: 'center' }}>
                    <select 
                        value={logLevel} 
                        onChange={(e) => onLevelChange(e.target.value)}
                        className="form-control-sm"
                        style={{ background: 'var(--bg-input)', border: '1px solid var(--border-color)', color: 'var(--text-primary)', padding: '6px 12px', borderRadius: 'var(--radius-sm)' }}
                    >
                        <option value="">All Levels</option>
                        <option value="INFO">Info</option>
                        <option value="WARN">Warn</option>
                        <option value="ERROR">Error</option>
                        <option value="DEBUG">Debug</option>
                    </select>
                    <button className="btn btn-secondary btn-sm" onClick={onRefresh}>Refresh</button>
                </div>
            </div>
            <div className="logs-console card" style={{ padding: '16px', background: 'var(--bg-input)', border: '1px solid var(--border-color)', borderRadius: 'var(--radius-lg)', maxHeight: '500px', overflowY: 'auto' }}>
                <pre style={{ margin: 0, whiteSpace: 'pre-wrap', fontFamily: 'monospace', fontSize: '13px', color: 'var(--text-primary)' }}>
                    {formattedLogs || 'No logs matching current criteria'}
                </pre>
            </div>
        </div>
    );
}
