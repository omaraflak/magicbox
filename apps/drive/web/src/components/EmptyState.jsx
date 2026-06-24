export default function EmptyState({ volumeName }) {
  const isTrash = volumeName === 'Trash';

  return (
    <div className="empty-state">
      <div className="empty-state-icon">{isTrash ? '🗑️' : '📂'}</div>
      <h3 className="empty-state-title">{isTrash ? 'Trash is empty' : 'No files yet'}</h3>
      <p className="empty-state-text">
        {isTrash ? (
          'Deleted files and folders will appear here for 30 days before they are permanently deleted.'
        ) : (
          <>
            Drag and drop files here or click <strong>Upload</strong> to add files to{' '}
            <strong>{volumeName}</strong>
          </>
        )}
      </p>
    </div>
  );
}
