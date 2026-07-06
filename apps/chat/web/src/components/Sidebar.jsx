import React from 'react';
import { IconPlus, IconSettings } from './Icons';

export function Sidebar({
  profile,
  searchQuery,
  setSearchQuery,
  filteredConvs,
  selectedConv,
  setSelectedConv,
  showSettings,
  setShowSettings,
  setShowModal,
  getConversationName,
  formatTime
}) {
  return (
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
            onClick={() => {
              setSelectedConv(c);
              setShowSettings(false);
            }}
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

      <div className="sidebar-footer">
        <button 
          className={`settings-btn ${showSettings ? 'active' : ''}`} 
          onClick={() => {
            setShowSettings(true);
            setSelectedConv(null);
          }}
        >
          <IconSettings /> Settings
        </button>
      </div>
    </div>
  );
}
