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
          <el-card shadow="never">我已提取总数：{{ summary.todayExtracted }}</el-card>
        </el-col>
      </el-row>

      <div class="extract-panel">
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
const page = ref(1)
const pageSize = ref(10)
const total = ref(0)
const historyData = ref([])
const extracting = ref(false)

const extractMax = computed(() => Math.max(Number(summary.value.available) || 0, 0))

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

const handleZipDownload = async (res, fallbackName) => {
  const ct = String(res?.headers?.['content-type'] || '').toLowerCase()
  const blob = res?.data instanceof Blob ? res.data : null
  if (!blob) {
    ElMessage.error('提取失败')
    return false
  }
  if (ct.includes('application/json')) {
    const text = await blob.text()
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
  const name = pickZipFilename(res.headers?.['content-disposition']) || fallbackName
  const url = window.URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = name.endsWith('.zip') ? name : `${name}.zip`
  a.click()
  window.URL.revokeObjectURL(url)
  ElMessage.success('已开始下载')
  return true
}

const fetchSummary = async () => {
  const { data } = await getQQCacheSalesSummary()
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
    const res = await exportSalesQQCacheIniZip({ count })
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
