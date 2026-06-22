import React, { useState, useRef, useEffect } from 'react';
import FileCard from './FileCard';

export default function FileGrid({ 
  files, 
  volume, 
  path, 
  searchQuery, 
  viewMode, 
  onFolderClick, 
  onContextMenu,
  selectedFileNames,
  onSelectionChange,
  onMoveFiles
}) {
  const [marquee, setMarquee] = useState(null);
  const containerRef = useRef(null);
  const cardRects = useRef([]);
  const initialSelection = useRef([]);
  const lastClickedName = useRef(null);

  const filtered = (files || [])
    .filter((f) => {
      if (!searchQuery) return true;
      return f.name.toLowerCase().includes(searchQuery.toLowerCase());
    })
    .sort((a, b) => {
      if (a.is_dir && !b.is_dir) return -1;
      if (!a.is_dir && b.is_dir) return 1;
      return a.name.localeCompare(b.name);
    });

  // Global Ctrl+A / Cmd+A listener
  useEffect(() => {
    const handleGlobalKeyDown = (e) => {
      if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === 'a') {
        e.preventDefault();
        onSelectionChange(filtered.map(f => f.name));
      }
    };
    window.addEventListener('keydown', handleGlobalKeyDown);
    return () => window.removeEventListener('keydown', handleGlobalKeyDown);
  }, [filtered, onSelectionChange]);

  // Window mouse event listeners for selection marquee box dragging
  useEffect(() => {
    if (!marquee) return;

    const handleWindowMouseMove = (e) => {
      if (!containerRef.current) return;
      const rect = containerRef.current.getBoundingClientRect();
      const currentX = e.clientX - rect.left + containerRef.current.scrollLeft;
      const currentY = e.clientY - rect.top + containerRef.current.scrollTop;

      setMarquee(prev => {
        if (!prev) return null;

        const x1 = Math.min(prev.startX, currentX);
        const y1 = Math.min(prev.startY, currentY);
        const x2 = Math.max(prev.startX, currentX);
        const y2 = Math.max(prev.startY, currentY);

        const intersectedNames = [];
        cardRects.current.forEach(r => {
          const cardX1 = r.left;
          const cardY1 = r.top;
          const cardX2 = r.left + r.width;
          const cardY2 = r.top + r.height;

          const intersect = !(x2 < cardX1 || x1 > cardX2 || y2 < cardY1 || y1 > cardY2);
          if (intersect) {
            intersectedNames.push(r.name);
          }
        });

        let finalSelection = [...initialSelection.current];
        intersectedNames.forEach(name => {
          if (!finalSelection.includes(name)) {
            finalSelection.push(name);
          }
        });

        // Deselect items that are not in initial selection and are no longer intersected
        finalSelection = finalSelection.filter(name => 
          initialSelection.current.includes(name) || intersectedNames.includes(name)
        );

        onSelectionChange(finalSelection);

        return {
          ...prev,
          currentX,
          currentY
        };
      });
    };

    const handleWindowMouseUp = () => {
      setMarquee(null);
    };

    window.addEventListener('mousemove', handleWindowMouseMove);
    window.addEventListener('mouseup', handleWindowMouseUp);
    return () => {
      window.removeEventListener('mousemove', handleWindowMouseMove);
      window.removeEventListener('mouseup', handleWindowMouseUp);
    };
  }, [marquee, onSelectionChange]);

  const handleMouseDown = (e) => {
    // Start marquee select on background left-click only
    if (e.button !== 0 || e.target.closest('.file-card')) return;

    e.preventDefault();
    containerRef.current = e.currentTarget;
    const rect = e.currentTarget.getBoundingClientRect();
    const startX = e.clientX - rect.left + e.currentTarget.scrollLeft;
    const startY = e.clientY - rect.top + e.currentTarget.scrollTop;

    setMarquee({
      startX,
      startY,
      currentX: startX,
      currentY: startY,
    });

    initialSelection.current = (e.ctrlKey || e.metaKey) ? [...selectedFileNames] : [];

    // Cache card offsets relative to container by walking offsetParent chain
    const cards = e.currentTarget.querySelectorAll('.file-card');
    const tempRects = [];
    cards.forEach(card => {
      const name = card.getAttribute('data-name');
      if (name) {
        let left = card.offsetLeft;
        let top = card.offsetTop;
        let parent = card.offsetParent;
        while (parent && parent !== e.currentTarget) {
          left += parent.offsetLeft;
          top += parent.offsetTop;
          parent = parent.offsetParent;
        }
        tempRects.push({
          name,
          left,
          top,
          width: card.offsetWidth,
          height: card.offsetHeight
        });
      }
    });
    cardRects.current = tempRects;

    if (!e.ctrlKey && !e.metaKey) {
      onSelectionChange([]);
    }
  };

  const handleItemClick = (e, file) => {
    e.stopPropagation();

    let newSelection = [];
    if (e.ctrlKey || e.metaKey) {
      if (selectedFileNames.includes(file.name)) {
        newSelection = selectedFileNames.filter(name => name !== file.name);
      } else {
        newSelection = [...selectedFileNames, file.name];
      }
      lastClickedName.current = file.name;
    } else if (e.shiftKey && lastClickedName.current) {
      const namesList = filtered.map(f => f.name);
      const startIdx = namesList.indexOf(lastClickedName.current);
      const endIdx = namesList.indexOf(file.name);
      if (startIdx !== -1 && endIdx !== -1) {
        const minIdx = Math.min(startIdx, endIdx);
        const maxIdx = Math.max(startIdx, endIdx);
        const rangeNames = namesList.slice(minIdx, maxIdx + 1);
        newSelection = Array.from(new Set([...selectedFileNames, ...rangeNames]));
      }
    } else {
      newSelection = [file.name];
      lastClickedName.current = file.name;
    }

    onSelectionChange(newSelection);
  };

  const handleItemDoubleClick = (e, file) => {
    e.stopPropagation();
    if (file.is_dir && onFolderClick) {
      onFolderClick(file.name);
    }
  };

  const handleItemContextMenu = (e, file) => {
    e.preventDefault();
    e.stopPropagation();

    if (!selectedFileNames.includes(file.name)) {
      onSelectionChange([file.name]);
    }
    onContextMenu(e, file);
  };

  // Drag and Drop implementation
  const handleDragStart = (e, file) => {
    let dragFiles = [file];
    if (selectedFileNames.includes(file.name)) {
      dragFiles = filtered.filter(f => selectedFileNames.includes(f.name));
    }
    e.dataTransfer.setData('application/json', JSON.stringify(dragFiles));
    e.dataTransfer.effectAllowed = 'move';

    // Create custom drag feedback element if dragging multiple files
    if (dragFiles.length > 1) {
      const badge = document.createElement('div');
      badge.id = 'drag-selection-badge';
      badge.style.position = 'absolute';
      badge.style.top = '-1000px';
      badge.style.left = '-1000px';
      badge.style.padding = '8px 16px';
      badge.style.background = 'rgba(6, 182, 212, 0.95)';
      badge.style.border = '1px solid var(--accent-cyan)';
      badge.style.color = '#fff';
      badge.style.borderRadius = '20px';
      badge.style.fontSize = '0.875rem';
      badge.style.fontWeight = '600';
      badge.style.boxShadow = '0 4px 12px rgba(6, 182, 212, 0.3)';
      badge.style.display = 'flex';
      badge.style.alignItems = 'center';
      badge.style.gap = '8px';
      badge.style.zIndex = '999999';
      badge.innerHTML = `<span>📦</span> <span>Moving ${dragFiles.length} items</span>`;

      document.body.appendChild(badge);
      e.dataTransfer.setDragImage(badge, 20, 20);

      setTimeout(() => {
        if (badge.parentNode) {
          badge.parentNode.removeChild(badge);
        }
      }, 0);
    }

    // Add visual dragging class
    dragFiles.forEach(f => {
      const el = document.querySelector(`[data-name="${CSS.escape(f.name)}"]`);
      if (el) el.classList.add('dragging');
    });
  };

  const handleDragEnd = () => {
    const elements = document.querySelectorAll('.file-card.dragging');
    elements.forEach(el => el.classList.remove('dragging'));
  };

  // Drop zone configuration for folders
  const getFolderDropHandlers = (destFolder) => {
    return {
      onDragOver: (e) => {
        e.preventDefault();
        e.stopPropagation();
        e.currentTarget.classList.add('drag-over');
      },
      onDragLeave: (e) => {
        e.preventDefault();
        e.stopPropagation();
        e.currentTarget.classList.remove('drag-over');
      },
      onDrop: (e) => {
        e.preventDefault();
        e.stopPropagation();
        e.currentTarget.classList.remove('drag-over');
        try {
          const dragData = e.dataTransfer.getData('application/json');
          if (!dragData) return;
          const dragFiles = JSON.parse(dragData);
          if (!dragFiles || dragFiles.length === 0) return;

          // Filter out dropping onto itself
          const validFiles = dragFiles.filter(f => f.name !== destFolder.name);
          if (validFiles.length > 0 && onMoveFiles) {
            onMoveFiles(validFiles, destFolder);
          }
        } catch (err) {
          console.error('Failed to parse drag drop data:', err);
        }
      }
    };
  };

  if (filtered.length === 0 && searchQuery) {
    return (
      <div className="empty-search">
        <span className="empty-search-icon">🔍</span>
        <p>No files matching &ldquo;{searchQuery}&rdquo;</p>
      </div>
    );
  }

  return (
    <div 
      ref={containerRef}
      className={`file-grid ${viewMode === 'list' ? 'list' : ''}`}
      onMouseDown={handleMouseDown}
      style={{ position: 'relative', minHeight: '300px' }}
    >
      {marquee && (
        <div style={{
          position: 'absolute',
          left: `${Math.min(marquee.startX, marquee.currentX)}px`,
          top: `${Math.min(marquee.startY, marquee.currentY)}px`,
          width: `${Math.abs(marquee.startX - marquee.currentX)}px`,
          height: `${Math.abs(marquee.startY - marquee.currentY)}px`,
          border: '1.5px solid var(--accent-cyan)',
          background: 'rgba(6, 182, 212, 0.08)',
          pointerEvents: 'none',
          borderRadius: '2px',
          zIndex: 1000
        }} />
      )}

      {filtered.map((file, index) => {
        const isSel = selectedFileNames.includes(file.name);
        return (
          <div
            key={file.name}
            className="file-grid-item"
            style={{ animationDelay: `${index * 0.02}s` }}
          >
            <FileCard 
              file={file} 
              volume={volume} 
              path={path} 
              selected={isSel}
              onClick={(e) => handleItemClick(e, file)}
              onDoubleClick={(e) => handleItemDoubleClick(e, file)}
              onContextMenu={(e) => handleItemContextMenu(e, file)}
              onDragStart={(e) => handleDragStart(e, file)}
              onDragEnd={handleDragEnd}
              dropHandlers={file.is_dir ? getFolderDropHandlers(file) : {}}
              viewMode={viewMode}
            />
          </div>
        );
      })}
    </div>
  );
}
