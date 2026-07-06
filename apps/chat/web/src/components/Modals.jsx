import React from 'react';
import { IconCheck } from './Icons';

export function NewChatModal({
  showModal,
  setShowModal,
  selectedContactIDs,
  setSelectedContactIDs,
  newConvName,
  setNewConvName,
  contacts,
  handleCreateConversation
}) {
  if (!showModal) return null;

  const toggleContactSelection = (id) => {
    if (selectedContactIDs.includes(id)) {
      setSelectedContactIDs(selectedContactIDs.filter(x => x !== id));
    } else {
      setSelectedContactIDs([...selectedContactIDs, id]);
    }
  };

  return (
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
            disabled={selectedContactIDs.length === 0}
          >
            Create
          </button>
        </div>
      </div>
    </div>
  );
}

export function RenameChatModal({
  showRenameModal,
  setShowRenameModal,
  renameInput,
  setRenameInput,
  handleRenameConversation
}) {
  if (!showRenameModal) return null;

  return (
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
  );
}

export function DeleteChatModal({
  showDeleteModal,
  setShowDeleteModal,
  selectedConv,
  getConversationName,
  handleDeleteConversation
}) {
  if (!showDeleteModal) return null;

  return (
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
  );
}
