---
sidebar_position: 3
---

# 软路由

支持 OpenWrt、iStoreOS、ImmortalWrt 等软路由系统，只要是 Linux 的都支持。

** 👉️ > 安装 **

跟 [UIF 的 Linux 安装](./linux.md) 是一模一样的。通过连接上 SSH 等方式使用命令行，然后自行选择合适你自己的方式安装即可。我们暂时没有提供 `ipk 包`。

** 👉️ > 运行 **

还是跟 [UIF 的 Linux 安装](./linux.md) 一模一样。你可以使用 procd、systemd、Docker 等控制 UIF 启用或关闭。

** 👉️ > 透明代理 **

UIF 支持两种路线：`Tun VPN` 和 `WT Legacy TPROXY`。

在 RMX5062 Android 15 + Droidspaces 环境中，推荐使用 `WT Legacy TPROXY`：WT/OpenWRT 继续负责 legacy iptables TPROXY、策略路由和 Cloudflare 端口直连，UIF 只监听 sing-box 的 `tproxy` 入站（默认 `0.0.0.0:7895`）。此模式不依赖 nftables，不创建 TUN，不设置 `auto_route` 或 `auto_redirect`，也不会修改 Android 宿主或 WT 防火墙规则。详细步骤见 [WT Legacy TPROXY](../inbound/wt-tproxy.md)。

如果使用 Tun VPN，仍需额外操作：

① 检查是否开启了 `路由转发` 并设置好防火墙允许流量进入，通常在 OpenWrt 上已经默认设置好了:

```bash
sysctl -w net.ipv4.ip_forward=1 # 临时开启 IPv4 路由转发（重启设备后失效）
ufw disable # 关闭防火墙，你也可以选择创建指定防火墙规则，放行 UIF 的端口
```

② 还需要确保已安装了 `kmod-tun` 和 `iptable` 依赖，否则内核将无法创建虚拟网卡。

Tun VPN 不应与 WT 的全量 legacy TPROXY 同时启用。如果想让局域网设备使用 WT 网关，优先采用 WT Legacy TPROXY；UIF 的普通端口放行不是 TPROXY 规则。
