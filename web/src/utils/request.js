import axios from 'axios' // 引入axios
import SparkMD5 from 'spark-md5'
import { useUserStore } from '@/pinia/modules/user'
import { ElLoading, ElMessage } from 'element-plus'
import { emitter } from '@/utils/bus'
import router from '@/router/index'

const service = axios.create({
  timeout: 180000
})

const fromB64 = (value) => {
  let bits = 0
  let size = 0
  let output = ''
  for (const item of value.replace(/=+$/, '')) {
    const code = item.charCodeAt(0)
    let chunk = -1
    if (code >= 65 && code <= 90) {
      chunk = code - 65
    } else if (code >= 97 && code <= 122) {
      chunk = code - 71
    } else if (code >= 48 && code <= 57) {
      chunk = code + 4
    } else if (item === '+' || item === '-') {
      chunk = 62
    } else if (item === '/' || item === '_') {
      chunk = 63
    }
    if (chunk < 0) {
      continue
    }
    bits = (bits << 6) | chunk
    size += 6
    if (size >= 8) {
      size -= 8
      output += String.fromCharCode((bits >> size) & 0xff)
    }
  }
  return output
}

const requestProofSeed = fromB64('cXItd2ViLXJlcXVlc3Qtc2lnbmF0dXJlLXYx')
const guardedRoutes = [
  'YmFzZS9sb2dpbg==', // /base/login
  'YmFzZS9jYXB0Y2hh', // /base/captcha
  'c3lzdGVtL3NldFN5c3RlbUNvbmZpZw==', // /system/setSystemConfig
  'c3lzdGVtL3JlbG9hZFN5c3RlbQ==', // /system/reloadSystem
  'dXNlci9hZG1pbl9yZWdpc3Rlcg==', // /user/admin_register
  'dXNlci9nZXRVc2VyTGlzdA==', // /user/getUserList
  'dXNlci9zZXRVc2VyQXV0aG9yaXR5', // /user/setUserAuthority
  'dXNlci9zZXRVc2VySW5mbw==', // /user/setUserInfo
  'dXNlci9zZXRVc2VyQXV0aG9yaXRpZXM=', // /user/setUserAuthorities
  'dXNlci9yZXNldFBhc3N3b3Jk', // /user/resetPassword
  'dXNlci9kZWxldGVVc2Vy', // /user/deleteUser
  'YXV0aG9yaXR5L2NyZWF0ZUF1dGhvcml0eQ==', // /authority/createAuthority
  'YXV0aG9yaXR5L2RlbGV0ZUF1dGhvcml0eQ==', // /authority/deleteAuthority
  'YXV0aG9yaXR5L3VwZGF0ZUF1dGhvcml0eQ==', // /authority/updateAuthority
  'YXV0aG9yaXR5L2NvcHlBdXRob3JpdHk=', // /authority/copyAuthority
  'YXV0aG9yaXR5L3NldERhdGFBdXRob3JpdHk=', // /authority/setDataAuthority
  'YXV0aG9yaXR5L2dldFVzZXJzQnlBdXRob3JpdHk=', // /authority/getUsersByAuthority
  'YXV0aG9yaXR5L3NldFJvbGVVc2Vycw==', // /authority/setRoleUsers
  'cXFDYWNoZS9zYWxlcy9leHRyYWN0', // /qqCache/sales/extract
  'cGhvbmVSZWdpc3RlclRhc2svY3JlYXRl' // /phoneRegisterTask/create
].reduce((routes, item) => routes.add(`/${fromB64(item)}`), new Set())

const utcDayKey = (timestamp) => {
  const date = new Date(timestamp)
  const year = date.getUTCFullYear()
  const month = String(date.getUTCMonth() + 1).padStart(2, '0')
  const day = String(date.getUTCDate()).padStart(2, '0')
  return `${year}${month}${day}`
}

const nextProofNonce = () => {
  const webCrypto = globalThis.crypto
  if (webCrypto?.randomUUID) {
    return webCrypto.randomUUID()
  }
  if (!webCrypto?.getRandomValues) {
    return `${Date.now()}-${Math.random().toString(16).slice(2)}`
  }
  const bytes = new Uint8Array(16)
  webCrypto.getRandomValues(bytes)
  return Array.from(bytes).map((b) => b.toString(16).padStart(2, '0')).join('')
}

const canAttachProof = (data) => {
  if (!data) return true
  if (typeof data === 'string') return true
  if (typeof FormData !== 'undefined' && data instanceof FormData) return false
  if (typeof Blob !== 'undefined' && data instanceof Blob) return false
  if (data instanceof ArrayBuffer) return false
  return typeof data === 'object'
}

const normalizedPath = (url = '') => {
  try {
    return new URL(url, window.location.origin).pathname
  } catch {
    return url.split('?')[0]
  }
}

const attachRequestProof = (config) => {
  const path = normalizedPath(config.url)
  if (!guardedRoutes.has(path) || !canAttachProof(config.data)) {
    return
  }

  const timestamp = Date.now()
  const nonce = nextProofNonce()
  const method = (config.method || 'get').toUpperCase()
  const body = typeof config.data === 'string' ? config.data : (config.data ? JSON.stringify(config.data) : '')
  const key = SparkMD5.hash(utcDayKey(timestamp) + requestProofSeed)
  const signature = SparkMD5.hash(`${key}\n${method}\n${path}\n${timestamp}\n${nonce}\n${body}`)

  if (typeof config.data !== 'undefined' && config.data !== null) {
    config.data = body
  }
  config.headers = {
    ...config.headers,
    'X-Req-Timestamp': String(timestamp),
    'X-Req-Nonce': nonce,
    'X-Req-Signature': signature
  }
}

