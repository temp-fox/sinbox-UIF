import { nodeFingerprint, mergeSubscriptionNodes, subscriptionDefaults, applyProbeResult, stableSubscriptionID, normalizeSubscriptions } from '@/store/uif/parser/subscription'

describe('subscription state management', () => {
  const node = (address) => ({
    protocol: 'vless',
    tag: address,
    transport: { address, port: 443, protocol: 'tcp', tls_type: 'tls', tls: { server_name: address } },
    setting: { uuid: 'stable-id' },
  })

  it('fingerprint ignores local runtime fields', () => {
    const first = node('example.com')
    const second = { ...first, tag: 'renamed', id: 'new-id', enabled: true, delay: 123 }
    expect(nodeFingerprint(first)).toBe(nodeFingerprint(second))
  })

  it('merge preserves existing node state and appends new nodes', () => {
    const old = { ...node('a.example'), id: 'old-id', enabled: true, delay: 88 }
    const next = mergeSubscriptionNodes([old], [node('a.example'), node('b.example')], 'merge')
    expect(next).toHaveLength(2)
    expect(next[0].id).toBe('old-id')
    expect(next[0].enabled).toBe(true)
    expect(next[0].delay).toBe(88)
  })

  it('replace removes nodes missing from source', () => {
    const next = mergeSubscriptionNodes([node('old.example')], [node('new.example')], 'replace')
    expect(next).toHaveLength(1)
    expect(next[0].transport.address).toBe('new.example')
  })

  it('applies threshold and protects minimum keep count', () => {
    const nodes = [
      { enabled: true, quarantined: false, consecutive_failures: 0 },
      { enabled: true, quarantined: false, consecutive_failures: 0 },
    ]
    applyProbeResult(nodes, 0, { success: true, delay: 500 }, {
      threshold_ms: 200, max_consecutive_failures: 1, failure_action: 'delete', min_keep: 1,
    })
    expect(nodes[0].quarantined).toBe(true)
    expect(nodes[0].enabled).toBe(false)
    expect(nodes[1].enabled).toBe(true)
  })

  it('uses the same UTF-8 stable ID as the backend', () => {
    expect(stableSubscriptionID({ tag: '中文', data: 'https://example.test/sub' }, 0)).toBe('legacy-subscription-520efc1b')
  })

  it('fills IDs and policy/probe defaults for legacy subscriptions', () => {
    const subscriptions = [{ tag: 'legacy', data: 'https://legacy.example/list' }]
    expect(normalizeSubscriptions(subscriptions)).toBe(true)
    expect(subscriptions[0].id).toBe('legacy-subscription-30406cf9')
    expect(subscriptions[0].policy.update_mode).toBe('merge')
    expect(subscriptions[0].probe.default_url).toBe(subscriptionDefaults.probe.default_url)
    expect(normalizeSubscriptions(subscriptions)).toBe(false)
  })
  it('ships safe defaults for old subscriptions', () => {
    expect(subscriptionDefaults.policy.update_mode).toBe('merge')
    expect(subscriptionDefaults.probe.min_keep).toBeGreaterThan(0)
  })
})
