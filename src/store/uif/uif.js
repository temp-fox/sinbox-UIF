import {
  v4 as uuidv4
} from "uuid";

import {
  formatTime,
  GetAPIAddress,
  GetKey,
  InitSetting,
  MyPost,
  MyWS,
  SetAPIAddress,
  SetKey,
  SetSession,
  GetAllSession,
  DELAY_TIMEOUT,
  SetLang,
  DeepCopy,
  Translator,
  SetSimpleStyle,
  SetUIFAction,
  GetUIFActions,
  GetSimpleStyle,
} from "@/utils";

import {
  buildInboundPorts,
  validateEnabledInbounds,
} from "./parser/inbound_validation";
import { mergeSubscriptionNodes, applyProbeResult, normalizeSubscriptions } from "./parser/subscription";
import { InheritSubscriptionDetour } from "./parser/subscription_detour";

var subscriptionTimer = null;
var subscriptionJobs = {};

import {
  BuildCoreConfig,
  BuildShareCoreConfig,
  DEFAULT_DNS_LOCAL,
  DEFAULT_DNS_REMOTE,
} from "@/uif/template/tun_fakeip";

import {
  Notification,
  Message
} from "element-ui";

import moment from "moment";

import {
  configObj,
  InitConfigState,
  newFreedomOut,
  newLoginSession,
  newDefaultHttpIn,
  newDefaultTunIn,
  newSub,
  FindOutByID,
} from "./config";

import TryParse from "@/store/uif/parser";

import {
  BuildTestNodeTemplate
} from "@/uif/template/speedtest";

import {
  SUBSCRIPTION_SELECTOR_PREFIX,
  subscriptionOutboundTag,
} from "@/store/uif/parser/route_target";

import {
  getToken
} from "@/utils/auth";

var defaultState = {
  apiAddress: "http://192.168.0.230:9413",
  password: "",

  showToolTip: false,
  coreLog: "empty.",
  consoleAuto: true,
  startTime: 0,
  loginSession: [newLoginSession()],
  loginSessionInfo: {
    isOpen: false,
    isReloadWeb: false,
  },
  usingLoginSessionIndex: 0,
  usingOutObj: newFreedomOut(),
  clashConnection: {connections: [], downloadTotal: 0, uploadTotal: 0, memory: 0},
  clashProxies: {
    all: [],
    show: [],
    selecting: "autoSelete",
    isUpdatingDelay: false,
  },
  connection: {
    isConnected: false,
    isConnecting: false,
    path: "",
    version: "-",
    web_version: "26.03.11",
    coreVersion: "-",
    ip: "127.0.0.1",
    coreStatus: 3,
    osType: "",
    system_info: {
      memory: {
        used: 100,
        available: 100,
      },
      cpu_info: {},
      cpu_usage: [],
      host_info: {},
    },
    cert: {
      public: ``,
      key: ``,
      domain: "",
    },
    times: "00:00:00",
  },
  subscribe: {
    isAdding: false,
    isOpenSub: false,
    isQuicImport: false,
    info: {
      tag: "",
      type: "link",
      data: "",
    },
  },
  share: {
    isOpenShare: false,
    isSingle: false,
    info: {},
  },
  testWebsiteList: [{
    domain: "https://baidu.com",
    delay: "",
    delay2: 0,
    isTesting: false,
  },
  {
    domain: "https://github.com",
    delay: "",
    delay2: 0,
    isTesting: false,
  },
  {
    domain: "https://youtube.com",
    delay: "",
    delay2: 0,
    isTesting: false,
  },
  {
    domain: "https://google.com",
    delay: "",
    delay2: 0,
    isTesting: false,
  },
  {
    domain: "https://instagram.com",
    delay: "",
    delay2: 0,
    isTesting: false,
  },
  {
    domain: "https://netflix.com/title/70143836",
    delay: "",
    delay2: 0,
    isTesting: false,
  },
  ],
  pannel: {
    isOpen: false,
    isAdding: false,
    isClient: false,
    isLoadingWarp: false,
    isShowingJson: false,
    info: {},
    info_string: "",
    all_list: [],
    index: 0,
  },
  dnsServer: {
    localDNSList: [{
      value: "udp://223.5.5.5",
      label: "udp://223.5.5.5",
    },
    {
      value: "udp://1.12.12.12",
      label: "udp://1.12.12.12",
    },
    {
      value: "tls://223.5.5.5",
      label: "tls://223.5.5.5",
    },
    {
      value: "tls://1.12.12.12",
      label: "tls://1.12.12.12",
    },
    {
      value: "https://dns.alidns.com/dns-query",
      label: "https://dns.alidns.com/dns-query (ECS)",
    },
    {
      value: "https://doh.pub/dns-query",
      label: "https://doh.pub/dns-query (ECS)",
    },
    {
      value: "https://223.5.5.5/dns-query",
      label: "https://223.5.5.5/dns-query",
    },
    {
      value: "https://1.12.12.12/dns-query",
      label: "https://1.12.12.12/dns-query",
    },
    {
      value: "h3://223.5.5.5/dns-query",
      label: "h3://223.5.5.5/dns-query",
    },
    {
      value: "h3://1.12.12.12/dns-query",
      label: "h3://1.12.12.12/dns-query",
    },
    ],
    remoteDNSList: [{
      value: "udp://8.8.8.8",
      label: "udp://8.8.8.8",
    },
    {
      value: "udp://1.1.1.1",
      label: "udp://1.1.1.1",
    },
    {
      value: "tls://8.8.8.8",
      label: "tls://8.8.8.8",
    },
    {
      value: "tls://1.1.1.1",
      label: "tls://1.1.1.1",
    },
    {
      value: "https://dns.google/dns-query",
      label: "https://dns.google/dns-query",
    },
    {
      value: "https://1dot1dot1dot1.cloudflare-dns.com",
      label: "https://1dot1dot1dot1.cloudflare-dns.com",
    },
    {
      value: "https://8.8.8.8/dns-query",
      label: "https://8.8.8.8/dns-query",
    },
    {
      value: "https://1.1.1.1/dns-query",
      label: "https://1.1.1.1/dns-query",
    },
    {
      value: "h3://8.8.8.8/dns-query",
      label: "h3://8.8.8.8/dns-query",
    },
    {
      value: "h3://1.1.1.1/dns-query",
      label: "h3://1.1.1.1/dns-query",
    },
    ],
  },
  subNetList: [{
    value: "113.64.0.0/12",
    label: "广东电信",
  },
  {
    value: "210.21.0.0/16",
    label: "广东联通",
  },
  {
    value: "223.104.0.0/14",
    label: "广东移动",
  },
  {
    value: "",
    label: "不设置",
  },
  ],
  route: {
    isOpen: false,
    isAdding: false,
    info: {},
    all_list: [],
    index: 0,
    routeType: "route",
    matchType: "route",
  },
  user_manager: {
    isOpenAddNode: false,
    isOpenAddUser: false,
    isOpenAddDomain: false,
    addressList: [],
    apiAddress: "",
    apiPwd: "",
    info: {},
    nodesList: [],
    domainList: [],
    usersList: [],
    usersTodayTrafficList: [],
  },
  config: {
    // to save
    startup: false,
    popupWeb: true,
    lang: "cn",
    simplified: {
      enabled: false,
      inboundMode: "mixed",
      sortOpts: [],
    },
    ssm: {
      enabled: false,
      password: "",
      listen: "0.0.0.0",
      listen_port: 9877,
      cache_path: './ssm.json',
    },
    autoUpdateUIF: true,
    autoUpdateCore: true,
    urlTest: {
      testURL: "https://www.gstatic.com/generate_204",
      interval: "5",
      tolerance: 70,
    },
    ntp: {
      enabled: false,
      server: "ntp.aliyun.com",
      interval: "30", // minute
      server_port: 123,
    },
    share: {
      domain: "",
      tunShareMode: "fakeip",
      localDNSAddress: DEFAULT_DNS_LOCAL,
      remoteDNSAddress: DEFAULT_DNS_REMOTE,
    },
    subnet: {
      client: "", // local
      share: "",
      ip_mode_local: "local",
      ip_mode_share: "local",
    },
    clash: {
      enabled: true,
      apiAddress: "http://127.0.0.1:9191", // auto update
      apiIP: "127.0.0.1",
      apiPort: 9191,
      apiKey: "",
      external_ui_download_url: "",
      external_ui: "",
    },
    fakeip: {
      enabled: false, // RealIP or FakeIP
      inet4_range: "172.19.0.1/30",
      inet6_range: "fc00::/18",
      range_cidr: ["172.19.0.1/30", "fc00::/18"],
      rewrite_ttl: 60, //seconds
      store_fakeip: true,
    },
    geoIPAddress: "https://github.com/soffchen/sing-geoip/releases/latest/download/geoip.db",
    geoSiteAddress: "https://github.com/soffchen/sing-geosite/releases/latest/download/geosite.db",
    dnsAddress: DEFAULT_DNS_LOCAL, // local
    remoteDNSAddress: DEFAULT_DNS_REMOTE,
    routeType: "route",
    useAdguardRule: false,
    coreAutoRestart: "0",
    ipType: "ipv4_only",
    shareIPType: "ipv4_only",
  },
};

