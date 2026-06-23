import React, { useState, useEffect, useRef, useCallback } from 'react';
import { fetchInfo, listFiles, createFolder, deleteFile, getDownloadPlan, getFileUrl, moveFile, fetchContacts, shareFile } from './utils/api';
import Header from './components/Header';
import Toolbar from './components/Toolbar';
import DropZone from './components/DropZone';
import FileGrid from './components/FileGrid';
import EmptyState from './components/EmptyState';

const VOLUMES = [
  { id: 'storage', name: 'My Storage', icon: '💾' },
];

export default function App() {
  const [userID, setUserID] = useState('');
  const [username, setUsername] = useState('');
  const [activeVolume, setActiveVolume] = useState('storage');
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
  const [shareTarget, setShareTarget] = useState(null);
  const [contacts, setContacts] = useState([]);
  const [selectedContactId, setSelectedContactId] = useState('');
  const [sharing, setSharing] = useState(false);
  const [shareError, setShareError] = useState('');
  const [shareSuccess, setShareSuccess] = useState(false);
  const uploadRef = useRef(null);

  useEffect(() => {
    fetchInfo()
      .then((info) => {
        setUserID(info.user_id);
        setUsername(info.username || info.user_id);
      })
      .catch((err) => console.error('Failed to fetch info:', err));
  }, []);

  // Synchronize currentPath state changes with the URL query params
  useEffect(() => {
    const url = new URL(window.location.href);
    const existingPath = url.searchParams.get('path') || '';
    if (currentPath !== existingPath) {
      if (currentPath) {
        url.searchParams.set('path', currentPath);
      } else {
        url.searchParams.delete('path');
      }
      window.history.pushState(null, '', url.pathname + url.search);
    }
  }, [currentPath]);

  // Synchronize browser history navigation (back/forward) with currentPath state
  useEffect(() => {
    const handlePopState = () => {
      const params = new URLSearchParams(window.location.search);
      setCurrentPath(params.get('path') || '');
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
  // Fetch contacts when sharing dialog is opened
  useEffect(() => {
    if (!shareTarget) return;
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
        setShareError('Failed to load contacts list.');
      });
  }, [shareTarget]);

  // Listen to Escape key to close the share dialog
  useEffect(() => {
    if (!shareTarget) return;
    const handleKeyDown = (e) => {
      if (e.key === 'Escape') {
        setShareTarget(null);
        setShareError('');
        setShareSuccess(false);
      }
    };
    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [shareTarget]);

  const handleConfirmShare = async () => {
    if (!selectedContactId) {
      setShareError('Please select a contact to share with.');
      return;
    }
    setSharing(true);
    setShareError('');
    setShareSuccess(false);
    try {
      await shareFile(activeVolume, currentPath, shareTarget.name, selectedContactId);
      setShareSuccess(true);
      setTimeout(() => {
        setShareTarget(null);
        setShareSuccess(false);
      }, 1500);
    } catch (err) {
      setShareError(err.message || 'Failed to share file.');
    } finally {
      setSharing(false);
    }
  };
  const loadFiles = useCallback(async (volume, path, showSpinner = true) => {
    if (showSpinner) setLoading(true);
    try {
      const data = await listFiles(volume, path);
      setFiles(data || []);
      setSelectedFileNames([]);
      setFileCounts((prev) => ({ ...prev, [volume]: (data || []).length }));
    } catch (err) {
      console.error('Failed to list files:', err);
      setFiles([]);
    } finally {
      if (showSpinner) setLoading(false);
    }
  }, []);

  // Load all volume counts on mount
  useEffect(() => {
    VOLUMES.forEach((vol) => {
      listFiles(vol.id, '')
        .then((data) => {
          setFileCounts((prev) => ({ ...prev, [vol.id]: (data || []).length }));
        })
        .catch(() => {});
    });
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
  };

  const handleBreadcrumbClick = (index) => {
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
      await deleteFile(activeVolume, currentPath, deleteTarget.name);
      setDeleteTarget(null);
      handleRefresh();
    } catch (err) {
      window.alert('Failed to delete file/folder: ' + err.message);
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
      window.alert('Failed to rename file/folder: ' + err.message);
    }
  };

  const handleDownloadItem = async (item) => {
    setContextMenu(null);
    try {
      const plan = await getDownloadPlan(activeVolume, currentPath, item.name);
      if (!plan || !plan.volumes || plan.volumes.length === 0) {
        window.alert('No files to download.');
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
      window.alert('Failed to initiate download: ' + err.message);
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
      window.alert('Failed to move files: ' + err.message);
    }
  };

  return (
    <div className="app">
      <Header
        username={username}
        searchQuery={searchQuery}
        onSearchChange={setSearchQuery}
      />

      <div className="app-body">
        <main className="main-content" style={{ padding: '0 24px' }}>
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
          />

          <DropZone
            volume={activeVolume}
            path={currentPath}
            onUploadComplete={handleRefresh}
            uploadRef={uploadRef}
            onContextMenu={(e) => {
              if (!e.target.closest('.file-card')) {
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
            ) : files.length === 0 ? (
              <EmptyState volumeName={currentPath ? currentPath.split('/').pop() : 'My Storage'} />
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
                  setContextMenu({ x: e.clientX, y: e.clientY, item: file });
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
                      window.alert('Failed to create folder');
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
                    window.alert('Failed to create folder');
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
            {contextMenu.item ? (
              <>
                 <button className="menu-item" onClick={() => handleDownloadItem(contextMenu.item)}>
                  ⬇ Download
                </button>
                {!contextMenu.item.is_dir && (
                  <button className="menu-item" onClick={() => setShareTarget(contextMenu.item)}>
                    📤 Share
                  </button>
                )}
                <button className="menu-item" onClick={() => setRenameTarget(contextMenu.item)}>
                  ✏️ Rename
                </button>
                <button className="menu-item menu-item-danger" onClick={() => handleTriggerDelete(contextMenu.item)}>
                  🗑 Delete
                </button>
              </>
            ) : (
              <button className="menu-item" onClick={() => setFolderPromptOpen(true)}>
                📁 New Folder
              </button>
            )}
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
              Are you sure you want to delete <strong>{deleteTarget.name}</strong>?
            </p>
            <p style={{ fontSize: '0.8rem', color: 'var(--text-muted)', marginBottom: '24px' }}>
              This action is permanent and cannot be undone.
            </p>
            <div style={{ display: 'flex', justifyContent: 'flex-end', gap: '8px' }}>
              <button className="btn btn-secondary" onClick={() => setDeleteTarget(null)}>Cancel</button>
              <button className="btn btn-danger" onClick={handleConfirmDelete}>Delete</button>
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

      {shareTarget && (
        <div 
          onClick={() => {
            if (!sharing) {
              setShareTarget(null);
              setShareError('');
              setShareSuccess(false);
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
            <h3 style={{ marginBottom: '16px', fontSize: '1.1rem', fontWeight: 600 }}>Share File</h3>
            <p style={{ fontSize: '0.9rem', color: 'var(--text-secondary)', marginBottom: '16px' }}>
              Select a contact to share <strong>{shareTarget.name}</strong> with:
            </p>

            {shareError && (
              <div style={{ color: 'var(--status-danger)', fontSize: '0.85rem', marginBottom: '16px', background: 'rgba(239, 68, 68, 0.1)', padding: '10px', borderRadius: '6px', border: '1px solid rgba(239, 68, 68, 0.2)' }}>
                ⚠️ {shareError}
              </div>
            )}

            {shareSuccess && (
              <div style={{ color: 'var(--status-success)', fontSize: '0.85rem', marginBottom: '16px', background: 'rgba(34, 197, 94, 0.1)', padding: '10px', borderRadius: '6px', border: '1px solid rgba(34, 197, 94, 0.2)' }}>
                ✓ File shared successfully!
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
                  disabled={sharing || shareSuccess}
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
                  setShareTarget(null);
                  setShareError('');
                  setShareSuccess(false);
                }}
                disabled={sharing}
              >
                Cancel
              </button>
              {contacts.length > 0 && (
                <button 
                  className="btn btn-primary" 
                  onClick={handleConfirmShare}
                  disabled={sharing || shareSuccess}
                >
                  {sharing ? 'Sharing...' : 'Share'}
                </button>
              )}
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
