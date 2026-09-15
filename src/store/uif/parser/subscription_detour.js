import { SUBSCRIPTION_SELECTOR_PREFIX, subscriptionOutboundTag } from '@/store/uif/parser/route_target'

// 订阅级链式代理：当订阅配置了前置代理(dial.detour)时，继承给该订阅下
// 未单独设置前置代理的每一个节点。节点级 detour 优先于订阅级。
//
// 放在无副作用的 parser 模块里，便于单测，避免 import tun_fakeip.js 时
// 级联触发 uif store 的初始化副作用。
export function InheritSubscriptionDetour(node, sub) {
  if (!sub || !sub.dial || !sub.dial.detour) {
    return
  }
  var subDetourId = sub.dial.detour.id
  if (!Array.isArray(subDetourId) || subDetourId.length === 0) {
    return
  }
  var nodeId = node && node.dial && node.dial.detour && node.dial.detour.id
  if (Array.isArray(nodeId) && nodeId.length > 0) {
    return
  }
  if (!node.dial) {
    node.dial = {}
  }
  if (!node.dial.detour) {
    node.dial.detour = {}
  }
  node.dial.detour.id = subDetourId.slice()
  node.dial.detour.tag = ''
}

// 订阅自动选优前置的纯决策函数，抽离自 tun_fakeip.js 的 InitDetour，
// 便于单测钉死「前置代理=订阅自动选优」的运行时 tag 解析逻辑。
//
// 从 detour.id 数组的最后一项解析出订阅选优的 urltest 组 tag。
// 入参：
//   id          - dial.detour.id 的最后一项（具体节点 id 或 "subscription:<id>"）
//   hasEnabled  - 回调 (subscriptionId) => boolean，判断该订阅是否至少有一个
//                 启用且未隔离的节点（否则 urltest 组不会被生成，引用会指向
//                 不存在的 tag 导致内核解码失败）
// 返回：
//   { tag }     - 应写入 dial.detour.tag 的 urltest 组 tag
//   null        - 不是订阅选优（交由具体节点前置逻辑处理）
export function resolveSubscriptionDetourTag(id, hasEnabled) {
  if (typeof id !== 'string' || id.indexOf(SUBSCRIPTION_SELECTOR_PREFIX) !== 0) {
    return null
  }
  var subId = id.slice(SUBSCRIPTION_SELECTOR_PREFIX.length)
  if (!subId) {
    return null
  }
  if (typeof hasEnabled === 'function' && !hasEnabled(subId)) {
    return null
  }
  return { tag: subscriptionOutboundTag(subId) }
}
