import {
  DeepCopy,
  DeleteKeyFromDict
} from "./utils.js";

// sing-box 1.12.x 已移除 xhttp/splithttp 传输类型，旧订阅快照里仍可能保留
// 这两个值。若原样写入 transport.type，内核会以 "unknown transport type"
// 解码失败退出，导致 Clash API 挂掉、节点选择列表全空。这里统一归一为
// httpupgrade（XHTTP 的 sing-box 原生等价物），其余类型原样保留。
const normalizeTransportType = (type) => {
  if (type === 'xhttp' || type === 'splithttp') {
    return 'httpupgrade'
  }
  return type
}

function Bound(uif_config) {
  var singBoxStyle = DeepCopy(uif_config['setting']);
  var transport = uif_config['transport']
  var proxyProtocol = uif_config['protocol']
  // WT/OpenWRT 没有桌面环境:混合入站若携带 set_system_proxy=true,
  // sing-box 会在启动时因 unsupported desktop environment 直接退出。
  // 该分支的系统代理由 WT 的 legacy iptables TPROXY 管理,不能请求桌面代理。
  if (proxyProtocol === 'mixed') {
    delete singBoxStyle.set_system_proxy
  }
  var transportProtocol = transport['protocol'];

  singBoxStyle['type'] = proxyProtocol;
  singBoxStyle['tag'] = uif_config['tag'];

  if (transportProtocol != 'tcp' && transportProtocol != '') {
    // 后端 Transport.Setting 带 omitempty，空 map 会被省略成 undefined，
    // 这里必须兜底，否则 ws/grpc 等无额外参数节点会在赋值 type 时崩溃。
    var transportSetting = uif_config['transport']['setting'] || {}
    transportSetting['type'] = normalizeTransportType(transportProtocol)
    singBoxStyle['transport'] = transportSetting
  }

  if (transportProtocol == 'ws' && 'transport' in singBoxStyle &&
    'headers' in singBoxStyle['transport'] &&
    singBoxStyle['transport']['headers']['Host'] == "") {
    delete singBoxStyle['transport']['headers']
  }

  if ('multiplex' in transport) {
    singBoxStyle['multiplex'] = transport['multiplex']
  }

  if (transport['tls_type'] != 'none') {
    singBoxStyle['tls'] = transport['tls']
    // 防御：hysteria2/tuic/trojan 等协议强制要求 tls.enabled=true，
    // 某些 parser（或历史快照）生成的 tls 对象可能漏掉 enabled，
    // 内核会以 "TLS required" 拒绝启动。这里兜底补齐。
    if (singBoxStyle['tls'] && singBoxStyle['tls']['enabled'] !== true) {
      singBoxStyle['tls']['enabled'] = true
    }
  }

  if (proxyProtocol == 'hysteria2') {
    if (singBoxStyle['obfs'] != undefined && singBoxStyle['obfs']['type'] == '') {
      delete singBoxStyle['obfs']
    }
  }

  if ('dial' in uif_config && 'detour' in uif_config['dial'] &&
    uif_config['dial']['detour']['tag'] != '') {
    singBoxStyle['detour'] = uif_config['dial']['detour']['tag']
  }

  if ('dial' in uif_config && ["trojan", "vmess", "vless", "shadowsocks"].includes(uif_config['protocol'])) {
    if ('tcp_fast_open' in uif_config['dial'] && uif_config['dial']['tcp_fast_open']) {
      singBoxStyle['tcp_fast_open'] = true
    }
    if ('tcp_multi_path' in uif_config['dial'] && uif_config['dial']['tcp_multi_path']) {
      singBoxStyle['tcp_multi_path'] = true
    }
  }
  return singBoxStyle
}

export function Inbound(uifConfig) {
  var singConfig = Bound(uifConfig)
  singConfig['listen'] = uifConfig['transport']['address'];
  singConfig['listen_port'] = parseInt(uifConfig['transport']['port']);
  // singConfig['sniff'] = true;

  if ('multiplex' in singConfig) {
    if ('protocol' in singConfig['multiplex']) {
      delete singConfig['multiplex']['protocol']
    }

    if ('max_streams' in singConfig['multiplex']) {
      delete singConfig['multiplex']['max_streams']
    }
  }
  if ('tls' in singConfig && 'reality' in singConfig['tls']) {
    DeleteKeyFromDict('public_key', singConfig['tls']['reality'])
  }
  return singConfig
}

export function ParseSSPluginOpts(pluginOpts) {
  var res = []
  for (var item in pluginOpts) {
    res.push(item + '=' + String(pluginOpts[item]))
  }
  return res.join(';')
}

export function Outbound(uif_config) {
  var singBoxStyle = Bound(uif_config)
  var protocol = singBoxStyle['type']
  if (protocol == 'freedom') {
    singBoxStyle['type'] = 'direct'
  } else if (protocol != 'block') {
    singBoxStyle['server'] = uif_config['transport']['address'];
    singBoxStyle['server_port'] = parseInt(uif_config['transport']['port']);

    if (singBoxStyle['tls'] && singBoxStyle['tls']['reality'] && singBoxStyle['tls']['reality']['enabled']) {
      if (!singBoxStyle['tls']['utls']) {
        singBoxStyle['tls']['utls'] = {enabled: true, fingerprint: 'random'}
      }
      singBoxStyle['tls']['utls']['enabled'] = true
      if (!singBoxStyle['tls']['utls']['fingerprint']) {
        singBoxStyle['tls']['utls']['fingerprint'] = 'random'
      }
    }
  }

  if (protocol == 'wireguard') {
    var ep = {
      "type": "wireguard",
      "tag": singBoxStyle['tag'],
      "system": singBoxStyle['system_interface'],
      "name": singBoxStyle['interface_name'],
      "mtu": singBoxStyle['mtu'],
      "address": singBoxStyle['local_address'],
      "private_key": singBoxStyle['private_key'],
      // "listen_port": 10000,
      "peers": [
        {
          "address": singBoxStyle['server'],
          "port": singBoxStyle['server_port'],
          "public_key": singBoxStyle['peer_public_key'],
          "pre_shared_key": singBoxStyle['pre_shared_key'],
          "allowed_ips": [
            "0.0.0.0/0"
          ],
          // "persistent_keepalive_interval": 30,
          "reserved": singBoxStyle['reserved']
        }
      ]
    }
    return ep
  } else if (protocol == 'shadowsocks') {
    if ('plugin' in singBoxStyle) {
      if (singBoxStyle['plugin'] == '') {
        singBoxStyle['plugin_opts'] = ''
      } else {
        singBoxStyle['plugin_opts'] = ParseSSPluginOpts(singBoxStyle['plugin_opts'])
      }
    }
  }

  return singBoxStyle
}
