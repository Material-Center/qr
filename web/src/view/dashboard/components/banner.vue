<template>
  <el-carousel class="-mt-2">
    <el-carousel-item
      class="lg:h-40"
      v-for="(item, index) in banners"
      :key="index"
    >
      <el-image class="h-full w-full" :src="item.img" fit="fill"></el-image>
    </el-carousel-item>
  </el-carousel>
</template>

<script setup>
  import banner from '@/assets/banner.jpg'
  import banner2 from '@/assets/banner2.jpg'

  const banners = [
    {
      img: banner
    },
    {
      img: banner2
    }
  ]
</script>

<style scoped lang="scss"></style>
