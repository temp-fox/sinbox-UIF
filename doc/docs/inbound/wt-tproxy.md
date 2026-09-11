---
sidebar_position: 3
---

# WT Legacy TPROXY 方式

本文适用于以下链路：

```text
Android 宿主
└── Droidspaces
    ├── WT/OpenWRT：网关、legacy iptables TPROXY、Cloudflare 端口直连
    └── UIF/sing-box：下游代理核心，监听 0.0.0.0:7895
```

## 职责边界

| 组件 | 负责内容 |
| --- | --- |
| WT/OpenWRT | legacy iptables TPROXY、策略路由、端口例外、Cloudflare TCP/UDP 直连 |
| UIF | 面板、节点/规则管理、生成 sing-box 配置、监听 `tproxy` 入站 |
| Android/Droidspaces | 提供容器运行环境，不由 UIF 修改宿主规则 |

UIF 在此模式下**不使用 nftables**，不创建 TUN，不设置 `auto_route`、`auto_redirect` 或 Android 宿主策略路由。

## 配置步骤

1. 在 WT 中准备并验证现有 legacy iptables TPROXY 与策略路由。
2. 在 UIF 的“我的入口”中新增 `WT Legacy TPROXY`。
3. 保持默认监听地址和端口：

   ```text
   0.0.0.0:7895
   ```

4. 启用该入站并应用配置。
5. 将 WT 的 TPROXY 目标端口设置为同一个 `7895`。
6. 在 WT 规则中先处理 Cloudflare 的直连例外，再把其他流量送入 UIF。

端口可以修改，但必须同步修改 WT 规则。UIF 不会替你创建或修改这些规则。

## Cloudflare 直连

Cloudflare Tunnel 的直连判断必须留在 WT 层。例如当前方案使用的 TCP/UDP 7844 例外，应在 WT 的 TPROXY 规则之前 `RETURN`。不要把该例外迁移到 UIF 的 sing-box 路由中，因为流量应在进入 UIF 前就完成直连判定。

不要把所有 443 端口都设为直连，否则会绕过局域网其他 HTTPS 流量的代理分流。

## 与 TUN 模式的区别

`WT Legacy TPROXY` 和 `Tun VPN` 是两条互斥的流量接管路线：

- TUN 由 sing-box 创建虚拟网卡并配置系统路由；
- WT Legacy TPROXY 由 WT 的 legacy iptables 和策略路由把流量送至 UIF 监听端口；
- 在当前 RMX5062 原厂内核环境中，推荐 WT Legacy TPROXY；
- 不要同时启用 TUN 自动接管和 WT 全量 TPROXY，否则可能出现重复捕获、回环或路由冲突。

UIF 的普通端口放行逻辑也不是 TPROXY 规则；WT 仍然是唯一的网关流量分发方。

## 回滚

发生配置错误时：

1. 在 UIF 中关闭或删除 `WT Legacy TPROXY` 入站；
2. 应用配置，确认 sing-box 停止监听 `7895`；
3. 在 WT 中停用指向 `7895` 的 TPROXY 规则，或恢复原有代理目标；
4. 不修改 Android 内核、不启用 nftables、不刷入新的 boot 镜像。

若 UIF 配置无效，UIF 不应提交新的核心配置；当前运行中的旧配置保持不变。
