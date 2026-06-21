export function formatFileSize(bytes) {
  if (bytes === 0) return '0 B';
  const units = ['B', 'KB', 'MB', 'GB', 'TB'];
  const i = Math.floor(Math.log(bytes) / Math.log(1024));
  const size = bytes / Math.pow(1024, i);
  return `${size.toFixed(i === 0 ? 0 : 1)} ${units[i]}`;
}

export function formatDate(isoString) {
  if (!isoString) return '';
  const date = new Date(isoString);
  return date.toLocaleDateString('en-US', {
    month: 'short',
    day: 'numeric',
    year: 'numeric',
  });
}

const EXT_MAP = {
  image: {
    extensions: ['jpg', 'jpeg', 'png', 'gif', 'webp', 'svg', 'bmp', 'ico', 'tiff', 'heic', 'heif', 'avif'],
    icon: '📷',
  },
  document: {
    extensions: ['pdf', 'doc', 'docx', 'xls', 'xlsx', 'ppt', 'pptx', 'odt', 'ods', 'odp', 'rtf'],
    icon: '📄',
  },
  audio: {
    extensions: ['mp3', 'wav', 'flac', 'aac', 'ogg', 'wma', 'm4a', 'opus'],
    icon: '🎵',
  },
  video: {
    extensions: ['mp4', 'mkv', 'avi', 'mov', 'wmv', 'flv', 'webm', 'm4v'],
    icon: '🎬',
  },
  archive: {
    extensions: ['zip', 'rar', '7z', 'tar', 'gz', 'bz2', 'xz', 'zst'],
    icon: '📦',
  },
  config: {
    extensions: ['json', 'yaml', 'yml', 'toml', 'ini', 'cfg', 'conf', 'env', 'xml'],
    icon: '⚙️',
  },
  text: {
    extensions: ['txt', 'md', 'log', 'csv', 'tsv', 'rst'],
    icon: '📝',
  },
};

function getExtension(filename) {
  const parts = filename.split('.');
  if (parts.length < 2) return '';
  return parts[parts.length - 1].toLowerCase();
}

export function getFileIcon(filename, isDir) {
  if (isDir) return '📁';
  const ext = getExtension(filename);
  for (const category of Object.values(EXT_MAP)) {
    if (category.extensions.includes(ext)) return category.icon;
  }
  return '📄';
}

export function getFileCategory(filename) {
  const ext = getExtension(filename);
  for (const [category, data] of Object.entries(EXT_MAP)) {
    if (data.extensions.includes(ext)) return category;
  }
  return 'other';
}
