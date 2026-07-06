import React, { useState, useEffect, useRef } from 'react';
import { IconChat } from './components/Icons';
import { Sidebar } from './components/Sidebar';
import { ChatArea } from './components/ChatArea';
import { SettingsPage } from './components/SettingsPage';
import { NewChatModal, RenameChatModal, DeleteChatModal } from './components/Modals';
import { MediaPanel, ParticipantsPanel } from './components/Panels';

function App() {
  const [profile, setProfile] = useState(null);
  const [contacts, setContacts] = useState([]);
  const [conversations, setConversations] = useState([]);
  const [selectedConv, setSelectedConv] = useState(null);
  const [messages, setMessages] = useState([]);
  
  // Settings & Theme State
  const [showSettings, setShowSettings] = useState(false);
  const [theme, setTheme] = useState(() => {
    const saved = localStorage.getItem('theme');
    return saved || 'dark';
  });

  useEffect(() => {
    document.documentElement.setAttribute('data-theme', theme);
    localStorage.setItem('theme', theme);
  }, [theme]);

  // Search & Modal State
  const [searchQuery, setSearchQuery] = useState('');
  const [showModal, setShowModal] = useState(false);
  const [newConvName, setNewConvName] = useState('');
  const [selectedContactIDs, setSelectedContactIDs] = useState([]);
  const [newChatError, setNewChatError] = useState('');

  useEffect(() => {
    if (!showModal) {
      setNewChatError('');
    }
  }, [showModal]);

  // Menu & Modal States
  const [showMenu, setShowMenu] = useState(false);
  const [showRenameModal, setShowRenameModal] = useState(false);
  const [showDeleteModal, setShowDeleteModal] = useState(false);
  const [renameInput, setRenameInput] = useState('');

  const [showParticipantsPanel, setShowParticipantsPanel] = useState(false);

  // Shared Media & Search States
  const [showMediaPanel, setShowMediaPanel] = useState(false);
  const [mediaPanelWidth, setMediaPanelWidth] = useState(320);
  const [isResizingMedia, setIsResizingMedia] = useState(false);
  const [sharedMedia, setSharedMedia] = useState([]);
  const [hasMoreMedia, setHasMoreMedia] = useState(true);
  const [isLoadingMoreMedia, setIsLoadingMoreMedia] = useState(false);
  const [isSearchingMessages, setIsSearchingMessages] = useState(false);
  const [messageSearchQuery, setMessageSearchQuery] = useState('');

  // Pagination States
  const [hasMoreMessages, setHasMoreMessages] = useState(true);
  const [isLoadingMore, setIsLoadingMore] = useState(false);
  const containerRef = useRef(null);

  // Message Sending State
  const [messageText, setMessageText] = useState('');
  const [attachment, setAttachment] = useState(null);

  // Missing Contacts States
  const [addingContacts, setAddingContacts] = useState({});
  const [sentRequests, setSentRequests] = useState({});

  const startResizingMedia = (e) => {
    e.preventDefault();
    setIsResizingMedia(true);
  };

  const stopResizingMedia = () => {
    setIsResizingMedia(false);
  };

  const resizeMediaPanel = (e) => {
    if (!isResizingMedia) return;
    const newWidth = window.innerWidth - e.clientX;
    if (newWidth > 260 && newWidth < 600) {
      setMediaPanelWidth(newWidth);
    }
  };

  useEffect(() => {
    if (isResizingMedia) {
      window.addEventListener('mousemove', resizeMediaPanel);
      window.addEventListener('mouseup', stopResizingMedia);
    }
    return () => {
      window.removeEventListener('mousemove', resizeMediaPanel);
      window.removeEventListener('mouseup', stopResizingMedia);
    };
  }, [isResizingMedia]);

  const apiFetch = async (url, options = {}) => {
    const res = await window.fetch(url, options);
    if (res.status === 403) {
      const clone = res.clone();
      try {
        const err = await clone.json();
        if (err.error === 'consent_required' && err.request_id) {
          return new Promise((resolve, reject) => {
            const popup = window.open(`/consent?req_id=${err.request_id}`, 'ConsentRequired', 'width=500,height=600');
            if (!popup) {
              alert('Consent popup was blocked. Please allow popups for this site.');
              reject(new Error('Consent popup blocked'));
              return;
            }

            const listener = async (event) => {
              if (event.data?.type === 'consent_decision' && event.data?.request_id === err.request_id) {
                window.removeEventListener('message', listener);
                if (event.data?.approved) {
                  try {
                    const retryRes = await apiFetch(url, options);
                    resolve(retryRes);
                  } catch (retryErr) {
                    reject(retryErr);
                  }
                } else {
                  reject(new Error('Permission denied by user'));
                }
              }
            };
            window.addEventListener('message', listener);
          });
        }
      } catch (e) {
        // ignore JSON parse error
      }
    }
    return res;
  };

  const selectedConvRef = useRef(selectedConv);
  useEffect(() => {
    selectedConvRef.current = selectedConv;
  }, [selectedConv]);

  // Sync state to URL
  useEffect(() => {
    const params = new URLSearchParams(window.location.search);
    const convIdFromUrl = params.get('conv');
    const settingsFromUrl = params.get('settings') === 'true';

    if (showSettings) {
      if (!settingsFromUrl) {
        window.history.pushState(null, '', '?settings=true');
      }
    } else if (selectedConv) {
      if (convIdFromUrl !== selectedConv.id) {
        window.history.pushState(null, '', `?conv=${selectedConv.id}`);
      }
    } else {
      if (window.location.search) {
        window.history.pushState(null, '', window.location.pathname);
      }
    }
  }, [selectedConv?.id, showSettings]);

  // Handle browser back/forward buttons
  useEffect(() => {
    const handlePopState = () => {
      const params = new URLSearchParams(window.location.search);
      const convIdFromUrl = params.get('conv');
      const settingsFromUrl = params.get('settings') === 'true';

      if (settingsFromUrl) {
        setShowSettings(true);
        setSelectedConv(null);
      } else if (convIdFromUrl) {
        setShowSettings(false);
        const found = conversations.find(c => c.id === convIdFromUrl);
        if (found) {
          setSelectedConv(found);
        }
      } else {
        setShowSettings(false);
        setSelectedConv(null);
      }
    };
    window.addEventListener('popstate', handlePopState);
    return () => window.removeEventListener('popstate', handlePopState);
  }, [conversations]);

  useEffect(() => {
    const init = async () => {
      try {
        await fetchProfile();
        await fetchContacts();
        await fetchConversations();
      } catch (err) {
        console.error('Failed initialization', err);
      }
    };
    init();

    // Subscribe to SSE updates
    const eventSource = new EventSource('api/events');
    eventSource.onmessage = (event) => {
      if (event.data === 'update') {
        fetchConversations();
        const active = selectedConvRef.current;
        if (active) {
          fetchMessages(active.id, '', false);
        }
      }
    };

    return () => {
      eventSource.close();
    };
  }, []);

  useEffect(() => {
    if (selectedConv) {
      fetchMessages(selectedConv.id, '', false);
      setShowMenu(false);
      setShowMediaPanel(false);
      setShowParticipantsPanel(false);
    } else {
      setMessages([]);
    }
  }, [selectedConv?.id]);

  const handleAddContact = async (participant) => {
    setAddingContacts(prev => ({ ...prev, [participant.user_id]: true }));
    try {
      const res = await apiFetch('api/contacts/add', {
        method: 'POST',
        body: JSON.stringify({
          invite_link: participant.invite_link,
          display_name: participant.display_name,
        }),
      });
      if (res.ok) {
        setSentRequests(prev => ({ ...prev, [participant.user_id]: true }));
        fetchContacts();
      } else {
        const data = await res.json();
        alert(data.error || 'Failed to send contact request');
      }
    } catch (err) {
      console.error(err);
      alert('Error sending contact request');
    } finally {
      setAddingContacts(prev => ({ ...prev, [participant.user_id]: false }));
    }
  };

  const fetchProfile = async () => {
    const res = await apiFetch('api/profile');
    if (res.ok) {
      const data = await res.json();
      setProfile(data);
    }
  };

  const fetchContacts = async () => {
    const res = await apiFetch('api/contacts');
    if (res.ok) {
      const data = await res.json();
      setContacts(data || []);
    }
  };

  const fetchConversations = async () => {
    const res = await apiFetch('api/conversations');
    if (res.ok) {
      const data = await res.json();
      setConversations(data || []);
      const active = selectedConvRef.current;
      if (active) {
        const updated = (data || []).find(c => c.id === active.id);
        if (updated) {
          setSelectedConv(updated);
        }
      } else {
        const params = new URLSearchParams(window.location.search);
        const convIdFromUrl = params.get('conv');
        const settingsFromUrl = params.get('settings') === 'true';

        if (settingsFromUrl) {
          setShowSettings(true);
        } else if (convIdFromUrl) {
          const found = (data || []).find(c => c.id === convIdFromUrl);
          if (found) {
            setSelectedConv(found);
          }
        }
      }
    }
  };

  const fetchMessages = async (convID, before = '', append = false) => {
    let url = `api/conversations/${convID}/messages?limit=50`;
    if (before) {
      url += `&before=${encodeURIComponent(before)}`;
    }
    const res = await apiFetch(url);
    if (res.ok) {
      const data = await res.json();
      if (selectedConvRef.current?.id !== convID) {
        return;
      }
      if (append) {
        setMessages(prev => [...(data || []), ...prev]);
        setHasMoreMessages((data || []).length === 50);
      } else {
        setMessages(data || []);
        setHasMoreMessages((data || []).length === 50);
        setTimeout(scrollToBottom, 50);
      }
    }
  };

  const [searchResults, setSearchResults] = useState([]);
  const [searchBeforeTime, setSearchBeforeTime] = useState('');
  const [hasMoreSearch, setHasMoreSearch] = useState(true);

  const searchChatMessages = async (convID, query) => {
    if (!query) {
      setSearchResults([]);
      return;
    }
    const url = `api/conversations/${convID}/messages?q=${encodeURIComponent(query)}`;
    const res = await apiFetch(url);
    if (res.ok) {
      const data = await res.json();
      if (selectedConvRef.current?.id !== convID) {
        return;
      }
      setSearchResults(data || []);
      setHasMoreSearch(false); // Search queries match all matching directly
    }
  };

  useEffect(() => {
    if (isSearchingMessages && selectedConv) {
      const timer = setTimeout(() => {
        searchChatMessages(selectedConv.id, messageSearchQuery);
      }, 300);
      return () => clearTimeout(timer);
    }
  }, [messageSearchQuery, isSearchingMessages, selectedConv?.id]);

  const fetchSharedMedia = async (convID, before = '', append = false) => {
    setIsLoadingMoreMedia(true);
    let url = `api/conversations/${convID}/attachments?limit=20`;
    if (before) {
      url += `&before=${encodeURIComponent(before)}`;
    }
    try {
      const res = await apiFetch(url);
      if (res.ok) {
        const data = await res.json();
        if (selectedConvRef.current?.id !== convID) {
          return;
        }
        if (append) {
          setSharedMedia(prev => [...prev, ...(data || [])]);
        } else {
          setSharedMedia(data || []);
        }
        setHasMoreMedia((data || []).length === 20);
      }
    } catch (err) {
      console.error(err);
    } finally {
      setIsLoadingMoreMedia(false);
    }
  };

  const handleCreateConversation = async () => {
    setNewChatError('');
    try {
      const res = await apiFetch('api/conversations', {
        method: 'POST',
        body: JSON.stringify({
          name: newConvName,
          participant_ids: selectedContactIDs,
        }),
      });
      if (res.ok) {
        const newConv = await res.json();
        setShowModal(false);
        setNewConvName('');
        setSelectedContactIDs([]);
        setSelectedConv(newConv);
        fetchConversations();
      } else {
        const data = await res.json();
        setNewChatError(data.error || 'Failed to create conversation');
      }
    } catch (err) {
      console.error(err);
      setNewChatError('Error creating conversation');
    }
  };

  const handleSendMessage = async (e) => {
    e.preventDefault();
    if (messageText.trim() === '' && !attachment) return;

    const formData = new FormData();
    formData.append('text', messageText);
    if (attachment) {
      formData.append('attachment', attachment);
    }

    setMessageText('');
    setAttachment(null);
    if (fileInputRef.current) fileInputRef.current.value = '';

    try {
      const res = await apiFetch(`api/conversations/${selectedConv.id}/messages`, {
        method: 'POST',
        body: formData,
      });
      if (res.ok) {
        fetchConversations();
        fetchMessages(selectedConv.id, '', false);
      } else {
        const data = await res.json();
        alert(data.error || 'Failed to send message');
      }
    } catch (err) {
      console.error(err);
      alert('Error sending message');
    }
  };

  const handleRenameConversation = async () => {
    try {
      const res = await apiFetch(`api/conversations/${selectedConv.id}/rename`, {
        method: 'POST',
        body: JSON.stringify({ name: renameInput }),
      });
      if (res.ok) {
        setShowRenameModal(false);
        setRenameInput('');
        fetchConversations();
      } else {
        const data = await res.json();
        alert(data.error || 'Failed to rename conversation');
      }
    } catch (err) {
      console.error(err);
      alert('Error renaming conversation');
    }
  };

  const handleDeleteConversation = async () => {
    try {
      const res = await apiFetch(`api/conversations/${selectedConv.id}`, {
        method: 'DELETE',
      });
      if (res.ok) {
        setShowDeleteModal(false);
        setSelectedConv(null);
        fetchConversations();
      } else {
        const data = await res.json();
        alert(data.error || 'Failed to delete conversation');
      }
    } catch (err) {
      console.error(err);
      alert('Error deleting conversation');
    }
  };

  const handleScroll = () => {
    if (!containerRef.current || messages.length === 0 || isLoadingMore || !hasMoreMessages) return;
    const { scrollTop } = containerRef.current;
    if (scrollTop === 0) {
      setIsLoadingMore(true);
      const firstMsgTime = messages[0].sent_at;
      const prevScrollHeight = containerRef.current.scrollHeight;
      fetchMessages(selectedConv.id, firstMsgTime, true).finally(() => {
        setIsLoadingMore(false);
        setTimeout(() => {
          if (containerRef.current) {
            containerRef.current.scrollTop = containerRef.current.scrollHeight - prevScrollHeight;
          }
        }, 30);
      });
    }
  };

  const handleMediaScroll = (e) => {
    const { scrollTop, scrollHeight, clientHeight } = e.target;
    if (scrollHeight - scrollTop - clientHeight < 40 && hasMoreMedia && !isLoadingMoreMedia && sharedMedia.length > 0) {
      const oldestMediaSentAt = sharedMedia[sharedMedia.length - 1].sent_at;
      fetchSharedMedia(selectedConv.id, oldestMediaSentAt, true);
    }
  };

  const isImage = (mimeType) => {
    return mimeType && mimeType.startsWith('image/');
  };

  const isVideo = (mimeType) => {
    return mimeType && mimeType.startsWith('video/');
  };

  const messagesEndRef = useRef(null);
  const fileInputRef = useRef(null);

  const scrollToBottom = () => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  };

  const handleAttachmentChange = (e) => {
    if (e.target.files && e.target.files[0]) {
      setAttachment(e.target.files[0]);
    }
  };

  const getConversationName = (conv) => {
    if (!conv) return '';
    if (conv.name) return conv.name;
    const otherParts = conv.participants.filter(p => p.user_id !== profile?.user_id);
    if (otherParts.length === 0) return 'Self Chat';
    return otherParts.map(p => p.display_name).join(', ');
  };

  const formatTime = (timeStr) => {
    if (!timeStr) return '';
    const date = new Date(timeStr);
    return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
  };

  const filteredConvs = conversations.filter(c => 
    getConversationName(c).toLowerCase().includes(searchQuery.toLowerCase())
  );

  return (
    <div className="app-container">
      
      {/* Sidebar Panel */}
      <Sidebar
        profile={profile}
        searchQuery={searchQuery}
        setSearchQuery={setSearchQuery}
        filteredConvs={filteredConvs}
        selectedConv={selectedConv}
        setSelectedConv={setSelectedConv}
        showSettings={showSettings}
        setShowSettings={setShowSettings}
        setShowModal={setShowModal}
        getConversationName={getConversationName}
        formatTime={formatTime}
      />

      {/* Main Conversation Window or Settings Page */}
      {showSettings ? (
        <SettingsPage 
          theme={theme} 
          setTheme={setTheme} 
        />
      ) : selectedConv ? (
        <ChatArea
          selectedConv={selectedConv}
          profile={profile}
          contacts={contacts}
          getConversationName={getConversationName}
          showMenu={showMenu}
          setShowMenu={setShowMenu}
          setShowRenameModal={setShowRenameModal}
          setRenameInput={setRenameInput}
          isSearchingMessages={isSearchingMessages}
          setIsSearchingMessages={setIsSearchingMessages}
          messageSearchQuery={messageSearchQuery}
          setMessageSearchQuery={setMessageSearchQuery}
          setShowMediaPanel={setShowMediaPanel}
          setShowParticipantsPanel={setShowParticipantsPanel}
          setSharedMedia={setSharedMedia}
          setHasMoreMedia={setHasMoreMedia}
          fetchSharedMedia={fetchSharedMedia}
          setShowDeleteModal={setShowDeleteModal}
          sentRequests={sentRequests}
          addingContacts={addingContacts}
          handleAddContact={handleAddContact}
          containerRef={containerRef}
          handleScroll={handleScroll}
          searchResults={searchResults}
          messages={messages}
          isImage={isImage}
          isVideo={isVideo}
          formatTime={formatTime}
          messagesEndRef={messagesEndRef}
          attachment={attachment}
          setAttachment={setAttachment}
          fileInputRef={fileInputRef}
          handleAttachmentChange={handleAttachmentChange}
          messageText={messageText}
          setMessageText={setMessageText}
          handleSendMessage={handleSendMessage}
        />
      ) : (
        <div className="chat-empty-state">
          <div className="chat-empty-icon"><IconChat /></div>
          <h2>Magic Chat P2P</h2>
          <p style={{ marginTop: '8px', maxWidth: '320px', fontSize: '14px' }}>
            Select a conversation from the sidebar or click the "+" button to start a new chat with your contacts.
          </p>
        </div>
      )}

      {/* Modals & Slide-out Panels */}
      <NewChatModal
        showModal={showModal}
        setShowModal={setShowModal}
        selectedContactIDs={selectedContactIDs}
        setSelectedContactIDs={setSelectedContactIDs}
        newConvName={newConvName}
        setNewConvName={setNewConvName}
        contacts={contacts}
        handleCreateConversation={handleCreateConversation}
        newChatError={newChatError}
      />

      <RenameChatModal
        showRenameModal={showRenameModal}
        setShowRenameModal={setShowRenameModal}
        renameInput={renameInput}
        setRenameInput={setRenameInput}
        handleRenameConversation={handleRenameConversation}
      />

      <DeleteChatModal
        showDeleteModal={showDeleteModal}
        setShowDeleteModal={setShowDeleteModal}
        selectedConv={selectedConv}
        getConversationName={getConversationName}
        handleDeleteConversation={handleDeleteConversation}
      />

      <MediaPanel
        showMediaPanel={showMediaPanel}
        setShowMediaPanel={setShowMediaPanel}
        sharedMedia={sharedMedia}
        handleMediaScroll={handleMediaScroll}
        isImage={isImage}
        isVideo={isVideo}
        formatTime={formatTime}
      />

      <ParticipantsPanel
        showParticipantsPanel={showParticipantsPanel}
        setShowParticipantsPanel={setShowParticipantsPanel}
        selectedConv={selectedConv}
        profile={profile}
        contacts={contacts}
        addingContacts={addingContacts}
        handleAddContact={handleAddContact}
        mediaPanelWidth={mediaPanelWidth}
        startResizingMedia={startResizingMedia}
      />

    </div>
  );
}

export default App;
