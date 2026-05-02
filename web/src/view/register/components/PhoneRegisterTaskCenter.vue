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
        <el-card
          v-for="task in activeTasks"
          :key="task.id"
          shadow="never"
          class="mb-3"
        >
          <div class="task-title-row">
            <div>
              <div class="task-title">任务#{{ task.id }}</div>
              <div class="task-subtitle">
                创建于 {{ safeFormatDate(task.createdAt) }}，剩余 {{ remainText(task) }}
              </div>
            </div>
            <el-tag :type="statusTagType(task.status, task.finishedAt)">
              {{ statusText(task.status) }}
            </el-tag>
          </div>

          <el-descriptions
            :column="1"
            border
            size="small"
            class="mt-3"
          >
            <el-descriptions-item label="手机号">
              {{ task.phone || '-' }}
            </el-descriptions-item>
            <el-descriptions-item label="收码方式">
              {{ smsModeText(task.smsReceiveMode) }}
            </el-descriptions-item>
            <el-descriptions-item label="设备">
              {{ task.holderDeviceId || '-' }}
            </el-descriptions-item>
            <el-descriptions-item label="最近错误">
              <span :class="{ 'text-red-500': task.lastError }">{{ task.lastError || '-' }}</span>
            </el-descriptions-item>
          </el-descriptions>

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
              >提交验证码</el-button>
            </el-form-item>
          </el-form>
        </el-card>
      </div>
    </el-card>

    <el-card shadow="never">
      <template #header>我的手机号注册任务</template>
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
        >
          <el-table-column
            label="任务ID"
            prop="ID"
            width="90"
          />
          <el-table-column
            label="创建时间"
            min-width="170"
          >
            <template #default="scope">
              {{ safeFormatDate(scope.row.CreatedAt) }}
            </template>
          </el-table-column>
          <el-table-column
            label="手机号"
            prop="phone"
            min-width="130"
          />
          <el-table-column
            label="收码方式"
            min-width="120"
          >
            <template #default="scope">
              {{ smsModeText(scope.row.smsReceiveMode) }}
            </template>
          </el-table-column>
          <el-table-column
            label="状态"
            min-width="130"
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
            min-width="120"
          />
          <el-table-column
            label="设备"
            min-width="140"
          >
            <template #default="scope">
              {{ scope.row.holderDeviceId || '-' }}
            </template>
          </el-table-column>
          <el-table-column
            label="错误"
            prop="lastError"
            min-width="160"
            show-overflow-tooltip
          />
          <el-table-column
            label="完成时间"
            min-width="170"
          >
            <template #default="scope">
              {{ safeFormatDate(scope.row.finishedAt) }}
            </template>
          </el-table-column>
        </el-table>
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
const verifyCodeMap = ref({})
const refreshTimer = ref(null)
const countdownTimer = ref(null)
const refreshing = ref(false)
const nowTs = ref(Date.now())
const counters = ref({
  success: 0,
  fail: 0,
  processing: 0
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
    page: 1,
    pageSize: 20
  })
  myTasks.value = data?.list || []
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
    smsReceiveMode.value = loadLastSmsReceiveMode()
    await refreshAll()
    startCountdown()
  } catch (e) {
    ElMessage.error(e?.message || '加载失败')
  }
})

onBeforeUnmount(() => {
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

.task-title {
  font-size: 15px;
  font-weight: 600;
}

.task-subtitle {
  margin-top: 4px;
  color: var(--el-text-color-secondary);
  font-size: 12px;
}

.table-wrap {
  overflow-x: auto;
  -webkit-overflow-scrolling: touch;
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
}
</style>
