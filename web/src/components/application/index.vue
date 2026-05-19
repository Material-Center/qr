<template>
  <error-preview v-if="showError" :error-data="errorInfo" @close="handleClose" @confirm="handleConfirm" />
</template>

<script setup>
import { ref, onUnmounted } from 'vue'
import { ElMessage } from 'element-plus'
import { emitter } from '@/utils/bus'
import ErrorPreview from '@/components/errorPreview/index.vue'

const showError = ref(false)
const errorInfo = ref(null)
let cb = null
let lastToastKey = ''
let lastToastAt = 0

const showErrorToast = (data) => {
  const message = data?.message || '请求失败'
  const key = `${data?.code || ''}:${message}`
  const now = Date.now()
  if (key === lastToastKey && now - lastToastAt < 3000) return
  lastToastKey = key
  lastToastAt = now
  ElMessage({
    showClose: true,
    message,
    type: 'error'
  })
}

const showErrorDialog = (data) => {
  const tipOnlyCodes = ['network', 'request', 408, 502, 503, 504, '408', '502', '503', '504']
  const message = String(data?.message || '')
  const isTimeoutMessage = /timeout|time-out|gateway time-out/i.test(message)
  if (tipOnlyCodes.includes(data?.code) || isTimeoutMessage) {
    showErrorToast(data)
    return
  }
  // 这玩意同时只允许存在一个
  if(showError.value) return

  errorInfo.value = data
  showError.value = true
  cb = data?.fn || null
}

const handleClose = () => {
  showError.value = false
  errorInfo.value = null
  cb = null
}

const handleConfirm = (code) => {
  cb && cb(code)
  handleClose()
}

emitter.on('show-error', showErrorDialog)

onUnmounted(() => {
  emitter.off('show-error', showErrorDialog)
})
</script>
