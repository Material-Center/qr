<template>
  <div>
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
        <el-form-item>
          <el-button type="primary" icon="search" @click="fetchAll">查询</el-button>
          <el-button icon="refresh" @click="resetSearch">重置</el-button>
        </el-form-item>
      </el-form>
    </div>

    <div class="gva-table-box">
      <el-row v-if="showCounters" :gutter="12" class="mb-3">
        <el-col :span="8">
          <el-card shadow="never">成功任务：{{ counters.success }}</el-card>
        </el-col>
        <el-col :span="8">
          <el-card shadow="never">失败任务：{{ counters.fail }}</el-card>
        </el-col>
        <el-col :span="8">
          <el-card shadow="never">处理中任务：{{ counters.processing }}</el-card>
        </el-col>
      </el-row>

      <template v-if="showTaskList">
        <el-table :data="tableData" row-key="ID">
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
          <el-table-column label="状态" min-width="140">
            <template #default="scope">
              <el-tag :type="statusTagType(scope.row.status)">
                {{ statusText(scope.row.status) }}
              </el-tag>
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
        <el-row :gutter="12">
          <el-col :span="12">
            <el-card shadow="never">
              <template #header>团长汇总</template>
              <el-table :data="summary.leaders" size="small">
                <el-table-column label="团长ID" prop="leaderId" width="90" />
                <el-table-column label="团长名称" prop="leaderName" min-width="100" />
                <el-table-column label="成功" prop="successCount" width="80" />
                <el-table-column label="失败" prop="failCount" width="80" />
                <el-table-column label="处理中" prop="processingCount" width="90" />
                <el-table-column v-if="canSettle" label="已结算" prop="settledCount" width="90" />
                <el-table-column v-if="canSettle" label="待结算" prop="unsettledCount" width="90" />
                <el-table-column v-if="canSettle" label="操作" width="90" fixed="right">
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
                  </template>
                </el-table-column>
              </el-table>
            </el-card>
          </el-col>
          <el-col :span="12">
            <el-card shadow="never">
              <template #header>地推汇总</template>
              <el-table :data="summary.promoters" size="small">
                <el-table-column label="地推ID" prop="promoterId" width="90" />
                <el-table-column label="地推名称" prop="promoterName" min-width="100" />
                <el-table-column label="成功" prop="successCount" width="80" />
                <el-table-column label="失败" prop="failCount" width="80" />
                <el-table-column label="处理中" prop="processingCount" width="90" />
                <el-table-column v-if="canSettle" label="已结算" prop="settledCount" width="90" />
                <el-table-column v-if="canSettle" label="待结算" prop="unsettledCount" width="90" />
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
  </div>
</template>

<script setup>
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { getUserList } from '@/api/user'
import { getPhoneRegisterTaskList, getPhoneRegisterTaskLogs, getPhoneRegisterTaskSummary, settlePhoneRegisterTaskLeader } from '@/api/phoneRegisterTask'
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
const leaderOptions = ref([])
const promoterOptions = ref([])
const counters = ref({
  success: 0,
  fail: 0,
  processing: 0
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

const summaryQueryParams = () => {
  const params = {
    leaderId: searchInfo.value.leaderId || undefined
  }
  if (currentRoleId.value === ROLE_LEADER) {
    Object.assign(params, todayRangeParams())
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
    processing: data?.processingCount || 0
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
    const { data } = await settlePhoneRegisterTaskLeader({ leaderId: row.leaderId })
    ElMessage.success(`已结算 ${data?.settledCount || 0} 个任务`)
    await fetchAll()
  } catch (e) {
    if (e !== 'cancel' && e !== 'close') {
      ElMessage.error(e?.message || '结算失败')
    }
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
    smsReceiveMode: undefined,
    finishedAtRange: [],
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
  :deep(.task-log-dialog) {
    width: 94vw;
  }
}
</style>
