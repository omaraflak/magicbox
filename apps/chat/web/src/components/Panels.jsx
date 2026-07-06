import React from 'react';
import { IconFile, IconDownload } from './Icons';

export function MediaPanel({ 
  showMediaPanel, 
  setShowMediaPanel, 
  sharedMedia, 
  handleMediaScroll,
  isImage,
  isVideo,
  formatTime
}) {
  if (!showMediaPanel) return null;

  const visualMedia = sharedMedia.filter(m => isImage(m.attachment_type) || isVideo(m.attachment_type));
  const docFiles = sharedMedia.filter(m => !isImage(m.attachment_type) && !isVideo(m.attachment_type));

  return (
    <div className="media-panel">
      <div className="media-panel-header">
        <span className="media-panel-title">Shared Media</span>
        <button className="action-btn" onClick={() => setShowMediaPanel(false)}>✕</button>
      </div>
      <div className="media-panel-body" onScroll={handleMediaScroll}>
        {sharedMedia.length === 0 ? (
          <div style={{ textAlign: 'center', color: 'var(--text-mute)', padding: '32px' }}>
            No media or files shared in this chat.
          </div>
        ) : (
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
        )}
      </div>
    </div>
  );
}

export function ParticipantsPanel({
  showParticipantsPanel,
  setShowParticipantsPanel,
  selectedConv,
  profile,
  contacts,
  addingContacts,
  handleAddContact,
  mediaPanelWidth,
  startResizingMedia
}) {
  if (!showParticipantsPanel) return null;

  return (
    <div className="media-panel" style={{ width: `${mediaPanelWidth}px`, position: 'relative' }}>
      <div 
        className="resize-handle-left"
        onMouseDown={startResizingMedia}
      />
      <div className="media-panel-header">
        <span className="media-panel-title">Participants ({selectedConv.participants.length})</span>
        <button className="action-btn" onClick={() => setShowParticipantsPanel(false)}>✕</button>
      </div>
      <div className="media-panel-body">
        <div style={{ display: 'flex', flexDirection: 'column', gap: '12px' }}>
          {selectedConv.participants.map(p => {
            const isMe = profile && p.user_id === profile.user_id;
            const isContact = contacts.some(c => c.target_user_id === p.user_id && (c.status === 'active' || !c.status));
            const isPending = contacts.some(c => c.target_user_id === p.user_id && c.status === 'pending');
            return (
              <div key={p.user_id} className="participant-item">
                <div className="participant-info">
                  <div className="avatar" style={{ width: '32px', height: '32px', fontSize: '12px' }}>
                    {p.display_name.substring(0, 2)}
                  </div>
                  <div className="participant-text">
                    <span className="participant-name">
                      {p.display_name} {isMe && "(You)"}
                    </span>
                    <span className="participant-id">
                      ID: {p.user_id}
                    </span>
                  </div>
                </div>
                {!isMe && !isContact && p.invite_link && (
                  <button
                    className={`banner-btn ${isPending ? 'sent' : ''}`}
                    style={{ padding: '4px 8px', fontSize: '11.5px' }}
                    disabled={isPending || addingContacts[p.user_id]}
                    onClick={() => handleAddContact(p)}
                  >
                    {addingContacts[p.user_id] ? 'Adding...' : isPending ? 'Pending' : '+ Add'}
                  </button>
                )}
              </div>
            );
          })}
        </div>
      </div>
    </div>
  );
}
