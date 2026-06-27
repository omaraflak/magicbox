// Merge conversations by ID, deduplicating
export function mergeById(existing, incoming) {
  const map = {};
  existing.forEach(c => map[c.id] = c);
  incoming.forEach(c => map[c.id] = c);
  return Object.values(map);
}

// Safely parse params JSON (handles double-encoded strings)
export function parseParams(raw) {
  if (!raw) return {};
  try {
    let parsed = typeof raw === 'string' ? JSON.parse(raw) : raw;
    if (typeof parsed === 'string') {
      try { parsed = JSON.parse(parsed); } catch(e) { return {}; }
    }
    return parsed;
  } catch(e) {
    return {};
  }
}

// Default AI params
export const DEFAULT_PARAMS = {
  system_prompt: '',
  temperature: 1.0,
  top_k: 40,
  top_p: 1.0
};
