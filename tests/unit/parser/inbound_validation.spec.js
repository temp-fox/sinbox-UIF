import { normalizePort, validateEnabledInbounds, buildInboundPorts } from '@/store/uif/parser/inbound_validation'

describe('inbound validation', () => {
  it('normalizes valid ports', () => {
    expect(normalizePort('7895')).toBe(7895)
    expect(normalizePort(65535)).toBe(65535)
  })

  it('rejects invalid ports', () => {
    for (const port of ['', 0, -1, 65536, 1.5, '1.5', '7895abc']) {
      expect(() => normalizePort(port)).toThrow()
    }
  })

  it('detects duplicate enabled ports after normalization', () => {
    expect(() => validateEnabledInbounds([
      { enabled: true, tag: 'a', transport: { port: '7895' } },
      { enabled: true, tag: 'b', transport: { port: 7895 } },
    ])).toThrow(/7895/)
  })

  it('ignores disabled inbound conflicts', () => {
    expect(validateEnabledInbounds([
      { enabled: false, transport: { port: 7895 } },
      { enabled: true, transport: { port: 7895 } },
    ])).toBe(true)
  })

  it('does not expose tproxy to ordinary firewall port opening', () => {
    expect(buildInboundPorts({ inbounds: [
      { type: 'tproxy', listen: '0.0.0.0', listen_port: 7895 },
      { type: 'mixed', listen: '0.0.0.0', listen_port: 9110 },
    ] })).toEqual(['9110'])
  })
})
