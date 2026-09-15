<template>
  <el-card v-loading="isLoading">
    <div slot="header" class="clearfix">
      <span style="cursor: pointer">{{ uif.subscribe.info.tag }} </span>
      <el-button
        style="float: right; margin-left: 10px; padding: 3px 0"
        type="text"
        icon="el-icon-check"
        @click="SaveOrAdd()"
      >
        {{ this.$translator({ cn: "保存", en: "Save" }) }}
      </el-button>
    </div>

    <el-form label-width="120px">
      <el-form-item :label="$translator({ cn: '名字', en: 'name' })">
        <el-input
          placeholder="必填"
          v-model="uif.subscribe.info.tag"
        ></el-input>
      </el-form-item>

      <el-form-item label="自动更新间隔">
        <el-select
          v-model.number="uif.subscribe.info.policy.update_interval_sec"
          @change="OnUpdateIntervalChange"
        >
          <el-option label="关闭" :value="0"></el-option>
          <el-option label="每 1 小时" :value="3600"></el-option>
          <el-option label="每 5 小时" :value="18000"></el-option>
          <el-option label="每天" :value="86400"></el-option>
        </el-select>
      </el-form-item>

      <el-form-item label="更新模式">
        <el-radio v-model="uif.subscribe.info.policy.update_mode" label="merge">增量合并</el-radio>
        <el-radio v-model="uif.subscribe.info.policy.update_mode" label="replace">覆盖更新</el-radio>
      </el-form-item>

      <el-form-item label="自动测速">
        <el-switch v-model="uif.subscribe.info.probe.enabled"></el-switch>
      </el-form-item>

      <el-form-item :label="$translator({ cn: '链式代理', en: 'detour' })">
        <el-tooltip :disabled="uif.showToolTip" placement="top">
          <div slot="content">对整个订阅的所有节点统一套前置代理，需先启用负责转发的节点，小心本地回环</div>
          <out_seletor
            :placeholder="$translator({ cn: '选填', en: 'Optional' })"
            :outbound="detourInfo"
            :isDetour="true"
          />
        </el-tooltip>
      </el-form-item>

      <el-form-item label="测速并发" v-if="uif.subscribe.info.probe.enabled">
        <el-input-number v-model.number="uif.subscribe.info.probe.concurrency" :min="1" :max="5"></el-input-number>
        <span>个/批，最多 5 个</span>
      </el-form-item>

      <el-form-item label="测速超时" v-if="uif.subscribe.info.probe.enabled">
        <el-input-number v-model.number="uif.subscribe.info.probe.timeout_ms" :min="3000" :max="120000" :step="1000"></el-input-number>
        <span>毫秒</span>
      </el-form-item>

      <el-form-item label="测速阈值" v-if="uif.subscribe.info.probe.enabled">
        <el-input-number v-model.number="uif.subscribe.info.probe.threshold_ms" :min="1" :max="60000"></el-input-number>
        <span>毫秒，只标记慢速，不作为失败</span>
      </el-form-item>

      <el-form-item :label="$translator({ cn: '导入方式', en: 'Import type' })">
        <el-radio v-model="uif.subscribe.info.type" label="link">
          {{ $translator({ cn: "链接", en: "Link" }) }}
        </el-radio>
        <el-radio v-model="uif.subscribe.info.type" label="data">
          {{ $translator({ cn: "原始数据", en: "Meta Data" }) }}
        </el-radio>
      </el-form-item>

      <el-form-item
        :label="$translator({ cn: '订阅地址', en: 'Link' })"
        v-if="uif.subscribe.info.type == 'link'"
      >
        <el-input
          :placeholder="
            $translator({
              cn: '必填 (支持 Clash，Clash-Meta, V2rayN, Sing-Box, UIF)',
              en: 'Required (Support Clash，Clash-Meta, V2rayN, Sing-Box, UIF)',
            })
          "
          v-model="uif.subscribe.info.data"
          :rows="3"
          type="textarea"
        ></el-input>
      </el-form-item>

      <el-form-item
        :label="$translator({ cn: '数据', en: 'Data' })"
        v-if="uif.subscribe.info.type == 'data'"
      >
        <el-input
          :placeholder="
            $translator({
              cn: '必填 (支持 Clash，Clash-Meta, V2rayN, Sing-Box, UIF)',
              en: 'Required (Support Clash，Clash-Meta, V2rayN, Sing-Box, UIF)',
            })
          "
          v-model="uif.subscribe.info.data"
          :rows="5"
          type="textarea"
        ></el-input>
      </el-form-item>
    </el-form>
  </el-card>
