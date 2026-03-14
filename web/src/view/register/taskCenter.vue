<template>
  <div class="task-center">
    <el-card shadow="never" class="mb-3">
      <template #header>创建注册任务（H5友好）</template>

      <div v-if="activeTask">
        <el-alert
          :title="`当前任务 #${activeTask.id}，步骤：${stepText(activeTask.currentStep)}`"
          type="info"
          show-icon
          :closable="false"
        />
        <el-form label-width="90px" class="mt-3">
          <el-form-item label="手机号">
            <el-input v-model="activeTask.phone" disabled />
          </el-form-item>
          <el-form-item label="验证码">
            <el-input v-model="verifyCode" placeholder="请输入验证码（mock）" />
          </el-form-item>
          <el-form-item>
            <el-button type="primary" @click="submitStep">提交当前步骤</el-button>
            <el-button @click="retryStep">重试</el-button>
            <el-button type="danger" plain @click="markFail">标记失败</el-button>
          </el-form-item>
          <el-form-item label="提示">
            <div class="text-sm text-gray-500">
              mock规则：验证码 `0000` 触发系统失败；`1111` 触发业务失败；其他值视为成功推进步骤。
            </div>
          </el-form-item>
          <el-form-item v-if="activeTask.lastError" label="最近错误">
            <span class="text-red-500">{{ activeTask.lastError }}</span>
          </el-form-item>
          <el-form-item label="过期时间">
            <span>{{ formatDate(activeTask.expiresAt) }}</span>
          </el-form-item>
        </el-form>
      </div>

      <el-form v-else label-width="90px">
        <el-form-item label="手机号">
          <el-input v-model="phone" placeholder="请输入手机号" />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="createTask">创建任务</el-button>
        </el-form-item>
      </el-form>
    </el-card>

    <el-card shadow="never">
      <template #header>我的任务记录</template>
      <el-row :gutter="12" class="mb-3">
        <el-col :span="8">成功：{{ counters.success }}</el-col>
        <el-col :span="8">失败：{{ counters.fail }}</el-col>
        <el-col :span="8">处理中：{{ counters.processing }}</el-col>
      </el-row>
      <el-table :data="myTasks" row-key="ID">
        <el-table-column label="任务ID" prop="ID" width="90" />
        <el-table-column label="手机号" prop="phone" min-width="130" />
        <el-table-column label="步骤" prop="currentStep" min-width="120" />
        <el-table-column label="状态码" prop="statusCode" width="90" />
        <el-table-column label="错误" prop="lastError" min-width="140" show-overflow-tooltip />
        <el-table-column label="完成时间" min-width="170">
          <template #default="scope">
            {{ scope.row.finishedAt ? formatDate(scope.row.finishedAt) : '-' }}
          </template>
        </el-table-column>
      </el-table>
    </el-card>
  </div>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { createRegisterTask, getActiveRegisterTask, getRegisterTaskList, submitRegisterTaskStep } from '@/api/registerTask'
import { formatDate } from '@/utils/format'

defineOptions({
  name: 'RegisterTaskCenter'
})

const phone = ref('')
const verifyCode = ref('')
const activeTask = ref(null)
const myTasks = ref([])
const counters = ref({
  success: 0,
  fail: 0,
  processing: 0
})

const stepText = (step) => {
  if (step === 'phone_bind') return '手机号绑定QQ'
  if (step === 'change_password') return '改密'
  if (step === 'login') return '登录并查达人'
  return step || '-'
}

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
  await Promise.all([loadActiveTask(), loadMyTasks()])
}

const createTask = async () => {
  if (!phone.value) {
    ElMessage.warning('请先输入手机号')
    return
  }
  await createRegisterTask({ phone: phone.value })
  ElMessage.success('任务创建成功')
  phone.value = ''
  await refreshAll()
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
  if (!verifyCode.value) {
    ElMessage.warning('请输入验证码')
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

onMounted(async () => {
  try {
    await refreshAll()
  } catch (e) {
    ElMessage.error(e?.message || '加载失败')
  }
})
</script>

<style scoped>
.task-center {
  max-width: 720px;
  margin: 0 auto;
}
</style>
