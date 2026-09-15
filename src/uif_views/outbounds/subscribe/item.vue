<template>
  <el-card v-loading="isLoading">
    <div slot="header" class="clearfix">
      <span
        style="cursor: pointer; font-weight: bold"
        v-if="isSimple"
        @click="toggleCollapse"
      >
        {{ subscribe_item_info.tag }}
      </span>

      <span style="cursor: pointer" v-else @click="toggleCollapse">
        <i
          :class="isCollapsed ? 'el-icon-arrow-down' : 'el-icon-arrow-up'"
          style="margin-right: 5px"
        ></i>
        {{ subscribe_item_info.tag }}
        <el-divider direction="vertical"></el-divider>
        {{ subscribe_item_info.outbounds.length }}
        <el-divider direction="vertical"></el-divider>
        {{ LastUpdateTime() }}
        <el-tag size="mini" :type="subscribe_item_info.last_update_status === 'failed' ? 'danger' : 'info'">
          {{ subscribe_item_info.last_update_status || 'unknown' }}
        </el-tag>
      </span>

      <el-dropdown
        trigger="click"
        style="float: right; margin-left: 10px; padding: 3px 0; cursor: pointer"
      >
        <span class="el-dropdown-link">
          {{ $translator({ cn: "操作", en: "Actions" }) }}
          <i class="el-icon-arrow-down el-icon--right"></i>
        </span>
        <el-dropdown-menu slot="dropdown">
          <el-dropdown-item @click.native="Update" icon="el-icon-refresh-right">
            {{ $translator({ cn: "更新订阅", en: "Update" }) }}
          </el-dropdown-item>

          <el-dropdown-item @click.native="OpenHistoryTasks" icon="el-icon-time">
            {{ $translator({ cn: "历史/任务", en: "History / Tasks" }) }}
          </el-dropdown-item>

          <el-dropdown-item
            @click.native="SpeedTest"
            icon="el-icon-odometer"
            v-if="!isSimple"
          >
            {{ $translator({ cn: "测速并清理", en: "Probe and Clean" }) }}
          </el-dropdown-item>

          <el-dropdown-item
            divided
            @click.native="ChangeAllNodeStatus(true)"
            v-if="!isSimple"
            icon="el-icon-sort-down"
          >
            {{ $translator({ cn: "全启用", en: "Enable All" }) }}
          </el-dropdown-item>
          <el-dropdown-item
            @click.native="ChangeAllNodeStatus(false)"
            v-if="!isSimple"
            icon="el-icon-sort-up"
          >
            {{ $translator({ cn: "全不启用", en: "Disable All" }) }}
          </el-dropdown-item>
          <el-dropdown-item
            divided
            @click.native="ShareMuti"
            v-if="!isSimple"
            icon="el-icon-share"
          >
            {{ $translator({ cn: "分享", en: "Share" }) }}
          </el-dropdown-item>

          <el-dropdown-item @click.native="Config" icon="el-icon-edit-outline">
            {{ $translator({ cn: "配置", en: "Config" }) }}
          </el-dropdown-item>

          <el-dropdown-item
            style="color: red"
            @click.native="Delete"
            divided
            icon="el-icon-delete"
          >
            {{ $translator({ cn: "删除", en: "Delete" }) }}
          </el-dropdown-item>

          <el-dropdown-item
            @click.native="openWebURL"
            icon="el-icon-top-right"
            v-if="hasOpenURL()"
          >
            {{ $translator({ cn: "打开官网", en: "OpenURL" }) }}
          </el-dropdown-item>
        </el-dropdown-menu>
      </el-dropdown>
    </div>

    <el-collapse-transition>
      <div v-show="!isCollapsed || isSimple">
        <div
          v-if="isSimple"
          style="
            height: 100px;
            display: flex;
            flex-direction: column;
            justify-content: space-between;
          "
        >
          <div>
            <el-switch
              v-model="subscribe_item_info.enabled"
              active-color="#13ce66"
              inactive-color="#ff4949"
              @change="ChangeAllNodeStatus"
            >
            </el-switch>
            <el-divider direction="vertical"></el-divider>
            {{ subscribe_item_info["outbounds"].length }}
            <el-divider direction="vertical"></el-divider>
            <span style="color: grey">
              {{ LastUpdateTime() }}
            </span>
          </div>
          <div v-if="isShowExtra()">
            <div style="display: flex; justify-content: space-between">
              <div>{{ BuildExpireDate() }}</div>
              <div>{{ BuildTraffic() }}</div>
            </div>

            <el-progress :percentage="BuildPercent()"></el-progress>
          </div>
        </div>

        <out_table
          :outbound_list="subscribe_item_info.outbounds"
          :isSub="true"
          :subscription_info="subscribe_item_info"
          v-else
        />
      </div>
    </el-collapse-transition>

    <el-dialog
      :title="$translator({ cn: '订阅历史 / 任务', en: 'Subscription history / tasks' })"
      :visible.sync="historyDialogVisible"
      width="min(760px, 92vw)"
      @close="StopTaskPolling"
    >
      <el-tabs v-model="historyTaskTab" @tab-click="RefreshHistoryTasks">
        <el-tab-pane :label="$translator({ cn: '历史', en: 'History' })" name="history">
          <el-table
            v-loading="historyLoading"
            :data="subscriptionHistory"
            size="small"
            max-height="360"
            empty-text="暂无历史快照"
          >
            <el-table-column prop="name" label="文件" min-width="220" show-overflow-tooltip />
            <el-table-column label="时间" min-width="160">
              <template slot-scope="scope">{{ FormatHistoryTime(scope.row.created_at) }}</template>
            </el-table-column>
            <el-table-column label="大小" width="100" align="right">
              <template slot-scope="scope">{{ FormatHistorySize(scope.row.size) }}</template>
            </el-table-column>
            <el-table-column label="操作" width="90" align="center">
              <template slot-scope="scope">
                <el-button type="text" size="mini" @click="RestoreHistory(scope.row)">恢复</el-button>
              </template>
            </el-table-column>
          </el-table>
        </el-tab-pane>
        <el-tab-pane :label="$translator({ cn: '任务', en: 'Tasks' })" name="tasks">
          <el-table
            v-loading="tasksLoading"
            :data="subscriptionTasks"
            size="small"
            max-height="360"
            empty-text="暂无任务"
          >
            <el-table-column prop="state" label="状态" width="110">
              <template slot-scope="scope">
                <el-tag size="mini" :type="TaskTagType(scope.row.state)">{{ scope.row.state || 'unknown' }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="job_id" label="任务 ID" min-width="180" show-overflow-tooltip />
            <el-table-column label="解析摘要" min-width="180" show-overflow-tooltip>
              <template slot-scope="scope">{{ TaskParseSummary(scope.row) }}</template>
            </el-table-column>
            <el-table-column label="快照摘要" min-width="220" show-overflow-tooltip>
              <template slot-scope="scope">{{ TaskSnapshotSummary(scope.row) }}</template>
            </el-table-column>
            <el-table-column label="探测状态" min-width="180" show-overflow-tooltip>
              <template slot-scope="scope">{{ TaskProbeSummary(scope.row) }}</template>
            </el-table-column>
            <el-table-column label="错误" min-width="180" show-overflow-tooltip>
              <template slot-scope="scope">
                <span v-if="TaskError(scope.row)" style="color: #f56c6c">{{ TaskError(scope.row) }}</span>
                <span v-else style="color: #909399">-</span>
              </template>
            </el-table-column>
            <el-table-column label="开始时间" min-width="150">
              <template slot-scope="scope">{{ FormatHistoryTime(scope.row.last_started_at || (scope.row.job && scope.row.job.started_at)) }}</template>
            </el-table-column>
            <el-table-column label="操作" width="90" align="center">
              <template slot-scope="scope">
                <el-button
                  v-if="IsTaskActive(scope.row)"
                  type="text"
                  size="mini"
                  @click="CancelTask(scope.row)"
                >取消</el-button>
                <span v-else style="color: #909399">-</span>
              </template>
            </el-table-column>
          </el-table>
        </el-tab-pane>
      </el-tabs>
    </el-dialog>
  </el-card>
</template>

<script>
import { formatTime, ParseTraffic, MyGet, MyPost } from "@/utils/index.js";
import { stableSubscriptionID } from "@/store/uif/parser/subscription";
import { mapState, mapActions } from "vuex";
import detail from "./detail.vue";
import out_table from "@/uif_views/outbounds/my_servers/out_table.vue";
import uif_store from "@/store/uif/uif";
import moment from "moment";

export default {
  name: "subscribe_item",
  props: ["subscribe_item_info", "subscribe_index"],
  components: { detail, out_table },
  data() {
    return {
      show_detail: false,
      viewerText: "",
      isLoading: false,
      isCollapsed: false,
      historyDialogVisible: false,
      historyTaskTab: "history",
      historyLoading: false,
      tasksLoading: false,
      subscriptionHistory: [],
      subscriptionTasks: [],
      taskPollingTimer: null,
    };
  },
  mounted() {
    this.isCollapsed = this.subscribe_item_info["isCollapsed"];
  },
  computed: {
    ...mapState(["config", "uif"]),
    isSimple() {
      return this.uif.config.simplified.enabled;
    },
  },

  methods: {
    ...mapActions({
      SaveUIFConfig: "uif/SaveUIFConfig",
      ApplyCoreConfig: "uif/ApplyCoreConfig",
      UpdateSub: "uif/UpdateSub",
    }),
    toggleCollapse() {
      this.isCollapsed = !this.isCollapsed;
      this.subscribe_item_info["isCollapsed"] = this.isCollapsed;
      this.SaveUIFConfig();
    },
    SubscriptionID() {
      if (!this.subscribe_item_info.id) {
        this.$set(this.subscribe_item_info, "id", stableSubscriptionID(this.subscribe_item_info, this.subscribe_index || 0));
        this.SaveUIFConfig();
      }
      return this.subscribe_item_info.id;
    },
    APIAddress(path) {
      return this.uif.apiAddress.replace(/\/$/, "") + path;
    },
    async OpenHistoryTasks() {
      this.historyDialogVisible = true;
      await this.RefreshHistoryTasks();
      this.StartTaskPolling();
    },
    async RefreshHistoryTasks() {
      if (!this.historyDialogVisible && this.historyTaskTab === "history") return;
      if (this.historyTaskTab === "history") {
        this.historyLoading = true;
        try {
          const response = await MyGet(this.APIAddress("/subscriptions/history"), {
            id: this.SubscriptionID(),
          });
          const data = response.data;
          this.subscriptionHistory = Array.isArray(data)
            ? data
            : (data && (data.history || data.items || data.data)) || [];
        } catch (error) {
          this.$message.error(error.message || "获取订阅历史失败");
        } finally {
          this.historyLoading = false;
        }
      } else {
        await this.LoadTasks();
      }
    },
    async LoadTasks() {
      this.tasksLoading = true;
      try {
        const response = await MyGet(this.APIAddress("/subscriptions/tasks"), {});
        const data = response.data;
        const tasks = Array.isArray(data)
          ? data
          : (data && (data.tasks || data.items || data.data)) || [];
        const filteredTasks = tasks.filter((task) => {
          return !task.subscription_id || task.subscription_id === this.SubscriptionID();
        });
        const details = await Promise.all(filteredTasks.map(async (task) => {
          if (!task.subscription_id) return task;
          try {
            const detail = await MyGet(this.APIAddress("/subscriptions/task"), {
              id: task.subscription_id,
            });
            const payload = detail.data || {};
            return Object.assign({}, task, payload.task || {}, { job: payload.job || null });
          } catch (error) {
            return task;
          }
        }));
        this.subscriptionTasks = details;
      } catch (error) {
        this.$message.error(error.message || "获取订阅任务失败");
      } finally {
        this.tasksLoading = false;
      }
    },
    StartTaskPolling() {
      this.StopTaskPolling();
      this.taskPollingTimer = setInterval(() => {
        if (this.historyDialogVisible && this.historyTaskTab === "tasks") this.LoadTasks();
      }, 3000);
    },
    StopTaskPolling() {
      if (this.taskPollingTimer) {
        clearInterval(this.taskPollingTimer);
        this.taskPollingTimer = null;
      }
    },
    beforeDestroy() {
      this.StopTaskPolling();
    },
    async RestoreHistory(row) {
      try {
        await this.$confirm("恢复该历史快照将覆盖当前订阅快照，是否继续？", "提示", {
          type: "warning",
        });
        this.historyLoading = true;
        const response = await MyPost(this.APIAddress("/subscriptions/restore"), {
          id: this.SubscriptionID(),
          history: row.name || row.path,
        });
        const restored = response.data && response.data.snapshot;
        if (restored && Array.isArray(restored.nodes)) {
          this.subscribe_item_info.outbounds = restored.nodes;
          this.subscribe_item_info.updateTime = restored.updated_at || Date.now();
          this.SaveUIFConfig();
          this.ApplyCoreConfig();
        }
        this.$message.success("历史快照已恢复");
        await this.RefreshHistoryTasks();
      } catch (error) {
        if (error !== "cancel" && error !== "close") {
          this.$message.error((error && error.message) || "恢复历史快照失败");
        }
      } finally {
        this.historyLoading = false;
      }
    },
    IsTaskActive(task) {
      return task && ["queued", "running"].indexOf(task.state) !== -1;
    },
    TaskTagType(state) {
      if (state === "success") return "success";
      if (state === "failed" || state === "cancelled") return "danger";
      if (state === "running") return "warning";
      return "info";
    },
    async CancelTask(task) {
      const id = task.subscription_id || this.SubscriptionID();
      try {
        await MyPost(this.APIAddress("/subscriptions/task/" + encodeURIComponent(id) + "/cancel"), {});
        this.$message.success("任务已取消");
        await this.LoadTasks();
      } catch (error) {
        this.$message.error(error.message || "取消任务失败");
      }
    },
    TaskParseSummary(task) {
      const summary = this.TaskData(task, "parse_summary");
      if (!summary) return "-";
      const format = summary.format || "unknown";
      const nodes = summary.nodes === undefined ? "-" : summary.nodes;
      const skipped = summary.skipped === undefined ? 0 : summary.skipped;
      return `${format} · ${nodes} nodes · skipped ${skipped}`;
    },
    TaskSnapshotSummary(task) {
      const summary = this.TaskData(task, "snapshot_summary");
      if (!summary) return "-";
      return `+${summary.added || 0} ~${summary.updated || 0} missing ${summary.missing || 0} -${summary.removed || 0}`;
    },
    TaskProbeSummary(task) {
      const summary = this.TaskData(task, "probe_summary");
      if (summary) {
        return `${summary.status || "unknown"} · ${summary.healthy || 0}/${summary.total || 0} healthy`;
      }
      const probe = this.TaskData(task, "probe");
      if (probe) return `${probe.status || "unknown"} · ${probe.healthy || 0}/${probe.total || 0} healthy`;
      return "-";
    },
    TaskError(task) {
      const error = task && (task.error || (task.job && task.job.error));
      return error || "";
    },
    TaskData(task, key) {
      if (!task) return null;
      if (task[key]) return task[key];
      if (task.job && task.job[key]) return task.job[key];
      return null;
    },
    FormatHistoryTime(value) {
      if (!value) return "-";
      const parsed = moment(value);
      return parsed.isValid() ? parsed.format("YYYY-MM-DD HH:mm:ss") : formatTime(value, "");
    },
    FormatHistorySize(value) {
      if (!value) return "0 B";
      const units = ["B", "KB", "MB", "GB"];
      let size = Number(value);
      let unit = 0;
      while (size >= 1024 && unit < units.length - 1) {
        size /= 1024;
        unit += 1;
      }
      return size.toFixed(unit ? 1 : 0) + " " + units[unit];
    },
    BuildTraffic() {
      var extra = this.subscribe_item_info["extra"];
      return (
        ParseTraffic(
          extra["traffic"]["upload"] + extra["traffic"]["download"],
        ) +
        " / " +
        ParseTraffic(extra["traffic"]["total"])
      );
    },
    BuildExpireDate() {
      var expire = this.subscribe_item_info["extra"]["traffic"]["expire"];
      if (expire == 0) {
        return "-";
      }
      return moment.unix(expire).format("YYYY-MM-DD");
    },
    BuildPercent() {
      var extra = this.subscribe_item_info["extra"];
      if (this.isShowExtra()) {
        var p =
          (extra["traffic"]["upload"] + extra["traffic"]["download"]) /
          extra["traffic"]["total"];
        p = parseFloat(p) * 100;
        return parseInt(p);
      }
      return 0;
    },
    isShowExtra() {
      var extra = this.subscribe_item_info["extra"];
      if (
        extra != undefined &&
        extra != null &&
        extra["traffic"] != null &&
        extra["traffic"] != undefined &&
        extra["traffic"]["total"] != undefined &&
        extra["traffic"]["total"] != 0
      ) {
        return true;
      }
      return false;
    },
    hasOpenURL() {
      var extra = this.subscribe_item_info["extra"];
      if (
        extra != undefined &&
        extra != null &&
        extra["openWebURL"] != undefined &&
        extra["openWebURL"] != null &&
        extra["openWebURL"] != ""
      ) {
        return true;
      }
      return false;
    },
    LastUpdateTime() {
      return formatTime(this.subscribe_item_info.updateTime, "");
    },
    Config() {
      this.uif.subscribe.info = this.subscribe_item_info;
      this.uif.subscribe.isAdding = false;
      this.uif.subscribe.isOpenSub = true;
    },
    openWebURL() {
      window.open(this.subscribe_item_info["extra"]["openWebURL"], "_blank");
    },
    async Update() {
      this.uif.subscribe.info = this.subscribe_item_info;
      this.isLoading = true;
      var res = await this.UpdateSub();
      console.log(res);
      if (res) {
        this.$message({
          type: "success",
          message: "更新成功！",
        });
        if (this.isSimple && this.subscribe_item_info.enabled) {
          this.ChangeAllNodeStatus(true);
        } else {
          this.SaveUIFConfig();
        }
      }
      this.isLoading = false;
    },
    ShareMuti() {
      this.uif.share.isSingle = false;
      this.uif.share.info = this.subscribe_item_info;
      this.uif.share.isOpenShare = true;
    },
    SpeedTest() {
      uif_store.actions.TestNode(this.subscribe_item_info.outbounds, this.subscribe_item_info);
      this.$message({
        type: "info",
        message: "测速已启动，完成后会更新节点延迟；失败节点不会立即删除。",
      });
    },
    ChangeAllNodeStatus(res) {
      for (var item in this.subscribe_item_info.outbounds) {
        let row = this.subscribe_item_info.outbounds[item];
        if (row.transport.address.includes("127.0.0.1")) {
          continue;
        }
        row.enabled = res;
      }
      this.SaveUIFConfig();
      this.ApplyCoreConfig();
    },
    Delete() {
      this.$confirm("真的要删除该订阅?", "提示", {
        confirmButtonText: "确定",
        cancelButtonText: "取消",
        type: "warning",
      })
        .then(() => {
          var i = 0;
          for (var item in this.config.config.subscribe) {
            item = this.config.config.subscribe[item];
            if (item == this.subscribe_item_info) {
              this.config.config.subscribe.splice(i, 1);
              break;
            }
            i += 1;
          }
          this.SaveUIFConfig();
          this.ApplyCoreConfig();
        })
        .catch(() => {
          this.$message({
            type: "info",
            message: $translator({ cn: "取消删除", en: "Delete aborted." }),
          });
        });
    },
  },
};
</script>

<style>
.el-table .success-row {
  background: oldlace;
}
</style>
