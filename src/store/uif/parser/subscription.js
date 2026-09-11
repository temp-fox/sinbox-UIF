const isObject = (value) => value !== null && typeof value === 'object'

const sortValue = (value, omit = new Set()) => {
  if (Array.isArray(value)) return value.map((item) => sortValue(item, omit))
  if (!isObject(value)) return value
  return Object.keys(value).filter((key) => !omit.has(key)).sort().reduce((result, key) => {
    result[key] = sortValue(value[key], omit)
    return result
  }, {})
}

const hashString = (text) => {
  let a = 0x811c9dc5
  let b = 0x9e3779b9
  for (let i = 0; i < text.length; i += 1) {
    const code = text.charCodeAt(i)
    a ^= code
    a = Math.imul(a, 0x01000193)
    b ^= code + i
    b = Math.imul(b, 0x85ebca6b)
  }
  return `${(a >>> 0).toString(16).padStart(8, '0')}${(b >>> 0).toString(16).padStart(8, '0')}`
}

export const nodeFingerprint = (node) => {
  const identity = sortValue(node, new Set([
    'id', 'core_tag', 'tag', 'enabled', 'delay', 'fingerprint',
    'last_seen_at', 'last_probe_at', 'last_probe_delay_ms',
    'last_probe_status', 'consecutive_failures', 'quarantined',
    'stale_runs', 'probe_error',
  ]))
  return `fnv1a:${hashString(JSON.stringify(identity))}`
}

export const normalizeNodeState = (node, previous = {}) => ({
  ...node,
  id: previous.id || node.id,
  fingerprint: node.fingerprint || previous.fingerprint || nodeFingerprint(node),
  enabled: previous.enabled === undefined ? Boolean(node.enabled) : previous.enabled,
  delay: previous.delay === undefined ? (node.delay || '') : previous.delay,
  core_tag: previous.core_tag || '',
  last_seen_at: Date.now(),
  last_probe_at: previous.last_probe_at || 0,
  last_probe_delay_ms: previous.last_probe_delay_ms === undefined ? -1 : previous.last_probe_delay_ms,
  last_probe_status: previous.last_probe_status || 'unknown',
  consecutive_failures: previous.consecutive_failures || 0,
  quarantined: previous.quarantined || false,
  stale_runs: 0,
  probe_error: previous.probe_error || '',
})

export const mergeSubscriptionNodes = (oldNodes = [], newNodes = [], mode = 'merge') => {
  const oldByFingerprint = new Map()
  for (const node of oldNodes) oldByFingerprint.set(node.fingerprint || nodeFingerprint(node), node)
  const seen = new Set()
  const merged = []
  for (const node of newNodes) {
    const fingerprint = node.fingerprint || nodeFingerprint(node)
    if (seen.has(fingerprint)) continue
    seen.add(fingerprint)
    merged.push(normalizeNodeState(node, oldByFingerprint.get(fingerprint)))
  }
  if (mode === 'merge') {
    for (const node of oldNodes) {
      const fingerprint = node.fingerprint || nodeFingerprint(node)
      if (!seen.has(fingerprint)) merged.push({ ...node, fingerprint, stale_runs: (node.stale_runs || 0) + 1 })
    }
  }
  return merged
}

export const subscriptionDefaults = {
  schema_version: 2,
  policy: {
    update_enabled: false,
    update_interval_sec: 18000,
    update_mode: 'merge',
    startup_update: false,
    remove_missing: false,
    missing_grace_runs: 3,
  },
  probe: {
    enabled: false,
    interval_sec: 18000,
    target_mode: 'default',
    default_url: 'https://www.gstatic.com/generate_204',
    timeout_ms: 10000,
    concurrency: 4,
    threshold_ms: 200,
    max_consecutive_failures: 3,
    failure_action: 'quarantine',
    min_keep: 2,
    target_policy: 'any',
  },
  last_update_at: 0,
  last_update_status: 'unknown',
  last_update_error: '',
  last_probe_at: 0,
  last_probe_status: 'unknown',
}
