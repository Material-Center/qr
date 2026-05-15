<template>
  <div class="task-center">
    <el-card
      shadow="never"
      class="user-card mb-3"
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
    <PhoneRegisterTaskCenter />
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { useUserStore } from '@/pinia/modules/user'
import PhoneRegisterTaskCenter from './components/PhoneRegisterTaskCenter.vue'

defineOptions({
  name: 'PhoneTaskCenter'
})

const userStore = useUserStore()
const currentUser = computed(() => userStore.userInfo || {})
</script>

<style scoped>
.task-center {
  width: 100%;
  height: 100%;
  min-height: 100%;
  max-width: 1120px;
  margin: 0 auto;
  padding: 8px;
  box-sizing: border-box;
  overflow-y: auto;
  -webkit-overflow-scrolling: touch;
  overscroll-behavior-y: contain;
  touch-action: pan-y;
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

  .user-card {
    margin-bottom: 6px;
  }

  .user-card :deep(.el-card__body) {
    padding: 5px 8px;
  }

  .user-bar {
    min-height: 24px;
    font-size: 12px;
    line-height: 1.2;
  }

  .user-bar :deep(.el-button) {
    height: 22px;
    padding: 0;
    font-size: 12px;
  }
}
</style>
