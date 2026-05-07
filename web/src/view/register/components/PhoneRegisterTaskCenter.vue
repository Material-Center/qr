<template>
  <div class="phone-center compact-page">
    <el-card
      shadow="never"
      class="mb-3"
    >
      <template #header>创建手机号注册任务</template>

      <el-form
        label-width="88px"
        class="compact-form"
      >
        <el-form-item label="手机号">
          <el-input
            v-model="phoneInput"
            placeholder="请输入手机号"
          />
        </el-form-item>
        <el-form-item label="是否发码">
          <div class="sms-switch-row">
            <el-switch
              v-model="isPlatformSend"
              inline-prompt
              active-text="是"
              inactive-text="否"
            />
            <span class="sms-switch-hint">
              {{ isPlatformSend ? '平台发码' : '我已发码' }}
            </span>
          </div>
        </el-form-item>
        <el-form-item>
          <el-button
            size="small"
            type="primary"
            @click="createTask"
          >创建任务</el-button>
          <el-button
            size="small"
            @click="refreshAll"
          >刷新</el-button>
        </el-form-item>
      </el-form>

      <div v-if="activeTasks.length">
        <el-divider class="my-2">当前任务</el-divider>
        <div class="active-task-grid">
          <el-card
            v-for="task in activeTasks"
            :key="task.id"
            shadow="never"
          >
            <div class="task-title-row">
              <div>
                <div class="task-title-line">
                  <span class="task-title">任务#{{ task.id }}</span>
                  <el-tag
                    size="small"
                    :type="statusTagType(task.status, task.finishedAt)"
                  >
                    {{ statusText(task.status) }}
                  </el-tag>
                </div>
                <div class="task-subtitle">
                  创建于 {{ safeFormatDate(task.createdAt) }}，剩余 {{ remainText(task) }}
                </div>
              </div>
            </div>

            <div class="task-brief">
              <span>手机号：{{ task.phone || '-' }}，收码方式：{{ smsModeText(task.smsReceiveMode) }}</span>
            </div>
            <div
              v-if="promoterErrorText(task)"
              class="task-error"
            >
              最近错误：{{ promoterErrorText(task) }}
            </div>

            <el-form
              v-if="task.needPromoterCode"
              label-width="88px"
              class="compact-form mt-3"
            >
              <el-form-item label="验证码">
                <el-input
                  v-model="verifyCodeMap[task.id]"
                  placeholder="请输入验证码"
                />
              </el-form-item>
              <el-form-item>
                <el-button
                  size="small"
                  type="primary"
                  @click="submitCode(task)"
                >
                  提交验证码
                </el-button>
              </el-form-item>
            </el-form>
          </el-card>
        </div>
      </div>
    </el-card>

    <el-card shadow="never">
      <template #header>我的手机号注册任务</template>
      <div class="task-list-toolbar">
        <el-select
          v-model="taskListStatus"
          size="small"
          class="status-filter"
          placeholder="注册状态"
          @change="handleStatusChange"
        >
          <el-option label="全部" value="" />
          <el-option label="执行中" value="processing" />
          <el-option label="成功" value="succeeded" />
          <el-option label="失败" value="failed" />
        </el-select>
      </div>
      <el-row
        :gutter="12"
        class="mb-3"
        style="font-size: 12px;"
      >
        <el-col :span="8">成功：{{ counters.success }}</el-col>
        <el-col :span="8">失败：{{ counters.fail }}</el-col>
        <el-col :span="8">处理中：{{ counters.processing }}</el-col>
      </el-row>
      <div class="table-wrap">
        <el-table
          :data="myTasks"
          row-key="ID"
          size="small"
          class="my-task-table"
        >
          <el-table-column
            label="任务ID"
            prop="ID"
            :width="taskColumnWidth.id"
          />
          <el-table-column
            label="创建时间"
            :min-width="taskColumnWidth.createdAt"
          >
            <template #default="scope">
              <span class="task-time-cell">{{ safeFormatDate(scope.row.CreatedAt) }}</span>
            </template>
          </el-table-column>
          <el-table-column
            label="手机号"
            prop="phone"
            :min-width="taskColumnWidth.phone"
          />
          <el-table-column
            label="收码方式"
            :min-width="taskColumnWidth.smsMode"
          >
            <template #default="scope">
              {{ smsModeText(scope.row.smsReceiveMode) }}
            </template>
          </el-table-column>
          <el-table-column
            label="状态"
            :min-width="taskColumnWidth.status"
          >
            <template #default="scope">
              <el-tag :type="statusTagType(scope.row.status, scope.row.finishedAt)">
                {{ statusText(scope.row.status) }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column
            label="QQ号"
            prop="qqNum"
            :min-width="taskColumnWidth.qqNum"
          />
          <el-table-column
            label="错误"
            :min-width="taskColumnWidth.error"
            show-overflow-tooltip
          >
            <template #default="scope">
              {{ promoterErrorText(scope.row) }}
            </template>
          </el-table-column>
          <el-table-column
            label="完成时间"
            :min-width="taskColumnWidth.finishedAt"
          >
            <template #default="scope">
              <span class="task-time-cell">{{ safeFormatDate(scope.row.finishedAt) }}</span>
            </template>
          </el-table-column>
        </el-table>
      </div>
      <div class="task-pagination">
        <el-pagination
          v-model:current-page="taskPage"
          v-model:page-size="taskPageSize"
          :page-sizes="[10, 20, 50]"
          :total="taskTotal"
          size="small"
          layout="total, sizes, prev, pager, next"
          @current-change="loadMyTasks"
          @size-change="handlePageSizeChange"
        />
      </div>
    </el-card>
  </div>
</template>

<script setup>
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import {
  createPhoneRegisterTask,
  getActivePhoneRegisterTasks,
  getPhoneRegisterTaskList,
  submitPhoneRegisterTaskCode
} from '@/api/phoneRegisterTask'
import { formatDate } from '@/utils/format'

defineOptions({
  name: 'PhoneRegisterTaskCenter'
})

const DEFAULT_SMS_RECEIVE_MODE = 'PLATFORM_SEND'
const SMS_RECEIVE_MODE_STORAGE_KEY = 'phoneRegisterTask.lastSmsReceiveMode'
const phoneInput = ref('')
const smsReceiveMode = ref(DEFAULT_SMS_RECEIVE_MODE)
const activeTasks = ref([])
const myTasks = ref([])
const taskListStatus = ref('processing')
const taskPage = ref(1)
const taskPageSize = ref(10)
const taskTotal = ref(0)
const verifyCodeMap = ref({})
const refreshTimer = ref(null)
const countdownTimer = ref(null)
const refreshing = ref(false)
const nowTs = ref(Date.now())
const windowWidth = ref(typeof window === 'undefined' ? 1024 : window.innerWidth)
const counters = ref({
  success: 0,
  fail: 0,
  processing: 0
})

const isMobile = computed(() => windowWidth.value <= 768)
const taskColumnWidth = computed(() => {
  if (isMobile.value) {
    return {
      id: 64,
      createdAt: 120,
      phone: 105,
      smsMode: 76,
      status: 88,
      qqNum: 92,
      error: 100,
      finishedAt: 120
    }
  }
  return {
    id: 90,
    createdAt: 170,
    phone: 130,
    smsMode: 120,
    status: 130,
    qqNum: 120,
    error: 160,
    finishedAt: 170
  }
})

const normalizeSmsReceiveMode = (value) => {
  if (value === 'PLATFORM_SEND' || value === 'USER_SENT_TO_TX') {
    return value
  }
  return DEFAULT_SMS_RECEIVE_MODE
}

const loadLastSmsReceiveMode = () => {
  if (typeof window === 'undefined') {
    return DEFAULT_SMS_RECEIVE_MODE
  }
  try {
    const stored = window.localStorage.getItem(SMS_RECEIVE_MODE_STORAGE_KEY)
    return normalizeSmsReceiveMode(stored)
  } catch (_error) {
    return DEFAULT_SMS_RECEIVE_MODE
  }
}

const persistSmsReceiveMode = (value) => {
  if (typeof window === 'undefined') {
    return
  }
  try {
    window.localStorage.setItem(
      SMS_RECEIVE_MODE_STORAGE_KEY,
      normalizeSmsReceiveMode(value)
    )
  } catch (_error) {
    // ignore storage write failures and keep page usable
  }
}

const safeFormatDate = (value) => {
  if (!value) return '-'
  const ts = new Date(value).getTime()
  if (Number.isNaN(ts)) return '-'
  return formatDate(value)
}

const remainSeconds = computed(() => {
  return (expiresAt, finishedAt) => {
    if (!expiresAt || finishedAt) return null
    const diff = Math.floor((new Date(expiresAt).getTime() - nowTs.value) / 1000)
    return diff > 0 ? diff : 0
  }
})

const remainText = (task) => {
  const seconds = remainSeconds.value(task?.expiresAt, task?.finishedAt)
  if (seconds === null) return '--:--'
  const m = Math.floor(seconds / 60)
  const s = seconds % 60
  return `${String(m).padStart(2, '0')}:${String(s).padStart(2, '0')}`
}

const smsModeText = (mode) => {
  if (mode === 'PLATFORM_SEND') return '平台发码'
  if (mode === 'USER_SENT_TO_TX') return '我已发码'
  return mode || '-'
}

const isPlatformSend = computed({
  get() {
    return smsReceiveMode.value === 'PLATFORM_SEND'
  },
  set(value) {
    smsReceiveMode.value = value ? 'PLATFORM_SEND' : 'USER_SENT_TO_TX'
    persistSmsReceiveMode(smsReceiveMode.value)
  }
})

const statusText = (status) => {
  const map = {
    pending: '待领取',
    running: '执行中',
    waiting_promoter_code: '待地推验证码',
    registered_wait_upload: '待上传缓存',
    succeeded: '成功',
    failed: '失败'
  }
  return map[status] || status || '-'
}

const statusTagType = (status, finishedAt) => {
  if (status === 'succeeded') return 'success'
  if (status === 'failed') return 'danger'
  if (status === 'waiting_promoter_code') return 'warning'
  if (!finishedAt) return 'info'
  return 'info'
}

const promoterErrorText = (task) => {
  const raw = String(task?.lastError || '').trim()
  if (!raw) return ''
  if (task?.status === 'failed') return '注册失败'
  if (
    raw.includes('没有触发') ||
    raw.includes('未找到') ||
    raw.includes('超时') ||
    raw.includes('异常') ||
    raw.includes('失败')
  ) {
    return '注册失败'
  }
  return raw
}

const loadActiveTasks = async () => {
  const res = await getActivePhoneRegisterTasks()
  activeTasks.value = Array.isArray(res.data) ? res.data : []
  const nextMap = {}
  activeTasks.value.forEach((task) => {
    nextMap[task.id] = verifyCodeMap.value?.[task.id] || ''
  })
  verifyCodeMap.value = nextMap
}

const loadMyTasks = async () => {
  const { data } = await getPhoneRegisterTaskList({
    page: taskPage.value,
    pageSize: taskPageSize.value,
    status: taskListStatus.value
  })
  myTasks.value = data?.list || []
  taskTotal.value = data?.total || 0
  counters.value = {
    success: data?.successCount || 0,
    fail: data?.failCount || 0,
    processing: data?.processingCount || 0
  }
}

const refreshAll = async () => {
  if (refreshing.value) return
  refreshing.value = true
  try {
    await Promise.all([loadActiveTasks(), loadMyTasks()])
  } finally {
    refreshing.value = false
    syncAutoRefresh()
  }
}

const handleStatusChange = async () => {
  taskPage.value = 1
  await loadMyTasks()
}

const handlePageSizeChange = async () => {
  taskPage.value = 1
  await loadMyTasks()
}

const createTask = async () => {
  const phone = String(phoneInput.value || '').trim()
  if (!phone) {
    ElMessage.warning('请先输入手机号')
    return
  }
  await createPhoneRegisterTask({
    phone,
    smsReceiveMode: smsReceiveMode.value
  })
  persistSmsReceiveMode(smsReceiveMode.value)
  ElMessage.success('手机号注册任务创建成功')
  phoneInput.value = ''
  await refreshAll()
}

const submitCode = async (task) => {
  const verifyCode = String(verifyCodeMap.value?.[task.id] || '').trim()
  if (!verifyCode) {
    ElMessage.warning('验证码不能为空')
    return
  }
  await submitPhoneRegisterTaskCode({
    taskId: task.id,
    verifyCode
  })
  verifyCodeMap.value[task.id] = ''
  ElMessage.success('验证码已提交')
  await refreshAll()
}

const startAutoRefresh = () => {
  stopAutoRefresh()
  refreshTimer.value = window.setInterval(async () => {
    await refreshAll()
  }, 5000)
}

const stopAutoRefresh = () => {
  if (refreshTimer.value) {
    clearInterval(refreshTimer.value)
    refreshTimer.value = null
  }
}

const syncAutoRefresh = () => {
  if (activeTasks.value.length > 0) {
    if (!refreshTimer.value) {
      startAutoRefresh()
    }
  } else {
    stopAutoRefresh()
  }
}

const handleWindowResize = () => {
  windowWidth.value = window.innerWidth
}

const startCountdown = () => {
  stopCountdown()
  countdownTimer.value = window.setInterval(() => {
    nowTs.value = Date.now()
  }, 1000)
}

const stopCountdown = () => {
  if (countdownTimer.value) {
    clearInterval(countdownTimer.value)
    countdownTimer.value = null
  }
}

onMounted(async () => {
  try {
    window.addEventListener('resize', handleWindowResize)
    handleWindowResize()
    smsReceiveMode.value = loadLastSmsReceiveMode()
    await refreshAll()
    startCountdown()
  } catch (e) {
    ElMessage.error(e?.message || '加载失败')
  }
})

onBeforeUnmount(() => {
  window.removeEventListener('resize', handleWindowResize)
  stopAutoRefresh()
  stopCountdown()
})
</script>

<style scoped>
.phone-center {
  padding-top: 8px;
}

.compact-form :deep(.el-form-item) {
  margin-bottom: 10px;
}

.compact-page :deep(.el-card__header) {
  padding: 10px 12px;
}

.compact-page :deep(.el-card__body) {
  padding: 10px;
}

.sms-switch-row {
  display: flex;
  align-items: center;
  gap: 10px;
}

.sms-switch-hint {
  color: #606266;
  font-size: 13px;
}

.task-title-row {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
}

.active-task-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 10px;
}

