<template>
  <div class="register-debug-login">
    <el-card shadow="never" style="margin-top: 8px;">
      <template #header>注册登录调试（管理员）</template>

      <el-form label-width="130px">
        <el-form-item label="手机号">
          <el-input v-model="form.phone" placeholder="用于代理地区解析，例如 19146017340" />
        </el-form-item>
        <el-form-item label="UIN">
          <el-input v-model="form.uin" placeholder="QQ号（纯数字）" />
        </el-form-item>
        <el-form-item label="密码">
          <el-input v-model="form.password" show-password placeholder="QQ密码" />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" :loading="starting" @click="startDebug">启动登录调试</el-button>
          <el-button :disabled="!taskId" @click="refreshTask">刷新任务状态</el-button>
        </el-form-item>
      </el-form>
    </el-card>

    <el-card v-if="taskId" shadow="never" style="margin-top: 8px;">
      <template #header>调试任务状态（#{{ taskId }}）</template>
      <el-descriptions :column="1" border>
        <el-descriptions-item label="步骤">{{ task.currentStep || '-' }}</el-descriptions-item>
        <el-descriptions-item label="状态提示">{{ task.stepHint || '-' }}</el-descriptions-item>
        <el-descriptions-item label="最近错误">{{ task.lastError || '-' }}</el-descriptions-item>
        <el-descriptions-item label="是否需验证码">{{ task.needVerifyCode ? '是' : '否' }}</el-descriptions-item>
        <el-descriptions-item label="提交按钮文案">{{ task.submitText || '-' }}</el-descriptions-item>
      </el-descriptions>

      <el-form label-width="130px" style="margin-top: 12px;">
        <el-form-item :label="task.verifyLabel || '验证码'">
          <el-input v-model="verifyCode" :placeholder="task.verifyPlace || '请输入验证码'" />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" :loading="submitting" :disabled="!taskId" @click="submitCode">提交验证码</el-button>
        </el-form-item>
      </el-form>
    </el-card>
  </div>
</template>

<script setup>
import { ref, onBeforeUnmount } from 'vue'
import { ElMessage } from 'element-plus'
import {
  getRegisterTaskDebugLoginTask,
  startRegisterTaskDebugLogin,
  submitRegisterTaskDebugLoginCode
} from '@/api/registerTask'

defineOptions({
  name: 'RegisterDebugLogin'
})

const form = ref({
  phone: '',
  uin: '',
  password: ''
})
const taskId = ref(0)
const task = ref({})
const verifyCode = ref('')
const starting = ref(false)
const submitting = ref(false)
const timer = ref(null)

const stopAutoRefresh = () => {
  if (timer.value) {
    clearInterval(timer.value)
    timer.value = null
  }
}

const startAutoRefresh = () => {
  stopAutoRefresh()
  timer.value = setInterval(async () => {
    if (!taskId.value) return
    await refreshTask()
  }, 5000)
}

const startDebug = async () => {
  const phone = String(form.value.phone || '').trim()
  const uin = String(form.value.uin || '').trim()
  const password = String(form.value.password || '').trim()
  if (!phone || !uin || !password) {
    ElMessage.warning('手机号、UIN、密码不能为空')
    return
  }
  if (!/^\d+$/.test(uin)) {
    ElMessage.warning('UIN必须为纯数字QQ号')
    return
  }
  starting.value = true
  try {
    const res = await startRegisterTaskDebugLogin({ phone, uin, password })
    const data = res?.data || {}
    task.value = data
    taskId.value = data.id || 0
    verifyCode.value = ''
    ElMessage.success('已启动调试任务')
    startAutoRefresh()
  } finally {
    starting.value = false
  }
}

const refreshTask = async () => {
  if (!taskId.value) return
  const res = await getRegisterTaskDebugLoginTask({ taskId: taskId.value })
  task.value = res?.data || {}
}

const submitCode = async () => {
  if (!taskId.value) return
  const code = String(verifyCode.value || '').trim()
  if (!code) {
    ElMessage.warning('验证码不能为空')
    return
  }
  submitting.value = true
  try {
    await submitRegisterTaskDebugLoginCode({
      taskId: taskId.value,
      verifyCode: code
    })
    verifyCode.value = ''
    ElMessage.success('验证码已提交')
    await refreshTask()
  } finally {
    submitting.value = false
  }
}

onBeforeUnmount(() => {
  stopAutoRefresh()
})
</script>

<style scoped>
.register-debug-login {
  max-width: 820px;
}
</style>
