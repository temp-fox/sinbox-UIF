import { domainPresets, domainSuffixPresets, geoSitePresets } from '@/store/uif/parser/route_presets'

describe('route presets', () => {
  it('provides documented common geosite values', () => {
    const values = geoSitePresets.map((item) => item.value)
    expect(new Set(values).size).toBe(values.length)
    expect(values).toEqual(expect.arrayContaining([
      'openai', 'google', 'google-scholar', 'github', 'netflix', 'telegram', 'microsoft',
    ]))
  })

  it('keeps YouTube and GitHub exact-domain presets available', () => {
    expect(domainPresets).toEqual(expect.arrayContaining([
      { value: 'youtube.com', label: 'YouTube' },
      { value: 'github.com', label: 'GitHub（完全匹配）' },
    ]))
  })

  it('represents Grok as domain suffixes, not a fake geosite', () => {
    const values = domainSuffixPresets.map((item) => item.value)
    expect(values).toEqual(expect.arrayContaining(['grok.com', 'x.ai']))
    expect(geoSitePresets.map((item) => item.value)).not.toContain('grok')
  })
})
