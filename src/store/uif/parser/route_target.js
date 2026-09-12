export const SUBSCRIPTION_SELECTOR_PREFIX = 'subscription:'

export const subscriptionOutboundTag = (subscriptionId) => `sub::${subscriptionId}::urltest`

export const subscriptionUrltestOutbound = (subscriptionId, nodeTags, settings = {}) => ({
  type: 'urltest',
  tag: subscriptionOutboundTag(subscriptionId),
  outbounds: nodeTags,
  url: settings.url,
  interval: settings.interval,
  idle_timeout: settings.idle_timeout,
  tolerance: settings.tolerance,
})

const asValues = (selection) => (Array.isArray(selection) ? selection : [selection])

export const subscriptionIdFromSelector = (selection) => {
  const values = asValues(selection)
  for (let index = values.length - 1; index >= 0; index -= 1) {
    const value = values[index]
    if (typeof value === 'string' && value.indexOf(SUBSCRIPTION_SELECTOR_PREFIX) === 0) {
      const id = value.slice(SUBSCRIPTION_SELECTOR_PREFIX.length).trim()
      if (id) return id
    }
  }
  return ''
}

export const subscriptionIdFromRoute = (route) => {
  if (!route || typeof route !== 'object') return ''
  const target = route.target
  if (target && target.kind === 'subscription' && typeof target.subscription_id === 'string') {
    return target.subscription_id.trim()
  }
  return subscriptionIdFromSelector(route.id)
}

export const applyRouteSelector = (route, selection) => {
  const values = asValues(selection).filter((value) => value !== undefined && value !== null && value !== '')
  const subscriptionId = subscriptionIdFromSelector(values)
  route.id = values
  if (subscriptionId) {
    route.target = { kind: 'subscription', subscription_id: subscriptionId }
    route.outbound = `${SUBSCRIPTION_SELECTOR_PREFIX}${subscriptionId}`
  } else {
    delete route.target
    route.outbound = values.length ? values[values.length - 1] : ''
  }
  return route
}

export const selectorPathForRoute = (route, subscriptions = []) => {
  const subscriptionId = subscriptionIdFromRoute(route)
  if (!subscriptionId) return Array.isArray(route && route.id) ? route.id : []
  const group = subscriptions.find((item) => item && item.id === subscriptionId)
  if (group) return [group.id, `${SUBSCRIPTION_SELECTOR_PREFIX}${subscriptionId}`]
  return [`${SUBSCRIPTION_SELECTOR_PREFIX}${subscriptionId}`]
}

export const normalizeRouteTarget = (route) => {
  if (!route || typeof route !== 'object') return route
  const subscriptionId = subscriptionIdFromRoute(route)
  if (subscriptionId) {
    route.target = { kind: 'subscription', subscription_id: subscriptionId }
    route.outbound = `${SUBSCRIPTION_SELECTOR_PREFIX}${subscriptionId}`
  }
  return route
}
