const DECIMAL_PORT = /^(0|[1-9][0-9]*)$/

export function normalizePort(value) {
  if (typeof value === 'number') {
    if (!Number.isInteger(value)) throw new Error(`invalid inbound port: ${value}`)
    value = String(value)
  }
  if (typeof value !== 'string' || !DECIMAL_PORT.test(value.trim())) {
    throw new Error(`invalid inbound port: ${String(value)}`)
  }
  const port = Number(value.trim())
  if (!Number.isInteger(port) || port < 1 || port > 65535) {
    throw new Error(`inbound port must be between 1 and 65535: ${String(value)}`)
  }
  return port
}

export function validateEnabledInbounds(inbounds = []) {
  const used = new Map()
  for (const inbound of Array.isArray(inbounds) ? inbounds : []) {
    if (!inbound || !inbound.enabled) continue
    const port = normalizePort(inbound.transport && inbound.transport.port)
    const tag = inbound.tag || inbound.protocol || 'unnamed inbound'
    const previous = used.get(port)
    if (previous) {
      throw new Error(`inbound port ${port} is used by ${previous} and ${tag}`)
    }
    used.set(port, tag)
    if (inbound.transport) inbound.transport.port = port
  }
  return true
}

export function buildInboundPorts(coreConfig) {
  const ports = []
  for (const inbound of (coreConfig && coreConfig.inbounds) || []) {
    if (!inbound || inbound.type === 'tproxy') continue
    if (inbound.listen === '127.0.0.1' || !('listen_port' in inbound)) continue
    ports.push(normalizePort(inbound.listen_port).toString())
  }
  return ports
}
