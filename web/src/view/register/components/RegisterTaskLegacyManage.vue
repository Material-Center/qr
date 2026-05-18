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
        <el-form-item label="状态">
          <el-select v-model="searchInfo.status" clearable style="width: 120px">
            <el-option value="success" label="成功" />
            <el-option value="fail" label="失败" />
          </el-select>
        </el-form-item>
        <el-form-item label="是否导出">
          <el-select v-model="searchInfo.exported" clearable style="width: 120px">
            <el-option :value="false" label="否" />
            <el-option :value="true" label="是" />
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
          <el-card shadow="never">成功登录QQ：{{ counters.success }}</el-card>
        </el-col>
        <el-col :span="8">
          <el-card shadow="never">失败任务：{{ counters.fail }}</el-card>
        </el-col>
        <el-col :span="8">
          <el-card shadow="never">处理中任务：{{ counters.processing }}</el-card>
        </el-col>
      </el-row>

      <template v-if="showTaskList">
        <div v-if="canDownloadCache" class="gva-btn-list">
          <el-button
            type="primary"
            :disabled="multipleSelection.length === 0"
            @click="openBatchDownloadDialog"
          >
            批量下载压缩包
          </el-button>
        </div>

        <el-table :data="tableData" row-key="ID" @selection-change="handleSelectionChange">
          <el-table-column v-if="canDownloadCache" type="selection" width="55" />
          <el-table-column label="任务ID" min-width="90" prop="ID" />
          <el-table-column label="手机号" min-width="140" prop="phone" />
          <el-table-column label="当前步骤" min-width="120">
            <template #default="scope">
              {{ stepText(scope.row.currentStep) }}
            </template>
          </el-table-column>
          <el-table-column label="状态" min-width="100">
            <template #default="scope">
              <el-tag :type="statusTagType(scope.row)">
                {{ statusText(scope.row) }}
              </el-tag>
            </template>
          </el-table-column>
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
          <el-table-column label="登录成功数" min-width="110" prop="loginSuccessCount" />
          <el-table-column label="失败原因" min-width="160" prop="lastError" show-overflow-tooltip />
          <el-table-column label="完成时间" min-width="170">
            <template #default="scope">
              {{ scope.row.finishedAt ? formatDate(scope.row.finishedAt) : '-' }}
            </template>
          </el-table-column>
          <el-table-column v-if="canDownloadCache" label="操作" width="130" fixed="right">
            <template #default="scope">
              <el-button
                type="primary"
                link
                :disabled="!scope.row.loginCacheIni"
                @click="openSingleDownloadDialog(scope.row)"
              >
                下载压缩包
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

      <el-divider />
      <el-row :gutter="12">
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
              <el-table-column label="成功QQ" prop="successCount" width="80" />
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
              <el-table-column label="成功QQ" prop="successCount" width="80" />
              <el-table-column label="失败" prop="failCount" width="80" />
              <el-table-column label="处理中" prop="processingCount" width="90" />
              <el-table-column v-if="canSettle" label="已结算" prop="settledCount" width="90" />
              <el-table-column v-if="canSettle" label="待结算" prop="unsettledCount" width="90" />
            </el-table>
          </el-card>
        </el-col>
      </el-row>
    </div>

    <el-dialog v-model="downloadDialogVisible" title="下载配置" width="420px">
      <el-form label-width="120px">
        <el-form-item label="下载任务数">
          <span>{{ downloadTaskIds.length }}</span>
        </el-form-item>
        <el-form-item label="仅下载缓存">
          <el-switch v-model="downloadOnlyCache" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="downloadDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="confirmDownloadZip">开始下载</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="settlementHistoryVisible" :title="settlementHistoryTitle" width="520px">
      <el-table v-loading="settlementHistoryLoading" :data="settlementHistory" size="small">
        <el-table-column label="结算时间" min-width="180">
          <template #default="scope">
            {{ formatDate(scope.row.settledAt) }}
          </template>
        </el-table-column>
        <el-table-column label="数量" prop="settledCount" width="120" />
      </el-table>
    </el-dialog>
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { downloadRegisterTaskCache, getRegisterTaskList, getRegisterTaskSettlementHistory, getRegisterTaskSummary, settleRegisterTaskLeader } from '@/api/registerTask'
import { getUserList } from '@/api/user'
import { formatDate } from '@/utils/format'
import { useUserStore } from '@/pinia/modules/user'

defineOptions({
  name: 'RegisterTaskLegacyManage'
})

const page = ref(1)
const pageSize = ref(10)
const total = ref(0)
const tableData = ref([])
const multipleSelection = ref([])
const ROLE_SUPER = 888
const ROLE_ADMIN = 100
const ROLE_LEADER = 200
const ROLE_PROMOTER = 300
const userStore = useUserStore()
const currentRoleId = computed(() => userStore.userInfo?.authority?.authorityId)
const currentUserId = computed(() => userStore.userInfo?.ID)
const canDownloadCache = computed(() => [ROLE_SUPER, ROLE_ADMIN].includes(currentRoleId.value))
const canSettle = computed(() => [ROLE_SUPER, ROLE_ADMIN].includes(currentRoleId.value))
const showLeaderFilter = computed(() => [ROLE_SUPER, ROLE_ADMIN].includes(currentRoleId.value))
const showTaskList = computed(() => currentRoleId.value !== ROLE_LEADER)
const showCounters = computed(() => [ROLE_SUPER, ROLE_ADMIN].includes(currentRoleId.value))
const showDailyResetTip = computed(() => currentRoleId.value === ROLE_LEADER)
const searchInfo = ref({
  promoterId: undefined,
  leaderId: undefined,
  status: 'success',
  exported: false,
  finishedAtRange: [],
  phone: ''
})
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
const downloadDialogVisible = ref(false)
const downloadOnlyCache = ref(false)
const downloadTaskIds = ref([])
const settlementHistoryVisible = ref(false)
const settlementHistoryLoading = ref(false)
const settlementHistoryTitle = ref('结算历史')
const settlementHistory = ref([])

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

