<template>
  <div class="register-manage-page">
    <RegisterTaskLegacyManage />
  </div>
</template>

<script setup>
import RegisterTaskLegacyManage from './components/RegisterTaskLegacyManage.vue'

defineOptions({
  name: 'RegisterTaskManage'
})
</script>

<style scoped>
.register-manage-page {
  min-height: 100%;
}
</style>
