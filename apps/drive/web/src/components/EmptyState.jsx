export default function EmptyState({ volumeName }) {
  return (
    <div className="empty-state">
      <div className="empty-state-icon">📂</div>
      <h3 className="empty-state-title">No files yet</h3>
      <p className="empty-state-text">
        Drag and drop files here or click <strong>Upload</strong> to add files to{' '}
        <strong>{volumeName}</strong>
      </p>
    </div>
  );
}