var state = DeepCopy(defaultState);

export function InitUIFState() {
  state = DeepCopy(defaultState);
}

var isStartedGlobalClock = false;
var CLOCK_INTERNAL = 3000;

function StartClock() {
  if (isStartedGlobalClock) {
    return;
  }
  isStartedGlobalClock = true;
  setTimeout(heartBeat, CLOCK_INTERNAL);
}

// only update when it is connected.
function UpdateInfo(res) {
  state.connection.isConnected = true;
  state.connection.path = res.data.path;
  state.connection.version = res.data.version;
  state.connection.coreVersion = res.data.coreVersion;
  state.connection.ip = res.data.ip;
  state.connection.cert = res.data.cert;
  state.connection.coreStatus = res.data.coreStatus;
  state.connection.system_info = res.data.system_info;
  state.startTime = moment(res.data.startTime * 1000);
  state.connection.times = `${formatTime(state.startTime.toDate(), "")}`;
  if ("osType" in res.data) {
    state.connection.osType = res.data.osType;
  }

  if ("coreLog" in res.data && state.consoleAuto) {
    state.coreLog = res.data.coreLog;
    state.connection.coreStatus = res.data.coreStatus;
    state.connection.system_info = res.data.system_info;
  }
  ClashConnection();
  StartClock(); // update next time.
}

function StartSubscriptionScheduler() {
  // Refreshes are owned by the backend scheduler so manual and periodic jobs
  // share one task registry and /subscriptions/tasks status.
  normalizeSubscriptions(configObj.state.config.subscribe || []);
}

function StopSubscriptionScheduler() {
  subscriptionJobs = {};
}

async function heartBeat() {
  if (state.connection.isConnected) {
    try {
      var res = await MyPost(state.apiAddress + "/connect", {});
      if ("status" in res.data) {
        console.warn("UIF authentication status returned by /connect");
        state.connection.isConnecting = false;
      } else {
        UpdateInfo(res);
      }
    } catch (e) {
      // A transient heartbeat timeout must not tear down the management
      // session. Clash API health is independent from the UIF API session.
      console.warn("UIF heartbeat failed", e);
    }
  }
  setTimeout(heartBeat, CLOCK_INTERNAL);
}

function findClashNodeStateByName(name) {
  if (!name) return null;
  var lists = [state.clashProxies.show || [], state.clashProxies.all || []];
  for (var l = 0; l < lists.length; l++) {
    for (var i = 0; i < lists[l].length; i++) {
      var item = lists[l][i];
      if (item && item.name === name) {
        return item;
      }
    }
  }
  return null;
}

function sourceProbeStateByClashName(name) {
  var source = findProbeSourceByClashName(name);
  if (!source || !source.node) return null;
  if (source.node.last_probe_status === undefined && source.node.delay === undefined) {
    return null;
  }
  return source.node;
}

function applySavedDelayState(clashNode) {
  if (!clashNode) return;
  var cached = findClashNodeStateByName(clashNode.name);
  var source = sourceProbeStateByClashName(clashNode.name);
  var hasHistory = "history" in clashNode && clashNode["history"].length > 0 && "delay" in clashNode["history"][0];

  clashNode.isTestingDelay = cached ? !!cached.isTestingDelay : false;
  if (hasHistory) {
    clashNode.delay = clashNode["history"][0]["delay"];
  } else {
    clashNode.delay = "";
  }

  if (cached && cached.isTestingDelay) {
    clashNode.delay = cached.delay !== undefined ? cached.delay : " ";
  }
  if (source) {
    if (source.delay !== undefined) clashNode.delay = source.delay;
    if (source.last_probe_delay_ms !== undefined) clashNode.last_probe_delay_ms = source.last_probe_delay_ms;
    if (source.last_probe_status !== undefined) clashNode.last_probe_status = source.last_probe_status;
    if (source.probe_error !== undefined) clashNode.probe_error = source.probe_error;
  } else if (cached) {
    if (cached.delay !== undefined && cached.delay !== -1 && cached.delay !== "-1") clashNode.delay = cached.delay;
    if (cached.last_probe_delay_ms !== undefined) clashNode.last_probe_delay_ms = cached.last_probe_delay_ms;
    if (cached.last_probe_status !== undefined) clashNode.last_probe_status = cached.last_probe_status;
    if (cached.probe_error !== undefined) clashNode.probe_error = cached.probe_error;
  }
}

function UpdateClashNode() {
  if (!state.config.clash.enabled) {
    return;
  }

  DoClashReqeust("/proxies", "GET", {})
    .then(function (r) {
      if (state.clashProxies.isUpdatingDelay) {
        return;
      }
      var urlTest = null;
      state.clashProxies.all = [];
      for (var item in r.data["proxies"]) {
        if (item == "GLOBAL") {
          continue;
        }
        if (item == "proxy") {
          state.clashProxies.selecting = r.data["proxies"][item]["now"];
        }
        var type = r.data["proxies"][item]["type"];
        type = type.toLowerCase();
        if (["reject", "dns", "direct", "selector"].includes(type)) {
          continue;
        }
        var item = r.data["proxies"][item];
        applySavedDelayState(item);

        if (type == "urltest") {
          urlTest = item;
        } else {
          state.clashProxies.all.push(item);
        }
      }
      if (urlTest != null) {
        state.clashProxies.all = [urlTest].concat(state.clashProxies.all);
      }
      SortClashProxyNode();
    })
    .catch((error) => {});
}

