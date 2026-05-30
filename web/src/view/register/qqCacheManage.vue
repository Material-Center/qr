<template>
  <div>
    <div class="gva-search-box">
      <el-form :inline="true" :model="searchInfo">
        <el-form-item label="创建时间">
          <el-date-picker
            v-model="searchInfo.createdAtRange"
            type="datetimerange"
            value-format="YYYY-MM-DD HH:mm:ss"
            :shortcuts="createdAtShortcuts"
            range-separator="至"
            start-placeholder="开始时间"
            end-placeholder="结束时间"
            style="width: 360px"
          />
        </el-form-item>
        <el-form-item label="QQ账号">
          <el-input v-model="searchInfo.qqNum" clearable placeholder="请输入QQ账号" />
        </el-form-item>
        <el-form-item label="设备ID">
          <el-input v-model="searchInfo.deviceId" clearable placeholder="请输入设备ID" />
        </el-form-item>
        <el-form-item label="版本号">
          <el-input v-model="searchInfo.clientVersion" clearable placeholder="请输入版本号" />
        </el-form-item>
        <el-form-item label="提取状态">
          <el-select v-model="searchInfo.extracted" clearable style="width: 140px">
            <el-option :value="true" label="已提取" />
            <el-option :value="false" label="未提取" />
          </el-select>
        </el-form-item>
        <el-form-item label="提取人">
          <el-select
            v-model="searchInfo.extractorId"
            clearable
            filterable
            placeholder="请选择提取人"
            style="width: 180px"
          >
            <el-option
              v-for="item in salesSummaryList"
              :key="item.extractorId"
              :label="item.extractorName || item.username || `ID ${item.extractorId}`"
              :value="item.extractorId"
            />
          </el-select>
        </el-form-item>
        <el-form-item>
          <el-button type="primary" icon="search" @click="fetchList">查询</el-button>
          <el-button icon="refresh" @click="resetSearch">重置</el-button>
        </el-form-item>
      </el-form>
      <div class="qq-cache-tool-row">
        <el-button type="success" :disabled="!selectedRows.length" @click="onExportIniZip">下载INI</el-button>
        <el-upload
          class="qq-file-upload"
          accept=".txt,text/plain"
          :show-file-list="false"
          :auto-upload="false"
          :on-change="onExportByQQFile"
        >
          <el-button type="info">按TXT导出</el-button>
        </el-upload>
        <el-upload
          class="qq-file-upload"
          accept=".txt,text/plain"
          :show-file-list="false"
          :auto-upload="false"
          :on-change="onExtractByQQFile"
        >
          <el-button type="warning" plain>按TXT提取</el-button>
        </el-upload>
        <input
          ref="importZipInputRef"
          class="qq-cache-hidden-input"
          type="file"
          accept=".zip,application/zip"
          multiple
          @change="onImportZipChange"
        >
        <el-button type="success" plain @click="triggerImportZipInput">导入缓存包</el-button>
        <div class="account-list-export-tool">
          <el-tooltip
            effect="dark"
            :content="accountListExportHint"
            placement="top"
          >
            <el-button type="primary" plain @click="onExportAccountList">导出账号列表</el-button>
          </el-tooltip>
        </div>
        <div class="extract-tool">
          <span class="extract-label">提取数量</span>
          <el-input-number
            v-model="extractCount"
            :min="1"
            :max="extractMax"
            :disabled="extractMax <= 0"
            controls-position="right"
            style="width: 140px"
          />
          <el-button
            type="warning"
            :disabled="extractMax <= 0"
            @click="onExportPendingIniZip"
          >
            提取INI
          </el-button>
        </div>
      </div>
    </div>

    <div class="gva-table-box">
      <el-row :gutter="12" class="mb-3">
        <el-col :xs="24" :sm="12" :md="6">
          <el-card shadow="never">待提取数量：{{ extractStats.pending }}</el-card>
        </el-col>
        <el-col :xs="24" :sm="12" :md="6">
          <el-card shadow="never">已提取数量：{{ extractStats.extracted }}</el-card>
        </el-col>
        <el-col :xs="24" :sm="12" :md="6">
          <el-card shadow="never">总数：{{ extractStats.total }}</el-card>
        </el-col>
        <el-col :xs="24" :sm="12" :md="6">
          <el-card shadow="never">
            <div class="billing-stat-card">
              <span>待结算数量：{{ extractStats.billingUnsettled }}</span>
              <div class="billing-actions">
                <el-button
                  type="primary"
                  size="small"
                  :disabled="extractStats.billingUnsettled <= 0"
                  @click="onSettleBilling"
                >
                  结算
                </el-button>
                <el-button size="small" @click="onOpenBillingHistory">
                  历史结算
                </el-button>
              </div>
            </div>
          </el-card>
        </el-col>
      </el-row>

      <el-table :data="tableData" row-key="ID" @selection-change="onSelectionChange">
        <el-table-column type="selection" width="48" reserve-selection />
        <el-table-column label="ID" prop="ID" width="80" />
        <el-table-column label="QQ账号" prop="qqNum" min-width="140" />
        <el-table-column label="版本号" prop="clientVersion" min-width="110" />
        <el-table-column label="设备ID" prop="deviceId" min-width="160" show-overflow-tooltip />
        <el-table-column label="提取人" min-width="120">
          <template #default="{ row }">
            {{ extractorDisplay(row.extractor) }}
          </template>
        </el-table-column>
        <el-table-column label="提取时间" min-width="170">
          <template #default="{ row }">
            {{ row.extractionAt ? formatDate(row.extractionAt) : '-' }}
          </template>
        </el-table-column>
        <el-table-column label="创建时间" min-width="170">
          <template #default="{ row }">
            {{ getRowTime(row, 'createdAt') }}
          </template>
        </el-table-column>
        <el-table-column label="更新时间" min-width="170">
          <template #default="{ row }">
            {{ getRowTime(row, 'updatedAt') }}
          </template>
        </el-table-column>
        <el-table-column label="操作" width="120" fixed="right">
          <template #default="{ row }">
            <el-button type="primary" link :disabled="!row.extractor" @click="onResetExtract(row)">
              重置提取
            </el-button>
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

    <div class="gva-table-box">
      <div class="sales-summary-header">
        <span class="sales-summary-title">销售提取汇总</span>
        <el-button icon="refresh" @click="fetchSalesSummary">刷新</el-button>
      </div>
      <el-table :data="salesSummaryList" row-key="extractorId" size="small">
        <el-table-column label="销售" min-width="160">
          <template #default="{ row }">
            {{ row.extractorName || row.nickName || row.username || `ID ${row.extractorId}` }}
          </template>
        </el-table-column>
        <el-table-column label="提取数量" prop="extractedCount" min-width="100" />
        <el-table-column label="已结算总数" prop="settledCount" min-width="110" />
        <el-table-column label="待结算总数" prop="unsettledCount" min-width="110" />
        <el-table-column label="最近提取时间" min-width="170">
          <template #default="{ row }">
            {{ row.lastExtractionAt ? formatDate(row.lastExtractionAt) : '-' }}
          </template>
        </el-table-column>
        <el-table-column label="操作" width="140" fixed="right">
          <template #default="{ row }">
            <el-button
              type="primary"
              link
              :disabled="Number(row.unsettledCount) <= 0"
              @click="onSettleSales(row)"
            >
              标记已结算
            </el-button>
          </template>
        </el-table-column>
      </el-table>
    </div>

    <el-dialog v-model="billingHistoryVisible" title="QQ缓存结算历史" width="520px">
      <el-table v-loading="billingHistoryLoading" :data="billingHistory" size="small">
        <el-table-column label="结算时间" min-width="180">
          <template #default="{ row }">
            {{ row.settledAt ? formatDate(row.settledAt) : '-' }}
          </template>
        </el-table-column>
        <el-table-column label="数量" prop="settledCount" width="120" />
      </el-table>
    </el-dialog>
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { formatDate } from '@/utils/format'
import {
  exportQQCacheAccountList,
  exportPendingQQCacheIniZip,
  exportQQCacheIniZipByQQFile,
  exportQQCacheIniZip,
  getQQCacheBillingHistory,
  getQQCacheList,
  getQQCacheSalesSummaryList,
  importQQCacheZip,
  resetQQCacheExtract,
  settleQQCacheBilling,
  settleQQCacheSalesBilling
} from '@/api/qqCache'

