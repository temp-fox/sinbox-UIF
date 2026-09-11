<template>
  <el-form label-width="150px">
    <el-alert
      title="WT Legacy TPROXY"
      type="info"
      :closable="false"
      description="WT/OpenWRT 负责 legacy iptables TPROXY、策略路由和 Cloudflare 直连；UIF 只监听 sing-box tproxy 入站，不创建 TUN、不使用 nftables。"
    />
    <el-row :gutter="5">
      <el-col :xs="24" :sm="12" :md="8" :lg="8" :xl="8">
        <el-form-item label="监听地址">
          <el-input v-model="outbound_obj.transport.address" />
        </el-form-item>
      </el-col>
      <el-col :xs="24" :sm="12" :md="8" :lg="8" :xl="8">
        <el-form-item label="监听端口">
          <el-input-number v-model="outbound_obj.transport.port" :min="1" :max="65535" />
        </el-form-item>
      </el-col>
    </el-row>
    <el-alert
      title="端口必须与 WT 的 TPROXY 目标端口一致，默认 7895。该入站不支持分享。"
      type="warning"
      :closable="false"
    />
  </el-form>
</template>

<script>
export default {
  name: "tproxy",
  props: ["outbound_obj"],
  created() {
    this.outbound_obj.transport.address = this.outbound_obj.transport.address || "0.0.0.0";
    this.outbound_obj.transport.port = this.outbound_obj.transport.port || 7895;
    this.outbound_obj.transport.protocol = "tcp";
    this.outbound_obj.transport.tls_type = "none";
    this.outbound_obj.transport.tls = {};
    this.outbound_obj.setting = {};
  },
};
</script>