function ClashConnection() {
  if (!state.config.clash.enabled) {
    return;
  }

  UpdateSubExtraInfo();

  if (state.connection.isConnected && !state.connection.isConnecting) {
    DoClashReqeust("/connections", "GET", {}).then(function (r) {
      if (!state.clashConnection) {
        state.clashConnection = r.data;
        return;
      }

      // Update basic stats
      state.clashConnection.downloadTotal = r.data.downloadTotal;
      state.clashConnection.uploadTotal = r.data.uploadTotal;
      state.clashConnection.memory = r.data.memory;

      // Update connections list with diff logic
      const newConnections = r.data.connections || [];
      const oldConnections = state.clashConnection.connections || [];
      const newConnMap = new Map(newConnections.map(c => [c.id, c]));
      const oldConnMap = new Map(oldConnections.map(c => [c.id, c]));

      // 1. Remove closed connections (in old but not in new)
      for (let i = oldConnections.length - 1; i >= 0; i--) {
        const id = oldConnections[i].id;
        if (!newConnMap.has(id)) {
          oldConnections.splice(i, 1);
        }
      }

      // 2. Add new connections (in new but not in old)
      // 3. Update existing connections
      for (let i = 0; i < newConnections.length; i++) {
        const newConn = newConnections[i];
        if (oldConnMap.has(newConn.id)) {
          // Update existing
          const existingConn = oldConnMap.get(newConn.id);
          // Update properties
          existingConn.upload = newConn.upload;
          existingConn.download = newConn.download;
          existingConn.chains = newConn.chains;
          existingConn.rule = newConn.rule;
          existingConn.start = newConn.start;
          // Metadata usually doesn't change, but safer to update if needed or deep check
          // For now assuming metadata is static for same ID, but let's update if necessary
        } else {
          // Add new
          oldConnections.push(newConn);
        }
      }

      // Ensure the order matches new connections if needed, or just let Vue handle reactivity
      // If order matters (e.g. sorted by time), we might need to re-sort or match order.
      // API usually returns sorted list. Let's sync order if user expects it.
      // But diff update is mainly to keep Vue object references stable.
      // If we just want to avoid "refresh" flash, updating properties is key.

      // Optional: re-sort oldConnections to match newConnections order if needed
      // oldConnections.sort((a, b) => {
      //   const indexA = newConnections.findIndex(c => c.id === a.id);
      //   const indexB = newConnections.findIndex(c => c.id === b.id);
      //   return indexA - indexB;
      // });
    });
  }

  UpdateClashNode();
}

function DoClashReqeust(path, method, data, clientTimeoutMs) {
  if (method == "") {
    method = "GET";
  }
  if (method == "GET") {
    path += "?";
    for (let item in data) {
      path +=
        encodeURIComponent(item) + "=" + encodeURIComponent(data[item]) + "&";
    }
  }
  var payload = {
    dst: state.config.clash.apiAddress + path,
    method: method,
    data: data,
    authorization: state.config.clash.apiKey,
  };
  if (clientTimeoutMs) {
    payload.__client_timeout_ms = clientTimeoutMs;
  }
  return MyPost(state.apiAddress + "/http_with_port", payload);
}

async function ClashChangeProxy() {
  await DoClashReqeust("/proxies/proxy", "PUT", {
    name: state.clashProxies.selecting,
  })
  await DoClashReqeust("/connections", "DELETE", {})
  // await DoClashReqeust("cache/fakeip/flush", "POST", {})
  await DoClashReqeust("cache/dns/flush", "POST", {})
}

function clashDelayTimeoutMs() {
  var subs = configObj.state.config.subscribe || [];
  var maxTimeout = 10000;
  for (var i = 0; i < subs.length; i++) {
    var value = Number(subs[i] && subs[i].probe && subs[i].probe.timeout_ms);
    if (Number.isFinite(value) && value > maxTimeout) {
      maxTimeout = value;
    }
  }
  return Math.max(3000, Math.min(120000, Math.floor(maxTimeout)));
}

function findProbeSourceByClashName(name) {
  if (!name) return null;
  var config = configObj.state.config || {};
  for (var i = 0; i < (config.outbounds || []).length; i++) {
    var outbound = config.outbounds[i];
    if (outbound && (outbound.core_tag === name || outbound.tag === name)) {
      return { node: outbound, subscription: null };
    }
  }
  for (var s = 0; s < (config.subscribe || []).length; s++) {
    var sub = config.subscribe[s];
    var nodes = (sub && sub.outbounds) || [];
    for (var n = 0; n < nodes.length; n++) {
      var node = nodes[n];
      if (node && (node.core_tag === name || node.tag === name)) {
        return { node: node, subscription: sub };
      }
    }
  }
  return null;
}

function effectiveProbeDetourId(node, subscription) {
  var nodeIds = node && node.dial && node.dial.detour && node.dial.detour.id;
  if (Array.isArray(nodeIds) && nodeIds.length > 0) {
    return nodeIds[nodeIds.length - 1] || '';
  }
  var subIds = subscription && subscription.dial && subscription.dial.detour && subscription.dial.detour.id;
  if (Array.isArray(subIds) && subIds.length > 0) {
    return subIds[subIds.length - 1] || '';
  }
  return '';
}

function probeGroupKey(source) {
  var subId = source.subscription && source.subscription.id ? source.subscription.id : 'standalone';
  return subId + '::' + effectiveProbeDetourId(source.node, source.subscription);
}

function mirrorProbeResultToClashNode(clashNode, sourceNode) {
  if (!clashNode || !sourceNode) return;
  if (sourceNode.delay !== undefined) clashNode.delay = sourceNode.delay;
  if (sourceNode.last_probe_delay_ms !== undefined) clashNode.last_probe_delay_ms = sourceNode.last_probe_delay_ms;
  if (sourceNode.last_probe_status !== undefined) clashNode.last_probe_status = sourceNode.last_probe_status;
  if (sourceNode.probe_error !== undefined) clashNode.probe_error = sourceNode.probe_error;
  if (sourceNode.delay !== ' ') clashNode.isTestingDelay = false;
  for (var item in state.clashProxies.all) {
    var target = state.clashProxies.all[item];
    if (target && target.name === clashNode.name) {
      if (sourceNode.delay !== undefined) target.delay = sourceNode.delay;
      if (sourceNode.last_probe_delay_ms !== undefined) target.last_probe_delay_ms = sourceNode.last_probe_delay_ms;
      if (sourceNode.last_probe_status !== undefined) target.last_probe_status = sourceNode.last_probe_status;
      if (sourceNode.probe_error !== undefined) target.probe_error = sourceNode.probe_error;
      if (sourceNode.delay !== ' ') target.isTestingDelay = false;
      break;
    }
  }
}

function updateClashUpdatingFlag() {
  var stillTesting = state.clashProxies.show.some(function (item) {
    return item && item.isTestingDelay;
  });
  state.clashProxies.isUpdatingDelay = stillTesting;
}

function updateClashDelayFallback(node, url) {
  var timeout = clashDelayTimeoutMs();
  var params = {
    url: url,
    timeout: timeout,
  };
  node["isTestingDelay"] = true;
  state.clashProxies.isUpdatingDelay = true;
  return DoClashReqeust(
    "/proxies/" + encodeURIComponent(node["name"]) + "/delay",
    "GET",
    params,
    timeout + 20000,
  )
    .then(function (response) {
      var delay = Number(response.data && response.data["delay"]);
      node["delay"] = Number.isFinite(delay) && delay > 0 ? delay : -1;
      node["isTestingDelay"] = false;
      return node["delay"];
    })
    .catch((error) => {
      node["delay"] = -1;
      node["isTestingDelay"] = false;
      throw error;
    })
    .finally(updateClashUpdatingFlag);
}

async function UpdateClashDelay(node, url) {
  var source = findProbeSourceByClashName(node && node.name);
  if (!source) {
    return updateClashDelayFallback(node, url);
  }
  node.isTestingDelay = true;
  state.clashProxies.isUpdatingDelay = true;
  await TestNode([source.node], source.subscription);
  mirrorProbeResultToClashNode(node, source.node);
  updateClashUpdatingFlag();
  return source.node.delay;
}

