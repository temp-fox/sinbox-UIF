import { Base64 } from 'js-base64'
import { V2rayN2UIF } from '@/store/uif/parser/v2rayn2uif.js'

describe('real V2rayN subscription shape', () => {
  it('parses the Base64-wrapped six-node vless source', () => {
    const lines = Array.from({ length: 6 }, (_, index) =>
      `vless://5d7fc674-0d51-48ac-b207-79bb35f6fc98@yg${index + 8}.ygkkk.dpdns.org:443?encryption=none&security=tls&sni=yg.crf.dpdns.org&fp=chrome&type=ws&host=yg.crf.dpdns.org&path=%2F%3Fed%3D2560#CF_V${index + 8}`,
    ).join('\n')
    const nodes = V2rayN2UIF(Base64.encode(lines))
    expect(nodes).toHaveLength(6)
    expect(nodes.every((node) => node.protocol === 'vless')).toBe(true)
    expect(nodes[0].transport.protocol).toBe('ws')
    expect(nodes[0].transport.tls_type).toBe('tls')
  })
})
