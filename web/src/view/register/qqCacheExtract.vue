<template>
  <div>
    <div class="gva-table-box">
      <el-row :gutter="12" class="mb-3">
        <el-col :xs="24" :sm="8">
          <el-card shadow="never">当前可提取数量：{{ summary.available }}</el-card>
        </el-col>
        <el-col :xs="24" :sm="8">
          <el-card shadow="never">待结算数量：{{ summary.todayUnsettled }}</el-card>
        </el-col>
        <el-col :xs="24" :sm="8">
          <el-card shadow="never">今日已提取数量：{{ summary.todayExtracted }}</el-card>
        </el-col>
      </el-row>

      <div class="extract-panel">
        <span class="extract-label">提取范围</span>
        <el-button-group>
          <el-button
            size="small"
            :type="!extractRecentMinutesValue ? 'primary' : 'default'"
            @click="setExtractRecentMinutes(undefined)"
          >
            不限
          </el-button>
          <el-button
            v-for="item in recentMinuteOptions"
            :key="item.value"
            size="small"
            :type="extractRecentMinutesValue === item.value ? 'primary' : 'default'"
            @click="setExtractRecentMinutes(item.value)"
          >
            {{ item.shortLabel }}
          </el-button>
        </el-button-group>
        <el-input-number
          v-model="extractCustomHours"
          :min="1"
          :precision="0"
          :step="1"
          controls-position="right"
          placeholder="自定义小时"
          style="width: 150px"
          @change="onExtractCustomHoursChange"
        />
        <el-tag v-if="extractRangeAvailableText" class="extract-range-count" type="info" effect="plain">
          {{ extractRangeAvailableText }}
        </el-tag>
        <span class="extract-label">提取数量</span>
        <el-input-number
          v-model="extractCount"
          :min="1"
          :max="extractMax"
          :disabled="extractMax <= 0"
          controls-position="right"
          style="width: 160px"
        />
        <el-button
          type="primary"
          :disabled="extractMax <= 0 || extracting"
          :loading="extracting"
          @click="onExtract"
        >
          提取
        </el-button>
      </div>
    </div>

    <div class="gva-table-box">
      <div class="table-header">
        <span class="table-title">今日提取历史</span>
        <el-button icon="refresh" @click="fetchAll">刷新</el-button>
      </div>
      <el-table :data="historyData" row-key="id">
        <el-table-column label="提取时间" min-width="180">
          <template #default="{ row }">
            {{ row.extractedAt ? formatDate(row.extractedAt) : '-' }}
          </template>
        </el-table-column>
        <el-table-column label="提取账号数量" prop="extractCount" min-width="140" />
        <el-table-column label="结算状态" min-width="120">
          <template #default="{ row }">
            <el-tag :type="row.settlementStatus === 'settled' ? 'success' : 'warning'">
              {{ row.settlementStatusText || settlementStatusText(row.settlementStatus) }}
            </el-tag>
          </template>
        </el-table-column>
      </el-table>
      <div class="gva-pagination">
        <el-pagination
          :current-page="page"
          :page-size="pageSize"
          :page-sizes="[10, 30, 50, 100]"
          :total="total"
          layout="total, sizes, prev, pager, next, jumper"
          @current-change="handleCurrentChange"
          @size-change="handleSizeChange"
        />
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { formatDate } from '@/utils/format'
import {
  exportSalesQQCacheIniZip,
  getQQCacheSalesHistory,
  getQQCacheSalesSummary
} from '@/api/qqCache'

defineOptions({
  name: 'QQCacheExtract'
})

const summary = ref({
  available: 0,
  todayExtracted: 0,
  todayUnsettled: 0
})
const extractCount = ref(1)
const extractRecentMinutes = ref(undefined)
const extractCustomHours = ref(undefined)
const page = ref(1)
const pageSize = ref(10)
const total = ref(0)
const historyData = ref([])
const extracting = ref(false)

const recentMinuteOptions = [
  { label: '最近15分钟', shortLabel: '15分钟', value: 15 },
  { label: '最近30分钟', shortLabel: '30分钟', value: 30 },
  { label: '最近1小时', shortLabel: '1小时', value: 60 },
  { label: '最近2小时', shortLabel: '2小时', value: 120 },
  { label: '3小时以上', shortLabel: '3小时以上', value: -180 }
]

const normalizePositiveInteger = (value) => {
  if (value === undefined || value === null || value === '') return undefined
  const text = String(value).trim()
  if (!/^[1-9]\d*$/.test(text)) return null
  return Number(text)
}

const recentMinutesParam = () => {
  const customHours = normalizePositiveInteger(extractCustomHours.value)
  if (customHours) return customHours * 60
  const value = Number(extractRecentMinutes.value)
  return Number.isInteger(value) && value !== 0 ? value : undefined
}

const extractMax = computed(() => Math.max(Number(summary.value.available) || 0, 0))

const extractRecentMinutesValue = computed(() => recentMinutesParam())

const recentMinutesText = (minutes) => {
  if (!minutes) return ''
  if (minutes < 0) return `${Math.abs(minutes) / 60}小时以上`
  if (minutes < 60) return `${minutes}分钟`
  if (minutes % 60 === 0) return `${minutes / 60}小时`
  return `${minutes}分钟`
}