async function UpdateClashGroupDelay(url) {
  var clashNodes = state.clashProxies.show.filter(function (item) {
    return item && item["name"] && item["type"] !== "URLTest";
  });
  var groups = {};
  var groupList = [];
  var fallbackNodes = [];
  state.clashProxies.isUpdatingDelay = true;

  for (var i = 0; i < clashNodes.length; i++) {
    var clashNode = clashNodes[i];
    var source = findProbeSourceByClashName(clashNode.name);
    if (!source) {
      fallbackNodes.push(clashNode);
      continue;
    }
    var key = probeGroupKey(source);
    if (!groups[key]) {
      groups[key] = { subscription: source.subscription, sourceNodes: [], clashNodes: [] };
      groupList.push(groups[key]);
    }
    groups[key].sourceNodes.push(source.node);
    groups[key].clashNodes.push(clashNode);
    clashNode.isTestingDelay = true;
  }

  // 已能映射回 UIF 配置的节点一律走同一套链式测速逻辑：订阅级前置继承、
  // 前置订阅先测速选最快、目标节点再按 probe.concurrency（上限 5）分批测速。
  for (var g = 0; g < groupList.length; g++) {
    var group = groupList[g];
    await TestNode(group.sourceNodes, group.subscription);
    for (var n = 0; n < group.sourceNodes.length; n++) {
      mirrorProbeResultToClashNode(group.clashNodes[n], group.sourceNodes[n]);
    }
  }

  // 只对映射不到 UIF 源节点的 Clash 临时项使用兼容 fallback。
  var limit = 5;
  var cursor = 0;
  async function worker() {
    while (cursor < fallbackNodes.length) {
      var index = cursor;
      cursor += 1;
      try {
        await updateClashDelayFallback(fallbackNodes[index], url);
      } catch (error) {
        console.warn("Clash node delay failed", fallbackNodes[index] && fallbackNodes[index].name, error);
      }
    }
  }
  var workers = [];
  for (var w = 0; w < Math.min(limit, fallbackNodes.length); w++) {
    workers.push(worker());
  }
  await Promise.all(workers);
  updateClashUpdatingFlag();
}

function SortClashProxyNode() {
  var sortOpts = state.config.simplified.sortOpts;
  state.clashProxies.show = DeepCopy(state.clashProxies.all);

  var autoSelete = null;
  if (state.clashProxies.show.length > 0) {
    var autoSelete = state.clashProxies.show.shift();
  }

  if (sortOpts.includes("按延迟排序") || sortOpts.includes("Sort By Delay")) {
    state.clashProxies.show = state.clashProxies.show.sort((a, b) => {
      var a1 = a["delay"];
      var b1 = b["delay"];
      if (a1 == -1) {
        a1 = 100000;
      }
      if (b1 == -1) {
        b1 = 100000;
      }
      return a1 - b1;
    });
  }
  if (
    sortOpts.includes("过滤不可用") ||
    sortOpts.includes("Only Show Available")
  ) {
    state.clashProxies.show = state.clashProxies.show.filter((item) => {
      return item["delay"] !== -1;
    });
  }
  if (autoSelete != null) {
    state.clashProxies.show = [autoSelete].concat(state.clashProxies.show);
  }
}

function ConnectErrorMsg(msg, duration) {
  DisConnect();
  Notification({
    message: msg,
    type: "error",
    duration: duration, // 0 means forever
  });
}

function CloseCore() {
  if (!state.connection.isConnected) {
    return;
  }
  MyPost(state.apiAddress + "/close_core", {})
    .then(function (_) {
      Message({
        type: "success",
        message: Translator({
          cn: "内核已关闭！",
          en: "Core Closed!",
        }),
      });
    })
    .catch(function (error) {
      console.log(error);
      Message.error({
        message: Translator({
          cn: "内核关闭失败",
          en: "Failed to close Core",
        }),
      });
    });
}

function GetWarp() {
  if (!state.connection.isConnected) {
    return;
  }
  state.pannel.isLoadingWarp = true;
  var outbound_obj = state.pannel.info;
  MyPost(state.apiAddress + "/get_warp", {})
    .then(function (data) {
      state.pannel.isLoadingWarp = false;
      data = data.data["res"];
      console.log(data);
      console.log(outbound_obj);
      outbound_obj.setting["private_key"] = data["PrivateKey"];
      outbound_obj.setting["peer_public_key"] = data["PublicKey"];
      outbound_obj.setting["peer_public_key"] = data["PublicKey"];
      outbound_obj.setting["reserved"] = data["ClientID"];
      outbound_obj.setting["local_address"] = [
        data["Address1"] + "/32",
        data["Address2"] + "/128",
      ];
      outbound_obj.transport["address"] = data["EndpointAddress"];
      outbound_obj.transport["port"] = parseInt(data["EndpointPort"]);

      Message({
        type: "success",
        message: Translator({
          cn: "已注册并生成Warp",
          en: "warp ok",
        }),
      });
    })
    .catch(function (error) {
      state.pannel.isLoadingWarp = false;
      console.log(error);
      Message.error({
        message: Translator({
          cn: "Warp 生成失败",
          en: "Failed to build warp",
        }),
      });
    });
}

function CloseUIF() {
  if (!state.connection.isConnected) {
    return;
  }
  MyPost(state.apiAddress + "/close_uif", {})
    .then(function (_) {
      Message({
        type: "success",
        message: Translator({
          cn: "UIF 已关闭！",
          en: "UIF Closed!",
        }),
      });
    })
    .catch(function (error) {
      Message({
        type: "success",
        message: Translator({
          cn: "UIF 已关闭！",
          en: "UIF Closed!",
        }),
      });
    });
}

function GetUIFConfig() {
  if (!state.connection.isConnected) {
    return;
  }
  MyPost(state.apiAddress + "/get_uif_config", {})
    .then(function (res) {
      if (res.data == "") {
        return;
      }
      state.config = InitSetting(res.data.uif, state.config);
      if (res.data.data != undefined) {
        const savedConfig = configObj.state.config;
        const savedSubscriptions = savedConfig.subscribe || [];
        const loadedConfig = res.data.data;
        const loadedSubscriptions = loadedConfig.subscribe || [];
        const mergedSubscriptions = [...loadedSubscriptions];
        for (const local of savedSubscriptions) {
          const exists = mergedSubscriptions.some((remote) =>
            (remote.id && local.id && remote.id === local.id) ||
            (remote.data && local.data && remote.data === local.data),
          );
          if (!exists) mergedSubscriptions.push(local);
        }
        configObj.state.config = { ...loadedConfig, subscribe: mergedSubscriptions };
        if (normalizeSubscriptions(configObj.state.config.subscribe || [])) {
          SaveUIFConfig();
        }
      }

      if ("lang" in state.config) {
        SetLang(state.config.lang);
      } else {
        SetLang("cn");
      }

      if (ParseUIFAction()) {
        return;
      }

      if ("simplified" in state.config) {
        if (GetSimpleStyle() != state.config.simplified.enabled) {
          SetSimpleStyle(state.config.simplified.enabled);
          window.location.reload();
        }
      }

      ClashConnection();
    })
    .catch(function (error) {
      console.log(error);
      Message.error({
        message: "Failed: " + error,
      });
    });
}