.task-title-line {
  display: flex;
  align-items: center;
  gap: 8px;
}

.task-title {
  font-size: 15px;
  font-weight: 600;
}

.task-subtitle {
  margin-top: 4px;
  color: var(--el-text-color-secondary);
  font-size: 12px;
}

.task-brief {
  margin-top: 8px;
  color: var(--el-text-color-regular);
  font-size: 13px;
  line-height: 1.4;
}

.task-error {
  margin-top: 6px;
  color: var(--el-color-danger);
  font-size: 12px;
  line-height: 1.4;
  word-break: break-word;
}

.task-list-toolbar {
  display: flex;
  justify-content: flex-end;
  margin-bottom: 10px;
}

.status-filter {
  width: 140px;
}

.table-wrap {
  overflow-x: auto;
  -webkit-overflow-scrolling: touch;
}

.task-pagination {
  display: flex;
  justify-content: flex-end;
  margin-top: 10px;
}

.task-time-cell {
  line-height: 1;
  display: block;
}

@media (max-width: 768px) {
  .compact-page :deep(.el-card__header) {
    padding: 8px 10px;
  }

  .compact-page :deep(.el-card__body) {
    padding: 8px;
  }

  .task-title-row {
    flex-direction: column;
    align-items: stretch;
  }

  .active-task-grid {
    grid-template-columns: 1fr;
  }

  .task-list-toolbar,
  .task-pagination {
    justify-content: flex-start;
  }

  .status-filter {
    width: 120px;
  }

  .my-task-table {
    font-size: 12px;
  }

  .my-task-table :deep(.el-table__cell) {
    padding: 4px 0;
  }

  .my-task-table :deep(.cell) {
    padding: 0 4px;
    line-height: 1.3;
  }

  .my-task-table :deep(.el-tag) {
    height: 20px;
    padding: 0 5px;
    font-size: 11px;
  }
}
</style>
