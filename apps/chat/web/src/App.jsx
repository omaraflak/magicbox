import React, { useState, useEffect, useRef } from 'react';

// Inline SVG Icons
const IconPlus = () => (
  <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><line x1="12" y1="5" x2="12" y2="19"></line><line x1="5" y1="12" x2="19" y2="12"></line></svg>
);

const IconPaperclip = () => (
  <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M21.44 11.05l-9.19 9.19a6 6 0 0 1-8.49-8.49l9.19-9.19a4 4 0 0 1 5.66 5.66l-9.2 9.19a2 2 0 0 1-2.83-2.83l8.49-8.48"></path></svg>
);

const IconSend = () => (
  <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><line x1="22" y1="2" x2="11" y2="13"></line><polygon points="22 2 15 22 11 13 2 9 22 2"></polygon></svg>
);

const IconTrash = () => (
  <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><polyline points="3 6 5 6 21 6"></polyline><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"></path><line x1="10" y1="11" x2="10" y2="17"></line><line x1="14" y1="11" x2="14" y2="17"></line></svg>
);

const IconFile = () => (
  <svg xmlns="http://www.w3.org/2000/svg" width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"></path><polyline points="14 2 14 8 20 8"></polyline><line x1="16" y1="13" x2="8" y2="13"></line><line x1="16" y1="17" x2="8" y2="17"></line><polyline points="10 9 9 9 8 9"></polyline></svg>
);

const IconDownload = () => (
  <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"></path><polyline points="7 10 12 15 17 10"></polyline><line x1="12" y1="15" x2="12" y2="3"></line></svg>
);

const IconChat = () => (
  <svg xmlns="http://www.w3.org/2000/svg" width="64" height="64" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"></path></svg>
);

const IconEdit = () => (
  <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"></path><path d="M18.5 2.5a2.121 2.121 0 1 1 3 3L12 15l-4 1 1-4z"></path></svg>
);

const IconCheck = () => (
  <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><polyline points="20 6 9 17 4 12"></polyline></svg>
);

const IconDots = () => (
  <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><circle cx="12" cy="5" r="1.5"></circle><circle cx="12" cy="12" r="1.5"></circle><circle cx="12" cy="19" r="1.5"></circle></svg>
);

const IconImage = () => (
  <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><rect x="3" y="3" width="18" height="18" rx="2" ry="2"></rect><circle cx="8.5" cy="8.5" r="1.5"></circle><polyline points="21 15 16 10 5 21"></polyline></svg>
);

const IconSearch = () => (
  <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><circle cx="11" cy="11" r="8"></circle><line x1="21" y1="21" x2="16.65" y2="16.65"></line></svg>
);

