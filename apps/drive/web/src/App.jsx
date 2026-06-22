import React, { useState, useEffect, useRef, useCallback } from 'react';
import { fetchInfo, listFiles, createFolder } from './utils/api';
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
  const [currentPath, setCurrentPath] = useState('');
  const [files, setFiles] = useState([]);
  const [fileCounts, setFileCounts] = useState({});
  const [searchQuery, setSearchQuery] = useState('');
  const [viewMode, setViewMode] = useState('grid');
  const [loading, setLoading] = useState(true);
  const [folderPromptOpen, setFolderPromptOpen] = useState(false);
  const [contextMenu, setContextMenu] = useState(null);
  const uploadRef = useRef(null);

  useEffect(() => {
    fetchInfo()
      .then((info) => {
        setUserID(info.user_id);
        setUsername(info.username || info.user_id);
      })
      .catch((err) => console.error('Failed to fetch info:', err));
  }, []);

  const loadFiles = useCallback(async (volume, path) => {
    setLoading(true);
    try {
      const data = await listFiles(volume, path);
      setFiles(data || []);
      setFileCounts((prev) => ({ ...prev, [volume]: (data || []).length }));
    } catch (err) {
      console.error('Failed to list files:', err);
      setFiles([]);
    } finally {
      setLoading(false);
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
    loadFiles(activeVolume, currentPath);
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
          />

          <DropZone
            volume={activeVolume}
            path={currentPath}
            onUploadComplete={handleRefresh}
            uploadRef={uploadRef}
            onContextMenu={(e) => {
              if (!e.target.closest('.file-card')) {
                e.preventDefault();
                setContextMenu({ x: e.clientX, y: e.clientY });
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
                onDelete={handleRefresh}
              />
            )}
          </DropZone>
        </main>
      </div>

      {folderPromptOpen && (
        <div style={{
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
        }}>
          <div className="card" style={{ 
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
            <button className="menu-item" onClick={() => setFolderPromptOpen(true)}>
              📁 New Folder
            </button>
          </div>
        </>
      )}
    </div>
  );
}
