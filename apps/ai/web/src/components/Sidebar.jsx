import React from 'react';
import { PlusIcon, SettingsIcon } from './Icons';

function SidebarNode({ node, activeId, onSelect, getChildren, depth = 0 }) {
  const children = getChildren(node.id);
  const isActive = activeId === node.id;
  
  const hasActiveDescendant = (id) => {
    const directChildren = getChildren(id);
    for (const child of directChildren) {
      if (child.id === activeId || hasActiveDescendant(child.id)) {
        return true;
      }
    }
    return false;
  };

  const isExpanded = isActive || hasActiveDescendant(node.id);

  return (
    <div className="conv-tree-node">
      <div 
        className={`conv-item ${isActive ? 'active' : ''}`}
        style={{ paddingLeft: `${16 + depth * 12}px` }}
        onClick={() => onSelect(node.id)}
      >
        <span className="conv-title">{node.title || 'Untitled Chat'}</span>
      </div>
      {isExpanded && children.length > 0 && (
        <div className="conv-node-children">
          {children.map(child => (
            <SidebarNode 
              key={child.id} 
              node={child} 
              activeId={activeId} 
              onSelect={onSelect} 
              getChildren={getChildren} 
              depth={depth + 1}
            />
          ))}
        </div>
      )}
    </div>
  );
}

export default function Sidebar({ conversations, activeId, onSelect, onNew, openSettings, hasMoreRoots, onLoadMore }) {
  const handleNew = () => {
    onNew();
  };

  const getChildren = (parentId) => {
    return conversations.filter(c => c.parent_id === parentId);
  };

  const rootConversations = conversations.filter(c => !c.parent_id);

  return (
    <div className="sidebar glass-panel">
      <div className="sidebar-header">
        <h1 className="brand-title">Magic <span>AI</span></h1>
        <button className="btn-new" onClick={handleNew}>
          <PlusIcon size={18} />
          <span>New Chat</span>
        </button>
      </div>
      
      <div className="conversations-list">
        <div className="list-section-header">Conversations</div>
        {rootConversations.map(c => (
          <SidebarNode 
            key={c.id} 
            node={c} 
            activeId={activeId} 
            onSelect={onSelect} 
            getChildren={getChildren} 
          />
        ))}
        {hasMoreRoots && (
          <button className="btn-load-more" onClick={onLoadMore}>
            Load More
          </button>
        )}
        {conversations.length === 0 && (
          <div className="empty-sidebar">No conversations yet</div>
        )}
      </div>

      <div className="sidebar-footer">
        <button className="btn-icon-text" onClick={openSettings}>
          <SettingsIcon size={18} />
          <span>Settings</span>
        </button>
      </div>
    </div>
  );
}
