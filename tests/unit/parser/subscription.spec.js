import { nodeFingerprint, mergeSubscriptionNodes, subscriptionDefaults } from '@/store/uif/parser/subscription'

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

  it('ships safe defaults for old subscriptions', () => {
    expect(subscriptionDefaults.policy.update_mode).toBe('merge')
    expect(subscriptionDefaults.probe.min_keep).toBeGreaterThan(0)
  })
})