function InitSimple() {
  if (state.config.simplified.enabled) {
    var isFound = false;
    var inboundMode = state.config.simplified.inboundMode;
    var inbounds = configObj.state.config.inbounds;
    for (var item in inbounds) {
      item = inbounds[item];
      if (item["protocol"] == inboundMode) {
        isFound = true;
        item["enabled"] = true;
      } else if (item["protocol"].includes("tun", "mixed", "http", "socks")) {
        // disable others.
        item["enabled"] = false;
      }
    }
    if (!isFound) {
      var inb = newDefaultHttpIn();
      if (inboundMode == "tun") {
        inb = newDefaultTunIn("fakeip");
      }
      inb["enabled"] = true;
      inbounds.push(inb);
    }
    state.config.clash.enabled = true;
    ApplyCoreConfig();
  }
  SaveUIFConfig();
}

function ApplyCoreConfig() {
  if (!state.connection.isConnected) {
    return;
  }
  try {
    validateEnabledInbounds(configObj.state.config.inbounds);
    var coreConfig = BuildCoreConfig(
      state.config,
      configObj.state.config,
      true,
      false,
    );
    var inboudPorts = buildInboundPorts(coreConfig);

    var clash = state.config.clash;
    if (clash.enabled && !clash.apiIP.includes("127.0.0.1")) {
      inboudPorts.push(clash.apiPort.toString());
    }

    var content = {
      config: coreConfig,
      inboudPorts: inboudPorts,
    };
    MyPost(state.apiAddress + "/run_core", content)
      .then(function (_) {
        Message({
          type: "success",
          message: Translator({
            cn: "内核已更新！",
            en: "Core Updated.",
          }),
        });
        ClashConnection();
      })
      .catch(function (error) {
        console.log(error);
        Message.error({
          message: "Core Update Failed: " + error,
        });
      });
  } catch (error) {
    Message.error({
      message: "Core Update Failed: " + error.message,
    });
  }
}

// save and apply UIF config, then apply this config to core config.
function SaveUIFConfig() {
  if (!state.connection.isConnected) {
    return Promise.reject(new Error("UIF 后端未连接"));
  }
  var shareConfig = {};
  try {
    shareConfig = BuildShareCoreConfig(state.config, configObj.state.config);
  } catch (e) {
    console.log(e);
  }

  return MyPost(state.apiAddress + "/save_uif_config", {
    config: {
      uif: state.config,
      data: configObj.state.config,
    },
    shareConfig: shareConfig,
  })
    .then(function (_) {
      if (state.loginSession.isReloadWeb) {
        window.location.reload();
      }
      Message({
        type: "success",
        message: Translator({
          cn: "UIF 配置保存成功！",
          en: "Config Saved.",
        }),
      });
    })
    .catch(function (error) {
      console.log(error);
      Message.error({
        message: "Config save failed: " + error,
      });
      throw error;
    });
}

async function ResetAll() {
  if (!state.connection.isConnected) {
    return;
  }
  await MyPost(state.apiAddress + "/save_uif_config", {
    config: {},
  });

  SetAPIAddress("undefined");
  SetKey("undefined");
  CloseCore();
  location.reload();
}

function ParseUIFAction() {
  if (!state.connection.isConnected) {
    return false;
  }

  var savedAction = GetUIFActions();
  if (savedAction != null) {
    SetUIFAction(""); // clear
    DoAction(savedAction);
    return true;
  }

  const queryParams = getQueryParams();
  var action = queryParams.get("action");
  if (action == null) {
    return false;
  }
  var name = url.hash;
  if (name == "") {}

  var parsedAction = {
    actionType: action,
  };
  if (action == "add_subcription") {
    var importType = queryParams.get("import_type");
    if (importType == null || importType == "") {
      importType = "link";
    }
    parsedAction["import_type"] = importType;
    parsedAction["data"] = queryParams.get("data");
    if (parsedAction["data"] == null) {
      parsedAction["data"] = "";
    }
    parsedAction["tag"] = queryParams.get("tag");
    if (parsedAction["tag"] == null || parsedAction["tag"] == "") {
      parsedAction["tag"] = `UIF[${moment().format("YYYY-MM-DD HH:MM:SS")}]`;
    }
  }
  console.log(url);
  SetUIFAction(parsedAction);
  if (state.config.simplified.enabled) {
    window.location.replace(url.origin + "/#/simple/out");
  } else {
    window.location.replace(url.origin + "/#/out/subscribe");
  }
  return true;
}

function DoAction(action) {
  console.log(action);
  Notification.closeAll();
  var actionType = action["actionType"];
  if (actionType == "add_subcription" && action["data"] != "") {
    state.subscribe.info.tag = action["tag"];
    state.subscribe.info.type = action["import_type"];
    state.subscribe.info.data = action["data"];
    state.subscribe.isQuicImport = true;
    state.subscribe.isAdding = true;
    state.subscribe.isOpenSub = true;
  } else {
    var msg = Translator({
      cn: "无效的 UIF 动作，如需帮助请联系提供该动作的人员",
      en: "invalid UIF action.",
    });
    var msgType = "warning";
    Notification({
      message: msg,
      type: msgType,
      duration: 0, // 0 means forever
    });
  }
}

function Connect() {
  Notification.closeAll();
  if (state.connection.isConnected) {
    DisConnect();
    return;
  }
  try {
    var url = new URL(state.apiAddress);
  } catch (e) {
    return;
  }
  state.apiAddress = ResolveAPIAddress(url.origin);

  state.connection.isConnecting = true;
  SetKey(state.password);

  MyPost(state.apiAddress + "/connect", {})
    .then(function (res) {
      state.connection.isConnecting = false;

      if ("status" in res.data) {
        // failed to check.
        var msg = Translator({
          cn: "UIF 后端已运行，请先输入密码 验证登录",
          en: "UIF is running, but you must login with password",
        });
        var msgType = "warning";
        if (state.password != "") {
          msg = Translator({
            cn: `验证失败！UIF密码 错误`,
            en: "Wrong UIF password",
          });
          msgType = "error";
        }
        Notification({
          message: msg,
          type: msgType,
          duration: 0, // 0 means forever
        });
        return;
      }

      UpdateInfo(res);
      StartSubscriptionScheduler();
      SetKey(state.password);
      SetAPIAddress(state.apiAddress);
      SetSession({
        password: state.password,
        value: state.apiAddress,
      });

      if (res.data.isFirstTime) {
        InitUIFState();
        state.apiAddress = GetAPIAddress();
        state.password = GetKey();
        state.connection.isConnected = true;
        state.loginSession.isReloadWeb = true;
        InitConfigState();

        SetSimpleStyle(false);
        if (res.data.useSimplified === true) {
          state.config.simplified.enabled = true;
          SetSimpleStyle(true);
          InitSimple(); // will SaveUIFConfig
          return;
        }

        ApplyCoreConfig();
        SaveUIFConfig();
        return;
      } else {
        GetUIFConfig();
      }
      Notification({
        message: Translator({
          cn: "连接 UIF 成功！",
          en: "Connected!",
        }),
        type: "success",
        duration: 3000,
      });
    })
    .catch(function (error) {
      console.log(error);
      ConnectErrorMsg(
        Translator({
          cn: `无法连接到 ${state.apiAddress}，请确保在[管理接口]页面中提供了可用的地址和密码！`,
          en: `Can not connect to ${state.apiAddress}; Make sure a usable API address and password provided in [API Mannager]！`,
        }),
        0,
      );
    });
}

function DisConnect() {
  Notification.closeAll();
      state.connection.isConnected = false;
  state.connection.isConnecting = false;
  StopSubscriptionScheduler();
}

function Ping(row) {
  if (!state.connection.isConnected || row == undefined) {
    return;
  }

  row.delay = "";
  MyPost(state.apiAddress + "/ping", {
    address: row["transport"]["address"],
  })
    .then(function (res) {
      var rtt = res.data.res;
      if (rtt == "0") {
        rtt = "-1";
      }
      row.delay = rtt;
    })
    .catch(function (_) {});
}

