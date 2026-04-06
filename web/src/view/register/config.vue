<template>
  <div class="register-config-page">
    <el-card shadow="never" style="margin-top: 8px;">
      <template #header>注册配置管理</template>

      <el-form label-width="130px">
        <template v-if="isAdminRole">
          <el-form-item label="默认改密密码">
            <el-input v-model="form.defaultPassword" show-password placeholder="请输入默认改密密码" />
          </el-form-item>
          <el-form-item label="奶茶 AppID">
            <el-input v-model="form.naichaAppId" placeholder="请输入奶茶平台 appId" />
          </el-form-item>
          <el-form-item label="奶茶 Secret">
            <el-input v-model="form.naichaSecret" show-password placeholder="请输入奶茶平台 secret" />
          </el-form-item>
          <el-form-item label="奶茶 CKMd5">
            <el-input v-model="form.naichaCkMd5" placeholder="请输入奶茶平台 ckmd5（可选）" />
          </el-form-item>
          <el-form-item label="IP138 Token">
            <el-input v-model="form.ip138Token" show-password placeholder="可选：用于按手机号定位代理地区" />
          </el-form-item>
          <el-form-item label="签名 ApiBase">
            <el-input v-model="form.apiBase" placeholder="例如: http://sign9.owo.vin" />
          </el-form-item>
          <el-form-item label="签名 ApiToken">
            <el-input v-model="form.apiToken" show-password placeholder="请输入签名服务 apiToken" />
          </el-form-item>
          <el-form-item label="代理平台">
            <el-select v-model="form.proxyPlatform" style="width: 100%" placeholder="请选择代理平台">
              <el-option label="神龙代理(shenlong)" value="shenlong" />
              <el-option label="快代理(kuaidaili)" value="kuaidaili" />
              <el-option label="品赞代理(pingzan)" value="pingzan" />
            </el-select>
          </el-form-item>
          <el-form-item :label="proxyAccountLabel">
            <el-input v-model="form.proxyAccount" :placeholder="proxyAccountPlaceholder" />
          </el-form-item>
          <el-form-item :label="proxyPasswordLabel">
            <el-input v-model="form.proxyPassword" show-password :placeholder="proxyPasswordPlaceholder" />
          </el-form-item>
          <el-form-item v-if="form.proxyPlatform === 'kuaidaili'" label="快代理 SecretId">
            <el-input v-model="form.proxySecretId" placeholder="请输入快代理 SecretId" />
          </el-form-item>
          <el-form-item v-if="form.proxyPlatform === 'kuaidaili'" label="快代理 SecretKey">
            <el-input v-model="form.proxySecretKey" show-password placeholder="请输入快代理 SecretKey" />
          </el-form-item>
          <el-form-item label="验证码平台">
            <el-select v-model="form.captchaPlatform" style="width: 100%" placeholder="请选择验证码平台">
              <el-option label="yy（账号密码模式）" value="yy" />
              <el-option label="ac（baseURL + token 模式）" value="ac" />
              <el-option label="fj（服务器地址 + token 模式）" value="fj" />
            </el-select>
          </el-form-item>
          <el-form-item :label="captchaAccountLabel">
            <el-input v-model="form.captchaAccount" :placeholder="captchaAccountPlaceholder" />
          </el-form-item>
          <el-form-item :label="captchaPasswordLabel">
            <el-input v-model="form.captchaPassword" show-password :placeholder="captchaPasswordPlaceholder" />
          </el-form-item>
          <el-form-item :label="captchaTokenLabel">
            <el-input v-model="form.captchaToken" :placeholder="captchaTokenPlaceholder" />
          </el-form-item>
          <el-form-item>
            <el-alert
              type="info"
              show-icon
              :closable="false"
              :title="leaderConfigHint"
            />
          </el-form-item>
        </template>

        <template v-else>
          <el-alert type="warning" title="团长配置能力已迁移到管理员，请联系管理员维护配置" :closable="false" show-icon />
        </template>

        <el-form-item>
          <el-button type="primary" :disabled="!canEdit" @click="submit">保存配置</el-button>
          <el-button :loading="checking" :disabled="!canEdit" @click="checkConfig">检测连通性</el-button>
          <el-button :disabled="!isAdminRole" @click="loadConfig">刷新</el-button>
        </el-form-item>
        <el-form-item v-if="checkResultText">
          <el-alert
            :type="checkResultType"
            :closable="false"
            show-icon
            :title="checkResultText"
          />
        </el-form-item>
      </el-form>
    </el-card>
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { useUserStore } from '@/pinia/modules/user'
import { checkMyRegisterConfig, getMyRegisterConfig, setMyRegisterConfig } from '@/api/registerConfig'

