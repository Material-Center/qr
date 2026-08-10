<template>
  <div class="h-full app-container2 overflow-auto bg-slate-50/60 dark:bg-slate-900">
    <div class="space-y-4 p-4 lg:p-6">
      <section
        class="relative overflow-hidden rounded-xl border border-slate-200/80 bg-white px-5 py-6 shadow-sm dark:border-slate-700 dark:from-slate-900 dark:via-slate-800 dark:to-slate-900"
      >
        
        <div class="relative flex flex-col gap-4 lg:flex-row lg:items-end lg:justify-between">
          <div>
            <p class="text-xs tracking-[0.2em] text-slate-500 dark:text-slate-400">DASHBOARD</p>
            <h1 class="mt-2 text-xl font-semibold text-slate-900 dark:text-slate-100 lg:text-2xl">
              欢迎回来，查看今日业务概况
            </h1>
            <p class="mt-2 text-sm text-slate-600 dark:text-slate-300">
              {{ today }} · 已为你聚合核心业务数据和系统公告
            </p>
          </div>
        </div>
      </section>

      <div class="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-3">
        <app-card>
          <app-chart :type="1" title="访问人数" />
        </app-card>
        <app-card>
          <app-chart :type="2" title="新增客户" />
        </app-card>
        <app-card>
          <app-chart :type="3" title="解决数量" />
        </app-card>
      </div>

      <div class="grid grid-cols-1 items-stretch gap-4 xl:grid-cols-12">
        <div class="grid grid-cols-1 gap-4 content-start xl:col-span-8 xl:h-full">
          <app-card title="内容数据">
            <app-chart :type="4" />
          </app-card>

          <app-card title="系统公告">
            <app-notice />
          </app-card>
        </div>

        <div class="flex flex-col gap-4 xl:col-span-4 xl:h-full">
          <app-card title="快捷功能" show-action custom-class="min-h-[300px]">
            <app-quick-link />
          </app-card>
          <div
            class="relative min-h-[200px] flex-1 overflow-hidden rounded-lg border border-slate-200 bg-slate-900 p-5 text-white shadow-sm dark:border-slate-700"
          >
            
            <div class="relative">
              <div class="inline-flex rounded-full bg-white/10 px-3 py-1 text-xs">运营提醒</div>
              <h3 class="mt-3 text-lg font-semibold">关注登录安全与系统状态</h3>
              <p class="mt-2 text-sm text-slate-200/90">
                建议定期检查登录日志、异常账号和关键配置变更，及时处理异常访问。
              </p>
              <div class="mt-4 flex flex-wrap gap-2 text-xs">
                <span class="rounded-full bg-white/10 px-2.5 py-1">登录审计</span>
                <span class="rounded-full bg-white/10 px-2.5 py-1">权限核查</span>
                <span class="rounded-full bg-white/10 px-2.5 py-1">配置保护</span>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
  import { computed } from 'vue'
  import {
    AppChart,
    AppNotice,
    AppQuickLink,
    AppCard
  } from './components'

  const today = computed(() => {
    try {
      const d = new Date()
      return d.toLocaleDateString('zh-CN', {
        year: 'numeric',
        month: '2-digit',
        day: '2-digit'
      })
    } catch (e) {
      return new Date().toISOString().slice(0, 10)
    }
  })

  defineOptions({
    name: 'Dashboard'
  })
</script>

<style lang="scss" scoped></style>