defineOptions({
  name: 'QQCacheManage'
})

const page = ref(1)
const pageSize = ref(10)
const total = ref(0)
const tableData = ref([])
const selectedRows = ref([])
const importZipInputRef = ref(null)
const extractCount = ref(1)
const extractStats = ref({
  pending: 0,
  extracted: 0,
  total: 0,
  billingUnsettled: 0,
  billingSettled: 0
})
const billingHistoryVisible = ref(false)
const billingHistoryLoading = ref(false)
const billingHistory = ref([])
const salesSummaryList = ref([])
const searchInfo = ref({
  createdAtRange: [],
  qqNum: '',
  clientVersion: '',
  deviceId: '',
  extracted: undefined,
  extractorId: undefined
})

const onSelectionChange = (rows) => {
  selectedRows.value = rows || []
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

const getRowTime = (row, key) => {
  const value = row?.[key] || row?.[key.charAt(0).toUpperCase() + key.slice(1)]
  return value ? formatDate(value) : '-'
}

const extractMax = computed(() => Math.max(Number(extractStats.value.pending) || 0, 0))

const accountListExportHint = computed(() => {
  const selectedCount = selectedRows.value.length
  if (selectedCount > 0) {
    return `将导出选中的 ${selectedCount} 条记录`
  }
  return '未勾选时按当前筛选条件导出'
})

const salesSummaryMap = computed(() => {
  const map = new Map()
  ;(salesSummaryList.value || []).forEach((item) => {
    map.set(Number(item.extractorId), item)
  })
  return map
})

const extractorDisplay = (extractorId) => {
  if (!extractorId) return '-'
  const item = salesSummaryMap.value.get(Number(extractorId))
  if (!item) return `ID ${extractorId}`
  return item.extractorName || item.nickName || item.username || `ID ${extractorId}`
}

const dayStart = (base = new Date()) => {
  const d = new Date(base)
  d.setHours(0, 0, 0, 0)
  return d
}

const dayEnd = (base = new Date()) => {
  const d = new Date(base)
  d.setHours(23, 59, 59, 999)
  return d
}

const formatQueryDateTime = (date) => {
  const pad = (n) => String(n).padStart(2, '0')
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())} ${pad(date.getHours())}:${pad(date.getMinutes())}:${pad(date.getSeconds())}`
}

const todayDateTimeRange = () => {
  const now = new Date()
  return [formatQueryDateTime(dayStart(now)), formatQueryDateTime(dayEnd(now))]
}

const shiftDay = (base, days) => {
  const d = new Date(base)
  d.setDate(d.getDate() + days)
  return d
}

const createdAtShortcuts = [
  {
    text: '今天',
    value: () => {
      const now = new Date()
      return [dayStart(now), now]
    }
  },
  {
    text: '昨天',
    value: () => {
      const now = new Date()
      const target = shiftDay(now, -1)
      return [dayStart(target), dayEnd(target)]
    }
  },
  {
    text: '前天',
    value: () => {
      const now = new Date()
      const target = shiftDay(now, -2)
      return [dayStart(target), dayEnd(target)]
    }
  },
  {
    text: '近一周',
    value: () => {
      const now = new Date()
      return [dayStart(shiftDay(now, -6)), now]
    }
  },
  {
    text: '近一个月',
    value: () => {
      const now = new Date()
      return [dayStart(shiftDay(now, -29)), now]
    }
  }
]

const handleZipDownload = async (res, fallbackName) => {
  const ct = String(res?.headers?.['content-type'] || '').toLowerCase()
  const blob = res?.data instanceof Blob ? res.data : null
  if (!blob) {
    ElMessage.error('导出失败')
    return false
  }
  if (ct.includes('application/json')) {
    const text = await blob.text()
    let msg = '导出失败'
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

const qqCacheExtractZipName = (count) => {
  const d = new Date()
  const pad = (n) => String(n).padStart(2, '0')
  const timestamp = `${d.getFullYear()}${pad(d.getMonth() + 1)}${pad(d.getDate())}-${pad(d.getHours())}${pad(d.getMinutes())}`
  return `qq-${Number(count) || 0}个-${timestamp}.zip`
}

const countQQNumsFromTextFile = async (file) => {
  const raw = await file.text()
  const seen = new Set()
  raw.replace(/\r\n/g, '\n').split('\n').forEach((line) => {
    const text = String(line || '').trim()
    if (!text) return
    const qqNum = text.split('----')[0].trim()
    if (/^\d+$/.test(qqNum)) {
      seen.add(qqNum)
    }
  })
  return seen.size
}

const handleFileDownload = async (res, fallbackName) => {
  const ct = String(res?.headers?.['content-type'] || '').toLowerCase()
  const blob = res?.data instanceof Blob ? res.data : null
  if (!blob) {
    ElMessage.error('导出失败')
    return false
  }
  if (ct.includes('application/json')) {
    const text = await blob.text()
    let msg = '导出失败'
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
  a.download = name
  a.click()
  window.URL.revokeObjectURL(url)
  ElMessage.success('已开始下载')
  return true
}

const onExportIniZip = async () => {
  const ids = (selectedRows.value || []).map((r) => r.ID).filter(Boolean)
  if (!ids.length) {
    ElMessage.warning('请先勾选要导出的记录')
    return
  }
  try {
    const res = await exportQQCacheIniZip(ids)
    await handleZipDownload(res, qqCacheExtractZipName(ids.length))
  } catch (e) {
    ElMessage.error(e?.message || '导出失败')
  }
}

const buildAccountListExportPayload = () => {
  const ids = (selectedRows.value || []).map((r) => r.ID).filter(Boolean)
  if (ids.length) {
    return { ids }
  }
  const [createdAtStart, createdAtEnd] = searchInfo.value.createdAtRange || []
  return {
    qqNum: searchInfo.value.qqNum || undefined,
    clientVersion: searchInfo.value.clientVersion || undefined,
    deviceId: searchInfo.value.deviceId || undefined,
    extractorId: searchInfo.value.extractorId || undefined,
    extracted: searchInfo.value.extracted,
    createdAtStart: createdAtStart || undefined,
    createdAtEnd: createdAtEnd || undefined
  }
}

const onExportAccountList = async () => {
  try {
    const res = await exportQQCacheAccountList(buildAccountListExportPayload())
    await handleFileDownload(res, `qq_account_list_${Date.now()}.txt`)
  } catch (e) {
    ElMessage.error(e?.message || '导出失败')
  }
}

const onExportPendingIniZip = async () => {
  const count = Number(extractCount.value) || 0
  if (count <= 0) {
    ElMessage.warning('请输入提取数量')
    return
  }
  if (count > extractMax.value) {
    ElMessage.warning('提取数量不能超过待提取数量')
    return
  }
  try {
    await ElMessageBox.confirm(`确认提取 ${count} 个未提取缓存？`, '提示', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning'
    })
    const res = await exportPendingQQCacheIniZip({
      count
    })
    const ok = await handleZipDownload(res, qqCacheExtractZipName(count))
    if (ok) {
      await fetchList()
    }
  } catch (e) {
    if (e === 'cancel' || e === 'close') return
    ElMessage.error(e?.message || '提取失败')
  }
}

const onExportByQQFile = async (uploadFile) => {
  const file = uploadFile?.raw
  if (!file) {
    ElMessage.warning('请先选择TXT文件')
    return
  }
  try {
    const fallbackCount = await countQQNumsFromTextFile(file)
    const res = await exportQQCacheIniZipByQQFile(file)
    await handleZipDownload(res, qqCacheExtractZipName(fallbackCount))
  } catch (e) {
    ElMessage.error(e?.message || '导出失败')
  }
}

const onExtractByQQFile = async (uploadFile) => {
  const file = uploadFile?.raw
  if (!file) {
    ElMessage.warning('请先选择TXT文件')
    return
  }
  try {
    await ElMessageBox.confirm('确认按TXT提取并标记这些账号为已提取？', '提示', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning'
    })
    const fallbackCount = await countQQNumsFromTextFile(file)
    const res = await exportQQCacheIniZipByQQFile(file, { markExtracted: true })
    const ok = await handleZipDownload(res, qqCacheExtractZipName(fallbackCount))
    if (ok) {
      await fetchList()
    }
  } catch (e) {
    if (e === 'cancel' || e === 'close') return
    ElMessage.error(e?.message || '提取失败')
  }
}

const triggerImportZipInput = () => {
  if (!importZipInputRef.value) return
  importZipInputRef.value.value = ''
  importZipInputRef.value.click()
}

const onImportZipChange = async (event) => {
  const files = Array.from(event?.target?.files || [])
  if (!files.length) {
    ElMessage.warning('请先选择ZIP文件')
    return
  }
  const invalid = files.find((file) => !String(file.name || '').toLowerCase().endsWith('.zip'))
  if (invalid) {
    ElMessage.warning('请上传ZIP缓存包')
    return
  }
  try {
    const res = await importQQCacheZip(files)
    if (res?.code !== 0) {
      return
    }
    const { data } = res
    if (Array.isArray(data?.results)) {
      const success = Number(data?.success) || 0
      const failed = Number(data?.failed) || 0
      const message = `导入完成：成功 ${success} 个，失败 ${failed} 个`
      if (failed > 0) {
        ElMessage.warning(message)
      } else {
        ElMessage.success(message)
      }
    } else {
      const action = data?.action === 'updated' ? '已覆盖' : '已导入'
      ElMessage.success(`${data?.qqNum || '缓存包'}${action}`)
    }
    await fetchList()
  } catch (e) {
    ElMessage.error(e?.message || '导入失败')
  } finally {
    if (event?.target) {
      event.target.value = ''
    }
  }
}

const onSettleBilling = async () => {
  const count = Number(extractStats.value.billingUnsettled) || 0
  if (count <= 0) return
  try {
    await ElMessageBox.confirm(`确认结算当前 ${count} 个待结算账号？`, '确认结算', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning'
    })
    const { data } = await settleQQCacheBilling()
    ElMessage.success(`已结算 ${data?.settledCount || 0} 个账号`)
    await fetchList()
  } catch (e) {
    if (e === 'cancel' || e === 'close') return
    ElMessage.error(e?.message || '结算失败')
  }
}

const onSettleSales = async (row) => {
  const count = Number(row?.unsettledCount) || 0
  if (!row?.extractorId || count <= 0) return
  const name = row.extractorName || row.nickName || row.username || `ID ${row.extractorId}`
  try {
    await ElMessageBox.confirm(`确认将 ${name} 的 ${count} 个待结算账号标记为已结算？`, '确认结算', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning'
    })
    const { data } = await settleQQCacheSalesBilling({ extractorId: row.extractorId })
    ElMessage.success(`已结算 ${data?.settledCount || 0} 个账号`)
    await Promise.all([fetchList(), fetchSalesSummary()])
  } catch (e) {
    if (e === 'cancel' || e === 'close') return
    ElMessage.error(e?.message || '结算失败')
  }
}

const onOpenBillingHistory = async () => {
  billingHistory.value = []
  billingHistoryVisible.value = true
  billingHistoryLoading.value = true
  try {
    const { data } = await getQQCacheBillingHistory()
    billingHistory.value = data || []
  } catch (e) {
    ElMessage.error(e?.message || '结算历史加载失败')
  } finally {
    billingHistoryLoading.value = false
  }
}

const fetchList = async () => {
  try {
    const [createdAtStart, createdAtEnd] = searchInfo.value.createdAtRange || []
    const { data } = await getQQCacheList({
      page: page.value,
      pageSize: pageSize.value,
      qqNum: searchInfo.value.qqNum || undefined,
      clientVersion: searchInfo.value.clientVersion || undefined,
      deviceId: searchInfo.value.deviceId || undefined,
      extractorId: searchInfo.value.extractorId || undefined,
      extracted: searchInfo.value.extracted,
      createdAtStart: createdAtStart || undefined,
      createdAtEnd: createdAtEnd || undefined
    })
    tableData.value = data?.list || []
    total.value = data?.total || 0
    extractStats.value = {
      pending: Number(data?.stats?.pending) || 0,
      extracted: Number(data?.stats?.extracted) || 0,
      total: Number(data?.stats?.total) || 0,
      billingUnsettled: Number(data?.stats?.billingUnsettled) || 0,
      billingSettled: Number(data?.stats?.billingSettled) || 0
    }
    if (extractMax.value > 0 && extractCount.value > extractMax.value) {
      extractCount.value = extractMax.value
    }
  } catch (e) {
    ElMessage.error(e?.message || '加载失败')
  }
}

const fetchSalesSummary = async () => {
  try {
    const { data } = await getQQCacheSalesSummaryList()
    salesSummaryList.value = data || []
  } catch (e) {
    ElMessage.error(e?.message || '销售汇总加载失败')
  }
}

const resetSearch = () => {
  searchInfo.value = {
    createdAtRange: todayDateTimeRange(),
    qqNum: '',
    clientVersion: '',
    deviceId: '',
    extracted: undefined,
    extractorId: undefined
  }
  page.value = 1
  fetchList()
}

const handleCurrentChange = (val) => {
  page.value = val
  fetchList()
}

const handleSizeChange = (val) => {
  pageSize.value = val
  page.value = 1
  fetchList()
}

const onResetExtract = async (row) => {
  if (!row?.ID) return
  await ElMessageBox.confirm('确认重置该账号的提取状态？', '提示', {
    confirmButtonText: '确定',
    cancelButtonText: '取消',
    type: 'warning'
  })
  await resetQQCacheExtract({ id: row.ID })
  ElMessage.success('重置成功')
  await fetchList()
}

onMounted(() => {
  searchInfo.value.createdAtRange = todayDateTimeRange()
  fetchSalesSummary()
  fetchList()
})
</script>

<style scoped>
.qq-file-upload {
  display: inline-flex;
  vertical-align: middle;
}

.qq-cache-hidden-input {
  display: none;
}

.qq-cache-tool-row {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 10px;
  margin-top: 8px;
  padding-top: 12px;
  border-top: 1px solid var(--el-border-color-lighter);
}

.extract-tool {
  display: inline-flex;
  align-items: center;
  gap: 8px;
}

.account-list-export-tool {
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

.extract-label {
  color: var(--el-text-color-regular);
  font-size: 14px;
  white-space: nowrap;
}

.billing-stat-card {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}

.billing-actions {
  display: flex;
  align-items: center;
  gap: 6px;
  white-space: nowrap;
}

.billing-actions :deep(.el-button + .el-button) {
  margin-left: 0;
}

.sales-summary-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 12px;
}

.sales-summary-title {
  color: var(--el-text-color-primary);
  font-size: 15px;
  font-weight: 600;
}

@media (max-width: 900px) {
  .billing-stat-card {
    align-items: flex-start;
    flex-direction: column;
  }
}
</style>