const stepText = (step) => {
  if (step === 'phone_bind') return '查绑'
  if (step === 'change_password') return '改密'
  if (step === 'login') return '登录'
  return step || '-'
}

const statusText = (task) => {
  if (!task?.finishedAt) return '处理中'
  if (task?.statusCode === 0) return '成功'
  return '失败'
}

const statusTagType = (task) => {
  if (!task?.finishedAt) return 'warning'
  if (task?.statusCode === 0) return 'success'
  return 'danger'
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
  const { data } = await getRegisterTaskList({
    page: page.value,
    pageSize: pageSize.value,
    promoterId: searchInfo.value.promoterId,
    leaderId: searchInfo.value.leaderId,
    status: searchInfo.value.status || undefined,
    exported: searchInfo.value.exported,
    phone: String(searchInfo.value.phone || '').trim() || undefined,
    finishedAtStart: finishedAtStart || undefined,
    finishedAtEnd: finishedAtEnd || undefined
  })
  tableData.value = data.list || []
  total.value = data.total || 0
  counters.value = {
    success: data.successCount || 0,
    fail: data.failCount || 0,
    processing: data.processingCount || 0
  }
}

const fetchSummary = async () => {
  const { data } = await getRegisterTaskSummary(summaryQueryParams())
  summary.value = data || { leaders: [], promoters: [] }
}

const confirmSettleLeader = async (row) => {
  if (!row?.leaderId || !row.unsettledCount) return
  try {
    await ElMessageBox.confirm(
      `确认结算团长 ${row.leaderName || row.leaderId} 的 ${row.unsettledCount} 个待结算QQ？`,
      '确认结算',
      { type: 'warning' }
    )
    const { data } = await settleRegisterTaskLeader({ leaderId: row.leaderId })
    ElMessage.success(`已结算 ${data?.settledCount || 0} 个QQ`)
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
    const { data } = await getRegisterTaskSettlementHistory({ leaderId: row.leaderId })
    settlementHistory.value = data || []
  } catch (e) {
    ElMessage.error(e?.message || '结算历史加载失败')
  } finally {
    settlementHistoryLoading.value = false
  }
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
    status: 'success',
    exported: false,
    finishedAtRange: defaultFinishedAtRange(),
    phone: ''
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

const handleSelectionChange = (rows) => {
  multipleSelection.value = rows || []
}

const parseFilenameFromDisposition = (disposition) => {
  const source = String(disposition || '')
  if (!source) return ''
  const utfMatch = source.match(/filename\*=UTF-8''([^;]+)/i)
  if (utfMatch?.[1]) {
    try {
      return decodeURIComponent(utfMatch[1])
    } catch (e) {
      return utfMatch[1]
    }
  }
  const basicMatch = source.match(/filename="?([^";]+)"?/i)
  return basicMatch?.[1] || ''
}

const openSingleDownloadDialog = (row) => {
  if (!row?.ID) return
  downloadTaskIds.value = [row.ID]
  downloadOnlyCache.value = false
  downloadDialogVisible.value = true
}

const openBatchDownloadDialog = () => {
  const ids = (multipleSelection.value || []).map((item) => item?.ID).filter(Boolean)
  if (ids.length === 0) {
    ElMessage.warning('请先选择任务')
    return
  }
  downloadTaskIds.value = ids
  downloadOnlyCache.value = false
  downloadDialogVisible.value = true
}

const confirmDownloadZip = async () => {
  if (!downloadTaskIds.value.length) return
  try {
    const ids = Array.from(new Set(downloadTaskIds.value))
    const params = {
      onlyCache: downloadOnlyCache.value
    }
    if (ids.length === 1) {
      params.taskId = ids[0]
    } else {
      params.taskIds = ids.join(',')
    }
    const rsp = await downloadRegisterTaskCache(params)
    const contentType = String(rsp?.headers?.['content-type'] || rsp?.headers?.['Content-Type'] || '').toLowerCase()
    const buffer = rsp?.data
    if (contentType.includes('application/json')) {
      const text = new TextDecoder('utf-8').decode(buffer)
      try {
        const parsed = JSON.parse(text)
        ElMessage.error(parsed?.msg || '下载失败')
      } catch (e) {
        ElMessage.error(text || '下载失败')
      }
      return
    }
    const blob = new Blob([buffer], { type: 'application/zip' })
    const disposition = rsp?.headers?.['content-disposition'] || rsp?.headers?.['Content-Disposition']
    const filename = parseFilenameFromDisposition(disposition) || 'register_task_cache.zip'
    const url = window.URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = filename
    document.body.appendChild(a)
    a.click()
    a.remove()
    window.setTimeout(() => {
      window.URL.revokeObjectURL(url)
    }, 10000)
    ElMessage.success('压缩包下载成功')
    downloadDialogVisible.value = false
    await fetchList()
  } catch (e) {
    ElMessage.error(e?.message || '下载失败')
  }
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
</script>

<style scoped>
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
