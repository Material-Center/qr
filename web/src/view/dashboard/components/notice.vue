<template>
  <div class="space-y-3">
    <el-scrollbar max-height="320px">
      <div class="space-y-2 pr-1">
        <div
          v-for="(item, index) in notices"
          :key="index"
          class="group rounded-lg border border-black/10 bg-white/70 p-3 transition-all duration-200 hover:-translate-y-0.5 hover:border-black/20 hover:shadow-sm dark:border-white/10 dark:bg-white/[0.02] dark:hover:border-white/20"
        >
          <div class="flex items-start gap-3">
            <span class="mt-1.5 h-2.5 w-2.5 shrink-0 rounded-full" :class="item.dotClass" />
            <div class="min-w-0 flex-1">
              <div class="flex items-center justify-between gap-2">
                <span class="rounded-md px-2 py-0.5 text-[11px] font-semibold leading-4" :class="item.tagClass">
                  {{ item.typeTitle }}
                </span>
                <span class="shrink-0 text-[11px] text-black/45 dark:text-white/45">{{ item.time }}</span>
              </div>
              <el-tooltip effect="light" :content="item.title" placement="top">
                <p class="mt-1.5 line-clamp-2 text-sm text-black/75 dark:text-white/75">
                  {{ item.title }}
                </p>
              </el-tooltip>
            </div>
          </div>
        </div>
      </div>
    </el-scrollbar>
  </div>
</template>

<script setup>

  const notices = [
    {
      typeTitle: '通知',
      time: '今天',
      title: '建议定期检查登录日志、异常账号和关键配置变更。',
      dotClass: 'bg-cyan-500',
      tagClass: 'bg-cyan-100 text-cyan-700 dark:bg-cyan-900/40 dark:text-cyan-200'
    },
    {
      typeTitle: '活动',
      time: '2天前',
      title: '请关注高频登录失败、异常 User-Agent 和来源 IP。',
      dotClass: 'bg-emerald-500',
      tagClass: 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/40 dark:text-emerald-200'
    },
    {
      typeTitle: '合规',
      time: '3天前',
      title: '关键接口已启用请求签名、限流和权限校验。',
      dotClass: 'bg-amber-500',
      tagClass: 'bg-amber-100 text-amber-700 dark:bg-amber-900/40 dark:text-amber-200'
    },
    {
      typeTitle: '服务',
      time: '5天前',
      title: '如发现异常账号，请先禁用账号并复核角色权限。',
      dotClass: 'bg-violet-500',
      tagClass: 'bg-violet-100 text-violet-700 dark:bg-violet-900/40 dark:text-violet-200'
    }
  ]
</script>

<style scoped lang="scss"></style>
