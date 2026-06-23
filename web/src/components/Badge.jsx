import React from 'react';

export default function Badge({ type = 'secondary', children, style, className = '', ...props }) {
    let bg = 'rgba(255, 255, 255, 0.05)';
    let color = 'var(--text-muted)';
    let border = '1px solid var(--border-color)';

    switch (type) {
        case 'success':
        case 'running':
            bg = 'rgba(16, 185, 129, 0.08)';
            color = 'var(--accent-success)';
            border = '1px solid rgba(16, 185, 129, 0.2)';
            break;
        case 'warning':
        case 'stopped':
            bg = 'rgba(245, 158, 11, 0.08)';
            color = 'var(--accent-warning)';
            border = '1px solid rgba(245, 158, 11, 0.2)';
            break;
        case 'danger':
        case 'error':
        case 'uninstalling':
            bg = 'rgba(239, 68, 68, 0.08)';
            color = 'var(--accent-error)';
            border = '1px solid rgba(239, 68, 68, 0.2)';
            break;
        case 'info':
        case 'installing':
        case 'admin':
            bg = 'rgba(6, 182, 212, 0.08)';
            color = 'var(--accent-cyan)';
            border = '1px solid rgba(6, 182, 212, 0.2)';
            break;
        case 'updating':
            bg = 'rgba(139, 92, 246, 0.08)';
            color = '#8b5cf6';
            border = '1px solid rgba(139, 92, 246, 0.2)';
            break;
        case 'starting':
            bg = 'rgba(6, 182, 212, 0.08)';
            color = 'var(--accent-cyan)';
            border = '1px solid rgba(6, 182, 212, 0.2)';
            break;
        case 'stopping':
            bg = 'rgba(245, 158, 11, 0.08)';
            color = 'var(--accent-warning)';
            border = '1px solid rgba(245, 158, 11, 0.2)';
            break;
    }

    const defaultStyle = {
        display: 'inline-flex',
        alignItems: 'center',
        gap: '6px',
        fontSize: '0.78rem',
        fontWeight: 600,
        padding: '4px 10px',
        borderRadius: '50px',
        background: bg,
        color: color,
        border: border,
        textTransform: 'capitalize',
        ...style
    };

    return (
        <span style={defaultStyle} className={`badge-bubble ${className}`} {...props}>
            {children}
        </span>
    );
}
