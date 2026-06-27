import { useState, useEffect } from 'react';

export default function TransfersDrawer({ transfers }) {
  const [minimized, setMinimized] = useState(false);
  const [dismissedIds, setDismissedIds] = useState([]);
  const [completedAt, setCompletedAt] = useState({});
  const [, setTick] = useState(0);

  // Tick every second to trigger state-update for expiring rows
  useEffect(() => {
    const timer = setInterval(() => {
      setTick(t => t + 1);
    }, 1000);
    return () => clearInterval(timer);
  }, []);

  // Record completion timestamps
  useEffect(() => {
    transfers.forEach(t => {
      if (t.status !== 'sending' && !completedAt[t.id]) {
        setCompletedAt(prev => ({ ...prev, [t.id]: Date.now() }));
      }
    });
  }, [transfers, completedAt]);

  const visibleTransfers = transfers.filter(t => {
    if (dismissedIds.includes(t.id)) return false;
    if (t.status !== 'sending') {
      const time = completedAt[t.id];
      if (time && Date.now() - time > 3000) {
        return false;
      }
    }
    return true;
  });

  if (visibleTransfers.length === 0) return null;

  const activeCount = visibleTransfers.filter(t => t.status === 'sending').length;

  const handleDismiss = (id) => {
    setDismissedIds(prev => [...prev, id]);
  };

  return (
    <div 
      className="card"
      style={{
        position: 'fixed',
        bottom: '20px',
        right: '20px',
        width: '340px',
        maxHeight: '400px',
        zIndex: 9999,
        background: 'var(--bg-secondary)',
        border: '1px solid var(--border-color)',
        borderRadius: 'var(--radius-lg)',
        boxShadow: 'var(--shadow-premium)',
        display: 'flex',
        flexDirection: 'column',
        overflow: 'hidden',
      }}
    >
      <style>{`
        @keyframes indeterminate {
          0% { left: -35%; right: 100%; }
          60% { left: 100%; right: -90%; }
          100% { left: 100%; right: -90%; }
        }
        .progress-indeterminate-bar {
          position: absolute;
          height: 100%;
          background: var(--primary-color);
          animation: indeterminate 1.8s infinite ease-in-out;
        }
        .transfers-drawer-dismiss-btn:hover {
          color: var(--text-primary) !important;
          background: rgba(255, 255, 255, 0.08) !important;
        }
        @keyframes spin {
          from { transform: rotate(0deg); }
          to { transform: rotate(360deg); }
        }
        .drawer-ring-spinner {
          display: inline-block;
          width: 12px;
          height: 12px;
          border: 1.5px solid rgba(255, 255, 255, 0.1);
          border-radius: 50%;
          border-top-color: var(--primary-color);
          animation: spin 1s linear infinite;
        }
      `}</style>

      {/* Header */}
      <div 
        onClick={() => setMinimized(!minimized)}
        style={{ 
          background: 'rgba(255,255,255,0.02)', 
          padding: '12px 16px', 
          borderBottom: minimized ? 'none' : '1px solid var(--border-color)', 
          display: 'flex', 
          justifyContent: 'space-between', 
          alignItems: 'center',
          cursor: 'pointer',
          userSelect: 'none'
        }}
      >
        <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
          <span style={{ fontSize: '1.1rem' }}>
            {activeCount > 0 ? '🔄' : '📤'}
          </span>
          <span style={{ fontWeight: 600, fontSize: '0.85rem' }}>
            {activeCount > 0 ? `Sending ${activeCount} file${activeCount === 1 ? '' : 's'}...` : 'Transfers Complete'}
          </span>
        </div>
        <div style={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
          <span style={{ fontSize: '0.8rem', color: 'var(--text-muted)' }}>
            {minimized ? '▲ Expand' : '▼ Collapse'}
          </span>
        </div>
      </div>

      {/* Body */}
      {!minimized && (
        <div style={{ display: 'flex', flexDirection: 'column', height: '100%', overflow: 'hidden' }}>
          <div style={{ flex: 1, overflowY: 'auto', padding: '8px 6px 8px 12px', maxHeight: '280px' }}>
            {visibleTransfers.map(t => {
              const isSending = t.status === 'sending';
              const isFailed = t.status === 'failed';

              return (
                <div 
                  key={t.id} 
                  style={{ 
                    padding: '8px 0', 
                    borderBottom: '1px solid rgba(255,255,255,0.02)', 
                    display: 'flex', 
                    flexDirection: 'column',
                    gap: '4px',
                    paddingRight: '6px'
                  }}
                >
                  <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                    <span 
                      title={t.filename}
                      style={{ 
                        fontSize: '0.8rem', 
                        fontWeight: 500, 
                        overflow: 'hidden', 
                        textOverflow: 'ellipsis', 
                        whiteSpace: 'nowrap',
                        maxWidth: '180px'
                      }}
                    >
                      {t.filename}
                    </span>
                    <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                      {isSending ? (
                        <span className="drawer-ring-spinner"></span>
                      ) : isFailed ? (
                        <span style={{ fontSize: '0.75rem', color: 'var(--danger-color)' }}>⚠️ Failed</span>
                      ) : (
                        <span style={{ fontSize: '0.8rem', color: 'var(--success-color)' }}>✅</span>
                      )}
                      
                      {!isSending && (
                        <button 
                          className="transfers-drawer-dismiss-btn"
                          onClick={() => handleDismiss(t.id)}
                          style={{
                            background: 'none',
                            border: 'none',
                            color: 'var(--text-muted)',
                            cursor: 'pointer',
                            fontSize: '0.75rem',
                            padding: '2px 6px',
                            borderRadius: '4px',
                            transition: 'background 0.2s, color 0.2s'
                          }}
                        >
                          ✕
                        </button>
                      )}
                    </div>
                  </div>

                  {/* Progress Bar */}
                  <div 
                    style={{ 
                      width: '100%', 
                      height: '4px', 
                      background: 'rgba(255,255,255,0.05)', 
                      borderRadius: '2px', 
                      overflow: 'hidden',
                      position: 'relative'
                    }}
                  >
                    {isSending ? (
                      <div className="progress-indeterminate-bar"></div>
                    ) : (
                      <div 
                        style={{ 
                          width: '100%', 
                          height: '100%', 
                          background: isFailed ? 'var(--danger-color)' : 'var(--success-color)'
                        }}
                      ></div>
                    )}
                  </div>
                  
                  <span style={{ fontSize: '0.7rem', color: 'var(--text-muted)' }}>
                    To: {t.contact_name}
                  </span>
                </div>
              );
            })}
          </div>
        </div>
      )}
    </div>
  );
}
