<template>
  <div class="task-center compact-page">
    <el-card
      shadow="never"
      class="mb-3"
    >
      <div class="user-bar">
        <span>当前登录用户：{{ currentUser.nickName || '-' }}</span>
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

      <div v-if="activeTask">
        <el-alert
          :title="taskTitle"
          :description="taskHint"
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
              v-model="activeTask.phone"
              disabled
            />
          </el-form-item>
          <el-form-item :label="verifyLabel">
            <el-input
              v-model="verifyCode"
              :placeholder="verifyPlaceholder"
            />
          </el-form-item>
          <el-form-item>
            <el-button
              size="small"
              type="primary"
              @click="submitStep"
            >{{ submitText }}</el-button>
            <el-button
              size="small"
              @click="retryStep"
            >{{ retryText }}</el-button>
            <el-button
              size="small"
              type="danger"
              plain
              @click="markFail"
            >{{ failText }}</el-button>
          </el-form-item>
          <el-form-item
            v-if="activeTask.lastError"
            label="最近错误"
          >
            <span class="text-red-500">{{ activeTask.lastError }}</span>
          </el-form-item>
          <el-form-item label="过期时间">
            <span>{{ safeFormatDate(activeTask.expiresAt) }}</span>
          </el-form-item>
        </el-form>
      </div>

      <el-form
        v-else
        label-width="72px"
        class="compact-form"
      >
        <el-form-item label="手机号">
          <el-input
            v-model="phone"
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
    </el-card>
  </div>
</template>

<script setup>
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import {
  createRegisterTask,
  getActiveRegisterTask,
  getRegisterTaskList,
  submitRegisterTaskStep
} from '@/api/registerTask'
import { formatDate } from '@/utils/format'
import { useUserStore } from '@/pinia/modules/user'

defineOptions({
  name: 'RegisterTaskCenter'
})

const phone = ref('')
const verifyCode = ref('')
const activeTask = ref(null)
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

const verifyLabel = computed(() => activeTask.value?.verifyLabel || '验证码')
const verifyPlaceholder = computed(
  () => activeTask.value?.verifyPlace || '请输入验证码'
)
const submitText = computed(() => activeTask.value?.submitText || '提交')
const retryText = computed(() => activeTask.value?.retryText || '重试')
const failText = computed(() => activeTask.value?.failText || '失败')
const taskHint = computed(() => activeTask.value?.stepHint || '')
const needVerifyCode = computed(() => activeTask.value?.needVerifyCode !== false)

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
  if (!activeTask.value?.expiresAt || activeTask.value?.finishedAt) return null
  const diff = Math.floor(
    (new Date(activeTask.value.expiresAt).getTime() - nowTs.value) / 1000
  )
  return diff > 0 ? diff : 0
})

const remainText = computed(() => {
  if (remainSeconds.value === null) return '--:--'
  const m = Math.floor(remainSeconds.value / 60)
  const s = remainSeconds.value % 60
  return `${String(m).padStart(2, '0')}:${String(s).padStart(2, '0')}`
})

const taskTitle = computed(() => {
  if (!activeTask.value?.id) return '当前任务'
  const title =
    activeTask.value?.stepTitle || stepText(activeTask.value.currentStep)
  const progress = activeTask.value?.progress
    ? `，${activeTask.value.progress}`
    : ''
  return `当前任务 #${activeTask.value.id}，${title}${progress}，剩余：${remainText.value}`
})

const loadActiveTask = async () => {
  const res = await getActiveRegisterTask()
  activeTask.value = res.data || null
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
  if (!phone.value) {
    ElMessage.warning('请先输入手机号')
    return
  }
  const res = await createRegisterTask({ phone: phone.value })
  if (res?.code !== 0) {
    ElMessage.error(res?.msg || '任务创建失败')
    return
  }
  ElMessage.success('任务创建成功')
  phone.value = ''
  await refreshAll()
  // 创建后可能存在短暂写库/状态同步延迟，补一次短轮询确保拿到最新进度
  if (!activeTask.value?.id) {
    for (let i = 0; i < 3; i++) {
      await new Promise((resolve) => window.setTimeout(resolve, 800))
      await loadActiveTask()
      if (activeTask.value?.id) break
    }
  }
}

const submitStepCommon = async (payload) => {
  if (!activeTask.value) return
  const { data } = await submitRegisterTaskStep({
    taskId: activeTask.value.id,
    step: activeTask.value.currentStep,
    ...payload
  })
  activeTask.value = data?.finishedAt ? null : data
  verifyCode.value = ''
  await loadMyTasks()
}

const submitStep = async () => {
  if (needVerifyCode.value && !verifyCode.value) {
    ElMessage.warning(`${verifyLabel.value}不能为空`)
    return
  }
  await submitStepCommon({
    action: 'submit',
    verifyCode: verifyCode.value
  })
}

const retryStep = async () => {
  await submitStepCommon({
    action: 'retry'
  })
}

const markFail = async () => {
  await submitStepCommon({
    action: 'fail',
    failMessage: '地推手动标记失败'
  })
}

const startAutoRefresh = () => {
  stopAutoRefresh()
  refreshTimer.value = window.setInterval(async () => {
    if (activeTask.value?.id && !activeTask.value?.finishedAt) {
      const isTimeout = activeTask.value?.expiresAt
        ? Date.now() >= new Date(activeTask.value.expiresAt).getTime()
        : false
      if (isTimeout) {
        await refreshAll()
        return
      }
    }
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
}

.user-bar {
  display: flex;
  justify-content: space-between;
  align-items: center;
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

@media (max-width: 768px) {
  .task-center {
    max-width: 100%;
    padding: 6px;
  }
}
</style>