function App() {
  const [profile, setProfile] = useState(null);
  const [contacts, setContacts] = useState([]);
  const [conversations, setConversations] = useState([]);
  const [selectedConv, setSelectedConv] = useState(null);
  const [messages, setMessages] = useState([]);
  
  // Search & Modal State
  const [searchQuery, setSearchQuery] = useState('');
  const [showModal, setShowModal] = useState(false);
  const [newConvName, setNewConvName] = useState('');
  const [selectedContactIDs, setSelectedContactIDs] = useState([]);

  // Menu & Modal States
  const [showMenu, setShowMenu] = useState(false);
  const [showRenameModal, setShowRenameModal] = useState(false);
  const [showDeleteModal, setShowDeleteModal] = useState(false);
  const [renameInput, setRenameInput] = useState('');

  // Shared Media & Search States
  const [showMediaPanel, setShowMediaPanel] = useState(false);
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
  
  // Refs
  const messagesEndRef = useRef(null);
  const fileInputRef = useRef(null);

  // Load baseline profile and contacts list
  useEffect(() => {
    fetchProfile();
    fetchContacts();
    fetchConversations();
  }, []);

  const getConversationName = (conv) => {
    if (!conv) return '';
    if (conv.name) return conv.name;
    const others = conv.participants ? conv.participants.filter(p => p.user_id !== profile?.user_id) : [];
    if (others.length === 0) return 'Solo Chat';
    return others.map(p => p.display_name).join(', ');
  };

  // Poll databases and notifications via EventSource
  useEffect(() => {
    // SSE event stream
    const eventSource = new EventSource('api/events');
    
    eventSource.onmessage = (event) => {
      if (event.data === 'update') {
        fetchConversations();
      }
    };

    eventSource.onerror = () => {
      // EventSource reconnects automatically, but log just in case
      console.log('SSE connection error, waiting for reconnect...');
    };

    // Fallback polling every 4 seconds
    const interval = setInterval(() => {
      fetchConversations();
    }, 4000);

    return () => {
      eventSource.close();
      clearInterval(interval);
    };
  }, []);

  // Fetch conversations messages when a conversation is selected
  useEffect(() => {
    if (selectedConv) {
      fetchMessages(selectedConv.id);
    } else {
      setMessages([]);
    }
    setShowMediaPanel(false);
    setSharedMedia([]);
    setHasMoreMedia(true);
    setIsLoadingMoreMedia(false);
    setIsSearchingMessages(false);
    setMessageSearchQuery('');
  }, [selectedConv]);

  // Click outside to close dropdown menu
  useEffect(() => {
    if (!showMenu) return;
    const closeMenu = () => setShowMenu(false);
    document.addEventListener('click', closeMenu);
    return () => document.removeEventListener('click', closeMenu);
  }, [showMenu]);

  // Debounced search trigger when messageSearchQuery changes
  useEffect(() => {
    if (!selectedConv) return;
    
    if (messageSearchQuery.trim() === '') {
      if (isSearchingMessages) {
        fetchMessages(selectedConv.id);
      }
      return;
    }
    
    const delayDebounce = setTimeout(() => {
      searchChatMessages(selectedConv.id, messageSearchQuery);
    }, 300);
    
    return () => clearTimeout(delayDebounce);
  }, [messageSearchQuery]);

  // Fetch calls

  const fetchProfile = async () => {
    try {
      const res = await fetch('api/profile');
      if (res.ok) {
        const data = await res.json();
        setProfile(data);
      }
    } catch (e) {
      console.error('Failed to load profile', e);
    }
  };

  const fetchContacts = async () => {
    try {
      const res = await fetch('api/contacts');
      if (res.ok) {
        const data = await res.json();
        setContacts(data || []);
      }
    } catch (e) {
      console.error('Failed to load contacts', e);
    }
  };

  const fetchConversations = async () => {
    try {
      const res = await fetch('api/conversations');
      if (res.ok) {
        const data = await res.json();
        setConversations(data || []);
        
        // If we currently have a conversation open, refresh its data to check for any updates
        if (selectedConv) {
          const updated = (data || []).find(c => c.id === selectedConv.id);
          if (updated) {
            setSelectedConv(updated);
            if (!isSearchingMessages) {
              fetchMessages(selectedConv.id);
            }
          }
        }
      }
    } catch (e) {
      console.error('Failed to load conversations', e);
    }
  };

  const fetchMessages = async (convID, before = '', append = false) => {
    if (isSearchingMessages && before !== '') return;

    try {
      const url = before 
        ? `api/conversations/${convID}/messages?limit=50&before=${before}` 
        : `api/conversations/${convID}/messages?limit=50`;
        
      const res = await fetch(url);
      if (res.ok) {
        const data = await res.json();
        const newMsgs = data || [];
        
        if (append) {
          const container = containerRef.current;
          const oldScrollHeight = container ? container.scrollHeight : 0;
          const oldScrollTop = container ? container.scrollTop : 0;
          
          setMessages(prev => [...newMsgs, ...prev]);
          setHasMoreMessages(newMsgs.length === 50);
          
          setTimeout(() => {
            if (container) {
              const newScrollHeight = container.scrollHeight;
              container.scrollTop = oldScrollTop + (newScrollHeight - oldScrollHeight);
            }
          }, 0);
        } else {
          setMessages(newMsgs);
          setHasMoreMessages(newMsgs.length === 50);
          setTimeout(() => {
            if (containerRef.current) {
              containerRef.current.scrollTop = containerRef.current.scrollHeight;
            }
          }, 0);
        }
      }
    } catch (e) {
      console.error('Failed to load messages', e);
    }
  };

  const searchChatMessages = async (convID, query) => {
    try {
      const res = await fetch(`api/conversations/${convID}/messages?q=${encodeURIComponent(query)}`);
      if (res.ok) {
        const data = await res.json();
        setMessages(data || []);
        setHasMoreMessages(false);
      }
    } catch (e) {
      console.error('Failed to search messages', e);
    }
  };

  const handleScroll = () => {
    if (!selectedConv || !containerRef.current || !hasMoreMessages || isLoadingMore || isSearchingMessages) return;
    
    const container = containerRef.current;
    if (container.scrollTop < 150) {
      setIsLoadingMore(true);
      const oldestMsg = messages[0];
      if (oldestMsg) {
        fetchMessages(selectedConv.id, oldestMsg.sent_at, true).then(() => {
          setIsLoadingMore(false);
        });
      } else {
        setIsLoadingMore(false);
      }
    }
  };

  const fetchSharedMedia = async (convID, before = '', append = false) => {
    try {
      const url = before 
        ? `api/conversations/${convID}/attachments?limit=20&before=${before}` 
        : `api/conversations/${convID}/attachments?limit=20`;
        
      const res = await fetch(url);
      if (res.ok) {
        const data = await res.json();
        const newMedia = data || [];
        
        if (append) {
          setSharedMedia(prev => [...prev, ...newMedia]);
        } else {
          setSharedMedia(newMedia);
        }
        setHasMoreMedia(newMedia.length === 20);
      }
    } catch (e) {
      console.error('Failed to load shared media', e);
    }
  };

  const handleMediaScroll = (e) => {
    if (!selectedConv || !hasMoreMedia || isLoadingMoreMedia) return;
    
    const target = e.target;
    if (target.scrollHeight - target.scrollTop - target.clientHeight < 60) {
      setIsLoadingMoreMedia(true);
      const oldestMedia = sharedMedia[sharedMedia.length - 1];
      if (oldestMedia) {
        fetchSharedMedia(selectedConv.id, oldestMedia.sent_at, true).then(() => {
          setIsLoadingMoreMedia(false);
        });
      } else {
        setIsLoadingMoreMedia(false);
      }
    }
  };

  // Actions

  const handleCreateConversation = async () => {
    if (selectedContactIDs.length === 0) return;

    try {
      const res = await fetch('api/conversations', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          name: selectedContactIDs.length > 1 ? newConvName : '',
          participant_ids: selectedContactIDs
        })
      });

      if (res.ok) {
        const newConv = await res.json();
        setConversations(prev => [newConv, ...prev]);
        setSelectedConv(newConv);
        setShowModal(false);
        // Reset modal forms
        setNewConvName('');
        setSelectedContactIDs([]);
      } else {
        const err = await res.json();
        alert(err.error || 'Failed to create conversation');
      }
    } catch (e) {
      alert('Network error: ' + e.message);
    }
  };

  const handleSendMessage = async (e) => {
    e.preventDefault();
    if (!selectedConv) return;
    if (messageText.trim() === '' && !attachment) return;

    try {
      let res;
      if (attachment) {
        const formData = new FormData();
        formData.append('text', messageText);
        formData.append('attachment', attachment);

        res = await fetch(`api/conversations/${selectedConv.id}/messages`, {
          method: 'POST',
          body: formData // Content-Type header set automatically by browser
        });
      } else {
        res = await fetch(`api/conversations/${selectedConv.id}/messages`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ text: messageText })
        });
      }

      if (res.ok) {
        const newMsg = await res.json();
        setMessages(prev => [...prev, newMsg]);
        setMessageText('');
        setAttachment(null);
        if (fileInputRef.current) fileInputRef.current.value = '';
        fetchConversations(); // Update side panel state
      } else {
        const err = await res.json();
        alert(err.error || 'Failed to send message');
      }
    } catch (e) {
      alert('Network error: ' + e.message);
    }
  };

  const handleRenameConversation = async () => {
    const trimmed = renameInput.trim();
    if (trimmed === '' || !selectedConv) return;
    try {
      const res = await fetch(`api/conversations/${selectedConv.id}/rename`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ name: trimmed })
      });
      if (res.ok) {
        setSelectedConv(prev => ({ ...prev, name: trimmed }));
        setShowRenameModal(false);
        fetchConversations();
      } else {
        alert('Failed to rename conversation');
      }
    } catch (e) {
      alert('Network error: ' + e.message);
    }
  };

  const handleDeleteConversation = async () => {
    if (!selectedConv) return;
    try {
      const res = await fetch(`api/conversations/${selectedConv.id}`, {
        method: 'DELETE'
      });

      if (res.ok) {
        setSelectedConv(null);
        setShowDeleteModal(false);
        fetchConversations();
      } else {
        alert('Failed to delete chat');
      }
    } catch (e) {
      alert('Network error: ' + e.message);
    }
  };

  const handleAttachmentChange = (e) => {
    if (e.target.files && e.target.files[0]) {
      setAttachment(e.target.files[0]);
    }
  };

  const toggleContactSelection = (contactID) => {
    setSelectedContactIDs(prev => {
      if (prev.includes(contactID)) {
        return prev.filter(id => id !== contactID);
      } else {
        return [...prev, contactID];
      }
    });
  };

  // Rendering Helpers

  const formatTime = (timeStr) => {
    if (!timeStr) return '';
    try {
      const date = new Date(timeStr);
      return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
    } catch (e) {
      return '';
    }
  };

  const isImage = (mimeType) => {
    return mimeType && mimeType.startsWith('image/');
  };

  const isVideo = (mimeType) => {
    return mimeType && mimeType.startsWith('video/');
  };

  // Filter conversations based on search text
  const filteredConvs = conversations.filter(c => 
    getConversationName(c).toLowerCase().includes(searchQuery.toLowerCase())
  );

  return (
    <div className="app-container">
      
      {/* Sidebar Panel */}
      <div className="sidebar">
        <div className="sidebar-header">
          <div className="user-profile">
            <div className="avatar">
              {profile ? profile.username.substring(0, 2) : 'MC'}
            </div>
            <div className="user-info">
              <span className="username">{profile ? profile.username : 'Loading...'}</span>
              <span className="user-status">Online</span>
            </div>
          </div>
          <button className="action-btn" onClick={() => setShowModal(true)} title="New Conversation">
            <IconPlus />
          </button>
        </div>

        <div className="search-bar-container">
          <input 
            type="text" 
            placeholder="Search or start new chat" 
            className="search-input" 
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
          />
        </div>

        <div className="chats-list">
          {filteredConvs.map(c => (
            <div 
              key={c.id} 
              className={`chat-item ${selectedConv?.id === c.id ? 'active' : ''}`}
              onClick={() => setSelectedConv(c)}
            >
              <div className="avatar">
                {getConversationName(c).substring(0, 2)}
              </div>
              <div className="chat-item-details">
                <div className="chat-item-header">
                  <span className="chat-item-name">{getConversationName(c)}</span>
                  <span className="chat-item-time">{c.last_message ? formatTime(c.last_message.sent_at) : ''}</span>
                </div>
                <div className="chat-item-body">
                  <span className="chat-item-lastmsg">
                    {c.last_message ? (
                      c.last_message.attachment_name ? `📎 [File] ${c.last_message.attachment_name}` : c.last_message.text
                    ) : (
                      <em style={{ opacity: 0.5 }}>No messages yet</em>
                    )}
                  </span>
                  {c.unread_count > 0 && (
                    <span className="unread-badge">{c.unread_count}</span>
                  )}
                </div>
              </div>
            </div>
          ))}
        </div>
      </div>

      {/* Main Conversation Window */}
      {selectedConv ? (
        <>
          <div className="chat-area animate-fade-in">
          <div className="chat-header">
            <div className="chat-header-info">
              <div className="avatar">
                {getConversationName(selectedConv).substring(0, 2)}
              </div>
              <div>
                <div className="chat-title">{getConversationName(selectedConv)}</div>
                <div className="chat-subtitle">
                  {selectedConv.participants.length > 2 
                    ? `(${selectedConv.participants.length}) Group Chat`
                    : 'Direct Message'
                  }
                </div>
              </div>
            </div>
            
            <div className="menu-container">
              <button 
                className="action-btn" 
                onClick={(e) => {
                  e.stopPropagation();
                  setShowMenu(!showMenu);
                }}
                title="Menu"
              >
                <IconDots />
              </button>
              {showMenu && (
                <div className="dropdown-menu">
                  <button 
                    className="dropdown-item" 
                    onClick={() => {
                      setShowRenameModal(true);
                      setRenameInput(getConversationName(selectedConv));
                    }}
                  >
                    <IconEdit /> Rename Chat
                  </button>
                  <button 
                    className="dropdown-item" 
                    onClick={() => {
                      setIsSearchingMessages(true);
                      setMessageSearchQuery('');
                    }}
                  >
                    <IconSearch /> Search Chat Text
                  </button>
                  <button 
                    className="dropdown-item" 
                    onClick={() => {
                      setShowMediaPanel(true);
                      setSharedMedia([]);
                      setHasMoreMedia(true);
                      fetchSharedMedia(selectedConv.id);
                    }}
                  >
                    <IconImage /> View Shared Media
                  </button>
                  <button 
                    className="dropdown-item danger" 
                    onClick={() => setShowDeleteModal(true)}
                  >
                    <IconTrash /> Delete Chat
                  </button>
                </div>
              )}
            </div>
          </div>

          {isSearchingMessages && (
            <div className="chat-search-bar">
              <div className="chat-search-input-container">
                <span style={{ color: 'var(--text-mute)', display: 'flex', alignItems: 'center' }}><IconSearch /></span>
                <input 
                  type="text" 
                  placeholder="Search messages..." 
                  className="chat-search-input"
                  value={messageSearchQuery}
                  onChange={(e) => setMessageSearchQuery(e.target.value)}
                  autoFocus
                />
              </div>
              <button className="cancel-attach-btn" onClick={() => {
                setIsSearchingMessages(false);
                setMessageSearchQuery('');
              }}>
                Close
              </button>
            </div>
          )}

          {/* Messages container */}
          <div className="messages-container" ref={containerRef} onScroll={handleScroll}>
            {(() => {
              const filtered = messages;
              if (filtered.length === 0) {
                return (
                  <div style={{ textAlign: 'center', color: 'var(--text-mute)', padding: '40px', fontSize: '14.5px' }}>
                    {messageSearchQuery ? "No matching text messages found." : "No messages yet."}
                  </div>
                );
              }
              return filtered.map(m => {
                const isSentByMe = profile && m.sender_id === profile.user_id;
                const attachmentURL = `api/attachments/${m.conversation_id}/${encodeURIComponent(m.attachment_name)}`;

                return (
                  <div 
                    key={m.id} 
                    className={`message-bubble-wrapper ${isSentByMe ? 'sent' : 'received'}`}
                  >
                    <div className="message-bubble">
                      {!isSentByMe && selectedConv.participants.length > 2 && (
                        <span className="message-sender">{m.sender_name}</span>
                      )}

                      {m.attachment_name && (
                        <div className="message-media">
                          {isImage(m.attachment_type) ? (
                            <img 
                              src={attachmentURL} 
                              alt={m.attachment_name} 
                              className="message-image" 
                              onClick={() => window.open(attachmentURL, '_blank')}
                            />
                          ) : isVideo(m.attachment_type) ? (
                            <video 
                              src={attachmentURL} 
                              controls 
                              className="message-video"
                            />
                          ) : (
                            <div className="message-file-card">
                              <span className="message-file-icon"><IconFile /></span>
                              <div className="message-file-info">
                                <div className="message-file-name" title={m.attachment_name}>
                                  {m.attachment_name}
                                </div>
                                <div className="message-file-size">
                                  Attachment File
                                </div>
                              </div>
                              <a 
                                href={attachmentURL} 
                                download={m.attachment_name} 
                                className="message-file-download"
                                title="Download to computer"
                              >
                                <IconDownload />
                              </a>
                            </div>
                          )}
                        </div>
                      )}

                      {m.text && <div className="message-text">{m.text}</div>}
                      
                      <div className="message-meta">
                        <span>{formatTime(m.sent_at)}</span>
                      </div>
                    </div>
                  </div>
                );
              });
            })()}
            <div ref={messagesEndRef} />
          </div>

          {/* Attachment Preview Bar */}
          {attachment && (
            <div className="attachment-preview-bar">
              <div className="attachment-preview-info">
                <span>📎</span>
                <strong>{attachment.name}</strong>
                <span style={{ opacity: 0.7 }}>({(attachment.size / 1024).toFixed(1)} KB)</span>
              </div>
              <button className="cancel-attach-btn" onClick={() => {
                setAttachment(null);
                if (fileInputRef.current) fileInputRef.current.value = '';
              }}>
                Cancel
              </button>
            </div>
          )}

          {/* Input Area */}
          <form className="input-area" onSubmit={handleSendMessage}>
            <input 
              type="file" 
              ref={fileInputRef} 
              style={{ display: 'none' }} 
              onChange={handleAttachmentChange}
            />
            <button 
              type="button" 
              className="action-btn" 
              onClick={() => fileInputRef.current?.click()}
              title="Attach File"
            >
              <IconPaperclip />
            </button>
            
            <div className="text-input-container">
              <input
                type="text"
                placeholder="Type a message..."
                className="message-input"
                value={messageText}
                onChange={(e) => setMessageText(e.target.value)}
              />
            </div>
            
            <button 
              type="submit" 
              className="send-btn" 
              disabled={messageText.trim() === '' && !attachment}
            >
              <IconSend />
            </button>
          </form>
        </div>
        {showMediaPanel && (
          <div className="media-panel">
            <div className="media-panel-header">
              <span className="media-panel-title">Shared Media</span>
              <button className="action-btn" onClick={() => setShowMediaPanel(false)}>✕</button>
            </div>
            <div className="media-panel-body" onScroll={handleMediaScroll}>
              {(() => {
                const mediaMessages = sharedMedia;
                const visualMedia = mediaMessages.filter(m => isImage(m.attachment_type) || isVideo(m.attachment_type));
                const docFiles = mediaMessages.filter(m => !isImage(m.attachment_type) && !isVideo(m.attachment_type));

                if (mediaMessages.length === 0) {
                  return (
                    <div style={{ textAlign: 'center', color: 'var(--text-mute)', padding: '32px' }}>
                      No media or files shared in this chat.
                    </div>
                  );
                }

                return (
                  <div>
                    {visualMedia.length > 0 && (
                      <div>
                        <div className="media-files-header">Photos & Videos ({visualMedia.length})</div>
                        <div className="media-grid">
                          {visualMedia.map(m => {
                            const url = `api/attachments/${m.conversation_id}/${encodeURIComponent(m.attachment_name)}`;
                            const isImg = isImage(m.attachment_type);
                            return (
                              <div 
                                key={m.id} 
                                className="media-grid-item"
                                onClick={() => window.open(url, '_blank')}
                                title={`Shared by ${m.sender_name} at ${formatTime(m.sent_at)}`}
                              >
                                {isImg ? (
                                  <img src={url} alt={m.attachment_name} className="media-grid-img" />
                                ) : (
                                  <video src={url} className="media-grid-video" muted playsInline />
                                )}
                                {!isImg && <span className="media-grid-video-badge">VIDEO</span>}
                              </div>
                            );
                          })}
                        </div>
                      </div>
                    )}

                    {docFiles.length > 0 && (
                      <div style={{ marginTop: visualMedia.length > 0 ? '20px' : '0' }}>
                        <div className="media-files-header">Documents & Files ({docFiles.length})</div>
                        <div className="media-files-list">
                          {docFiles.map(m => {
                            const url = `api/attachments/${m.conversation_id}/${encodeURIComponent(m.attachment_name)}`;
                            return (
                              <div key={m.id} className="message-file-card" style={{ margin: 0 }}>
                                <span className="message-file-icon"><IconFile /></span>
                                <div className="message-file-info">
                                  <div className="message-file-name" title={m.attachment_name}>{m.attachment_name}</div>
                                  <div className="message-file-size">Shared by {m.sender_name}</div>
                                </div>
                                <a href={url} download={m.attachment_name} className="message-file-download" title="Download">
                                  <IconDownload />
                                </a>
                              </div>
                            );
                          })}
                        </div>
                      </div>
                    )}
                  </div>
                );
              })()}
            </div>
          </div>
        )}
      </>
      ) : (
        <div className="chat-empty-state">
          <div className="chat-empty-icon"><IconChat /></div>
          <h2>Magic Chat P2P</h2>
          <p style={{ marginTop: '8px', maxWidth: '320px', fontSize: '14px' }}>
            Select a conversation from the sidebar or click the "+" button to start a new chat with your contacts.
          </p>
        </div>
      )}

      {/* New Conversation Modal */}
      {showModal && (
        <div className="modal-overlay">
          <div className="modal-content">
            <div className="modal-header">
              <span className="modal-title">New Conversation</span>
              <button className="action-btn" onClick={() => setShowModal(false)}>✕</button>
            </div>
            <div className="modal-body">
              
              {selectedContactIDs.length > 1 && (
                <div className="form-group">
                  <label className="form-label">Group Name</label>
                  <input 
                    type="text" 
                    placeholder="Enter group name" 
                    className="form-input"
                    value={newConvName}
                    onChange={(e) => setNewConvName(e.target.value)}
                  />
                </div>
              )}

              <div className="form-group">
                <label className="form-label">Select Contacts</label>
                {contacts.length === 0 ? (
                  <div style={{ padding: '12px', fontSize: '13px', color: 'var(--text-mute)', border: '1px solid var(--border)', borderRadius: '8px' }}>
                    No contacts found. Please add contacts in the Magicbox dashboard first!
                  </div>
                ) : (
                  <div className="contacts-picker">
                    {contacts.map(c => {
                      const isSelected = selectedContactIDs.includes(c.id);
                      return (
                        <div 
                          key={c.id} 
                          className={`contact-picker-item ${isSelected ? 'selected' : ''}`}
                          onClick={() => toggleContactSelection(c.id)}
                        >
                          <div className="contact-picker-item-left">
                            <div className="avatar">
                              {c.display_name.substring(0, 2)}
                            </div>
                            <span className="contact-picker-name">{c.display_name}</span>
                          </div>
                          {isSelected && (
                            <span className="contact-picker-check">
                              <IconCheck />
                            </span>
                          )}
                        </div>
                      );
                    })}
                  </div>
                )}
              </div>

            </div>
            <div className="modal-footer">
              <button className="btn btn-secondary" onClick={() => setShowModal(false)}>Cancel</button>
              <button 
                className="btn btn-primary" 
                onClick={handleCreateConversation}
                disabled={selectedContactIDs.length === 0 || (selectedContactIDs.length > 1 && newConvName.trim() === '')}
              >
                Create
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Rename Conversation Modal */}
      {showRenameModal && (
        <div className="modal-overlay" onClick={() => setShowRenameModal(false)}>
          <div className="modal-content" onClick={(e) => e.stopPropagation()}>
            <div className="modal-header">
              <span className="modal-title">Rename Chat</span>
              <button className="action-btn" onClick={() => setShowRenameModal(false)}>✕</button>
            </div>
            <div className="modal-body">
              <div className="form-group">
                <label className="form-label">New Name</label>
                <input 
                  type="text" 
                  className="form-input" 
                  value={renameInput}
                  onChange={(e) => setRenameInput(e.target.value)}
                  onKeyDown={(e) => {
                    if (e.key === 'Enter') handleRenameConversation();
                    if (e.key === 'Escape') setShowRenameModal(false);
                  }}
                  autoFocus
                />
              </div>
            </div>
            <div className="modal-footer">
              <button className="btn btn-secondary" onClick={() => setShowRenameModal(false)}>Cancel</button>
              <button className="btn btn-primary" onClick={handleRenameConversation} disabled={renameInput.trim() === ''}>
                Rename
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Delete Confirmation Modal */}
      {showDeleteModal && (
        <div className="modal-overlay" onClick={() => setShowDeleteModal(false)}>
          <div className="modal-content" onClick={(e) => e.stopPropagation()}>
            <div className="modal-header">
              <span className="modal-title">Delete Chat</span>
              <button className="action-btn" onClick={() => setShowDeleteModal(false)}>✕</button>
            </div>
            <div className="modal-body">
              <p style={{ fontSize: '14.5px', lineHeight: '1.5', opacity: 0.9 }}>
                Are you sure you want to delete the chat <strong>"{getConversationName(selectedConv)}"</strong>?
              </p>
              <p style={{ fontSize: '13px', color: 'var(--text-mute)', marginTop: '8px' }}>
                This will delete the conversation and erase all sent/received messages and files. This action cannot be undone.
              </p>
            </div>
            <div className="modal-footer">
              <button className="btn btn-secondary" onClick={() => setShowDeleteModal(false)}>Cancel</button>
              <button className="btn btn-danger" onClick={handleDeleteConversation}>
                Delete
              </button>
            </div>
          </div>
        </div>
      )}

    </div>
  );
}

export default App;
