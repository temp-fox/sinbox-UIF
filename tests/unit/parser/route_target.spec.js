import {
  applyRouteSelector,
  normalizeRouteTarget,
  selectorPathForRoute,
  subscriptionIdFromRoute,
  subscriptionOutboundTag,
  subscriptionUrltestOutbound,
} from '@/store/uif/parser/route_target'

describe('route target helpers', () => {
  it('writes an independent subscription target and selector value', () => {
    const route = { id: ['freedom'], outbound: 'freedom' }
    applyRouteSelector(route, ['sub-id', 'subscription:sub-id'])
    expect(route.target).toEqual({ kind: 'subscription', subscription_id: 'sub-id' })
    expect(route.outbound).toBe('subscription:sub-id')
    expect(subscriptionIdFromRoute(route)).toBe('sub-id')
  })

  it('does not let a stale legacy id override target', () => {
    const route = {
      id: ['old-node-id'],
      outbound: 'old-node',
      target: { kind: 'subscription', subscription_id: 'sub-id' },
    }
    normalizeRouteTarget(route)
    expect(route.outbound).toBe('subscription:sub-id')
    expect(selectorPathForRoute(route, [{ id: 'sub-id', tag: 'OpenAI' }])).toEqual([
      'sub-id',
      'subscription:sub-id',
    ])
  })

  it('creates a standalone urltest outbound with valid fields', () => {
    expect(subscriptionUrltestOutbound('sub-id', ['node-a'], {
      url: 'https://www.gstatic.com/generate_204',
      interval: '5m',
      idle_timeout: '5m',
      tolerance: 50,
    })).toEqual({
      type: 'urltest',
      tag: 'sub::sub-id::urltest',
      outbounds: ['node-a'],
      url: 'https://www.gstatic.com/generate_204',
      interval: '5m',
      idle_timeout: '5m',
      tolerance: 50,
    })
  })

})
