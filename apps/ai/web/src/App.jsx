import React, { useState, useEffect } from 'react';
import Sidebar from './components/Sidebar';
import ChatArea from './components/ChatArea';
import ParamsPanel from './components/ParamsPanel';
import SettingsPage from './components/SettingsPage';
import { getSettings, getConversations, getConversation, getConversationForks, getConversationTreeContext } from './api';
import { mergeById, parseParams } from './utils';
import './index.css';

let appBase = '/';
if (window.location.pathname.startsWith('/u/')) {
  const segments = window.location.pathname.split('/');
  appBase = segments.slice(0, 4).join('/') + '/';
}

export default function App() {
  const [showSettings, setShowSettings] = useState(false);
  const [settingsTab, setSettingsTab] = useState('apikey');
  const [showParamsPanel, setShowParamsPanel] = useState(false);
  const [conversations, setConversations] = useState([]);
  const [activeId, setActiveId] = useState(null);
  const [activeTitle, setActiveTitle] = useState('');
  const [activeParams, setActiveParams] = useState({});

  // Pagination states
  const [hasMoreRoots, setHasMoreRoots] = useState(true);
  const [rootsOffset, setRootsOffset] = useState(0);

  useEffect(() => {
    getSettings().then(res => {
      if (!res.has_api_key) {
        setShowSettings(true);
        setSettingsTab('apikey');
        window.history.replaceState({}, '', `${appBase}settings/apikey`);
      }
    }).catch(err => console.error('Settings error:', err));

    // Parse URL on load
    const path = window.location.pathname;
    let targetId = null;
    if (path.includes('/settings')) {
      setShowSettings(true);
      setActiveId(null);
      if (path.endsWith('/settings/presets')) {
        setSettingsTab('presets');
      } else {
        setSettingsTab('apikey');
        if (path.endsWith('/settings') || path.endsWith('/settings/')) {
          window.history.replaceState({}, '', `${appBase}settings/apikey`);
        }
      }
    } else {
      const match = path.match(/conversations\/([^/]+)$/);
      if (match) {
        const id = match[1];
        if (id !== 'new') {
          targetId = id;
          setActiveId(id);
        } else {
          setActiveId('new');
        }
        setShowSettings(false);
      }
    }

    loadInitial(targetId);
  }, []);

  useEffect(() => {
    const handlePopState = () => {
      const path = window.location.pathname;
      if (path.includes('/settings')) {
        setShowSettings(true);
        setActiveId(null);
        if (path.endsWith('/settings/presets')) {
          setSettingsTab('presets');
        } else {
          setSettingsTab('apikey');
        }
      } else {
        const match = path.match(/conversations\/([^/]+)$/);
        if (match) {
          setActiveId(match[1]);
          setShowSettings(false);
        } else {
          setActiveId(null);
          setShowSettings(false);
        }
      }
    };

    window.addEventListener('popstate', handlePopState);
    return () => window.removeEventListener('popstate', handlePopState);
  }, []);

  const loadInitial = async (targetId) => {
    try {
      let initialConvs = [];
      if (targetId) {
        const context = await getConversationTreeContext(targetId);
        if (context) initialConvs = [...context];
      }
      const roots = await getConversations(20, 0);
      if (roots) {
        setConversations(mergeById(initialConvs, roots));
        
        setRootsOffset(roots.length);
        if (roots.length < 20) {
          setHasMoreRoots(false);
        }

        if (targetId) {
          const forks = await getConversationForks(targetId);
          if (forks && forks.length > 0) {
            setConversations(prev => mergeById(prev, forks));
          }
        }
      }
    } catch(e) {
      console.error('Initial load error:', e);
    }
  };

  const loadMoreRoots = () => {
    getConversations(20, rootsOffset).then(res => {
      if (res && res.length > 0) {
        setConversations(prev => mergeById(prev, res));
        setRootsOffset(prev => prev + res.length);
        if (res.length < 20) {
          setHasMoreRoots(false);
        }
      } else {
        setHasMoreRoots(false);
      }
    }).catch(err => console.error('Load more roots error:', err));
  };

  const reloadParams = () => {
    if (activeId) {
      getConversation(activeId).then(res => {
        if (res) {
          setActiveParams(parseParams(res.params));
        }
      });
    }
  };

  useEffect(() => {
    if (activeId === 'new') {
      setActiveTitle('New Chat');
      setActiveParams({});
    } else if (activeId) {
      getConversation(activeId).then(res => {
        if (res) {
          setActiveTitle(res.title || 'Untitled Chat');
          setActiveParams(parseParams(res.params));
        } else {
          setActiveParams({});
          setActiveTitle('');
        }
      });
    } else {
      setActiveTitle('');
    }
  }, [activeId]);

  const handleTitleChange = (id, newTitle) => {
    setActiveTitle(newTitle);
    setConversations(prev => prev.map(c => c.id === id ? { ...c, title: newTitle } : c));
  };

  const handleSelectConversation = async (id) => {
    setActiveId(id);
    setShowSettings(false);
    window.history.pushState({}, '', `${appBase}conversations/${id}`);

    try {
      const forks = await getConversationForks(id);
      if (forks && forks.length > 0) {
        setConversations(prev => mergeById(prev, forks));
      }
    } catch(e) {
      console.error('Failed to load forks on click:', e);
    }
  };

  const handleCreateNewChat = () => {
    setActiveId('new');
    setShowSettings(false);
    setShowParamsPanel(false);
    window.history.pushState({}, '', `${appBase}conversations/new`);
  };

  const handleDeleteChat = (id) => {
    setConversations(prev => prev.filter(c => c.id !== id));
    setActiveId(null);
    window.history.pushState({}, '', `${appBase}`);
  };

  return (
    <div className="app-container">
      <div className="bg-glow"></div>

      <Sidebar 
        conversations={conversations} 
        activeId={activeId} 
        onSelect={handleSelectConversation}
        onNew={handleCreateNewChat}
        openSettings={() => {
          setShowSettings(true);
          setSettingsTab('apikey');
          window.history.pushState({}, '', `${appBase}settings/apikey`);
        }}
        hasMoreRoots={hasMoreRoots}
        onLoadMore={loadMoreRoots}
      />

      <main className="main-content">
        {showSettings ? (
          <SettingsPage 
            activeTab={settingsTab} 
            onTabChange={(tab) => {
              setSettingsTab(tab);
              window.history.pushState({}, '', `${appBase}settings/${tab}`);
            }}
            onClose={() => setShowSettings(false)} 
          />
        ) : (
          <>
            <ChatArea 
              activeId={activeId} 
              activeTitle={activeTitle}
              activeParams={activeParams}
              onTitleChange={handleTitleChange}
              onRefreshParams={reloadParams}
              onChatCreated={(newChat) => {
                setConversations(prev => [newChat, ...prev]);
                setActiveId(newChat.id);
                window.history.replaceState({}, '', `${appBase}conversations/${newChat.id}`);
              }}
              onForkCreated={(newFork) => {
                setConversations(prev => [newFork, ...prev]);
                setActiveId(newFork.id);
                window.history.pushState({}, '', `${appBase}conversations/${newFork.id}`);
              }}
              onDeleteChat={handleDeleteChat}
              toggleParams={() => setShowParamsPanel(!showParamsPanel)} 
            />
            {showParamsPanel && activeId && (
              <ParamsPanel 
                activeId={activeId} 
                initialParams={activeParams} 
                onParamsChange={(newParams) => {
                  setActiveParams(newParams);
                }}
              />
            )}
          </>
        )}
      </main>
    </div>
  );
}