async function UpdateSub() {
  return await UpdateSub2(state.subscribe.info, false);
}

function UpdateSubExtraInfo() {
  if (!state.connection.isConnected || !state.config.simplified.enabled) {
    return;
  }

  for (var item in configObj.state.config.subscribe) {
    var item = configObj.state.config.subscribe[item];
    if (!("updateTime" in item)) {
      continue;
    }
    var pastTime = moment(item["updateTime"]);
    var currentTime = moment();
    var hoursDifference = currentTime.diff(pastTime, "hours");

    if (hoursDifference < 1) {
      continue;
    }

    var extra = item["extra"];
    if (
      extra != undefined &&
      extra != null &&
      extra["traffic"] != null &&
      extra["traffic"] != undefined &&
      extra["traffic"]["total"] != undefined &&
      extra["traffic"]["total"] != 0
    ) {
      UpdateSub2(item, true);
    }
  }
}

async function RunSubscriptionJob(sub) {
  const payload = await MyPost(state.apiAddress + "/subscriptions/job", {
    subscription_id: sub.id,
    kind: "refresh",
    source: sub.data,
    dst: sub.data,
  });
  const job = payload.data;
  if (!job || !job.job_id) throw new Error("订阅任务创建失败");
  for (let i = 0; i < 120; i += 1) {
    await new Promise((resolve) => setTimeout(resolve, 500));
    const status = await MyPost(state.apiAddress + "/subscriptions/job/status", { job_id: job.job_id });
    const current = status.data;
    if (current.status === -1) throw new Error(current.error || "订阅任务不存在");
    if (current.state === "success") {
      try {
        const envelope = typeof current.result === "string"
          ? JSON.parse(current.result || "{}")
          : (current.result || {});
        if (envelope.body !== undefined) {
          return { body: String(envelope.body), extraInfo: envelope.extra_info || "", nodes: envelope.nodes || [] };
        }
      } catch (error) {
        console.warn("subscription result envelope parse failed", error);
      }
      const fallbackBody = typeof current.result === "string" ? current.result : "";
      return { body: fallbackBody, extraInfo: "" };
    }
    if (["failed", "cancelled"].includes(current.state)) throw new Error(current.error || "订阅任务失败");
  }
  throw new Error("订阅任务超时");
}

// 将后端/前端解析出的原始节点统一补齐运行时字段，并强制使用前端
// nodeFingerprint（fnv1a）作为身份，避免后端 sha256 指纹与旧节点不匹配。
// 后端 Go 的 Transport.Setting/TLS/Multiplex 带 omitempty，空 map 会被省略，
// 这里补全 transport 结构，避免 BuildCoreConfig 时访问 undefined 崩溃。
function prepareParsedNodes(nodes) {
  return (nodes || []).map(function (node) {
    var copy = DeepCopy(node || {});
    delete copy.fingerprint;
    copy.enabled = false;
    copy.delay = "";
    copy.core_tag = "";
    copy.id = uuidv4();
    if (!copy.setting || typeof copy.setting !== "object") {
      copy.setting = {};
    }
    var transport = copy.transport;
    if (!transport || typeof transport !== "object") {
      transport = { protocol: "tcp", tls_type: "none", address: "", port: 0 };
      copy.transport = transport;
    }
    if (!transport.setting || typeof transport.setting !== "object") {
      transport.setting = {};
    }
    if (!transport.tls || typeof transport.tls !== "object") {
      transport.tls = {};
    }
    if (!transport.multiplex || typeof transport.multiplex !== "object") {
      transport.multiplex = {};
    }
    if (!transport.protocol) {
      transport.protocol = "tcp";
    }
    if (!transport.tls_type) {
      transport.tls_type = "none";
    }
    return copy;
  });
}

async function UpdateSub2(info, isUpdatingExtraData) {
  if (!state.connection.isConnected) {
    if (isUpdatingExtraData) {
      return;
    }
    Message.error({
      message: Translator({
        cn: "需先连接后端！",
        en: "Not connected.",
      }),
    });
    return;
  }

  // 订阅拉取/解析是纯后端操作（fetch + parse + snapshot），不依赖 sing-box
  // 内核是否运行。内核状态只影响“应用配置到内核”，不能阻断订阅更新，
  // 否则内核起不来时用户连订阅都无法添加/更新。
  var rawData = info.data;
  const previousUpdateTime = info.updateTime;
  const previousExtra = DeepCopy(info.extra || {});
  if (info.type == "link") {
    let res = { data: { status: 0, res: rawData }, headers: {} };
    try {
      let fetched = await RunSubscriptionJob(info);
      rawData = fetched.body || fetched;
      if (fetched.extraInfo) res.headers["extra-info"] = fetched.extraInfo;
      if (fetched.nodes && fetched.nodes.length > 0) {
        info.outbounds = mergeSubscriptionNodes(
          info.outbounds || [],
          prepareParsedNodes(fetched.nodes),
          info.policy && info.policy.update_mode === "replace" ? "replace" : "merge",
        );
        info.updateTime = moment().valueOf();
        info.last_update_status = "success";
        info.last_update_error = "";
        return true;
      }
    } catch (error) {
      console.log(error);
      Message.error({
        message: "请求出错！" + error,
      });
      info.updateTime = previousUpdateTime;
      info.extra = previousExtra;
      info.last_update_status = "failed";
      info.last_update_error = error.message || String(error);
      return false;
    }
    if (res.data["status"] != 0) {
      info.updateTime = previousUpdateTime;
      info.extra = previousExtra;
      info.last_update_status = "failed";
      info.last_update_error = res.data["res"] || "subscription request failed";
      Message.error({ message: info.last_update_error });
      return false;
    }
    console.log(res.data);
    rawData = res.data["res"];
    if ("extra-info" in res.headers) {
      var ExtraInfo = JSON.parse(res.headers["extra-info"]);
      if (!("extra" in info)) {
        info["extra"] = {};
      }
      info["extra"]["openWebURL"] = ExtraInfo["profile-web-page-url"];
      info["extra"]["traffic"] = parseTraffic(
        ExtraInfo["subscription-userinfo"],
      );
    }
  }

  if (isUpdatingExtraData) {
    return;
  }

  var outList = rawData ? TryParse(rawData) : (info.outbounds || []);
  if (outList.length == 0) {
    info.updateTime = previousUpdateTime;
    info.last_update_status = "failed";
    info.last_update_error = "subscription parser returned no nodes";
    Message.error({
      message: Translator({
        cn: "导入数据解析出错！可能不支持该订阅格式",
        en: "Parse Failed.",
      }),
    });
    return false;
  }
  console.log(outList);

  for (var item in outList) {
    item = outList[item];
    item["enabled"] = false;
    item["delay"] = "";
    item["core_tag"] = "";
    item["id"] = uuidv4();
  }
  info.outbounds = mergeSubscriptionNodes(
    info.outbounds || [],
    outList,
    info.policy && info.policy.update_mode === "replace" ? "replace" : "merge",
  );
  info.updateTime = moment().valueOf();
  info.last_update_status = "success";
  info.last_update_error = "";
  return true;
}

function parseTraffic(data) {
  if (data != "" && data != undefined) {
    const pairs = data.split("; ");
    const result = {};
    pairs.forEach((pair) => {
      const [key, value] = pair.split("=");
      result[key] = Number(value);
    });
    return result;
  }
  return null;
}

