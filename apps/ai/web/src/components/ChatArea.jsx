import React, { useState, useEffect, useRef } from 'react';
import { SettingsIcon, SendIcon, UserIcon, BotIcon, TrashIcon, ForkIcon, RefreshIcon, EditIcon } from './Icons';
import { getConversation, getPresets, updateTitle, updateParams, API_BASE, deleteConversation, forkConversation } from '../api';
import ReactMarkdown from 'react-markdown';
import remarkMath from 'remark-math';
import rehypeKatex from 'rehype-katex';
import 'katex/dist/katex.min.css';
import ConfirmationModal from './ConfirmationModal';

export default function ChatArea({ activeId, activeTitle, activeParams, onTitleChange, toggleParams, onRefreshParams, onChatCreated, onDeleteChat, onForkCreated }) {
  const [messages, setMessages] = useState([]);
  const [input, setInput] = useState('');
  const [loading, setLoading] = useState(false);
  const [localTitle, setLocalTitle] = useState('');
  const [presets, setPresets] = useState([]);
  const [selectedPreset, setSelectedPreset] = useState(null);
  const [showDeleteModal, setShowDeleteModal] = useState(false);
  const messagesEndRef = useRef(null);
  const containerRef = useRef(null);
  const inputRef = useRef(null);
  const isCreatingChatRef = useRef(false);

  useEffect(() => {
    if (activeId && activeId !== 'new') {
      if (isCreatingChatRef.current) {
        isCreatingChatRef.current = false;
        return;
      }
      getConversation(activeId).then(res => {
        setMessages(res.messages || []);
        setLocalTitle(res.title || 'Untitled Chat');
      }).catch(err => {
        setMessages([]);
        setLocalTitle('');
      });
    } else {
      setMessages([]);
      setLocalTitle('');
    }
    setSelectedPreset(null);
  }, [activeId]);

  useEffect(() => {
    setLocalTitle(activeTitle || '');
  }, [activeTitle]);

  useEffect(() => {
    if (activeId && messages.length === 0) {
      getPresets().then(res => setPresets(res || []));
    }
  }, [activeId, messages.length]);

  useEffect(() => {
    if (containerRef.current) {
      containerRef.current.scrollTop = containerRef.current.scrollHeight;
    }
  }, [messages]);

  const handleTitleBlur = async () => {
    if (!activeId || !localTitle.trim() || localTitle === activeTitle) return;
    await updateTitle(activeId, localTitle.trim());
    if (onTitleChange) onTitleChange(activeId, localTitle.trim());
  };

  const handleTitleKeyDown = (e) => {
    if (e.key === 'Enter') {
      e.target.blur();
    }
  };

  const handleApplyPreset = async (preset) => {
    setSelectedPreset(preset);
    if (activeId && activeId !== 'new') {
      await updateParams(activeId, preset ? preset.params : '{}');
      if (onRefreshParams) onRefreshParams();
    }
  };

  const handleDelete = async () => {
    setShowDeleteModal(true);
  };

  const handleFork = async (messageId) => {
    if (!activeId || activeId === 'new') return;
    const newConv = await forkConversation(activeId, messageId);
    if (newConv && newConv.id && onForkCreated) {
      onForkCreated(newConv);
    }
  };

  const handleEdit = async () => {
    if (!activeId || activeId === 'new') return;
    try {
      const response = await fetch(`${API_BASE}/conversations/${activeId}/undo-last-turn`, {
        method: 'POST'
      });
      if (response.ok) {
        const res = await response.json();
        setInput(res.content);
        setMessages(prev => {
          const copy = [...prev];
          if (copy.length > 0 && copy[copy.length - 1].role === 'model') {
            copy.pop();
          }
          if (copy.length > 0 && copy[copy.length - 1].role === 'user') {
            copy.pop();
          }
          return copy;
        });
        setTimeout(() => {
          if (inputRef.current) {
            inputRef.current.focus();
          }
        }, 50);
      }
    } catch(e) {
      console.error('Edit error:', e);
    }
  };

  const handleRetry = async () => {
    if (!activeId || activeId === 'new' || loading) return;
    setLoading(true);

    setMessages(prev => {
      const copy = [...prev];
      if (copy.length > 0 && copy[copy.length - 1].role === 'model') {
        copy.pop();
      }
      return copy;
    });

    try {
      const response = await fetch(`${API_BASE}/conversations/${activeId}/regenerate`, {
        method: 'POST'
      });

      if (!response.ok) {
        throw new Error('Failed to regenerate response');
      }

      const reader = response.body.getReader();
      const decoder = new TextDecoder();
      
      setMessages(prev => [...prev, { role: 'model', content: '' }]);

      while (true) {
        const { done, value } = await reader.read();
        if (done) break;

        const chunk = decoder.decode(value);
        const lines = chunk.split('\n');
        
        for (const line of lines) {
          if (line.startsWith('data: ')) {
            try {
              const data = JSON.parse(line.slice(6));
              if (data.content) {
                setMessages(prev => {
                  const copy = [...prev];
                  const last = copy[copy.length - 1];
                  if (last && last.role === 'model') {
                    last.content += data.content;
                  }
                  return copy;
                });
              }
            } catch(e) {}
          }
        }
      }
    } catch(err) {
      console.error(err);
    } finally {
      setLoading(false);
      getConversation(activeId).then(res => {
        setMessages(res.messages || []);
      });
    }
  };

  const confirmDelete = async () => {
    if (!activeId || activeId === 'new') return;
    await deleteConversation(activeId);
    setShowDeleteModal(false);
    if (onDeleteChat) onDeleteChat(activeId);
  };

  const handleSubmit = async (e) => {
    e.preventDefault();
    if (!input.trim() || !activeId || loading) return;

    const userMsg = input.trim();
    setInput('');
    setMessages(prev => [...prev, { role: 'user', content: userMsg }]);
    setLoading(true);

    let targetId = activeId;
    try {
      if (activeId === 'new') {
        isCreatingChatRef.current = true;
        const { createConversation } = await import('../api');
        const res = await createConversation();
        if (!res || !res.id) {
          setLoading(false);
          return;
        }
        targetId = res.id;
        if (activeParams && Object.keys(activeParams).length > 0) {
          await updateParams(targetId, JSON.stringify(activeParams));
        } else if (selectedPreset) {
          await updateParams(targetId, selectedPreset.params);
        }
        if (onChatCreated) onChatCreated(res);
      }

      // Add empty model message immediately so the typing indicator shows
      setMessages(prev => [...prev, { role: 'model', content: '' }]);

      const res = await fetch(`${API_BASE}/conversations/${targetId}/chat`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ message: userMsg })
      });

      if (!res.body) {
        setMessages(prev => prev.filter((m, i) => !(i === prev.length - 1 && m.role === 'model' && m.content === '')));
        setLoading(false);
        return;
      }

      const reader = res.body.getReader();
      const decoder = new TextDecoder();
      let modelContent = '';

      while (true) {
        const { done, value } = await reader.read();
        if (done) break;
        
        const chunk = decoder.decode(value, { stream: true });
        const lines = chunk.split('\n').filter(l => l.trim() !== '');
        
        for (const line of lines) {
          if (line.startsWith('data: ')) {
            try {
              const data = JSON.parse(line.slice(6));
              if (data.content) {
                modelContent += data.content;
                setMessages(prev => {
                  const newMsgs = [...prev];
                  newMsgs[newMsgs.length - 1] = { role: 'model', content: modelContent };
                  return newMsgs;
                });
              }
            } catch(e) {}
          }
        }
      }
    } catch (err) {
      console.error(err);
    } finally {
      setLoading(false);
      if (targetId && targetId !== 'new') {
        getConversation(targetId).then(res => {
          setMessages(res.messages || []);
        });
      }
    }
  };

  const handleKeyDown = (e) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      handleSubmit(e);
    }
  };

  if (!activeId) {
    return (
      <div className="chat-area empty-chat">
        <div className="empty-message-box glass">
          <BotIcon size={48} className="empty-icon" />
          <h2>Magic AI</h2>
          <p>Select a conversation or start a new one.</p>
        </div>
      </div>
    );
  }

  return (
    <div className="chat-area">
      <div className="chat-header">
        <input 
          className="title-input" 
          value={localTitle} 
          onChange={e => setLocalTitle(e.target.value)}
          onBlur={handleTitleBlur}
          onKeyDown={handleTitleKeyDown}
          placeholder="Chat Title..."
        />
        <div className="chat-header-actions">
          {activeId !== 'new' && (
            <button onClick={handleDelete} className="btn-icon btn-delete" title="Delete Chat">
              <TrashIcon size={20} />
            </button>
          )}
          <button onClick={toggleParams} className="btn-icon">
            <SettingsIcon size={20} />
          </button>
        </div>
      </div>
      <div className="messages-container" ref={containerRef}>
        {messages.length === 0 && (
          <div className="preset-picker-container">
            <h4>Start with a preset</h4>
            <div className="preset-cards">
              <div 
                className={"preset-card glass" + (selectedPreset && selectedPreset.id === 'default' ? " active-preset" : (!selectedPreset ? " active-preset" : ""))}
                onClick={() => handleApplyPreset({ id: 'default', params: '{}' })}
              >
                <h5>Default</h5>
                <p>No special instructions</p>
              </div>
              {presets.map(p => {
                let desc = '';
                try {
                  const parsed = JSON.parse(p.params);
                  desc = parsed.description || 'Custom preset';
                } catch(e) { desc = 'Custom preset'; }
                return (
                  <div 
                    key={p.id} 
                    className={"preset-card glass" + (selectedPreset && selectedPreset.id === p.id ? " active-preset" : "")} 
                    onClick={() => handleApplyPreset(p)}
                  >
                    <h5>{p.name}</h5>
                    <p>{desc}</p>
                  </div>
                );
              })}
            </div>
          </div>
        )}
        {messages.map((m, i) => {
          const isLatestUser = m.role === 'user' && m.id && (
            i === messages.length - 1 || 
            (i === messages.length - 2 && messages[messages.length - 1].role === 'model')
          );
          
          return (
            <div key={i} className={`message-row ${m.role}`}>
              <div className="message-avatar">
                {m.role === 'user' ? <UserIcon size={20} /> : <BotIcon size={20} />}
              </div>
              <div className="message-content">
                <div className="message-header">
                  <span className="message-sender">{m.role === 'user' ? 'You' : 'Magic AI'}</span>
                </div>
                <div className="message-bubble">
                  {m.content ? (
                    <ReactMarkdown remarkPlugins={[remarkMath]} rehypePlugins={[rehypeKatex]}>
                      {m.content}
                    </ReactMarkdown>
                  ) : (
                    m.role === 'model' && loading && i === messages.length - 1 && (
                      <div className="typing-indicator">
                        <span></span>
                        <span></span>
                        <span></span>
                      </div>
                    )
                  )}
                </div>
                {isLatestUser && (
                  <div className="message-actions">
                    <button className="btn-action-icon" onClick={handleEdit} title="Edit message">
                      <EditIcon size={14} />
                    </button>
                  </div>
                )}
                {m.role === 'model' && m.id && (
                  <div className="message-actions">
                    {i === messages.length - 1 && (
                      <button className="btn-action-icon" onClick={handleRetry} title="Regenerate response">
                        <RefreshIcon size={14} />
                      </button>
                    )}
                    <button className="btn-action-icon" onClick={() => handleFork(m.id)} title="Fork conversation from here">
                      <ForkIcon size={14} />
                    </button>
                  </div>
                )}
              </div>
            </div>
          );
        })}
        <div ref={messagesEndRef} />
      </div>

      <div className="input-container">
        <form className="input-form glass" onSubmit={handleSubmit}>
          <textarea
            ref={inputRef}
            value={input}
            onChange={e => setInput(e.target.value)}
            onKeyDown={handleKeyDown}
            placeholder="Type your message..."
            rows={1}
            disabled={loading}
          />
          <button type="submit" className="btn-send" disabled={!input.trim() || loading}>
            <SendIcon size={20} />
          </button>
        </form>
      </div>

      <ConfirmationModal 
        isOpen={showDeleteModal}
        title="Delete Chat"
        message="Are you sure you want to delete this conversation? This action cannot be undone."
        onConfirm={confirmDelete}
        onCancel={() => setShowDeleteModal(false)}
      />
    </div>
  );
}
