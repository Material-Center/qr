<template>
  <div class="task-center compact-page">
    <el-card
      shadow="never"
      class="mb-3"
    >
      <div class="user-bar">
        <span>当前登录：{{ currentUser.nickName || '-' }}</span>
        <el-button
          type="danger"
          link
          @click="userStore.LoginOut"
        >退出登录</el-button>
      </div>
    </el-card>

    <el-card
      shadow="never"
      class="mb-3"
    >
      <template #header>创建注册任务</template>

      <el-form
        label-width="72px"
        class="compact-form"
      >
        <el-form-item label="手机号">
          <el-input
            v-model="phoneInput"
            placeholder="请输入手机号"
          />
        </el-form-item>
        <el-form-item>
          <el-button
            size="small"
            type="primary"
            @click="createTask"
          >创建任务</el-button>
        </el-form-item>
      </el-form>

      <div v-if="activeTasks.length">
        <el-divider class="my-2">当前可操作任务</el-divider>
        <el-card
          v-for="task in activeTasks"
          :key="task.id"
          shadow="never"
          class="mb-3"
        >
          <el-alert
            :title="taskTitle(task)"
            :description="task.stepHint || ''"
            type="info"
            show-icon
            :closable="false"
          />
          <el-form
            label-width="72px"
            class="mt-2 compact-form"
          >
            <el-form-item label="手机号">
              <el-input
                :model-value="task.phone"
                disabled
              />
            </el-form-item>
            <el-form-item :label="task.verifyLabel || '验证码'">
              <el-input
                v-model="verifyCodeMap[task.id]"
                :placeholder="task.verifyPlace || '请输入验证码'"
              />
            </el-form-item>
            <el-form-item>
              <div class="task-actions">
                <el-button
                  size="small"
                  type="primary"
                  class="action-btn"
                  @click="submitStep(task)"
                >{{ task.submitText || '提交' }}</el-button>
                <el-button
                  size="small"
                  class="action-btn"
                  @click="retryStep(task)"
                >{{ task.retryText || '重试' }}</el-button>
                <el-button
                  size="small"
                  type="danger"
                  plain
                  class="action-btn action-btn-danger"
                  @click="markFail(task)"
                >{{ task.failText || '失败' }}</el-button>
              </div>
            </el-form-item>
            <el-form-item
              v-if="task.lastError"
              label="最近错误"
            >
              <span class="text-red-500">{{ task.lastError }}</span>
            </el-form-item>
            <el-form-item label="过期时间">
              <span>{{ safeFormatDate(task.expiresAt) }}</span>
            </el-form-item>
          </el-form>
        </el-card>
      </div>
    </el-card>

    <el-card shadow="never">
      <template #header>我的任务记录</template>
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
            label="手机号"
            prop="phone"
            min-width="130"
          />
          <el-table-column
            label="状态"
            min-width="100"
          >
            <template #default="scope">
              <el-tag :type="statusTagType(scope.row)">
                {{ statusText(scope.row) }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column
            label="步骤"
            min-width="100"
          >
            <template #default="scope">
              {{ stepText(scope.row.currentStep) }}
            </template>
          </el-table-column>
          <el-table-column
            label="错误"
            prop="lastError"
            min-width="140"
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
  createRegisterTask,
  getActiveRegisterTasks,
  getRegisterTaskList,
  submitRegisterTaskStep
} from '@/api/registerTask'
import { formatDate } from '@/utils/format'
import { useUserStore } from '@/pinia/modules/user'

defineOptions({
  name: 'RegisterTaskCenter'
})

const phoneInput = ref('')
const activeTasks = ref([])
const verifyCodeMap = ref({})
const myTasks = ref([])
const userStore = useUserStore()
const currentUser = computed(() => userStore.userInfo || {})
const refreshTimer = ref(null)
const countdownTimer = ref(null)
const refreshing = ref(false)
const nowTs = ref(Date.now())
const counters = ref({
  success: 0,
  fail: 0,
  processing: 0
})

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

const taskTitle = (task) => {
  if (!task?.id) return '当前任务'
  const title = task?.stepTitle || stepText(task.currentStep)
  const progress = task?.progress
    ? ` ${task.progress}`
    : ''
  return `任务#${task.id} ${title}${progress} 剩余 ${remainText(task)}`
}

