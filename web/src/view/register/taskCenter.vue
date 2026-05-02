<template>
  <div class="task-center">
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
    <RegisterTaskLegacyCenter />
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { useUserStore } from '@/pinia/modules/user'
import RegisterTaskLegacyCenter from './components/RegisterTaskLegacyCenter.vue'

defineOptions({
  name: 'RegisterTaskCenter'
})

const userStore = useUserStore()
const currentUser = computed(() => userStore.userInfo || {})
</script>

<style scoped>
.task-center {
  max-width: 960px;
  margin: 0 auto;
  padding: 8px;
  box-sizing: border-box;
  min-height: 100vh;
  min-height: 100dvh;
}

.user-bar {
  display: flex;
  justify-content: space-between;
  align-items: center;
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
}
</style>
