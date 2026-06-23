<template>
  <div class="phone-register-task-manage">
    <div v-if="showTaskList" class="gva-search-box">
      <el-form :inline="true" :model="searchInfo">
        <el-form-item label="完成时间">
          <el-date-picker
            v-model="searchInfo.finishedAtRange"
            type="datetimerange"
            value-format="YYYY-MM-DD HH:mm:ss"
            :shortcuts="finishedAtShortcuts"
            range-separator="至"
            start-placeholder="开始时间"
            end-placeholder="结束时间"
            style="width: 360px"
          />
        </el-form-item>
        <el-form-item label="手机号">
          <el-input
            v-model="searchInfo.phone"
            clearable
            placeholder="请输入手机号"
            style="width: 180px"
          />
        </el-form-item>
        <el-form-item label="QQ号">
          <el-input
            v-model="searchInfo.qqNum"
            clearable
            placeholder="请输入QQ号"
            style="width: 180px"
          />
        </el-form-item>
        <el-form-item v-if="showLeaderFilter" label="团长">
          <el-select
            v-model="searchInfo.leaderId"
            clearable
            filterable
            style="width: 180px"
            placeholder="请选择团长"
            @change="onLeaderChange"
          >
            <el-option
              v-for="item in leaderOptions"
              :key="item.ID"
              :label="`${item.nickName}(${item.userName})`"
              :value="item.ID"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="地推">
          <el-select
            v-model="searchInfo.promoterId"
            clearable
            filterable
            style="width: 180px"
            placeholder="请选择地推"
          >
            <el-option
              v-for="item in promoterOptions"
              :key="item.ID"
              :label="`${item.nickName}(${item.userName})`"
              :value="item.ID"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="收码方式">
          <el-select
            v-model="searchInfo.smsReceiveMode"
            clearable
            style="width: 180px"
          >
            <el-option label="平台发码" value="PLATFORM_SEND" />
            <el-option label="用户转发到腾讯" value="USER_SENT_TO_TX" />
          </el-select>
        </el-form-item>
        <el-form-item label="状态">
          <el-select
            v-model="searchInfo.status"
            clearable
            style="width: 180px"
          >
            <el-option
              v-for="item in statusOptions"
              :key="item.value"
              :label="item.label"
              :value="item.value"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="缓存上报">
          <el-select
            v-model="searchInfo.cacheStatus"
            clearable
            style="width: 180px"
          >
            <el-option label="已上传" value="uploaded" />
            <el-option label="未上传" value="not_uploaded" />
          </el-select>
        </el-form-item>
        <el-form-item>
          <el-button type="primary" icon="search" @click="fetchAll">查询</el-button>
          <el-button icon="refresh" @click="resetSearch">重置</el-button>
        </el-form-item>
      </el-form>
    </div>

    <div class="gva-table-box">
      <el-row v-if="showCounters" :gutter="12" class="mb-3 counter-row">
        <el-col :span="6">
          <el-card shadow="never" class="counter-card">成功任务：{{ counters.success }}</el-card>
        </el-col>
        <el-col :span="6">
          <el-card shadow="never" class="counter-card">失败任务：{{ counters.fail }}</el-card>
        </el-col>
        <el-col :span="6">
          <el-card shadow="never" class="counter-card">处理中任务：{{ counters.processing }}</el-card>
        </el-col>
        <el-col :span="6">
          <el-card shadow="never" class="counter-card">当前设备：在线 {{ counters.deviceOnline }} / 空闲 {{ counters.deviceIdle }}</el-card>
        </el-col>
      </el-row>

      <template v-if="showTaskList">
        <el-table :data="tableData" row-key="ID" class="phone-task-table">
          <el-table-column label="任务ID" min-width="90" prop="ID" />
          <el-table-column label="创建时间" min-width="170">
            <template #default="scope">
              {{ safeFormatDate(scope.row.CreatedAt) }}
            </template>
          </el-table-column>
          <el-table-column label="手机号" min-width="140" prop="phone" />
          <el-table-column label="收码方式" min-width="130">
            <template #default="scope">
              {{ smsModeText(scope.row.smsReceiveMode) }}
            </template>
          </el-table-column>
          <el-table-column label="创建来源" min-width="110">
            <template #default="scope">
              {{ taskSourceText(scope.row.taskSource) }}
            </template>
          </el-table-column>
          <el-table-column label="状态" min-width="140">
            <template #default="scope">
              <el-tag :type="statusTagType(scope.row.status)">
                {{ statusText(scope.row.status) }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column label="缓存上报" min-width="120">
            <template #default="scope">
              <el-tag
                v-if="scope.row.cacheStatus"
                :type="cacheStatusTagType(scope.row.cacheStatus)"
              >
                {{ cacheStatusText(scope.row.cacheStatus) }}
              </el-tag>
              <span v-else>-</span>
            </template>
          </el-table-column>
          <el-table-column label="当前设备" min-width="140">
            <template #default="scope">
              {{ scope.row.holderDeviceId || '-' }}
            </template>
          </el-table-column>
          <el-table-column label="QQ号" min-width="120" prop="qqNum" />
          <el-table-column label="地推" min-width="120">
            <template #default="scope">
              {{ scope.row.promoter?.nickName || '-' }}
            </template>
          </el-table-column>
          <el-table-column label="团长" min-width="120">
            <template #default="scope">
              {{ scope.row.leader?.nickName || '-' }}
            </template>
          </el-table-column>
          <el-table-column label="失败原因" min-width="200" prop="lastError" show-overflow-tooltip />
          <el-table-column label="完成时间" min-width="170">
            <template #default="scope">
              {{ safeFormatDate(scope.row.finishedAt) }}
            </template>
          </el-table-column>
          <el-table-column label="操作" fixed="right" width="90">
            <template #default="scope">
              <el-button
                link
                type="primary"
                size="small"
                @click="openLogDialog(scope.row)"
              >
                日志
              </el-button>
            </template>
          </el-table-column>
        </el-table>

        <div class="gva-pagination">
          <el-pagination
            :current-page="page"
            :page-size="pageSize"
            :page-sizes="[10, 30, 50, 100]"
            :total="total"
            layout="total, sizes, prev, pager, next, jumper"
            @current-change="handleCurrentChange"
            @size-change="handleSizeChange"
          />
        </div>
      </template>

      <template v-if="showSummary">
        <el-divider v-if="showTaskList" />
        <el-row :gutter="12" class="summary-row">
          <el-col :span="12">
            <el-card shadow="never">
              <template #header>
                <span class="header-with-tip">
                  <span>团长汇总</span>
                  <el-tooltip
                    v-if="showDailyResetTip"
                    content="当前仅展示当天数据，每天 00:00 清空历史展示数据"
                    placement="top"
                  >
                    <span class="daily-reset-tip-icon">?</span>
                  </el-tooltip>
                </span>
              </template>
              <el-table :data="summary.leaders" size="small">
                <el-table-column label="团长ID" prop="leaderId" width="90" />
                <el-table-column label="团长名称" prop="leaderName" min-width="100" />
                <el-table-column label="成功" prop="successCount" width="80" />
                <el-table-column label="失败" prop="failCount" width="80" />
                <el-table-column label="处理中" prop="processingCount" width="90" />
                <el-table-column v-if="canSettle" label="已结算" prop="settledCount" width="90" />
                <el-table-column v-if="canSettle" label="待结算" prop="unsettledCount" width="90" />
                <el-table-column v-if="canSettle" label="操作" width="140" fixed="right">
                  <template #default="scope">
                    <el-button
                      link
                      type="primary"
                      size="small"
                      :disabled="!scope.row.unsettledCount"
                      @click="confirmSettleLeader(scope.row)"
                    >
                      结算
                    </el-button>
                    <el-button
                      link
                      type="primary"
                      size="small"
                      @click="openSettlementHistory(scope.row)"
                    >
                      历史
                    </el-button>
                  </template>
                </el-table-column>
              </el-table>
            </el-card>
          </el-col>
          <el-col :span="12">
            <el-card shadow="never">
              <template #header>
                <span class="header-with-tip">
                  <span>地推汇总</span>
                  <el-tooltip
                    v-if="showDailyResetTip"
                    content="当前仅展示当天数据，每天 00:00 清空历史展示数据"
                    placement="top"
                  >
                    <span class="daily-reset-tip-icon">?</span>
                  </el-tooltip>
                </span>
              </template>
              <el-table :data="summary.promoters" size="small">
                <el-table-column label="地推ID" prop="promoterId" width="90" />
                <el-table-column label="地推名称" prop="promoterName" min-width="100" />
                <el-table-column v-if="showPromoterLeaderColumn" label="所属团长" min-width="110">
                  <template #default="scope">
                    <span>{{ scope.row.leaderName || '-' }}</span>
                  </template>
                </el-table-column>
                <el-table-column label="成功" prop="successCount" width="80" />
                <el-table-column label="失败" prop="failCount" width="80" />
                <el-table-column v-if="canSettle" label="风控数" prop="riskFailCount" width="90" />
                <el-table-column label="处理中" prop="processingCount" width="90" />
                <el-table-column v-if="showAdminPromoterSummaryMetrics" label="总数" width="80">
                  <template #default="scope">
                    {{ promoterSummaryTotal(scope.row) }}
                  </template>
                </el-table-column>
                <el-table-column v-if="showAdminPromoterSummaryMetrics" label="成功率" width="90">
                  <template #default="scope">
                    {{ promoterSummarySuccessRate(scope.row) }}
                  </template>
                </el-table-column>
                <el-table-column v-if="showPromoterSettlementColumns" label="已结算" prop="settledCount" width="90" />
                <el-table-column v-if="showPromoterSettlementColumns" label="待结算" prop="unsettledCount" width="90" />
              </el-table>
            </el-card>
          </el-col>
        </el-row>
      </template>
    </div>

    <el-dialog
      v-model="logDialogVisible"
      :title="logDialogTitle"
      width="80vw"
      class="task-log-dialog"
      destroy-on-close
      @closed="stopLogRefresh"
    >
      <div v-loading="logPanelLoading" class="task-log-panel">
        <el-empty v-if="!taskLogs.length" description="暂无日志" />
        <div v-else class="task-log-list">
          <div class="task-log-line task-log-head">
            <span>时间</span>
            <span>设备</span>
            <span>内容</span>
          </div>
          <div
            v-for="item in taskLogs"
            :key="item.ID"
            class="task-log-line"
          >
            <span class="task-log-time">{{ safeFormatDate(item.clientTime) }}</span>
            <span class="task-log-device">{{ item.deviceId || '-' }}</span>
            <span class="task-log-message">{{ item.message }}</span>
          </div>
        </div>
      </div>
      <template #footer>
        <el-pagination
          v-model:current-page="logPage"
          v-model:page-size="logPageSize"
          :page-sizes="[50, 100, 200]"
          :total="logTotal"
          small
          layout="total, sizes, prev, pager, next"
          @current-change="fetchTaskLogs"
          @size-change="handleLogSizeChange"
        />
      </template>
    </el-dialog>

    <el-dialog v-model="settlementHistoryVisible" :title="settlementHistoryTitle" width="520px">
      <el-table v-loading="settlementHistoryLoading" :data="settlementHistory" size="small">
        <el-table-column label="结算时间" min-width="180">
          <template #default="scope">
            {{ safeFormatDate(scope.row.settledAt) }}
          </template>
        </el-table-column>
        <el-table-column label="数量" prop="settledCount" width="120" />
      </el-table>
    </el-dialog>
  </div>
</template>

<script setup>
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { getUserList } from '@/api/user'
import { getPhoneRegisterTaskList, getPhoneRegisterTaskLogs, getPhoneRegisterTaskSettlementHistory, getPhoneRegisterTaskSummary, settlePhoneRegisterTaskLeader } from '@/api/phoneRegisterTask'
import { formatDate } from '@/utils/format'
import { useUserStore } from '@/pinia/modules/user'

defineOptions({
  name: 'PhoneRegisterTaskManage'
})

const ROLE_SUPER = 888
const ROLE_ADMIN = 100
const ROLE_LEADER = 200
const ROLE_PROMOTER = 300

const page = ref(1)
const pageSize = ref(10)
const total = ref(0)
const tableData = ref([])
const logDialogVisible = ref(false)
const logLoading = ref(false)
const logTask = ref(null)
const taskLogs = ref([])
const logPage = ref(1)
const logPageSize = ref(100)
const logTotal = ref(0)
const logRefreshTimer = ref(null)
const settlementHistoryVisible = ref(false)
const settlementHistoryLoading = ref(false)
const settlementHistoryTitle = ref('结算历史')
const settlementHistory = ref([])
const leaderOptions = ref([])
const promoterOptions = ref([])
const counters = ref({
  success: 0,
  fail: 0,
  processing: 0,
  deviceOnline: 0,
  deviceIdle: 0
})
const summary = ref({
  leaders: [],
  promoters: []
})
const userStore = useUserStore()
const currentRoleId = computed(() => userStore.userInfo?.authority?.authorityId)
const currentUserId = computed(() => userStore.userInfo?.ID)
const showLeaderFilter = computed(() => [ROLE_SUPER, ROLE_ADMIN].includes(currentRoleId.value))
const showSummary = computed(() => [ROLE_SUPER, ROLE_ADMIN, ROLE_LEADER].includes(currentRoleId.value))
const showTaskList = computed(() => currentRoleId.value !== ROLE_LEADER)
const showCounters = computed(() => [ROLE_SUPER, ROLE_ADMIN].includes(currentRoleId.value))
const canSettle = computed(() => [ROLE_SUPER, ROLE_ADMIN].includes(currentRoleId.value))
const showPromoterLeaderColumn = computed(() => [ROLE_SUPER, ROLE_ADMIN].includes(currentRoleId.value))
const showAdminPromoterSummaryMetrics = computed(() => currentRoleId.value === ROLE_ADMIN)
const showPromoterSettlementColumns = computed(() => canSettle.value && currentRoleId.value !== ROLE_ADMIN)
const showDailyResetTip = computed(() => currentRoleId.value === ROLE_LEADER)
const logDialogTitle = computed(() => {
  if (!logTask.value) return '任务日志'
  return `任务日志 #${logTask.value.ID}`
})
const shouldRefreshTaskLogs = computed(() => {
  return logDialogVisible.value && logTask.value?.status === 'running'
})
const logPanelLoading = computed(() => logLoading.value && taskLogs.value.length === 0)

const searchInfo = ref({
  promoterId: undefined,
  leaderId: undefined,
  status: undefined,
  cacheStatus: undefined,
  smsReceiveMode: undefined,
  finishedAtRange: [],
  phone: '',
  qqNum: ''
})

const statusOptions = [
  { label: '待执行', value: 'pending' },
  { label: '执行中', value: 'running' },
  { label: '待地推验证码', value: 'waiting_promoter_code' },
  { label: '待上传缓存', value: 'registered_wait_upload' },
  { label: '成功', value: 'succeeded' },
  { label: '失败', value: 'failed' }
]

const dayStart = (base = new Date()) => {
  const d = new Date(base)
  d.setHours(0, 0, 0, 0)
  return d
}

const dayEnd = (base = new Date()) => {
  const d = new Date(base)
  d.setHours(23, 59, 59, 999)
  return d
}

const formatQueryDateTime = (date) => {
  const pad = (n) => String(n).padStart(2, '0')
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())} ${pad(date.getHours())}:${pad(date.getMinutes())}:${pad(date.getSeconds())}`
}

const todayRangeParams = () => {
  const now = new Date()
  return {
    finishedAtStart: formatQueryDateTime(dayStart(now)),
    finishedAtEnd: formatQueryDateTime(dayEnd(now)),
    dayScoped: true
  }
}

const todayDateTimeRange = () => {
  const now = new Date()
  return [formatQueryDateTime(dayStart(now)), formatQueryDateTime(dayEnd(now))]
}

const defaultFinishedAtRange = () => {
  return [ROLE_SUPER, ROLE_ADMIN].includes(currentRoleId.value) ? todayDateTimeRange() : []
}

const summaryQueryParams = () => {
  const [finishedAtStart, finishedAtEnd] = searchInfo.value.finishedAtRange || []
  const params = {
    leaderId: searchInfo.value.leaderId || undefined,
    finishedAtStart: finishedAtStart || undefined,
    finishedAtEnd: finishedAtEnd || undefined
  }
  if (currentRoleId.value === ROLE_LEADER) {
    Object.assign(params, finishedAtStart || finishedAtEnd ? { dayScoped: true } : todayRangeParams())
  }
  return params
}

const shiftDay = (base, days) => {
  const d = new Date(base)
  d.setDate(d.getDate() + days)
  return d
}

const finishedAtShortcuts = [
  {
    text: '今天',
    value: () => {
      const now = new Date()
      return [dayStart(now), now]
    }
  },
  {
    text: '昨天',
    value: () => {
      const now = new Date()
      const target = shiftDay(now, -1)
      return [dayStart(target), dayEnd(target)]
    }
  },
  {
    text: '前天',
    value: () => {
      const now = new Date()
      const target = shiftDay(now, -2)
      return [dayStart(target), dayEnd(target)]
    }
  },
  {
    text: '近一周',
    value: () => {
      const now = new Date()
      return [dayStart(shiftDay(now, -6)), now]
    }
  },
  {
    text: '近一月',
    value: () => {
      const now = new Date()
      return [dayStart(shiftDay(now, -29)), now]
    }
  }
]

const safeFormatDate = (value) => {
  if (!value) return '-'
  const ts = new Date(value).getTime()
  if (Number.isNaN(ts)) return '-'
  return formatDate(value)
}

const smsModeText = (mode) => {
  if (mode === 'PLATFORM_SEND') return '平台发码'
  if (mode === 'USER_SENT_TO_TX') return '自己发码'
  return mode || '-'
}

const taskSourceText = (source) => {
  if (source === 'OPENAPI') return 'OpenAPI'
  return '手动'
}

const statusText = (status) => {
  const map = {
    pending: '待执行',
    running: '执行中',
    waiting_promoter_code: '待地推验证码',
    registered_wait_upload: '待上传缓存',
    succeeded: '成功',
    failed: '失败'
  }
  return map[status] || status || '-'
}

const statusTagType = (status) => {
  if (status === 'succeeded') return 'success'
  if (status === 'failed') return 'danger'
  if (status === 'waiting_promoter_code') return 'warning'
  return 'info'
}

const cacheStatusText = (status) => {
  const map = {
    pending: '待上传',
    uploaded: '已上传',
    timeout: '超时未传'
  }
  return map[status] || status || '-'
}

const cacheStatusTagType = (status) => {
  if (status === 'uploaded') return 'success'
  if (status === 'timeout') return 'danger'
  if (status === 'pending') return 'warning'
  return 'info'
}

const promoterSummaryTotal = (row) => {
  return (Number(row?.successCount) || 0) + (Number(row?.failCount) || 0) + (Number(row?.processingCount) || 0)
}

const promoterSummarySuccessRate = (row) => {
  const total = promoterSummaryTotal(row)
  if (total <= 0) return '0%'
  return `${((Number(row?.successCount) || 0) * 100 / total).toFixed(2)}%`
}

const loadLeaderOptions = async () => {
  if (!showLeaderFilter.value) {
    leaderOptions.value = []
    return
  }
  const { data } = await getUserList({
    page: 1,
    pageSize: 1000,
    authorityId: ROLE_LEADER
  })
  leaderOptions.value = data?.list || []
}

const loadPromoterOptions = async () => {
  const params = {
    page: 1,
    pageSize: 1000,
    authorityId: ROLE_PROMOTER
  }
  if (currentRoleId.value === ROLE_LEADER) {
    params.leaderId = currentUserId.value
  } else if (showLeaderFilter.value && searchInfo.value.leaderId) {
    params.leaderId = searchInfo.value.leaderId
  }
  const { data } = await getUserList(params)
  promoterOptions.value = data?.list || []
}

const onLeaderChange = async () => {
  searchInfo.value.promoterId = undefined
  await loadPromoterOptions()
}

const fetchList = async () => {
  const [finishedAtStart, finishedAtEnd] = searchInfo.value.finishedAtRange || []
  const { data } = await getPhoneRegisterTaskList({
    page: page.value,
    pageSize: pageSize.value,
    promoterId: searchInfo.value.promoterId,
    leaderId: searchInfo.value.leaderId,
    status: searchInfo.value.status || undefined,
    cacheStatus: searchInfo.value.cacheStatus || undefined,
    smsReceiveMode: searchInfo.value.smsReceiveMode || undefined,
    phone: String(searchInfo.value.phone || '').trim() || undefined,
    qqNum: String(searchInfo.value.qqNum || '').trim() || undefined,
    finishedAtStart: finishedAtStart || undefined,
    finishedAtEnd: finishedAtEnd || undefined
  })
  tableData.value = data?.list || []
  total.value = data?.total || 0
  counters.value = {
    success: data?.successCount || 0,
    fail: data?.failCount || 0,
    processing: data?.processingCount || 0,
    deviceOnline: data?.deviceOnlineCount || 0,
    deviceIdle: data?.deviceIdleCount || 0
  }
}

const fetchSummary = async () => {
  if (!showSummary.value) {
    summary.value = { leaders: [], promoters: [] }
    return
  }
  const { data } = await getPhoneRegisterTaskSummary(summaryQueryParams())
  summary.value = data || { leaders: [], promoters: [] }
}

const confirmSettleLeader = async (row) => {
  if (!row?.leaderId || !row.unsettledCount) return
  try {
    await ElMessageBox.confirm(
      `确认结算团长 ${row.leaderName || row.leaderId} 的 ${row.unsettledCount} 个待结算任务？`,
      '确认结算',
      { type: 'warning' }
    )
    const [finishedAtStart, finishedAtEnd] = searchInfo.value.finishedAtRange || []
    const { data } = await settlePhoneRegisterTaskLeader({
      leaderId: row.leaderId,
      finishedAtStart: finishedAtStart || undefined,
      finishedAtEnd: finishedAtEnd || undefined
    })
    ElMessage.success(`已结算 ${data?.settledCount || 0} 个任务`)
    await fetchAll()
  } catch (e) {
    if (e !== 'cancel' && e !== 'close') {
      ElMessage.error(e?.message || '结算失败')
    }
  }
}

const openSettlementHistory = async (row) => {
  if (!row?.leaderId) return
  settlementHistoryTitle.value = `结算历史 - ${row.leaderName || row.leaderId}`
  settlementHistory.value = []
  settlementHistoryVisible.value = true
  settlementHistoryLoading.value = true
  try {
    const { data } = await getPhoneRegisterTaskSettlementHistory({ leaderId: row.leaderId })
    settlementHistory.value = data || []
  } catch (e) {
    ElMessage.error(e?.message || '结算历史加载失败')
  } finally {
    settlementHistoryLoading.value = false
  }
}

const fetchTaskLogs = async () => {
  if (!logTask.value?.ID) return
  logLoading.value = true
  try {
    const { data } = await getPhoneRegisterTaskLogs({
      taskId: logTask.value.ID,
      page: logPage.value,
      pageSize: logPageSize.value
    })
    taskLogs.value = data?.list || []
    logTotal.value = data?.total || 0
  } catch (e) {
    ElMessage.error(e?.message || '日志加载失败')
  } finally {
    logLoading.value = false
  }
}

const stopLogRefresh = () => {
  if (logRefreshTimer.value) {
    clearInterval(logRefreshTimer.value)
    logRefreshTimer.value = null
  }
}

const syncLogRefresh = () => {
  stopLogRefresh()
  if (!shouldRefreshTaskLogs.value) return
  logRefreshTimer.value = window.setInterval(async () => {
    if (!shouldRefreshTaskLogs.value) {
      stopLogRefresh()
      return
    }
    await fetchTaskLogs()
  }, 3000)
}

const openLogDialog = async (row) => {
  stopLogRefresh()
  logTask.value = row
  logPage.value = 1
  taskLogs.value = []
  logTotal.value = 0
  logDialogVisible.value = true
  await fetchTaskLogs()
  syncLogRefresh()
}

const handleLogSizeChange = async () => {
  logPage.value = 1
  await fetchTaskLogs()
}

const fetchAll = async () => {
  try {
    await Promise.all([fetchList(), fetchSummary()])
  } catch (e) {
    ElMessage.error(e?.message || '加载失败')
  }
}

const resetSearch = () => {
  searchInfo.value = {
    promoterId: undefined,
    leaderId: currentRoleId.value === ROLE_LEADER ? currentUserId.value : undefined,
    status: undefined,
    cacheStatus: undefined,
    smsReceiveMode: undefined,
    finishedAtRange: defaultFinishedAtRange(),
    phone: '',
    qqNum: ''
  }
  page.value = 1
  loadPromoterOptions()
  fetchAll()
}

const handleCurrentChange = (val) => {
  page.value = val
  fetchList()
}

const handleSizeChange = (val) => {
  pageSize.value = val
  page.value = 1
  fetchList()
}

onMounted(async () => {
  if (currentRoleId.value === ROLE_LEADER) {
    searchInfo.value.leaderId = currentUserId.value
  } else if ([ROLE_SUPER, ROLE_ADMIN].includes(currentRoleId.value)) {
    searchInfo.value.finishedAtRange = todayDateTimeRange()
  }
  await Promise.all([loadLeaderOptions(), loadPromoterOptions()])
  await fetchAll()
})

onBeforeUnmount(() => {
  stopLogRefresh()
})
</script>

<style scoped>
.task-log-panel {
  min-height: 420px;
}

.task-log-list {
  max-height: 680px;
  overflow: auto;
  padding: 0;
  background: #0f172a;
  border: 1px solid #1f2937;
  border-radius: 6px;
  color: #e5e7eb;
  font-family: Menlo, Monaco, Consolas, "Courier New", monospace;
  font-size: 12px;
  line-height: 1.45;
}

.task-log-line {
  display: grid;
  grid-template-columns: 150px 76px minmax(0, 1fr);
  gap: 12px;
  padding: 6px 10px;
  border-bottom: 1px solid rgba(255, 255, 255, 0.06);
}

.task-log-line:last-child {
  border-bottom: none;
}

.task-log-head {
  position: sticky;
  top: 0;
  z-index: 1;
  background: #111827;
  color: #9ca3af;
  font-weight: 600;
}

.task-log-time {
  color: #93c5fd;
}

.task-log-device {
  color: #fbbf24;
}

.task-log-message {
  white-space: pre-wrap;
  word-break: break-word;
}

@media (max-width: 768px) {
  .phone-register-task-manage :deep(.gva-search-box .el-form) {
    display: block;
  }

  .phone-register-task-manage :deep(.gva-search-box .el-form-item) {
    display: flex;
    margin-right: 0;
    margin-bottom: 8px;
  }

  .phone-register-task-manage :deep(.gva-search-box .el-form-item__label) {
    width: 72px;
    justify-content: flex-start;
    padding-right: 8px;
  }

  .phone-register-task-manage :deep(.gva-search-box .el-form-item__content) {
    flex: 1;
    min-width: 0;
  }

  .phone-register-task-manage :deep(.gva-search-box .el-input),
  .phone-register-task-manage :deep(.gva-search-box .el-select),
  .phone-register-task-manage :deep(.gva-search-box .el-date-editor) {
    width: 100% !important;
  }

  .counter-row :deep(.el-col) {
    flex: 0 0 50%;
    max-width: 50%;
    margin-bottom: 8px;
  }

  .counter-card :deep(.el-card__body) {
    min-height: 40px;
    padding: 8px;
    font-size: 12px;
    line-height: 1.25;
  }

  .summary-row > .el-col {
    flex: 0 0 100%;
    max-width: 100%;
    margin-bottom: 12px;
  }

  .phone-task-table :deep(.el-table__cell) {
    padding: 6px 0;
  }

  .phone-task-table :deep(.cell) {
    padding: 0 6px;
    line-height: 1.25;
  }

  .phone-register-task-manage :deep(.gva-pagination) {
    overflow-x: auto;
    padding-bottom: 4px;
  }

  .phone-register-task-manage :deep(.el-pagination) {
    min-width: max-content;
  }

  :deep(.task-log-dialog) {
    width: 94vw;
  }

  .task-log-panel {
    min-height: 300px;
  }

  .task-log-list {
    max-height: 60vh;
    font-size: 11px;
  }

  .task-log-line {
    grid-template-columns: 118px 62px minmax(180px, 1fr);
    gap: 8px;
    padding: 5px 8px;
  }

  :deep(.el-dialog) {
    max-width: 94vw;
  }
}

.header-with-tip {
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

.daily-reset-tip-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 16px;
  height: 16px;
  border-radius: 50%;
  color: var(--el-color-info);
  border: 1px solid var(--el-color-info-light-5);
  font-size: 12px;
  line-height: 1;
  cursor: help;
}
</style>
