import { mount } from '@vue/test-utils'
import addSubscribe from '@/uif_views/outbounds/subscribe/add.vue'

describe('subscription add flow', () => {
  it('registers the subscription before fetching and waits for save', async () => {
    const subscription = { id: 'sub-a', data: 'https://example.test/a', tag: 'A', outbounds: [], policy: {}, probe: {} }
    const config = { config: { subscribe: [] } }
    const uif = { subscribe: { info: subscription, isAdding: true, isOpenSub: true }, config: { simplified: { enabled: false } } }
    const saveCalls = []
    const wrapper = mount(addSubscribe, {
      computed: { config: () => config, uif: () => uif },
      methods: {
        SaveUIFConfig: jest.fn(() => { saveCalls.push(config.config.subscribe.length); return Promise.resolve() }),
        UpdateSub: jest.fn(async () => true),
      },
      mocks: { $message: { error: jest.fn() }, $translator: (x) => x.cn },
    })
    await wrapper.vm.AddNew()
    expect(config.config.subscribe).toContain(subscription)
    expect(saveCalls).toEqual([1, 1])
    expect(uif.subscribe.isOpenSub).toBe(false)
  })
})
