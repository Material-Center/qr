<script setup>
const props = defineProps({
  // logo 尺寸，单位为:rem
  size: {
    type: Number,
    default: 2
  }
})

const logoSrc = '/brand-mark.svg';

// 直接用 16px 作为默认的基准大小
const SPACING = 16
function getSize() {
  return {
    width: `${props.size * SPACING}px`,
    height: `${props.size * SPACING}px`,
  }
}
</script>

<template>
  <img
    :src="logoSrc"
    :alt="$APP_CONFIG.appName"
    class="object-contain"
    :style="{
      ...getSize()
    }"
  />
</template>
