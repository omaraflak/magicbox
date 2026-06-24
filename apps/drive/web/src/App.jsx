import React, { useState, useEffect, useRef, useCallback } from 'react';
import { 
  fetchInfo, 
  listFiles, 
  createFolder, 
  deleteFile, 
  getDownloadPlan, 
  getFileUrl, 
  moveFile, 
  fetchContacts, 
  sendFile,
  restoreTrashFile,
  emptyTrash,
  fetchTransfers,
  fetchFileTransfers,
  fetchAutoSendConfig,
  fetchRecentTransfers,
  fetchActiveTransfers
} from './utils/api';
import Header from './components/Header';
import Sidebar from './components/Sidebar';
import Toolbar from './components/Toolbar';
import DropZone from './components/DropZone';
import FileGrid from './components/FileGrid';
import EmptyState from './components/EmptyState';
import AutoSendModal from './components/AutoSendModal';
import TransfersDrawer from './components/TransfersDrawer';

export default function App() {
  const [userID, setUserID] = useState('');
  const [username, setUsername] = useState('');
  const [activeVolume, setActiveVolume] = useState(() => {
    const params = new URLSearchParams(window.location.search);
    return params.get('tab') || 'storage';
  });
  const [currentPath, setCurrentPath] = useState(() => {
    const params = new URLSearchParams(window.location.search);
    return params.get('path') || '';
  });
  const [files, setFiles] = useState([]);
  const [fileCounts, setFileCounts] = useState({});
  const [searchQuery, setSearchQuery] = useState('');
  const [viewMode, setViewMode] = useState('grid');
  const [loading, setLoading] = useState(true);
  const [folderPromptOpen, setFolderPromptOpen] = useState(false);
  const [contextMenu, setContextMenu] = useState(null);
  const [deleteTarget, setDeleteTarget] = useState(null);
  const [renameTarget, setRenameTarget] = useState(null);
  const [selectedFileNames, setSelectedFileNames] = useState([]);
  const [sendTarget, setSendTarget] = useState(null);
  const [fileTransfersTarget, setFileTransfersTarget] = useState(null);
  const [fileTransfers, setFileTransfers] = useState([]);
  const [contacts, setContacts] = useState([]);
  const [selectedContactId, setSelectedContactId] = useState('');
  const [sharing, setSharing] = useState(false);
  const [sendError, setSendError] = useState('');
  const [sendSuccess, setSendSuccess] = useState(false);
  const [emptyTrashConfirmOpen, setEmptyTrashConfirmOpen] = useState(false);
  const [autoSendTarget, setAutoSendTarget] = useState(null);
  const [autoSendConfig, setAutoSendConfig] = useState(null);
  const [activeTransfersCount, setActiveTransfersCount] = useState(0);
  const [recentTransfers, setRecentTransfers] = useState([]);
  const [errorModalMsg, setErrorModalMsg] = useState('');
  const uploadRef = useRef(null);

  useEffect(() => {
    fetchInfo()
      .then((info) => {
        setUserID(info.user_id);
        setUsername(info.username || info.user_id);
      })
      .catch((err) => console.error('Failed to fetch info:', err));
  }, []);

  useEffect(() => {
    let timer;
    async function loadTransfers() {
      try {
        const polledList = await fetchActiveTransfers();
        
        setRecentTransfers(prev => {
          const next = [...prev];
          
          // 1. Update existing items in next, or add new ones
          (polledList || []).forEach(p => {
            const idx = next.findIndex(item => item.id === p.id);
            if (idx !== -1) {
              next[idx] = { ...next[idx], ...p };
            } else {
              next.push(p);
            }
          });

          // 2. Identify items that were 'sending' in next but are ABSENT in polledList
          next.forEach((item, idx) => {
            if (item.status === 'sending') {
              const isStillActive = (polledList || []).some(p => p.id === item.id);
              if (!isStillActive) {
                // It completed!
                next[idx] = { ...item, status: 'completed' };
              }
            }
          });

          return next;
        });

        const activeCount = (polledList || []).filter(t => t.status === 'sending').length;
        setActiveTransfersCount(activeCount);
      } catch (err) {
        console.error('Failed to query active transfers list:', err);
      }
    }

    loadTransfers();
    timer = setInterval(loadTransfers, 2000);
    return () => clearInterval(timer);
  }, []);

  // Synchronize currentPath and activeVolume state changes with the URL query params
  useEffect(() => {
    const url = new URL(window.location.href);
    const existingPath = url.searchParams.get('path') || '';
    const existingTab = url.searchParams.get('tab') || 'storage';

    let updated = false;

    if (currentPath !== existingPath) {
      if (currentPath) {
        url.searchParams.set('path', currentPath);
      } else {
        url.searchParams.delete('path');
      }
      updated = true;
    }

    if (activeVolume !== existingTab) {
      if (activeVolume && activeVolume !== 'storage') {
        url.searchParams.set('tab', activeVolume);
      } else {
        url.searchParams.delete('tab');
      }
      updated = true;
    }

    if (updated) {
      window.history.pushState(null, '', url.pathname + url.search);
    }
  }, [currentPath, activeVolume]);

  // Synchronize browser history navigation (back/forward) with state
  useEffect(() => {
    const handlePopState = () => {
      const params = new URLSearchParams(window.location.search);
      const tab = params.get('tab') || 'storage';
      const path = params.get('path') || '';

      setActiveVolume(tab);
      setCurrentPath(path);
      setFiles([]);
    };
    window.addEventListener('popstate', handlePopState);
    return () => window.removeEventListener('popstate', handlePopState);
  }, []);

  // Listen to Escape key to close the new folder prompt dialog
  useEffect(() => {
    if (!folderPromptOpen) return;
    const handleKeyDown = (e) => {
      if (e.key === 'Escape') {
        setFolderPromptOpen(false);
      }
    };
    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [folderPromptOpen]);

  // Listen to Escape key to close the delete confirmation dialog
  useEffect(() => {
    if (!deleteTarget) return;
    const handleKeyDown = (e) => {
      if (e.key === 'Escape') {
        setDeleteTarget(null);
      }
    };
    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [deleteTarget]);

  useEffect(() => {
    if (!renameTarget) return;
    const handleKeyDown = (e) => {
      if (e.key === 'Escape') {
        setRenameTarget(null);
      }
    };
    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [renameTarget]);
  // Fetch contacts when sending dialog is opened
  useEffect(() => {
    if (!sendTarget) return;
    fetchContacts()
      .then((data) => {
        setContacts(data || []);
        if (data && data.length > 0) {
          setSelectedContactId(data[0].id);
        } else {
          setSelectedContactId('');
        }
      })
      .catch((err) => {
        console.error('Failed to fetch contacts:', err);
        setSendError('Failed to load contacts list.');
      });
  }, [sendTarget]);

  // Fetch file transfers history when file transfers dialog is opened
  useEffect(() => {
    if (!fileTransfersTarget) return;
    fetchFileTransfers(fileTransfersTarget.name, currentPath)
      .then(setFileTransfers)
      .catch((err) => console.error('Failed to fetch file sent history:', err));
  }, [fileTransfersTarget, currentPath]);

  // Listen to Escape key to close the file sent history dialog
  useEffect(() => {
    if (!fileTransfersTarget) return;
    const handleKeyDown = (e) => {
      if (e.key === 'Escape') {
        setFileTransfersTarget(null);
      }
    };
    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [fileTransfersTarget]);

  // Listen to Escape key to close the send dialog
  useEffect(() => {
    if (!sendTarget) return;
    const handleKeyDown = (e) => {
      if (e.key === 'Escape') {
        setSendTarget(null);
        setSendError('');
        setSendSuccess(false);
      }
    };
    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [sendTarget]);

  // Listen to Escape key to close the empty trash confirmation dialog
  useEffect(() => {
    if (!emptyTrashConfirmOpen) return;
    const handleKeyDown = (e) => {
      if (e.key === 'Escape') {
        setEmptyTrashConfirmOpen(false);
      }
    };
    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [emptyTrashConfirmOpen]);

  useEffect(() => {
    if (!errorModalMsg) return;
    const handleKeyDown = (e) => {
      if (e.key === 'Escape') {
        setErrorModalMsg('');
      }
    };
    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [errorModalMsg]);

  const handleConfirmSend = async () => {
    if (!selectedContactId) {
      setSendError('Please select a contact to send to.');
      return;
    }
    const targetFile = sendTarget;
    const targetContactId = selectedContactId;
    
    // Close modal immediately!
    setSendTarget(null);
    setSelectedContactId('');
    setSendError('');
    setSharing(false);

    try {
      if (targetFile.isMultiple) {
        for (const item of targetFile.items) {
          await sendFile(activeVolume, currentPath, item.name, targetContactId);
        }
      } else {
        await sendFile(activeVolume, currentPath, targetFile.name, targetContactId);
      }
      // Trigger a quick reload to refresh any active states
      loadFiles(activeVolume, currentPath, false);
    } catch (err) {
      console.error('Failed to start sending:', err);
      setErrorModalMsg('Failed to send file: ' + err.message);
    }
  };
  const loadFiles = useCallback(async (volume, path, showSpinner = true) => {
    if (showSpinner) setLoading(true);
    setFiles([]); // Reset files array immediately to prevent schema crashes from stale data
    try {
      let data;
      if (volume === 'shares') {
        data = await fetchTransfers();
      } else {
        data = await listFiles(volume, path);
      }
      setFiles(data || []);
      setSelectedFileNames([]);
      setFileCounts((prev) => ({ ...prev, [volume]: (data || []).length }));

      if (volume === 'storage') {
        try {
          const config = await fetchAutoSendConfig(path);
          setAutoSendConfig(config.is_auto_send ? config : null);
        } catch (err) {
          setAutoSendConfig(null);
        }
      } else {
        setAutoSendConfig(null);
      }
    } catch (err) {
      console.error('Failed to list files:', err);
      setFiles([]);
      setAutoSendConfig(null);
    } finally {
      if (showSpinner) setLoading(false);
    }
  }, []);

  // Load all volume counts on mount
  useEffect(() => {
    listFiles('storage', '')
      .then((data) => {
        setFileCounts((prev) => ({ ...prev, storage: (data || []).length }));
      })
      .catch(() => {});
    listFiles('trash', '')
      .then((data) => {
        setFileCounts((prev) => ({ ...prev, trash: (data || []).length }));
      })
      .catch(() => {});
  }, []);

  useEffect(() => {
    loadFiles(activeVolume, currentPath);
  }, [activeVolume, currentPath, loadFiles]);

  const handleRefresh = useCallback(() => {
    loadFiles(activeVolume, currentPath, false);
  }, [activeVolume, currentPath, loadFiles]);

  const handleUploadClick = () => {
    uploadRef.current?.trigger();
  };

  const handleFolderClick = (folderName) => {
    setCurrentPath((prev) => (prev ? `${prev}/${folderName}` : folderName));
    setFiles([]);
  };

  const handleBreadcrumbClick = (index) => {
    setFiles([]);
    if (index === -1) {
      setCurrentPath('');
      return;
    }
    const segments = currentPath.split('/');
    const newPath = segments.slice(0, index + 1).join('/');
    setCurrentPath(newPath);
  };

  const handleTriggerDelete = (item) => {
    setContextMenu(null);
    setDeleteTarget(item);
  };

  const handleConfirmDelete = async () => {
    if (!deleteTarget) return;
    try {
      if (deleteTarget.isMultiple) {
        for (const item of deleteTarget.items) {
          await deleteFile(activeVolume, currentPath, item.name);
        }
      } else {
        await deleteFile(activeVolume, currentPath, deleteTarget.name);
      }
      setDeleteTarget(null);
      setSelectedFileNames([]);
      handleRefresh();
    } catch (err) {
      setErrorModalMsg('Failed to delete file/folder: ' + err.message);
    }
  };

  const handleRestoreMultiple = async (items) => {
    try {
      for (const item of items) {
        await restoreTrashFile(item.name);
      }
      setSelectedFileNames([]);
      handleRefresh();
    } catch (err) {
      setErrorModalMsg('Failed to restore items: ' + err.message);
    }
  };

  const handleTriggerDeleteMultiple = (items) => {
    setDeleteTarget({
      isMultiple: true,
      count: items.length,
      items: items
    });
  };

  const handleDownloadMultiple = async (items) => {
    try {
      const names = items.map(i => i.name);
      const plan = await getDownloadPlan(activeVolume, currentPath, names);
      if (!plan || !plan.volumes || plan.volumes.length === 0) {
        setErrorModalMsg('No files to download.');
        return;
      }
      
      for (const vol of plan.volumes) {
        const downloadUrl = getFileUrl(activeVolume, currentPath, names, vol.index);
        const a = document.createElement('a');
        a.href = downloadUrl;
        a.download = vol.name;
        document.body.appendChild(a);
        a.click();
        a.remove();
        await new Promise(r => setTimeout(r, 600));
      }
    } catch (err) {
      setErrorModalMsg('Failed to initiate download: ' + err.message);
    }
  };

  const handleSendMultiple = (items) => {
    setSendTarget({
      isMultiple: true,
      count: items.length,
      items: items
    });
  };

  const handleRestoreTrashFile = async (item) => {
    setContextMenu(null);
    try {
      await restoreTrashFile(item.name);
      handleRefresh();
    } catch (err) {
      setErrorModalMsg('Failed to restore file: ' + err.message);
    }
  };

  const handleEmptyTrashClick = () => {
    setEmptyTrashConfirmOpen(true);
  };

  const handleConfirmEmptyTrash = async () => {
    setEmptyTrashConfirmOpen(false);
    try {
      await emptyTrash();
      handleRefresh();
    } catch (err) {
      setErrorModalMsg('Failed to empty trash: ' + err.message);
    }
  };

  const handleConfirmRename = async (newName) => {
    if (!renameTarget || !newName || newName === renameTarget.name) {
      setRenameTarget(null);
      return;
    }
    try {
      await moveFile(activeVolume, currentPath, renameTarget.name, currentPath, newName);
      setRenameTarget(null);
      handleRefresh();
    } catch (err) {
      setErrorModalMsg('Failed to rename file/folder: ' + err.message);
    }
  };

  const handleDownloadItem = async (item) => {
    setContextMenu(null);
    try {
      const plan = await getDownloadPlan(activeVolume, currentPath, item.name);
      if (!plan || !plan.volumes || plan.volumes.length === 0) {
        setErrorModalMsg('No files to download.');
        return;
      }
      
      for (const vol of plan.volumes) {
        const downloadUrl = getFileUrl(activeVolume, currentPath, item.name, vol.index);
        const a = document.createElement('a');
        a.href = downloadUrl;
        a.download = vol.name;
        document.body.appendChild(a);
        a.click();
        a.remove();
        await new Promise(r => setTimeout(r, 600));
      }
    } catch (err) {
      setErrorModalMsg('Failed to initiate download: ' + err.message);
    }
  };

  const handleMoveFiles = async (sourceFiles, destFolder) => {
    const destPath = currentPath ? `${currentPath}/${destFolder.name}` : destFolder.name;
    try {
      for (const f of sourceFiles) {
        await moveFile(activeVolume, currentPath, f.name, destPath);
      }
      setSelectedFileNames([]);
      handleRefresh();
    } catch (err) {
      setErrorModalMsg('Failed to move files: ' + err.message);
    }
  };

  return (
    <div className="app">
      <Header
        username={username}
        searchQuery={searchQuery}
        onSearchChange={setSearchQuery}
        activeTransfersCount={activeTransfersCount}
      />

      <div className="app-body">
        <Sidebar
          activeVolume={activeVolume}
          onVolumeChange={(vol) => {
            if (activeVolume === vol && currentPath === '') {
              return;
            }
            setActiveVolume(vol);
            setCurrentPath('');
            setFiles([]);
          }}
        />

        <main className="main-content">
          <Toolbar
            currentPath={currentPath}
            onBreadcrumbClick={handleBreadcrumbClick}
            fileCount={files.length}
            onUploadClick={handleUploadClick}
            onCreateFolderClick={() => setFolderPromptOpen(true)}
            viewMode={viewMode}
            onViewModeChange={setViewMode}
            selectedCount={selectedFileNames.length}
            onToggleSelectAll={() => {
              if (selectedFileNames.length === files.length) {
                setSelectedFileNames([]);
              } else {
                setSelectedFileNames(files.map(f => f.name));
              }
            }}
            activeVolume={activeVolume}
            onEmptyTrashClick={handleEmptyTrashClick}
          />

          {autoSendConfig && (
            <div 
              className="auto-send-banner" 
              style={{ 
                background: 'rgba(52, 152, 219, 0.08)', 
                border: '1px solid rgba(52, 152, 219, 0.25)', 
                padding: '12px 16px', 
                borderRadius: '8px', 
                margin: '16px', 
                display: 'flex', 
                alignItems: 'center', 
                justifyContent: 'space-between',
                color: 'var(--text-primary)'
              }}
            >
              <div style={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
                <span style={{ fontSize: '1.4rem' }}>📤</span>
                <div>
                  <strong style={{ display: 'block', fontSize: '0.875rem', marginBottom: '2px' }}>Auto-Send Folder</strong>
                  <span style={{ fontSize: '0.8rem', color: 'var(--text-muted)' }}>
                    Anything dropped or created in this folder is automatically sent to: <strong>{autoSendConfig.targets.map(t => t.contact_name).join(', ')}</strong>.
                    {activeTransfersCount > 0 && <span style={{ marginLeft: '8px', color: 'var(--primary-color)', fontWeight: 500 }}> (🔄 Syncing...)</span>}
                  </span>
                </div>
              </div>
              <button 
                className="btn btn-secondary" 
                style={{ padding: '4px 12px', fontSize: '0.75rem' }}
                onClick={() => setAutoSendTarget({ path: currentPath })}
              >
                ⚙ Settings
              </button>
            </div>
          )}

          <DropZone
            volume={activeVolume}
            path={currentPath}
            onUploadComplete={handleRefresh}
            uploadRef={uploadRef}
            onContextMenu={(e) => {
              if (!e.target.closest('.file-card') && activeVolume !== 'shares') {
                e.preventDefault();
                setContextMenu({ x: e.clientX, y: e.clientY, item: null });
              }
            }}
          >
            {loading ? (
              <div className="loading">
                <div className="loading-spinner" />
                <p>Loading files...</p>
              </div>
            ) : activeVolume === 'shares' ? (
              <div className="transfers-history" style={{ padding: '0 8px' }}>
                {files.length === 0 ? (
                  <div style={{ textAlign: 'center', padding: '48px', color: 'var(--text-muted)' }}>
                    No sent records found.
                  </div>
                ) : (
                  <table style={{ width: '100%', borderCollapse: 'collapse', marginTop: '8px' }}>
                    <thead>
                      <tr style={{ borderBottom: '1px solid var(--border-color)', textAlign: 'left', color: 'var(--text-muted)', fontSize: '0.8rem' }}>
                        <th style={{ padding: '12px' }}>File Name</th>
                        <th style={{ padding: '12px' }}>Original Path</th>
                        <th style={{ padding: '12px' }}>Sent To</th>
                        <th style={{ padding: '12px' }}>Sent At</th>
                      </tr>
                    </thead>
                    <tbody>
                      {files.map((rec) => (
                        <tr key={rec.id} style={{ borderBottom: '1px solid var(--border-color)', fontSize: '0.9rem' }}>
                          <td style={{ padding: '12px', fontWeight: 500 }}>{rec.filename}</td>
                          <td style={{ padding: '12px', color: 'var(--text-muted)' }}>{rec.path || '/'}</td>
                          <td style={{ padding: '12px' }}>{rec.contact_name}</td>
                          <td style={{ padding: '12px', color: 'var(--text-muted)' }}>
                            {new Date(rec.sent_at).toLocaleString()}
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                )}
              </div>
            ) : files.length === 0 ? (
              <EmptyState volumeName={activeVolume === 'trash' ? 'Trash' : (currentPath ? currentPath.split('/').pop() : 'My Storage')} />
            ) : (
              <FileGrid
                files={files}
                volume={activeVolume}
                path={currentPath}
                searchQuery={searchQuery}
                viewMode={viewMode}
                onFolderClick={handleFolderClick}
                selectedFileNames={selectedFileNames}
                onSelectionChange={setSelectedFileNames}
                onMoveFiles={handleMoveFiles}
                onContextMenu={(e, file) => {
                  e.preventDefault();
                  e.stopPropagation();
                  if (selectedFileNames.includes(file.name)) {
                    setContextMenu({ x: e.clientX, y: e.clientY, item: file });
                  } else {
                    setSelectedFileNames([file.name]);
                    setContextMenu({ x: e.clientX, y: e.clientY, item: file });
                  }
                }}
              />
            )}
          </DropZone>
        </main>
      </div>

      {folderPromptOpen && (
        <div 
          onClick={() => setFolderPromptOpen(false)}
          style={{
            position: 'fixed',
            top: 0,
            left: 0,
            right: 0,
            bottom: 0,
            background: 'rgba(0, 0, 0, 0.7)',
            backdropFilter: 'blur(8px)',
            display: 'flex',
            justifyContent: 'center',
            alignItems: 'center',
            zIndex: 9999,
          }}
        >
          <div className="card" onClick={(e) => e.stopPropagation()} style={{ 
            background: 'var(--bg-secondary)', 
            border: '1px solid var(--border-color)', 
            borderRadius: 'var(--radius-lg)', 
            padding: '24px', 
            maxWidth: '360px', 
            width: '90%',
            boxShadow: 'var(--shadow-premium)',
          }}>
            <h3 style={{ marginBottom: '16px', fontSize: '1.1rem', fontWeight: 600 }}>Create New Folder</h3>
            <input 
              type="text" 
              placeholder="Folder name" 
              id="new-folder-name"
              style={{ width: '100%', padding: '10px', marginBottom: '16px', borderRadius: '6px', border: '1px solid var(--border-color)', background: 'var(--bg-primary)', color: 'var(--text-primary)', outline: 'none' }}
              onKeyDown={async (e) => {
                if (e.key === 'Enter') {
                  const val = e.target.value;
                  if (val) {
                    try {
                      await createFolder(activeVolume, currentPath, val);
                      setFolderPromptOpen(false);
                      handleRefresh();
                    } catch (err) {
                      setErrorModalMsg('Failed to create folder: ' + err.message);
                    }
                  }
                }
              }}
              autoFocus
            />
            <div style={{ display: 'flex', justifyContent: 'flex-end', gap: '8px' }}>
              <button className="btn btn-secondary" onClick={() => setFolderPromptOpen(false)}>Cancel</button>
              <button className="btn btn-primary" onClick={async () => {
                const val = document.getElementById('new-folder-name')?.value;
                if (val) {
                  try {
                    await createFolder(activeVolume, currentPath, val);
                    setFolderPromptOpen(false);
                    handleRefresh();
                  } catch (err) {
                    setErrorModalMsg('Failed to create folder: ' + err.message);
                  }
                }
              }}>Create</button>
            </div>
          </div>
        </div>
      )}

      {contextMenu && (
        <>
          <div 
            style={{ position: 'fixed', top: 0, left: 0, right: 0, bottom: 0, zIndex: 98, background: 'transparent' }} 
            onClick={() => setContextMenu(null)}
            onContextMenu={(e) => { e.preventDefault(); setContextMenu(null); }}
          />
          <div 
            className="menu-dropdown" 
            style={{ 
              position: 'fixed', 
              left: `${contextMenu.x}px`, 
              top: `${contextMenu.y}px`, 
              zIndex: 99,
              marginTop: 0,
              boxShadow: '0 4px 12px rgba(0,0,0,0.15)',
              border: '1px solid var(--border-color)',
            }}
            onClick={() => setContextMenu(null)}
          >
            {(() => {
              const selectedItems = files.filter(f => selectedFileNames.includes(f.name));
              const allFiles = selectedItems.every(f => !f.is_dir);
              const allDirs = selectedItems.every(f => f.is_dir);
              const selectedCount = selectedItems.length;
              const isMultiMenu = selectedFileNames.length > 1 && selectedFileNames.includes(contextMenu.item?.name);

              if (!contextMenu.item) {
                return activeVolume === 'storage' && (
                  <button className="menu-item" onClick={() => setFolderPromptOpen(true)}>
                    📁 New Folder
                  </button>
                );
              }

              if (isMultiMenu) {
                return activeVolume === 'trash' ? (
                  <>
                    <button className="menu-item" onClick={() => handleRestoreMultiple(selectedItems)}>
                      🔄 Restore All ({selectedCount})
                    </button>
                    <button className="menu-item menu-item-danger" onClick={() => handleTriggerDeleteMultiple(selectedItems)}>
                      🗑 Delete All Permanently ({selectedCount})
                    </button>
                  </>
                ) : (
                  <>
                    <button className="menu-item" onClick={() => handleDownloadMultiple(selectedItems)}>
                      ⬇ Download All ({selectedCount})
                    </button>
                    {allFiles && (
                      <button className="menu-item" onClick={() => handleSendMultiple(selectedItems)}>
                        📤 Send All ({selectedCount})
                      </button>
                    )}
                    <button className="menu-item menu-item-danger" onClick={() => handleTriggerDeleteMultiple(selectedItems)}>
                      🗑 Delete All ({selectedCount})
                    </button>
                  </>
                );
              }

              // Single item menu options
              return activeVolume === 'trash' ? (
                <>
                  <button className="menu-item" onClick={() => handleRestoreTrashFile(contextMenu.item)}>
                    🔄 Restore
                  </button>
                  <button className="menu-item menu-item-danger" onClick={() => handleTriggerDelete(contextMenu.item)}>
                    🗑 Delete Permanently
                  </button>
                </>
              ) : (
                <>
                  <button className="menu-item" onClick={() => handleDownloadItem(contextMenu.item)}>
                    ⬇ Download
                  </button>
                  {!contextMenu.item.is_dir && (
                    <>
                      <button className="menu-item" onClick={() => setSendTarget(contextMenu.item)}>
                        📤 Send
                      </button>
                      <button className="menu-item" onClick={() => setFileTransfersTarget(contextMenu.item)}>
                        🕒 History
                      </button>
                    </>
                  )}
                  {contextMenu.item.is_dir && (
                    <button 
                      className="menu-item" 
                      onClick={() => {
                        const targetPath = currentPath ? `${currentPath}/${contextMenu.item.name}` : contextMenu.item.name;
                        setAutoSendTarget({ path: targetPath });
                      }}
                    >
                      📤 Auto-Send Settings
                    </button>
                  )}
                  <button className="menu-item" onClick={() => setRenameTarget(contextMenu.item)}>
                    ✏️ Rename
                  </button>
                  <button className="menu-item menu-item-danger" onClick={() => handleTriggerDelete(contextMenu.item)}>
                    🗑 Delete
                  </button>
                </>
              );
            })()}
          </div>
        </>
      )}

      {deleteTarget && (
        <div 
          onClick={() => setDeleteTarget(null)}
          style={{
            position: 'fixed',
            top: 0,
            left: 0,
            right: 0,
            bottom: 0,
            background: 'rgba(0, 0, 0, 0.7)',
            backdropFilter: 'blur(8px)',
            display: 'flex',
            justifyContent: 'center',
            alignItems: 'center',
            zIndex: 9999,
          }}
        >
          <div className="card" onClick={(e) => e.stopPropagation()} style={{ 
            background: 'var(--bg-secondary)', 
            border: '1px solid var(--border-color)', 
            borderRadius: 'var(--radius-lg)', 
            padding: '24px', 
            maxWidth: '380px', 
            width: '90%',
            boxShadow: 'var(--shadow-premium)',
          }}>
            <h3 style={{ marginBottom: '12px', fontSize: '1.1rem', fontWeight: 600 }}>Confirm Deletion</h3>
            <p style={{ fontSize: '0.9rem', color: 'var(--text-primary)', marginBottom: '8px' }}>
              Are you sure you want to delete <strong>{deleteTarget.isMultiple ? `${deleteTarget.count} items` : (deleteTarget.display_name || deleteTarget.name)}</strong>?
            </p>
            <p style={{ fontSize: '0.8rem', color: 'var(--text-muted)', marginBottom: '24px' }}>
              {activeVolume === 'trash' 
                ? (deleteTarget.isMultiple ? 'All selected items will be permanently deleted and cannot be undone.' : 'This action is permanent and cannot be undone.')
                : (deleteTarget.isMultiple ? 'All selected items will be moved to the Trash, where they will be kept for 30 days.' : 'This item will be moved to the Trash, where it will be kept for 30 days.')}
            </p>
            <div style={{ display: 'flex', justifyContent: 'flex-end', gap: '8px' }}>
              <button className="btn btn-secondary" onClick={() => setDeleteTarget(null)}>Cancel</button>
              <button className="btn btn-danger" onClick={handleConfirmDelete}>Delete</button>
            </div>
          </div>
        </div>
      )}

      {emptyTrashConfirmOpen && (
        <div 
          onClick={() => setEmptyTrashConfirmOpen(false)}
          style={{
            position: 'fixed',
            top: 0,
            left: 0,
            right: 0,
            bottom: 0,
            background: 'rgba(0, 0, 0, 0.7)',
            backdropFilter: 'blur(8px)',
            display: 'flex',
            justifyContent: 'center',
            alignItems: 'center',
            zIndex: 9999,
          }}
        >
          <div className="card" onClick={(e) => e.stopPropagation()} style={{ 
            background: 'var(--bg-secondary)', 
            border: '1px solid var(--border-color)', 
            borderRadius: 'var(--radius-lg)', 
            padding: '24px', 
            maxWidth: '380px', 
            width: '90%',
            boxShadow: 'var(--shadow-premium)',
          }}>
            <h3 style={{ marginBottom: '12px', fontSize: '1.1rem', fontWeight: 600 }}>Empty Trash</h3>
            <p style={{ fontSize: '0.9rem', color: 'var(--text-primary)', marginBottom: '8px' }}>
              Are you sure you want to empty the Trash?
            </p>
            <p style={{ fontSize: '0.8rem', color: 'var(--text-muted)', marginBottom: '24px' }}>
              All files in the Trash will be permanently deleted. This action cannot be undone.
            </p>
            <div style={{ display: 'flex', justifyContent: 'flex-end', gap: '8px' }}>
              <button className="btn btn-secondary" onClick={() => setEmptyTrashConfirmOpen(false)}>Cancel</button>
              <button className="btn btn-danger" onClick={handleConfirmEmptyTrash}>Empty Trash</button>
            </div>
          </div>
        </div>
      )}

      {renameTarget && (
        <div 
          onClick={() => setRenameTarget(null)}
          style={{
            position: 'fixed',
            top: 0,
            left: 0,
            right: 0,
            bottom: 0,
            background: 'rgba(0, 0, 0, 0.7)',
            backdropFilter: 'blur(8px)',
            display: 'flex',
            justifyContent: 'center',
            alignItems: 'center',
            zIndex: 9999,
          }}
        >
          <div className="card" onClick={(e) => e.stopPropagation()} style={{ 
            background: 'var(--bg-secondary)', 
            border: '1px solid var(--border-color)', 
            borderRadius: 'var(--radius-lg)', 
            padding: '24px', 
            maxWidth: '380px', 
            width: '90%',
            boxShadow: 'var(--shadow-premium)',
          }}>
            <h3 style={{ marginBottom: '16px', fontSize: '1.1rem', fontWeight: 600 }}>Rename {renameTarget.is_dir ? 'Folder' : 'File'}</h3>
            <input 
              type="text" 
              defaultValue={renameTarget.name}
              id="rename-target-input"
              style={{ width: '100%', padding: '10px', marginBottom: '16px', borderRadius: '6px', border: '1px solid var(--border-color)', background: 'var(--bg-primary)', color: 'var(--text-primary)', outline: 'none' }}
              onKeyDown={(e) => {
                if (e.key === 'Enter') {
                  handleConfirmRename(e.target.value);
                }
              }}
              autoFocus
              onFocus={(e) => {
                const val = e.target.value;
                const dotIdx = val.lastIndexOf('.');
                if (dotIdx > 0 && !renameTarget.is_dir) {
                  e.target.setSelectionRange(0, dotIdx);
                } else {
                  e.target.select();
                }
              }}
            />
            <div style={{ display: 'flex', justifyContent: 'flex-end', gap: '8px' }}>
              <button className="btn btn-secondary" onClick={() => setRenameTarget(null)}>Cancel</button>
              <button className="btn btn-primary" onClick={() => {
                const val = document.getElementById('rename-target-input')?.value;
                handleConfirmRename(val);
              }}>Rename</button>
            </div>
          </div>
        </div>
      )}

      {sendTarget && (
        <div 
          onClick={() => {
            if (!sharing) {
              setSendTarget(null);
              setSendError('');
              setSendSuccess(false);
            }
          }}
          style={{
            position: 'fixed',
            top: 0,
            left: 0,
            right: 0,
            bottom: 0,
            background: 'rgba(0, 0, 0, 0.7)',
            backdropFilter: 'blur(8px)',
            display: 'flex',
            justifyContent: 'center',
            alignItems: 'center',
            zIndex: 9999,
          }}
        >
          <div className="card" onClick={(e) => e.stopPropagation()} style={{ 
            background: 'var(--bg-secondary)', 
            border: '1px solid var(--border-color)', 
            borderRadius: 'var(--radius-lg)', 
            padding: '24px', 
            maxWidth: '420px', 
            width: '90%',
            boxShadow: 'var(--shadow-premium)',
          }}>
            <h3 style={{ marginBottom: '16px', fontSize: '1.1rem', fontWeight: 600 }}>
              {sendTarget.isMultiple ? `Send ${sendTarget.count} Files` : 'Send File'}
            </h3>
            <p style={{ fontSize: '0.9rem', color: 'var(--text-secondary)', marginBottom: '16px' }}>
              Select a contact to send <strong>{sendTarget.isMultiple ? `${sendTarget.count} selected files` : sendTarget.name}</strong> to:
            </p>

            {sendError && (
              <div style={{ color: 'var(--status-danger)', fontSize: '0.85rem', marginBottom: '16px', background: 'rgba(239, 68, 68, 0.1)', padding: '10px', borderRadius: '6px', border: '1px solid rgba(239, 68, 68, 0.2)' }}>
                ⚠️ {sendError}
              </div>
            )}

            {sendSuccess && (
              <div style={{ color: 'var(--status-success)', fontSize: '0.85rem', marginBottom: '16px', background: 'rgba(34, 197, 94, 0.1)', padding: '10px', borderRadius: '6px', border: '1px solid rgba(34, 197, 94, 0.2)' }}>
                ✓ File sent successfully!
              </div>
            )}

            {contacts.length === 0 ? (
              <div style={{ fontSize: '0.9rem', color: 'var(--text-muted)', marginBottom: '24px', textAlign: 'center', padding: '12px 0' }}>
                No contacts found. Please add contacts in settings first.
              </div>
            ) : (
              <div style={{ marginBottom: '24px' }}>
                <label style={{ display: 'block', fontSize: '0.8rem', color: 'var(--text-muted)', marginBottom: '6px', fontWeight: 500 }}>Recipient Contact</label>
                <select
                  value={selectedContactId}
                  onChange={(e) => setSelectedContactId(e.target.value)}
                  disabled={sharing || sendSuccess}
                  style={{
                    width: '100%',
                    padding: '10px',
                    borderRadius: '6px',
                    border: '1px solid var(--border-color)',
                    background: 'var(--bg-primary)',
                    color: 'var(--text-primary)',
                    outline: 'none',
                    fontSize: '0.9rem'
                  }}
                >
                  {contacts.map((c) => (
                    <option key={c.id} value={c.id}>
                      {c.display_name}
                    </option>
                  ))}
                </select>
              </div>
            )}

            <div style={{ display: 'flex', justifyContent: 'flex-end', gap: '8px' }}>
              <button 
                className="btn btn-secondary" 
                onClick={() => {
                  setSendTarget(null);
                  setSendError('');
                  setSendSuccess(false);
                }}
                disabled={sharing}
              >
                Cancel
              </button>
              {contacts.length > 0 && (
                <button 
                  className="btn btn-primary" 
                  onClick={handleConfirmSend}
                  disabled={sharing || sendSuccess}
                >
                  {sharing ? 'Sending...' : 'Send'}
                </button>
              )}
            </div>
          </div>
        </div>
      )}

      {fileTransfersTarget && (
        <div 
          onClick={() => setFileTransfersTarget(null)}
          style={{
            position: 'fixed',
            top: 0,
            left: 0,
            right: 0,
            bottom: 0,
            background: 'rgba(0, 0, 0, 0.7)',
            backdropFilter: 'blur(8px)',
            display: 'flex',
            justifyContent: 'center',
            alignItems: 'center',
            zIndex: 9999,
          }}
        >
          <div className="card" onClick={(e) => e.stopPropagation()} style={{ 
            background: 'var(--bg-secondary)', 
            border: '1px solid var(--border-color)', 
            borderRadius: 'var(--radius-lg)', 
            padding: '24px', 
            maxWidth: '500px', 
            width: '90%',
            boxShadow: 'var(--shadow-premium)',
          }}>
            <h3 style={{ marginBottom: '16px', fontSize: '1.1rem', fontWeight: 600 }}>History: {fileTransfersTarget.name}</h3>
            
            <div style={{ maxHeight: '300px', overflowY: 'auto', marginBottom: '24px' }}>
              {fileTransfers.length === 0 ? (
                <div style={{ textAlign: 'center', padding: '24px 0', color: 'var(--text-muted)', fontSize: '0.9rem' }}>
                  This file hasn't been sent to anyone yet.
                </div>
              ) : (
                <table style={{ width: '100%', borderCollapse: 'collapse' }}>
                  <thead>
                    <tr style={{ borderBottom: '1px solid var(--border-color)', textAlign: 'left', color: 'var(--text-muted)', fontSize: '0.8rem' }}>
                      <th style={{ padding: '8px 12px' }}>Sent To</th>
                      <th style={{ padding: '8px 12px' }}>Date & Time</th>
                    </tr>
                  </thead>
                  <tbody>
                    {fileTransfers.map((s) => (
                      <tr key={s.id} style={{ borderBottom: '1px solid var(--border-color)', fontSize: '0.85rem' }}>
                        <td style={{ padding: '8px 12px', fontWeight: 500 }}>{s.contact_name}</td>
                        <td style={{ padding: '8px 12px', color: 'var(--text-muted)' }}>
                          {new Date(s.sent_at).toLocaleString()}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              )}
            </div>

            <div style={{ display: 'flex', justifyContent: 'flex-end' }}>
              <button className="btn btn-secondary" onClick={() => setFileTransfersTarget(null)}>Close</button>
            </div>
          </div>
        </div>
      )}
      {autoSendTarget && (
        <AutoSendModal
          folderPath={autoSendTarget.path}
          onClose={() => setAutoSendTarget(null)}
          onSaveSuccess={handleRefresh}
        />
      )}
      <TransfersDrawer transfers={recentTransfers} />
      {errorModalMsg && (
        <div 
          className="modal-overlay" 
          onClick={() => setErrorModalMsg('')}
          style={{
            position: 'fixed',
            top: 0,
            left: 0,
            right: 0,
            bottom: 0,
            background: 'rgba(0, 0, 0, 0.7)',
            backdropFilter: 'blur(8px)',
            display: 'flex',
            justifyContent: 'center',
            alignItems: 'center',
            zIndex: 10000,
          }}
        >
          <div 
            className="card"
            onClick={(e) => e.stopPropagation()}
            style={{ 
              background: 'var(--bg-secondary)', 
              border: '1px solid var(--border-color)', 
              borderRadius: 'var(--radius-lg)', 
              padding: '24px', 
              maxWidth: '400px', 
              width: '90%',
              textAlign: 'center',
              boxShadow: 'var(--shadow-premium)',
            }}
          >
            <h3 style={{ fontSize: '1.05rem', fontWeight: 600, marginBottom: '12px', color: 'var(--danger-color)' }}>⚠️ Error</h3>
            <p style={{ fontSize: '0.85rem', color: 'var(--text-secondary)', marginBottom: '24px', lineHeight: 1.4 }}>
              {errorModalMsg}
            </p>
            <button className="btn btn-secondary" onClick={() => setErrorModalMsg('')} style={{ fontSize: '0.85rem', padding: '6px 16px' }}>
              Dismiss
            </button>
          </div>
        </div>
      )}
    </div>
  );
}