</template>

<script>
import { mapState, mapActions } from "vuex";
import out_seletor from "@/uif_views/outbounds/my_servers/out_seletor.vue";

export default {
  name: "add_subscribe_page",
  props: [],
  components: { out_seletor },
  data() {
    return {
      isLoading: false,
    };
  },
  mounted() {
    if (this.uif.subscribe.isQuicImport) {
      this.uif.subscribe.isQuicImport = false;
      var t = this;
      setTimeout(() => {
        t.SaveOrAdd();
      }, 1000);
    }
  },
  computed: {
    ...mapState(["config", "uif"]),
    detourInfo() {
      var info = this.uif.subscribe.info;
      if (!info.dial) {
        this.$set(info, "dial", {});
      }
      if (!info.dial.detour) {
        this.$set(info.dial, "detour", { id: [], tag: "" });
      }
      return info.dial.detour;
    },
  },
  methods: {
    ...mapActions({
      SaveUIFConfig: "uif/SaveUIFConfig",
      ApplyCoreConfig: "uif/ApplyCoreConfig",
      UpdateSub: "uif/UpdateSub",
    }),
    OnUpdateIntervalChange(value) {
      const interval = Number(value) || 0;
      this.uif.subscribe.info.policy.update_interval_sec = interval;
      this.uif.subscribe.info.policy.update_enabled = interval > 0;
    },
    async SaveOrAdd() {
      if (
        this.uif.subscribe.info.tag == "" ||
        this.uif.subscribe.info.data == ""
      ) {
        this.$message({
          type: "info",
          message: this.$translator({
            cn: "订阅链接 和 名字 都不能为空",
            en: "Link and name can not be empty.",
          }),
        });
        return;
      }

      if (this.uif.subscribe.isAdding) {
        await this.AddNew();
      } else {
        this.uif.subscribe.isOpenSub = false;
        await this.SaveUIFConfig();
        // 订阅级链式代理会影响生成的内核配置，保存后需重新应用到内核。
        this.ApplyCoreConfig();
      }
    },
    async AddNew() {
      this.isLoading = true;

      const subscription = this.uif.subscribe.info;
      if (!subscription.id) {
        subscription.id = `legacy-subscription-${Date.now().toString(16)}`;
      }
      const subscriptions = this.config.config.subscribe || [];
      const existingIndex = subscriptions.findIndex(
        (item) => item.id === subscription.id || item.data === subscription.data,
      );
      const previousSubscription = existingIndex >= 0 ? subscriptions[existingIndex] : null;
      const previousIndex = existingIndex;
      if (existingIndex >= 0) {
        subscriptions.splice(existingIndex, 1, subscription);
      } else {
        subscriptions.push(subscription);
      }

      try {
        // Register the subscription before fetching so a config reload during
        // the asynchronous job cannot erase it from the scheduler/UI state.
        await this.SaveUIFConfig();
        var isOK = await this.UpdateSub();
        if (!isOK) {
          throw new Error("订阅拉取或解析失败");
        }
        const saved = await this.SaveUIFConfig();
        if (saved === false) {
          throw new Error("订阅配置保存失败");
        }
        var isSimple = this.uif.config.simplified.enabled;
        if (isSimple) {
          subscription.enabled = true;
          for (var item of subscription.outbounds || []) {
            item.enabled = true;
          }
          await this.SaveUIFConfig();
          this.ApplyCoreConfig();
        }
        this.uif.subscribe.isOpenSub = false;
      } catch (error) {
        if (previousIndex >= 0 && previousSubscription) {
          subscriptions.splice(previousIndex, 1, previousSubscription);
        } else {
          const index = subscriptions.indexOf(subscription);
          if (index >= 0) subscriptions.splice(index, 1);
        }
        try {
          await this.SaveUIFConfig();
        } catch (rollbackError) {
          console.warn("subscription rollback save failed", rollbackError);
        }
        this.$message.error(error.message || String(error));
      } finally {
        this.isLoading = false;
      }
    },
  },
};
</script>

<style></style>
