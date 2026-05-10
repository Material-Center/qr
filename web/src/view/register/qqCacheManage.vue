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
        <el-form-item label="提取状态">
          <el-select v-model="searchInfo.extracted" clearable style="width: 140px">
            <el-option :value="true" label="已提取" />
            <el-option :value="false" label="未提取" />
          </el-select>
        </el-form-item>
        <el-form-item>
          <el-button type="primary" icon="search" @click="fetchList">查询</el-button>
          <el-button icon="refresh" @click="resetSearch">重置</el-button>
          <el-button type="success" :disabled="!selectedRows.length" @click="onExportIniZip">下载INI(ZIP)</el-button>
        </el-form-item>
        <el-form-item label="提取数量">
          <el-input-number
            v-model="extractCount"
            :min="1"
            :max="extractMax"
            :disabled="extractMax <= 0"
            controls-position="right"
            style="width: 140px"
          />
          <el-button
            class="extract-btn"
            type="warning"
            :disabled="extractMax <= 0"
            @click="onExportPendingIniZip"
          >
            提取INI
          </el-button>
        </el-form-item>
      </el-form>
    </div>

    <div class="gva-table-box">
      <el-row :gutter="12" class="mb-3">
        <el-col :span="8">
          <el-card shadow="never">待提取数量：{{ extractStats.pending }}</el-card>
        </el-col>
        <el-col :span="8">
          <el-card shadow="never">已提取数量：{{ extractStats.extracted }}</el-card>
        </el-col>
        <el-col :span="8">
          <el-card shadow="never">总数：{{ extractStats.total }}</el-card>
        </el-col>
      </el-row>

      <el-table :data="tableData" row-key="ID" @selection-change="onSelectionChange">
        <el-table-column type="selection" width="48" reserve-selection />
        <el-table-column label="ID" prop="ID" width="80" />
        <el-table-column label="QQ账号" prop="qqNum" min-width="140" />
        <el-table-column label="设备ID" prop="deviceId" min-width="160" show-overflow-tooltip />
        <el-table-column label="提取人ID" min-width="100">
          <template #default="{ row }">
            {{ row.extractor || '-' }}
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
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { formatDate } from '@/utils/format'
import { exportPendingQQCacheIniZip, exportQQCacheIniZip, getQQCacheList, resetQQCacheExtract } from '@/api/qqCache'

defineOptions({
  name: 'QQCacheManage'
})

const page = ref(1)
const pageSize = ref(10)
const total = ref(0)
const tableData = ref([])
const selectedRows = ref([])
const extractCount = ref(1)
const extractStats = ref({
  pending: 0,
  extracted: 0,
  total: 0
})
const searchInfo = ref({
  createdAtRange: [],
  qqNum: '',
  deviceId: '',
  extracted: undefined
})

const onSelectionChange = (rows) => {
  selectedRows.value = rows || []
}

const pickZipFilename = (contentDisposition) => {
  if (!contentDisposition) return null
  const m = /filename\*?=(?:UTF-8'')?["']?([^"';]+)/i.exec(contentDisposition)
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

const onExportIniZip = async () => {
  const ids = (selectedRows.value || []).map((r) => r.ID).filter(Boolean)
  if (!ids.length) {
    ElMessage.warning('请先勾选要导出的记录')
    return
  }
  try {
    const res = await exportQQCacheIniZip(ids)
    await handleZipDownload(res, `qq_cache_ini_${Date.now()}.zip`)
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
    const res = await exportPendingQQCacheIniZip(count)
    const ok = await handleZipDownload(res, `qq_cache_ini_${Date.now()}.zip`)
    if (ok) {
      await fetchList()
    }
  } catch (e) {
    if (e === 'cancel' || e === 'close') return
    ElMessage.error(e?.message || '提取失败')
  }
}

const fetchList = async () => {
  try {
    const [createdAtStart, createdAtEnd] = searchInfo.value.createdAtRange || []
    const { data } = await getQQCacheList({
      page: page.value,
      pageSize: pageSize.value,
      qqNum: searchInfo.value.qqNum || undefined,
      deviceId: searchInfo.value.deviceId || undefined,
      extracted: searchInfo.value.extracted,
      createdAtStart: createdAtStart || undefined,
      createdAtEnd: createdAtEnd || undefined
    })
    tableData.value = data?.list || []
    total.value = data?.total || 0
    extractStats.value = {
      pending: Number(data?.stats?.pending) || 0,
      extracted: Number(data?.stats?.extracted) || 0,
      total: Number(data?.stats?.total) || 0
    }
    if (extractMax.value > 0 && extractCount.value > extractMax.value) {
      extractCount.value = extractMax.value
    }
  } catch (e) {
    ElMessage.error(e?.message || '加载失败')
  }
}

const resetSearch = () => {
  searchInfo.value = {
    createdAtRange: [],
    qqNum: '',
    deviceId: '',
    extracted: undefined
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
  fetchList()
})
</script>

<style scoped>
.extract-btn {
  margin-left: 8px;
}
</style>