defineOptions({
  name: 'RegisterConfig'
})

const ROLE_SUPER = 888
const ROLE_ADMIN = 100

const userStore = useUserStore()
const currentRoleId = computed(() => userStore.userInfo?.authority?.authorityId)
const isAdminRole = computed(() => [ROLE_SUPER, ROLE_ADMIN].includes(currentRoleId.value))
const canEdit = computed(() => isAdminRole.value)
const checking = ref(false)
const checkResultText = ref('')
const checkResultType = ref('info')

const form = ref({
  defaultPassword: '',
  naichaAppId: '',
  naichaSecret: '',
  naichaCkMd5: '',
  ip138Token: '',
  apiBase: '',
  apiToken: '',
  proxyPlatform: '',
  proxyAccount: '',
  proxyPassword: '',
  proxySecretId: '',
  proxySecretKey: '',
  captchaPlatform: '',
  captchaAccount: '',
  captchaPassword: '',
  captchaToken: ''
})

const proxyAccountLabel = computed(() => {
  if (form.value.proxyPlatform === 'kuaidaili') return '代理账号(可空)'
  if (form.value.proxyPlatform === 'pingzan') return '品赞 no'
  return '代理账号'
})
const proxyAccountPlaceholder = computed(() => {
  if (form.value.proxyPlatform === 'kuaidaili') return '快代理模式此字段可留空'
  if (form.value.proxyPlatform === 'pingzan') return '请输入品赞套餐购买编号 no'
  return '神龙 key'
})
const proxyPasswordLabel = computed(() => {
  if (form.value.proxyPlatform === 'kuaidaili') return '代理密码(可空)'
  if (form.value.proxyPlatform === 'pingzan') return '品赞 secret'
  return '代理密码'
})
const proxyPasswordPlaceholder = computed(() => {
  if (form.value.proxyPlatform === 'kuaidaili') return '快代理模式此字段可留空'
  if (form.value.proxyPlatform === 'pingzan') return '请输入品赞套餐密匙 secret'
  return '神龙 sign'
})

const captchaAccountLabel = computed(() => {
  if (form.value.captchaPlatform === 'ac') return 'AC地址'
  if (form.value.captchaPlatform === 'fj') return 'FJ地址(可空)'
  return '验证码账号'
})
const captchaAccountPlaceholder = computed(() => {
  if (form.value.captchaPlatform === 'ac') return '例如: http://39.99.146.154:16168'
  if (form.value.captchaPlatform === 'fj') return '可留空，默认: http://156.238.235.35:8860/'
  return 'YY 平台账号'
})
const captchaPasswordLabel = computed(() => {
  if (form.value.captchaPlatform === 'ac') return 'AC密码(可空)'
  if (form.value.captchaPlatform === 'fj') return 'FJ密码(可空)'
  return '验证码密码'
})
const captchaPasswordPlaceholder = computed(() => {
  if (form.value.captchaPlatform === 'ac') return 'AC 模式可留空'
  if (form.value.captchaPlatform === 'fj') return 'FJ 模式可留空'
  return 'YY 平台密码'
})
const captchaTokenLabel = computed(() => {
  if (form.value.captchaPlatform === 'ac') return 'AC Token'
  if (form.value.captchaPlatform === 'fj') return 'FJ Token'
  return '验证码Token'
})
const captchaTokenPlaceholder = computed(() => {
  if (form.value.captchaPlatform === 'ac') return 'AC 必填 token'
  if (form.value.captchaPlatform === 'fj') return 'FJ 必填 token'
  return 'YY 模式可留空'
})
const leaderConfigHint = computed(() => {
  if (form.value.proxyPlatform === 'pingzan') {
    return '品赞代理：代理账号填写 no（套餐编号），代理密码填写 secret（密钥）；当前按固定 5 分钟 quality 池提取。'
  }
  if (form.value.captchaPlatform === 'ac') {
    return 'AC 平台：验证码账号填 baseURL，验证码Token 必填，验证码密码可空。'
  }
  if (form.value.captchaPlatform === 'fj') {
    return 'FJ 平台：验证码账号(服务器地址)可留空使用默认值，验证码Token 必填，验证码密码可空。'
  }
  return 'YY 平台：验证码账号/验证码密码必填，验证码Token 可留空。'
})