const loadActiveTask = async () => {
  const res = await getActiveRegisterTasks()
  activeTasks.value = Array.isArray(res.data) ? res.data : []
  const nextMap = {}
  activeTasks.value.forEach((task) => {
    nextMap[task.id] = verifyCodeMap.value?.[task.id] || ''
  })
  verifyCodeMap.value = nextMap
}

const loadMyTasks = async () => {
  const { data } = await getRegisterTaskList({
    page: 1,
    pageSize: 20
  })
  myTasks.value = data.list || []
  counters.value = {
    success: data.successCount || 0,
    fail: data.failCount || 0,
    processing: data.processingCount || 0
  }
}

const refreshAll = async () => {
  if (refreshing.value) return
  refreshing.value = true
  try {
    await Promise.all([loadActiveTask(), loadMyTasks()])
  } finally {
    refreshing.value = false
  }
}

const createTask = async () => {
  const phone = String(phoneInput.value || '').trim()
  if (!phone) {
    ElMessage.warning('请先输入手机号')
    return
  }
  const res = await createRegisterTask({ phone })
  if (res?.code !== 0) {
    ElMessage.error(res?.msg || '任务创建失败')
    return
  }
  ElMessage.success('任务创建成功')
  phoneInput.value = ''
  await refreshAll()
}

const submitStepCommon = async (task, payload) => {
  if (!task?.id) return
  await submitRegisterTaskStep({
    taskId: task.id,
    step: task.currentStep,
    ...payload
  })
  verifyCodeMap.value[task.id] = ''
  await refreshAll()
}

const submitStep = async (task) => {
  const needVerify = task?.needVerifyCode !== false
  const code = String(verifyCodeMap.value?.[task.id] || '').trim()
  const label = task?.verifyLabel || '验证码'
  if (needVerify && !code) {
    ElMessage.warning(`${label}不能为空`)
    return
  }
  await submitStepCommon(task, {
    action: 'submit',
    verifyCode: code
  })
}

const retryStep = async (task) => {
  await submitStepCommon(task, {
    action: 'retry'
  })
}

const markFail = async (task) => {
  await submitStepCommon(task, {
    action: 'fail',
    failMessage: '地推手动标记失败'
  })
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
    await refreshAll()
    startAutoRefresh()
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
.task-center {
  max-width: 720px;
  margin: 0 auto;
  padding: 8px;
  box-sizing: border-box;
  height: 100vh;
  min-height: 100vh;
  height: 100dvh;
  min-height: 100dvh;
  overflow-y: auto;
  -webkit-overflow-scrolling: touch;
  overscroll-behavior: contain;
}

.user-bar {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.compact-form :deep(.el-form-item) {
  margin-bottom: 10px;
}

.compact-form :deep(.el-form-item__label) {
  white-space: nowrap;
}

.compact-page :deep(.el-card__header) {
  padding: 10px 12px;
}

.compact-page :deep(.el-card__body) {
  padding: 10px;
}

.task-actions {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 8px;
  width: 100%;
}

.task-actions .action-btn {
  margin-left: 0 !important;
  width: 100%;
}

.task-actions .action-btn-danger {
  grid-column: 1 / -1;
}

.table-wrap {
  overflow-x: auto;
  -webkit-overflow-scrolling: touch;
}

@media (max-width: 768px) {
  .task-center {
    max-width: 100%;
    padding: 6px;
    padding-bottom: calc(6px + env(safe-area-inset-bottom, 0px));
  }

  .user-bar {
    font-size: 13px;
  }

  .compact-page :deep(.el-card__header) {
    padding: 8px 10px;
  }

  .compact-page :deep(.el-card__body) {
    padding: 8px;
  }

  .compact-form :deep(.el-form-item) {
    margin-bottom: 8px;
  }

  .compact-form :deep(.el-form-item__label) {
    width: 78px !important;
  }

  .compact-form :deep(.el-form-item__content) {
    min-width: 0;
  }

  .compact-page :deep(.el-alert__title) {
    font-size: 13px;
  }

  .compact-page :deep(.el-alert__description) {
    font-size: 12px;
    line-height: 1.35;
  }
}
</style>
