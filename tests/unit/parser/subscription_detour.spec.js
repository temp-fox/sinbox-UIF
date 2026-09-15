import { InheritSubscriptionDetour, resolveSubscriptionDetourTag } from '@/store/uif/parser/subscription_detour.js'

describe('订阅级链式代理继承', () => {
  it('订阅配置了前置代理时，继承给无 detour 的节点', () => {
    const node = { tag: 'node-1', enabled: true }
    const sub = { dial: { detour: { id: ['detour-node-id'], tag: '' } } }
    InheritSubscriptionDetour(node, sub)
    expect(node.dial.detour.id).toEqual(['detour-node-id'])
    expect(node.dial.detour.tag).toBe('')
  })

  it('节点级 detour 优先，不被订阅级覆盖', () => {
    const node = { tag: 'node-1', enabled: true, dial: { detour: { id: ['node-detour-id'], tag: 'x' } } }
    const sub = { dial: { detour: { id: ['sub-detour-id'], tag: '' } } }
    InheritSubscriptionDetour(node, sub)
    expect(node.dial.detour.id).toEqual(['node-detour-id'])
  })

  it('订阅没有 detour 时不改变节点', () => {
    const node = { tag: 'node-1', enabled: true }
    InheritSubscriptionDetour(node, {})
    expect(node.dial).toBeUndefined()
  })

  it('订阅 detour.id 为空数组时不继承', () => {
    const node = { tag: 'node-1', enabled: true }
    InheritSubscriptionDetour(node, { dial: { detour: { id: [], tag: '' } } })
    expect(node.dial).toBeUndefined()
  })

  it('继承时复制数组，不与订阅共享引用', () => {
    const node = { tag: 'node-1', enabled: true }
    const subDetourId = ['detour-node-id']
    InheritSubscriptionDetour(node, { dial: { detour: { id: subDetourId, tag: '' } } })
    expect(node.dial.detour.id).not.toBe(subDetourId)
    expect(node.dial.detour.id).toEqual(subDetourId)
  })
})

describe('订阅自动选优前置：运行时 tag 解析', () => {
  it('subscription:<id> 且订阅有启用节点时，指向 urltest 组', () => {
    expect(resolveSubscriptionDetourTag('subscription:sub1', () => true)).toEqual({
      tag: 'sub::sub1::urltest',
    })
  })

  it('订阅无启用节点时返回 null（回退直连，避免引用不存在的 urltest 组）', () => {
    expect(resolveSubscriptionDetourTag('subscription:sub1', () => false)).toBeNull()
  })

  it('具体节点 id（非 subscription: 前缀）返回 null，交由具体节点前置处理', () => {
    expect(resolveSubscriptionDetourTag('node-id', () => true)).toBeNull()
  })

  it('空 id 返回 null', () => {
    expect(resolveSubscriptionDetourTag('', () => true)).toBeNull()
    expect(resolveSubscriptionDetourTag('subscription:', () => true)).toBeNull()
  })
})