const loadConfig = async () => {
  const { data } = await getMyRegisterConfig()
  form.value = {
    defaultPassword: data?.defaultPassword || '',
    naichaAppId: data?.naichaAppId || '',
    naichaSecret: data?.naichaSecret || '',
    naichaCkMd5: data?.naichaCkMd5 || '',
    ip138Token: data?.ip138Token || '',
    apiBase: data?.apiBase || '',
    apiToken: data?.apiToken || '',
    proxyPlatform: data?.proxyPlatform || '',
    proxyAccount: data?.proxyAccount || '',
    proxyPassword: data?.proxyPassword || '',
    proxySecretId: data?.proxySecretId || '',
    proxySecretKey: data?.proxySecretKey || '',
    captchaPlatform: data?.captchaPlatform || '',
    captchaAccount: data?.captchaAccount || '',
    captchaPassword: data?.captchaPassword || '',
    captchaToken: data?.captchaToken || ''
  }
}

const submit = async () => {
  if (!canEdit.value) return
  if (isAdminRole.value && !form.value.defaultPassword) {
    ElMessage.warning('默认改密密码不能为空')
    return
  }
  if (isAdminRole.value && (!form.value.naichaAppId || !form.value.naichaSecret)) {
    ElMessage.warning('奶茶平台 appId 和 secret 不能为空')
    return
  }
  if (isAdminRole.value && (!form.value.apiBase || !form.value.apiToken)) {
    ElMessage.warning('签名服务 apiBase 和 apiToken 不能为空')
    return
  }
  if (isAdminRole.value && form.value.proxyPlatform === 'kuaidaili' && (!form.value.proxySecretId || !form.value.proxySecretKey)) {
    ElMessage.warning('快代理 SecretId 和 SecretKey 不能为空')
    return
  }
  if (isAdminRole.value && form.value.proxyPlatform === 'pingzan' && (!form.value.proxyAccount || !form.value.proxyPassword)) {
    ElMessage.warning('品赞代理 no 和 secret 不能为空')
    return
  }
  if (isAdminRole.value && form.value.proxyPlatform === 'shenlong' && (!form.value.proxyAccount || !form.value.proxyPassword)) {
    ElMessage.warning('神龙代理 key 和 sign 不能为空')
    return
  }
  await setMyRegisterConfig(form.value)
  ElMessage.success('保存成功')
  await loadConfig()
}

const checkConfig = async () => {
  if (!canEdit.value) return
  checking.value = true
  checkResultText.value = ''
  try {
    const { data } = await checkMyRegisterConfig()
    const proxy = data?.proxy || {}
    const captcha = data?.captcha || {}
    const defaultPwd = data?.defaultPassword || {}
    const naicha = data?.naicha || {}
    const qsign = data?.qsign || {}
    if (isAdminRole.value) {
      const lines = [
        defaultPwd.ok
          ? '默认改密密码已配置'
          : `默认改密密码未配置：${defaultPwd.message || '请先设置'}`,
        naicha.ok
          ? '奶茶平台配置已就绪'
          : `奶茶平台未配置：${naicha.message || '请先设置 appId/secret'}`,
        qsign.ok
          ? '签名服务配置已就绪'
          : `签名服务未配置：${qsign.message || '请先设置 apiBase/apiToken'}`,
        `代理: ${proxy.ok ? '可用' : '不可用'} (${proxy.message || '-'})`,
        `验证码: ${captcha.ok ? '可用' : '不可用'} (${captcha.message || '-'})`
      ]
      checkResultText.value = lines.join('；')
      checkResultType.value = defaultPwd.ok && naicha.ok && qsign.ok && proxy.ok && captcha.ok ? 'success' : 'warning'
    }
    ElMessage.success('检测完成')
  } catch (e) {
    checkResultType.value = 'error'
    checkResultText.value = e?.message || '检测失败'
    ElMessage.error(e?.message || '检测失败')
  } finally {
    checking.value = false
  }
}

onMounted(async () => {
  if (!isAdminRole.value) return
  try {
    await loadConfig()
  } catch (e) {
    ElMessage.error(e?.message || '加载配置失败')
  }
})
</script>

<style scoped>
.register-config-page {
  max-width: 760px;
}
</style>
