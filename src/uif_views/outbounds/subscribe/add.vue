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

      <el-form-item label="测速阈值" v-if="uif.subscribe.info.probe.enabled">
        <el-input-number v-model="uif.subscribe.info.probe.threshold_ms" :min="1" :max="60000"></el-input-number>
        <span>毫秒</span>
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

export default {
  name: "add_subscribe_page",
  props: [],
  components: {},
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
  },
  methods: {
    ...mapActions({
      SaveUIFConfig: "uif/SaveUIFConfig",
      ApplyCoreConfig: "uif/ApplyCoreConfig",
      UpdateSub: "uif/UpdateSub",
    }),
    OnUpdateIntervalChange(value) {
      this.uif.subscribe.info.policy.update_enabled = Number(value) > 0;
    },
    SaveOrAdd() {
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
        this.AddNew();
      } else {
        this.uif.subscribe.isOpenSub = false;
        this.SaveUIFConfig();
      }
    },
    async AddNew() {
      this.isLoading = true;

      var isOK = await this.UpdateSub();
      this.isLoading = false;
      if (!isOK) {
        return;
      }

      this.config.config.subscribe.push(this.uif.subscribe.info);
      var isSimple = this.uif.config.simplified.enabled;
      if (isSimple) {
        this.uif.subscribe.info["enabled"] = true;
        for (var item in this.uif.subscribe.info["outbounds"]) {
          this.uif.subscribe.info["outbounds"][item]["enabled"] = true;
        }
      }
      this.SaveUIFConfig();
      if (isSimple) {
        this.ApplyCoreConfig();
      }
      this.uif.subscribe.isOpenSub = false;
    },
  },
};
</script>

<style></style>
