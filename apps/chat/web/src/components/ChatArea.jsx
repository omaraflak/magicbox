import React from 'react';
import { 
  IconDots, IconEdit, IconSearch, IconImage, IconTrash, 
  IconPaperclip, IconSend, IconFile, IconDownload 
} from './Icons';

export function ChatArea({
  selectedConv,
  profile,
  contacts,
  getConversationName,
  showMenu,
  setShowMenu,
  setShowRenameModal,
  setRenameInput,
  isSearchingMessages,
  setIsSearchingMessages,
  messageSearchQuery,
  setMessageSearchQuery,
  setShowMediaPanel,
  setShowParticipantsPanel,
  setSharedMedia,
  setHasMoreMedia,
  fetchSharedMedia,
  setShowDeleteModal,
  sentRequests,
  addingContacts,
  handleAddContact,
  containerRef,
  handleScroll,
  searchResults,
  messages,
  isImage,
  isVideo,
  formatTime,
  messagesEndRef,
  attachment,
  setAttachment,
  fileInputRef,
  handleAttachmentChange,
  messageText,
  setMessageText,
  handleSendMessage
}) {

  const getMissingContacts = () => {
    if (!selectedConv || !profile) return [];
    return selectedConv.participants.filter(p => 
      p.user_id !== profile.user_id && 
      !contacts.some(c => c.target_user_id === p.user_id && (c.status === 'active' || !c.status))
    );
  };

  const missingContacts = getMissingContacts();

  return (
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
              {selectedConv.participants.length > 2 && (
                <button 
                  className="dropdown-item" 
                  onClick={() => {
                    setShowRenameModal(true);
                    setRenameInput(getConversationName(selectedConv));
                    setShowMenu(false);
                  }}
                >
                  <IconEdit /> Rename Chat
                </button>
              )}
              <button 
                className="dropdown-item" 
                onClick={() => {
                  setIsSearchingMessages(true);
                  setMessageSearchQuery('');
                  setShowMenu(false);
                }}
              >
                <IconSearch /> Search Chat Text
              </button>
              <button 
                className="dropdown-item" 
                onClick={() => {
                  setShowMediaPanel(true);
                  setShowParticipantsPanel(false);
                  setSharedMedia([]);
                  setHasMoreMedia(true);
                  fetchSharedMedia(selectedConv.id);
                  setShowMenu(false);
                }}
              >
                <IconImage /> View Shared Media
              </button>
              <button 
                className="dropdown-item" 
                onClick={() => {
                  setShowParticipantsPanel(true);
                  setShowMediaPanel(false);
                  setShowMenu(false);
                }}
              >
                <span style={{ marginRight: '8px', fontSize: '15.5px' }}>👥</span> View Participants
              </button>
              <button 
                className="dropdown-item danger" 
                onClick={() => {
                  setShowDeleteModal(true);
                  setShowMenu(false);
                }}
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

      {missingContacts.length > 0 && (
        <div className="missing-contacts-banner">
          <div className="banner-header">
            <span className="banner-icon">⚠️</span>
            <span className="banner-text">
              Some participants in this group are not in your contacts:
            </span>
          </div>
          <div className="banner-list">
            {missingContacts.map(p => {
              const contact = contacts.find(c => c.target_user_id === p.user_id);
              const isPending = contact && contact.status === 'pending';
              const isSent = sentRequests[p.user_id] || isPending;
              const isLoading = addingContacts[p.user_id];
              return (
                <div key={p.user_id} className="banner-list-item">
                  <span className="banner-item-name">{p.display_name}</span>
                  <button
                    className={`banner-btn ${isSent ? 'sent' : ''}`}
                    disabled={isSent || isLoading}
                    onClick={() => handleAddContact(p)}
                  >
                    {isLoading ? 'Adding...' : isSent ? 'Pending' : 'Add Contact'}
                  </button>
                </div>
              );
            })}
          </div>
        </div>
      )}

      {/* Messages container */}
      <div className="messages-container" ref={containerRef} onScroll={handleScroll}>
        {(() => {
          const displayList = isSearchingMessages && messageSearchQuery ? searchResults : messages;

          if (displayList.length === 0) {
            return (
              <div className="messages-empty-state">
                {isSearchingMessages ? "No matching messages found" : "No messages yet. Send a message to start the conversation!"}
              </div>
            );
          }

          return displayList.map(m => {
            const isMe = m.sender_id === profile?.user_id;

            if (m.is_system) {
              return (
                <div key={m.id} className="system-message animate-fade-in">
                  <span className="system-message-text">{m.text}</span>
                </div>
              );
            }

            return (
              <div key={m.id} className={`message-bubble-wrapper ${isMe ? 'sent' : 'received'} animate-fade-in`}>
                <div className={`message-bubble ${isMe ? 'sent' : 'received'}`}>
                  {!isMe && selectedConv.participants.length > 2 && (
                    <span className="message-sender">{m.sender_name}</span>
                  )}
                  
                  {m.attachment_name && (
                    <div className="message-file-card">
                      <span className="message-file-icon"><IconFile /></span>
                      <div className="message-file-info">
                        <div className="message-file-name" title={m.attachment_name}>{m.attachment_name}</div>
                        <div className="message-file-size">Attachment</div>
                      </div>
                      <a href={`api/attachments/${m.conversation_id}/${encodeURIComponent(m.attachment_name)}`} download={m.attachment_name} className="message-file-download" title="Download">
                        <IconDownload />
                      </a>
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
  );
}
