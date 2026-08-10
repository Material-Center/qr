<template>
  <div class="mt-2">
    <el-card>
      <template #header>
        <div>系统信息</div>
      </template>
      <el-descriptions :column="1" border>
        <el-descriptions-item label="系统名称">营销系统</el-descriptions-item>
        <el-descriptions-item label="运行状态">正常</el-descriptions-item>
      </el-descriptions>
    </el-card>
  </div>
</template>

<script setup>
  defineOptions({
    name: 'About'
  })
</script>
