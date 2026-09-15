// 链式测速的核心解析逻辑，抽成无副作用纯函数模块，便于单测。
// 只依赖 uif2singbox.js（Outbound）与 route_target.js（subscriptionUrltestOutbound），
// 二者都不 import 全局 uif store，不会在 jest 里触发 Init()→Connect() 副作用。
import { Outbound } from '@/store/uif/parser/uif2singbox.js'
import { subscriptionUrltestOutbound } from '@/store/uif/parser/route_target'

// 把 sing-box outbound 加入测试配置，统一处理 wireguard（endpoint + direct 包装）。
// 返回该节点最终在配置里可被 detour 引用的 tag。
export function addTestOutbound(res, out) {
  if (out['type'] == 'wireguard') {
    var tag = out['tag']
    var epTag = tag + '-ep'
    out['tag'] = epTag
    res['endpoints'].push(out)
    res['outbounds'].push({
      'type': 'direct',
      'tag': tag,
      'detour': epTag
    })
    return tag
  }
  res['outbounds'].push(out)
  return out['tag']
}

// 解析节点的前置代理（dial.detour.id 最后一项），把前置节点加入配置并返回
// 其 detour tag；无法解析时返回空串（回退直连）。
//
// resolveDetour 回调：入参是 id（具体节点 id 或 "subscription:<id>"），
// 返回 { kind: 'node', outbound } 或 { kind: 'subscription', nodes, urlTest, groupTag, subscriptionId }，
// 或 null/undefined 表示无法解析。
export function resolveNodeDetour(node, res, addedDetours, resolveDetour) {
  var detour = node && node.dial && node.dial.detour
  var ids = detour && detour.id
  if (!Array.isArray(ids) || ids.length === 0) {
    return ''
  }
  if (typeof resolveDetour !== 'function') {
    return ''
  }
  var id = ids[ids.length - 1]
  var info = resolveDetour(id)
  if (!info) {
    return ''
  }
  if (info.kind === 'subscription') {
    return addSubscriptionDetour(res, addedDetours, info)
  }
  // 具体节点前置
  var pre = info.outbound
  if (!pre || !pre['enabled']) {
    return ''
  }
  var preTag = pre['core_tag'] || pre['tag'] || pre['id'] || String(id)
  var detourTag = 'detour:' + preTag
  if (addedDetours[detourTag]) {
    return detourTag
  }
  addedDetours[detourTag] = true
  var preOut = Outbound(pre)
  preOut['tag'] = detourTag
  // 前置节点自身不再嵌套 detour，避免回环；仅作转发。
  delete preOut['detour']
  addTestOutbound(res, preOut)
  return detourTag
}

// 前置代理是「订阅自动选优」：把该订阅的启用节点全部加入测试配置，并生成
// 一个 urltest 组（自动选最快节点）作为前置，返回组 tag。
function addSubscriptionDetour(res, addedDetours, info) {
  var groupTag = info.groupTag
  if (!groupTag) {
    return ''
  }
  var nodes = Array.isArray(info.nodes) ? info.nodes : []
  var memberTags = []
  for (var k = 0; k < nodes.length; k++) {
    var member = nodes[k]
    if (!member || !member['enabled'] || member['quarantined']) {
      continue
    }
    var memberOut = Outbound(member)
    // 订阅成员节点自身不能有 detour（避免与组前置回环），也不在本次被单独测速。
    delete memberOut['detour']
    // 加 detour: 前缀，主循环据此跳过（不测速、不覆盖 tag），供 urltest 组引用。
    memberOut['tag'] = 'detour:' + (memberOut['tag'] || member['tag'] || member['id'])
    memberTags.push(addTestOutbound(res, memberOut))
  }
  if (memberTags.length === 0) {
    return ''
  }
  if (addedDetours[groupTag]) {
    return groupTag
  }
  addedDetours[groupTag] = true
  res['outbounds'].push(subscriptionUrltestOutbound(info.subscriptionId, memberTags, info.urlTest))
  return groupTag
}
