<template>
  <div class="register-config-page">
    <el-card shadow="never" style="margin-top: 8px;">
      <template #header>注册配置管理</template>

      <el-form label-width="130px">
        <template v-if="isAdminRole">
          <el-form-item label="默认改密密码">
            <el-input v-model="form.defaultPassword" show-password placeholder="请输入默认改密密码" />
          </el-form-item>
        </template>

        <template v-else-if="isLeaderRole">
          <el-form-item label="代理平台">
            <el-select v-model="form.proxyPlatform" style="width: 100%" placeholder="请选择代理平台">
              <el-option label="神龙代理(shenlong)" value="shenlong" />
            </el-select>
          </el-form-item>
          <el-form-item label="代理账号">
            <el-input v-model="form.proxyAccount" placeholder="请输入代理账号" />
          </el-form-item>
          <el-form-item label="代理密码">
            <el-input v-model="form.proxyPassword" show-password placeholder="请输入代理密码" />
          </el-form-item>
          <el-form-item label="验证码平台">
            <el-select v-model="form.captchaPlatform" style="width: 100%" placeholder="请选择验证码平台">
              <el-option label="yy" value="yy" />
              <el-option label="ac" value="ac" />
            </el-select>
          </el-form-item>
          <el-form-item label="验证码账号">
            <el-input v-model="form.captchaAccount" placeholder="请输入验证码账号" />
          </el-form-item>
          <el-form-item label="验证码密码">
            <el-input v-model="form.captchaPassword" show-password placeholder="请输入验证码密码" />
          </el-form-item>
          <el-form-item label="验证码Token">
            <el-input v-model="form.captchaToken" placeholder="请输入验证码Token（可选）" />
          </el-form-item>
        </template>

        <template v-else>
          <el-alert type="warning" title="当前角色无配置管理权限" :closable="false" show-icon />
        </template>

        <el-form-item>
          <el-button type="primary" :disabled="!canEdit" @click="submit">保存配置</el-button>
          <el-button @click="loadConfig">刷新</el-button>
        </el-form-item>
      </el-form>
    </el-card>
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { useUserStore } from '@/pinia/modules/user'
import { getMyRegisterConfig, setMyRegisterConfig } from '@/api/registerConfig'

defineOptions({
  name: 'RegisterConfig'
})

const ROLE_SUPER = 888
const ROLE_ADMIN = 100
const ROLE_LEADER = 200

const userStore = useUserStore()
const currentRoleId = computed(() => userStore.userInfo?.authority?.authorityId)
const isAdminRole = computed(() => [ROLE_SUPER, ROLE_ADMIN].includes(currentRoleId.value))
const isLeaderRole = computed(() => currentRoleId.value === ROLE_LEADER)
const canEdit = computed(() => isAdminRole.value || isLeaderRole.value)

const form = ref({
  defaultPassword: '',
  proxyPlatform: '',
  proxyAccount: '',
  proxyPassword: '',
  captchaPlatform: '',
  captchaAccount: '',
  captchaPassword: '',
  captchaToken: ''
})

const loadConfig = async () => {
  const { data } = await getMyRegisterConfig()
  form.value = {
    defaultPassword: data?.defaultPassword || '',
    proxyPlatform: data?.proxyPlatform || '',
    proxyAccount: data?.proxyAccount || '',
    proxyPassword: data?.proxyPassword || '',
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
  await setMyRegisterConfig(form.value)
  ElMessage.success('保存成功')
  await loadConfig()
}

onMounted(async () => {
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