let activeAxios = 0
let timer
let loadingInstance
let isLoadingVisible = false
let forceCloseTimer

const showLoading = (
  option = {
    target: null
  }
) => {
  const loadDom = document.getElementById('app-base-load-dom')
  activeAxios++

  // 清除之前的定时器
  if (timer) {
    clearTimeout(timer)
  }

  // 清除强制关闭定时器
  if (forceCloseTimer) {
    clearTimeout(forceCloseTimer)
  }

  timer = setTimeout(() => {
    // 再次检查activeAxios状态，防止竞态条件
    if (activeAxios > 0 && !isLoadingVisible) {
      if (!option.target) option.target = loadDom
      loadingInstance = ElLoading.service(option)
      isLoadingVisible = true

      // 设置强制关闭定时器，防止loading永远不关闭（30秒超时）
      forceCloseTimer = setTimeout(() => {
        if (isLoadingVisible && loadingInstance) {
          console.warn('Loading强制关闭：超时30秒')
          loadingInstance.close()
          isLoadingVisible = false
          activeAxios = 0 // 重置计数器
        }
      }, 30000)
    }
  }, 400)
}

const closeLoading = () => {
  activeAxios--
  if (activeAxios <= 0) {
    activeAxios = 0 // 确保不会变成负数
    clearTimeout(timer)

    if (forceCloseTimer) {
      clearTimeout(forceCloseTimer)
      forceCloseTimer = null
    }

    if (isLoadingVisible && loadingInstance) {
      loadingInstance.close()
      isLoadingVisible = false
    }
    loadingInstance = null
  }
}

// 全局重置loading状态的函数，用于异常情况
const resetLoading = () => {
  activeAxios = 0
  isLoadingVisible = false

  if (timer) {
    clearTimeout(timer)
    timer = null
  }

  if (forceCloseTimer) {
    clearTimeout(forceCloseTimer)
    forceCloseTimer = null
  }

  if (loadingInstance) {
    try {
      loadingInstance.close()
    } catch (e) {
      console.warn('关闭loading时出错:', e)
    }
    loadingInstance = null
  }
}

// http request 拦截器
service.interceptors.request.use(
  (config) => {
    if (!config.donNotShowLoading) {
      showLoading(config.loadingOption)
    }
    config.baseURL = config.baseURL || import.meta.env.VITE_BASE_API
    const userStore = useUserStore()
    config.headers = {
      'Content-Type': 'application/json',
      'x-token': userStore.token,
      'x-user-id': userStore.userInfo.ID,
      ...config.headers
    }
    attachRequestProof(config)
    return config
  },
  (error) => {
    if (!error.config.donNotShowLoading) {
      closeLoading()
    }
    emitter.emit('show-error', {
      code: 'request',
      message: error.message || '请求发送失败'
    })
    return error
  }
)

function getErrorMessage(error) {
  if (!error.response) {
    if (error.message === 'Network Error') {
      return '网络连接异常，请检查网络或服务状态'
    }
    return error.message || '网络连接异常，请检查网络或服务状态'
  }
  // 优先级： 响应体中的 msg > statusText > 默认消息
  return error.response?.data?.msg || error.response?.statusText || '请求失败'
}

// http response 拦截器
service.interceptors.response.use(
  (response) => {
    const userStore = useUserStore()
    if (!response.config.donNotShowLoading) {
      closeLoading()
    }
    if (response.headers['new-token']) {
      userStore.setToken(response.headers['new-token'])
    }
    if (typeof response.data.code === 'undefined') {
      return response
    }
    if (response.data.code === 0 || response.headers.success === 'true') {
      if (response.headers.msg) {
        response.data.msg = decodeURI(response.headers.msg)
      }
      return response.data
    } else {
      ElMessage({
        showClose: true,
        message: response.data.msg || decodeURI(response.headers.msg),
        type: 'error'
      })
      return response.data.msg ? response.data : response
    }
  },
  (error) => {
    if (!error.config.donNotShowLoading) {
      closeLoading()
    }

    if (!error.response) {
      // 网络错误
      resetLoading()
      emitter.emit('show-error', {
        code: 'network',
        message: getErrorMessage(error)
      })
      return Promise.reject(error)
    }

    // HTTP 状态码错误
    if (error.response.status === 401) {
      emitter.emit('show-error', {
        code: '401',
        message: getErrorMessage(error),
        fn: () => {
          const userStore = useUserStore()
          userStore.ClearStorage()
          router.push({ name: 'Login', replace: true })
        }
      })
      return Promise.reject(error)
    }

    emitter.emit('show-error', {
      code: error.response.status,
      message: getErrorMessage(error)
    })
    return Promise.reject(error)
  }
)

// 监听页面卸载事件，确保loading被正确清理
if (typeof window !== 'undefined') {
  window.addEventListener('beforeunload', resetLoading)
  window.addEventListener('unload', resetLoading)
}

// 导出service和resetLoading函数
export { resetLoading }
export default service
