import { BuildTestNodeTemplate } from '@/uif/template/speedtest.js'

const ssNode = (tag, extra = {}) => ({
  protocol: 'shadowsocks',
  tag,
  enabled: true,
  transport: {
    protocol: 'tcp',
    tls_type: 'none',
    tls: { enabled: false },
    address: '1.2.3.4',
    port: 443,
  },
  setting: { method: 'aes-128-gcm', password: 'p' },
  ...extra,
})

describe('链式测速：订阅自动选优作为前置代理', () => {
  it('被测节点 detour 指向订阅 urltest 组，成员节点加 detour: 前缀且自身无 detour', () => {
    const target = ssNode('target', {
      dial: { detour: { id: ['subscription:sub1'], tag: '' } },
    })
    const members = [
      ssNode('m1'),
      ssNode('m2'),
    ]
    const resolveDetour = (id) => {
      if (id === 'subscription:sub1') {
        return {
          kind: 'subscription',
          subscriptionId: 'sub1',
          groupTag: 'sub::sub1::urltest',
          nodes: members,
          urlTest: {
            url: 'https://www.gstatic.com/generate_204',
            interval: '5m',
            idle_timeout: '5m',
            tolerance: 50,
          },
        }
      }
      return null
    }

    const config = BuildTestNodeTemplate({ 0: target }, false, resolveDetour)

    const byTag = (t) => config.outbounds.find((o) => o.tag === t)
    const targetOut = byTag('target')
    expect(targetOut).toBeTruthy()
    // 被测节点 detour 指向订阅 urltest 组，而不是直连
    expect(targetOut.detour).toBe('sub::sub1::urltest')

    // 生成了 urltest 组
    const group = config.outbounds.find((o) => o.type === 'urltest')
    expect(group).toBeTruthy()
    expect(group.tag).toBe('sub::sub1::urltest')
    expect(group.outbounds).toEqual(['detour:m1', 'detour:m2'])

    // 成员节点被加入且加 detour: 前缀，自身无 detour（避免与组前置回环）
    const m1 = byTag('detour:m1')
    const m2 = byTag('detour:m2')
    expect(m1).toBeTruthy()
    expect(m2).toBeTruthy()
    expect(m1.detour).toBeUndefined()
    expect(m2.detour).toBeUndefined()
  })

  it('订阅成员无启用节点时，回退直连（不生成空 urltest 组）', () => {
    const target = ssNode('target', {
      dial: { detour: { id: ['subscription:sub2'], tag: '' } },
    })
    const resolveDetour = (id) => {
      if (id === 'subscription:sub2') {
        return {
          kind: 'subscription',
          subscriptionId: 'sub2',
          groupTag: 'sub::sub2::urltest',
          nodes: [ssNode('off', { enabled: false })],
          urlTest: { url: 'http://x', interval: '5m', idle_timeout: '5m', tolerance: 50 },
        }
      }
      return null
    }

    const config = BuildTestNodeTemplate({ 0: target }, false, resolveDetour)
    const targetOut = config.outbounds.find((o) => o.tag === 'target')
    expect(targetOut).toBeTruthy()
    expect(targetOut.detour).toBeUndefined()
    expect(config.outbounds.find((o) => o.type === 'urltest')).toBeUndefined()
  })

  it('具体节点前置：前置节点加 detour: 前缀并作转发，自身不再嵌套 detour', () => {
    const target = ssNode('target', {
      dial: { detour: { id: ['pre-node-id'], tag: '' } },
    })
    const resolveDetour = (id) => {
      if (id === 'pre-node-id') {
        return { kind: 'node', outbound: ssNode('pre-node-id') }
      }
      return null
    }

    const config = BuildTestNodeTemplate({ 0: target }, false, resolveDetour)
    const targetOut = config.outbounds.find((o) => o.tag === 'target')
    expect(targetOut.detour).toBe('detour:pre-node-id')

    const pre = config.outbounds.find((o) => o.tag === 'detour:pre-node-id')
    expect(pre).toBeTruthy()
    expect(pre.detour).toBeUndefined()
  })

  it('无法解析前置时回退直连，不残留 detour 字段', () => {
    const target = ssNode('target', {
      dial: { detour: { id: ['unknown-id'], tag: '' } },
    })
    const config = BuildTestNodeTemplate({ 0: target }, false, () => null)
    const targetOut = config.outbounds.find((o) => o.tag === 'target')
    expect(targetOut).toBeTruthy()
    expect(targetOut.detour).toBeUndefined()
  })
})
