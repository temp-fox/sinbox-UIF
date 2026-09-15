import {
  DeepCopy
} from '@/store/uif/parser/utils.js'

import {
  Outbound
} from '@/store/uif/parser/uif2singbox.js'
import {
  resolveNodeDetour,
  addTestOutbound
} from '@/store/uif/parser/test_detour.js'

// 测速临时配置的 DNS 上游。保持与运行时的 DEFAULT_REMOTE_IP_DNS 一致
// (udp://8.8.8.8)，但不 import tun_fakeip.js —— 它会级联 import 全局 uif store，
// 在 jest 环境下触发 Init()→Connect() 的 Cookies 副作用，导致模板无法单测。
const TEST_DNS = 'udp://8.8.8.8'

export var MutipleTemplate = {
  "experimental": {
    "clash_api": {
      "external_controller": '127.0.0.1:111111'
    }
  },
  "dns": {
    "servers": [{
      "tag": "dns_direct",
      "address": TEST_DNS,
      "detour": "freedom"
    }],
    "independent_cache": true
  },
  "outbounds": [{
    'tag': 'freedom',
    'type': 'direct'
  }],
  "route": {
    "auto_detect_interface": true
  }
}

export function BuildTestNodeTemplate(uifStyleNodeConfig, isAddHttpInbound, resolveDetour) {
  var res = DeepCopy(MutipleTemplate)

  if (isAddHttpInbound) {
    res['inbounds'].push({
      'tag': 'http',
      'type': 'http',
      "server": '127.0.0.1',
      'server_port': 222222
    })
  }

  res['dns']['servers'][0]['address'] = TEST_DNS
  res['endpoints'] = []
  var addedDetours = {}
  for (var item in uifStyleNodeConfig) {
    var out = Outbound(uifStyleNodeConfig[item])
    // 链式代理：把前置节点一并加入测试配置，被测节点 detour 指向前置节点，
    // 使测速真正经过前置代理，而不是直连绕过。
    var detourTag = resolveNodeDetour(uifStyleNodeConfig[item], res, addedDetours, resolveDetour)
    if (detourTag) {
      out['detour'] = detourTag
    } else {
      delete out['detour']
    }
    addTestOutbound(res, out)

    if (isAddHttpInbound) {
      res['route']['final'] = out['tag']
    }
  }
  return res
}