function TestWeb2(t, start) {
  t["isTesting"] = true;
  var dst = t["domain"];
  MyPost(state.apiAddress + "/proxy_get", {
    dst: dst,
    proxy_first: "true",
  })
    .then(function (r) {
      t["isTesting"] = false;
      var end = new Date().getTime();
      if (r.data["status"] != 0) {
        t["delay2"] = -1;
      } else {
        t["delay2"] = end - start;
      }
      t["delay"] = `[ ${t.delay2}ms ]`;
    })
    .catch(function (e) {
      t["isTesting"] = false;
      t["delay2"] = -1;
      console.log(e);
    });
}

function TestWeb() {
  if (!state.connection.isConnected) {
    Message.error({
      message: Translator({
        cn: "需先连接后端！",
        en: "Not connected.",
      }),
    });
    return;
  }
  for (var item in state.testWebsiteList) {
    TestWeb2(state.testWebsiteList[item], new Date().getTime());
  }
}

function BuildShareLink() {
  if (!state.connection.isConnected) {
    Message.error({
      message: Translator({
        cn: "需先连接后端！",
        en: "Not connected.",
      }),
    });
    return "";
  }
  var apiAddress = new URL(state.apiAddress);
  return apiAddress.origin + "/share?key=" + encodeURIComponent(getToken());
}

async function InstallAutoStartup() {
  try {
    var res = await MyPost(state.apiAddress + "/auto_startup", {
      isInstall: state.config.startup,
    });
  } catch (error) {
    console.log(error);
    Message.error({
      message: "Failed: " + error,
    });
    return false;
  }
  if (res.data["status"] != 0) {
    console.log(res);
    Message.error({
      message: res.data["res"],
    });
    return false;
  }
  Message({
    type: "success",
    message: Translator({
      cn: "'设置开机自启成功！'",
      en: "OK!",
    }),
  });
  return true;
}

// 测速用的前置代理解析器：根据 dial.detour.id 最后一项返回完整信息。
// 具体节点 id -> { kind: 'node', outbound }
// "subscription:<id>" -> { kind: 'subscription', subscriptionId, nodes, groupTag, urlTest }
function resolveTestDetour(id) {
  if (typeof id === 'string' && id.indexOf(SUBSCRIPTION_SELECTOR_PREFIX) === 0) {
    var subId = id.slice(SUBSCRIPTION_SELECTOR_PREFIX.length)
    var sub = null
    var subs = configObj.state.config.subscribe || []
    for (var s = 0; s < subs.length; s++) {
      if (subs[s] && subs[s].id === subId) {
        sub = subs[s]
        break
      }
    }
    if (!sub) {
      return null
    }
    var urlTest = state.config.urlTest || {}
    var interval = urlTest.interval + "m"
    return {
      kind: 'subscription',
      subscriptionId: subId,
      nodes: sub.outbounds || [],
      groupTag: subscriptionOutboundTag(subId),
      urlTest: {
        url: urlTest.testURL,
        interval: interval,
        idle_timeout: interval,
        tolerance: parseInt(urlTest.tolerance),
      },
      probe: sub.probe || {},
    }
  }
  var pre = FindOutByID(id)
  if (!pre) {
    return null
  }
  return { kind: 'node', outbound: pre }
}

function buildDelayConfigAndTags(nodes, resolveDetour) {
  var tags = [];
  var config = BuildTestNodeTemplate(nodes, false, resolveDetour);
  var i = 0;
  for (var item in config["outbounds"]) {
    if (item == 0) {
      continue;
    }
    item = config["outbounds"][item];
    // 前置转发节点（tag 前缀 detour:）及订阅 urltest 组保持 tag 供被测节点
    // detour 引用，不参与延迟上报。
    if ((item["tag"] || "").indexOf("detour:") === 0) {
      continue;
    }
    if (item["type"] === "urltest") {
      continue;
    }
    item["tag"] = i.toString();
    tags.push(i.toString());
    i += 1;
  }
  return { config: config, tags: tags };
}

function probeTimeoutMs(policy) {
  var value = Number(policy && policy.timeout_ms || DELAY_TIMEOUT);
  if (!Number.isFinite(value) || value <= 0) {
    return DELAY_TIMEOUT;
  }
  return Math.max(3000, Math.min(120000, Math.floor(value)));
}

function runDelayBatch(nodes, resolveDetour, policy, resultNodes, offset) {
  if (!state.connection.isConnected || nodes.length === 0) {
    return Promise.resolve([]);
  }
  var built = buildDelayConfigAndTags(nodes, resolveDetour);
  if (built.tags.length === 0) {
    return Promise.resolve([]);
  }
  return new Promise(function (resolve) {
    var results = [];
    var done = false;
    var timeoutMs = probeTimeoutMs(policy || {});
    var url = new URL(state.apiAddress + "/delay");
    var wsURL = `${url.protocol == 'https:' ? 'wss' : 'ws'}://${url.host}${url.pathname}?key=${encodeURIComponent(GetKey())}`;
    var socket = new WebSocket(wsURL);
    // 这里不能用前端短超时把未返回的节点批量判定为 -1。
    // 后端会等临时 sing-box Clash API 就绪，并按 timeout_ms 返回每个 tag 的真实结果；
    // 前端只负责接收后端结果。兜底只防止连接永久挂死，不写入任何节点失败状态。
    var timer = setTimeout(function () {
      if (done) return;
      done = true;
      try { socket.close(); } catch (e) {}
      Message.warning({ message: "测速连接等待过久，未返回的节点保持测速中状态。" });
      resolve(results);
    }, 30 * 60 * 1000);

    socket.onopen = function () {
      socket.send(JSON.stringify({
        config: JSON.stringify(built.config),
        tags: built.tags,
        timeout_ms: timeoutMs,
      }));
    };

    socket.onmessage = function (response) {
      var data = JSON.parse(response.data);
      var index = parseInt(data["tag"]);
      var globalIndex = Number(offset || 0) + index;
      var success = data["status"] == 0 && data["delay"] != 0;
      applyProbeResult(resultNodes || nodes, globalIndex, {
        success: success,
        delay: success ? data["delay"] : -1,
        msg: data["msg"] || "",
      }, policy || {});
      results.push({ index: index, success: success, delay: success ? Number(data["delay"]) : -1, data: data });
      if (results.length >= built.tags.length && !done) {
        done = true;
        clearTimeout(timer);
        try { socket.close(); } catch (e) {}
        resolve(results);
      }
    };

    socket.onerror = function () {
      if (done) return;
      done = true;
      clearTimeout(timer);
      try { socket.close(); } catch (e) {}
      Message.error({ message: "测速连接异常，未返回的节点保持测速中状态。" });
      resolve(results);
    };

    socket.onclose = function () {
      if (done) return;
      done = true;
      clearTimeout(timer);
      Message.warning({ message: "测速连接已关闭，未返回的节点保持测速中状态。" });
      resolve(results);
    };
  });
}

function chunkNodes(nodes, size) {
  var chunks = [];
  for (var i = 0; i < nodes.length; i += size) {
    chunks.push(nodes.slice(i, i + size));
  }
  return chunks;
}

function probeConcurrency(policy) {
  var value = Number(policy && policy.concurrency || 4);
  if (!Number.isFinite(value) || value <= 0) {
    return 4;
  }
  return Math.max(1, Math.min(5, Math.floor(value)));
}

async function runDelayInOrder(nodes, resolveDetour, policy) {
  var chunks = chunkNodes(nodes, probeConcurrency(policy));
  var offset = 0;
  for (var i = 0; i < chunks.length; i++) {
    await runDelayBatch(chunks[i], resolveDetour, policy, nodes, offset);
    offset += chunks[i].length;
  }
}