const extractRangeAvailableText = computed(() => {
  if (!extractRecentMinutesValue.value) return ''
  const prefix = extractRecentMinutesValue.value < 0 ? '' : '最近'
  return `${prefix}${recentMinutesText(extractRecentMinutesValue.value)}可提取：${extractMax.value} 个`
})

const setExtractRecentMinutes = async (value) => {
  extractRecentMinutes.value = value
  extractCustomHours.value = undefined
  await fetchSummary()
}

const settlementStatusText = (status) => {
  return status === 'settled' ? '已结算' : '待结算'
}

const pickZipFilename = (contentDisposition) => {
  if (!contentDisposition) return null
  const utf8Match = /filename\*=(?:UTF-8'')?["']?([^"';]+)/i.exec(contentDisposition)
  const fallbackMatch = /filename=["']?([^"';]+)/i.exec(contentDisposition)
  const m = utf8Match || fallbackMatch
  if (!m?.[1]) return null
  try {
    return decodeURIComponent(m[1].replace(/["']/g, '').trim())
  } catch {
    return m[1].replace(/["']/g, '').trim()
  }
}

const qqCacheExtractZipName = (count) => {
  const d = new Date()
  const pad = (n) => String(n).padStart(2, '0')
  const timestamp = `${d.getFullYear()}${pad(d.getMonth() + 1)}${pad(d.getDate())}-${pad(d.getHours())}${pad(d.getMinutes())}`
  return `qq-${Number(count) || 0}个-${timestamp}.zip`
}

const responseDataToText = async (data) => {
  if (data instanceof Blob) return data.text()
  return new TextDecoder('utf-8').decode(data)
}

const handleZipDownload = async (res, fallbackName) => {
  const ct = String(res?.headers?.['content-type'] || '').toLowerCase()
  const buffer = res?.data
  if (!buffer) {
    ElMessage.error('提取失败')
    return false
  }
  if (ct.includes('application/json')) {
    const text = await responseDataToText(buffer)
    let msg = '提取失败'
    try {
      const j = JSON.parse(text)
      msg = j.msg || j.message || msg
    } catch {
      msg = text || msg
    }
    ElMessage.error(msg)
    return false
  }
  const blob = buffer instanceof Blob ? buffer : new Blob([buffer], { type: 'application/zip' })
  const name = pickZipFilename(res.headers?.['content-disposition']) || fallbackName
  const url = window.URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = name.endsWith('.zip') ? name : `${name}.zip`
  document.body.appendChild(a)
  a.click()
  a.remove()
  window.setTimeout(() => {
    window.URL.revokeObjectURL(url)
  }, 10000)
  ElMessage.success('已开始下载')
  return true
}

const fetchSummary = async () => {
  const { data } = await getQQCacheSalesSummary({
    recentMinutes: recentMinutesParam()
  })
  summary.value = {
    available: Number(data?.available) || 0,
    todayExtracted: Number(data?.todayExtracted) || 0,
    todayUnsettled: Number(data?.todayUnsettled) || 0
  }
  if (extractMax.value > 0 && extractCount.value > extractMax.value) {
    extractCount.value = extractMax.value
  }
}

const fetchHistory = async () => {
  const { data } = await getQQCacheSalesHistory({
    page: page.value,
    pageSize: pageSize.value
  })
  historyData.value = data?.list || []
  total.value = data?.total || 0
}

const fetchAll = async () => {
  try {
    await Promise.all([fetchSummary(), fetchHistory()])
  } catch (e) {
    ElMessage.error(e?.message || '加载失败')
  }
}

const onExtract = async () => {
  if (extracting.value) return
  const count = Number(extractCount.value) || 0
  if (count <= 0) {
    ElMessage.warning('请输入提取数量')
    return
  }
  if (count > extractMax.value) {
    ElMessage.warning('提取数量不能超过当前可提取数量')
    return
  }
  try {
    await ElMessageBox.confirm(`确认提取 ${count} 个账号？`, '提示', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning'
    })
    extracting.value = true
    const res = await exportSalesQQCacheIniZip({
      count,
      recentMinutes: recentMinutesParam()
    })
    const ok = await handleZipDownload(res, qqCacheExtractZipName(count))
    if (ok) {
      page.value = 1
      await fetchAll()
    }
  } catch (e) {
    if (e === 'cancel' || e === 'close') return
    ElMessage.error(e?.message || '提取失败')
  } finally {
    extracting.value = false
  }
}

const handleCurrentChange = async (val) => {
  page.value = val
  await fetchHistory()
}

const handleSizeChange = async (val) => {
  pageSize.value = val
  page.value = 1
  await fetchHistory()
}

const onExtractCustomHoursChange = async () => {
  const normalized = normalizePositiveInteger(extractCustomHours.value)
  if (normalized === null) {
    ElMessage.warning('自定义范围请输入正整数小时')
    extractCustomHours.value = undefined
  } else {
    extractCustomHours.value = normalized
    extractRecentMinutes.value = undefined
  }
  await fetchSummary()
}

onMounted(() => {
  fetchAll()
})
</script>

<style scoped>
.extract-panel {
  display: flex;
  align-items: center;
  gap: 12px;
}

.extract-label {
  color: #606266;
  font-size: 14px;
}

.extract-range-count {
  white-space: nowrap;
}

.table-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 12px;
}

.table-title {
  color: #303133;
  font-size: 15px;
  font-weight: 600;
}
</style>
