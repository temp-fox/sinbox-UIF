import { In2Out } from '@/store/uif/parser/uif_in_2_out'

describe('tproxy inbound sharing', () => {
  it('rejects tproxy as a shareable outbound', () => {
    expect(() => In2Out({
      protocol: 'tproxy',
      tag: 'WT',
      enabled: true,
      transport: { address: '0.0.0.0', port: 7895, protocol: 'tcp', tls_type: 'none', tls: {} },
      setting: {},
    })).toThrow(/tproxy/)
  })
})