function fastestHealthyNode(nodes) {
  var best = null;
  var bestDelay = 0;
  for (var i = 0; i < nodes.length; i++) {
    var node = nodes[i];
    if (!node || !node.enabled || node.quarantined) {
      continue;
    }
    var delay = Number(node.last_probe_delay_ms || node.delay || -1);
    if (!Number.isFinite(delay) || delay <= 0) {
      continue;
    }
    if (best == null || delay < bestDelay) {
      best = node;
      bestDelay = delay;
    }
  }
  return best;
}

function cloneNodesWithSubscriptionDetour(nodes, subscriptionInfo) {
  var cloned = DeepCopy(nodes);
  if (subscriptionInfo) {
    for (var i = 0; i < cloned.length; i++) {
      InheritSubscriptionDetour(cloned[i], subscriptionInfo);
    }
  }
  return cloned;
}

function firstDetourId(nodes) {
  for (var i = 0; i < nodes.length; i++) {
    var ids = nodes[i] && nodes[i].dial && nodes[i].dial.detour && nodes[i].dial.detour.id;
    if (Array.isArray(ids) && ids.length > 0) {
      return ids[ids.length - 1];
    }
  }
  return '';
}

function hasPendingProbe(nodes) {
  for (var item in nodes) {
    if (nodes[item] && nodes[item]["delay"] == " ") {
      return true;
    }
  }
  return false;
}

function markProbeStarted(nodes) {
  for (var item in nodes) {
    if (nodes[item] && nodes[item]["enabled"] && !nodes[item]["quarantined"]) {
      nodes[item]["delay"] = " ";
    }
  }
}

async function TestNode(uifStyleNodeConfig, subscriptionInfo) {
  if (!state.connection.isConnected) {
    return;
  }
  markProbeStarted(uifStyleNodeConfig);

  var policy = (subscriptionInfo && subscriptionInfo.probe) || state.subscribe.info.probe || {};
  var testNodes = cloneNodesWithSubscriptionDetour(uifStyleNodeConfig, subscriptionInfo);
  var detourId = firstDetourId(testNodes);
  var chainedResolveDetour = resolveTestDetour;

  // 前置是「订阅自动选优」时，先测前置订阅，选最快可用节点，
  // 再把该具体节点作为前置去测当前订阅，避免 urltest 组并发竞态导致全 -1。
  var detourInfo = resolveTestDetour(detourId);
  if (detourInfo && detourInfo.kind === 'subscription') {
    await runDelayInOrder(detourInfo.nodes || [], null, detourInfo.probe || policy);
    var fastest = fastestHealthyNode(detourInfo.nodes || []);
    if (!fastest) {
      if (hasPendingProbe(detourInfo.nodes || [])) {
        Message.warning({ message: "前置订阅还有节点未返回，当前订阅暂不判定失败。" });
      } else {
        Message.error({ message: "前置订阅没有可用节点，链式测速失败。" });
      }
      return;
    }
    chainedResolveDetour = function (id) {
      if (id === detourId) {
        return { kind: 'node', outbound: fastest };
      }
      return resolveTestDetour(id);
    };
  } else if (detourInfo && detourInfo.kind === 'node') {
    await runDelayInOrder([detourInfo.outbound], null, policy);
    var preDelay = Number(detourInfo.outbound.last_probe_delay_ms || detourInfo.outbound.delay || -1);
    if (!Number.isFinite(preDelay) || preDelay <= 0) {
      if (detourInfo.outbound.delay == " ") {
        Message.warning({ message: "前置节点尚未返回测速结果，当前节点暂不判定失败。" });
      } else {
        Message.error({ message: "前置节点不可用，链式测速失败。" });
      }
      return;
    }
  }

  await runDelayInOrder(testNodes, chainedResolveDetour, policy);
  for (var i = 0; i < testNodes.length; i++) {
    if (testNodes[i].delay !== undefined) uifStyleNodeConfig[i].delay = testNodes[i].delay;
    if (testNodes[i].last_probe_delay_ms !== undefined) uifStyleNodeConfig[i].last_probe_delay_ms = testNodes[i].last_probe_delay_ms;
    if (testNodes[i].last_probe_status !== undefined) uifStyleNodeConfig[i].last_probe_status = testNodes[i].last_probe_status;
    if (testNodes[i].probe_error !== undefined) uifStyleNodeConfig[i].probe_error = testNodes[i].probe_error;
    if (testNodes[i].consecutive_failures !== undefined) uifStyleNodeConfig[i].consecutive_failures = testNodes[i].consecutive_failures;
    if (testNodes[i].quarantined !== undefined) uifStyleNodeConfig[i].quarantined = testNodes[i].quarantined;
  }
}

function TestNodeIPInfo(uifStyleNodeConfig) {
  if (!state.connection.isConnected) {
    return;
  }
  domainList = {
    1: function (data) {},
    2: function (data) {},
  };
  var tags = [];
  var config = BuildTestNodeTemplate(uifStyleNodeConfig, false);
  for (var item in domainList) {
    tags.push(item);
  }

  MyWS(
    state.apiAddress + "/delay", {
    config: JSON.stringify(config),
    is_ip_info: true,
    tags: tags,
  },
    function (response) {
      var data = JSON.parse(response.data);
      var i = data["tag"];
      domin[i].cb(data);
    },
    function (error) {
      console.log(error);
    },
  );
}

function AddNewSubcription() {
  state.subscribe.info = newSub();
  state.subscribe.isAdding = true;
  state.subscribe.isOpenSub = true;
}

function getQueryParams() {
  try {
    var currentURL = window.location.href;
    const url = new URL(currentURL);
    return url.searchParams;
  } catch (e) {
    const url = new URL('http://127.0.0.1:9413');
    return url.searchParams;
  }
}

function ResolveAPIAddress(address) {
  const fallback = "http://192.168.0.230:9413";
  const value = String(address || "").trim();
  try {
    const parsed = new URL(value || fallback);
    if (!["127.0.0.1", "localhost", "0.0.0.0"].includes(parsed.hostname)) {
      return parsed.origin;
    }
  } catch (_) {}
  if (typeof window !== "undefined" && window.location && window.location.hostname) {
    const host = window.location.hostname;
    if (!["127.0.0.1", "localhost", "0.0.0.0"].includes(host)) {
      return `${window.location.protocol}//${host}:9413`;
    }
  }
  return fallback;
}

function Init() {
  const queryParams = getQueryParams();
  var urlAddress = queryParams.get("a");
  var urlPwd = queryParams.get("p");

  // http://127.0.0.1:9528?a=http://127.0.0.1:9413&p=1
  if (urlAddress != null && urlAddress != "") {
    state.apiAddress = ResolveAPIAddress(urlAddress);
    if (urlPwd != null && urlPwd != "") {
      state.password = urlPwd;
    }
  } else {
    var address = GetAPIAddress();
    state.apiAddress = ResolveAPIAddress(address || state.apiAddress);
    state.password = GetKey();
  }

  var session = GetAllSession();
  if (session.length != 0) {
    state.loginSession = session;
  }

  Connect();
}

Init();

const actions = {
  Connect,
  SaveUIFConfig,
  GetUIFConfig,
  ApplyCoreConfig,
  ResetAll,
  UpdateSub,
  Ping,
  CloseCore,
  CloseUIF,
  TestWeb,
  BuildShareLink,
  TestNode,
  InstallAutoStartup,
  GetWarp,
  ClashChangeProxy,
  UpdateClashDelay,
  AddNewSubcription,
  UpdateClashGroupDelay,
  InitSimple,
  SortClashProxyNode,
  UpdateClashNode,
};

export default {
  namespaced: true,
  state,
  actions,
};
