<template>
  <div class="register-manage-page">
    <PhoneRegisterTaskManage />
  </div>
</template>

<script setup>
import PhoneRegisterTaskManage from './components/PhoneRegisterTaskManage.vue'

defineOptions({
  name: 'PhoneTaskManage'
})
</script>

<style scoped>
.register-manage-page {
  min-height: 100%;
}
</style>
