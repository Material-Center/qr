<template>
  <div>
    <div class="gva-search-box">
      <el-form :inline="true" :model="searchInfo">
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
      </el-form>
    </div>

    <div class="gva-table-box">
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
        <el-table-column label="更新时间" min-width="170">
          <template #default="{ row }">
            {{ row.updatedAt ? formatDate(row.updatedAt) : '-' }}
          </template>
        </el-table-column>
        <el-table-column label="缓存状态" min-width="100">
          <template #default="{ row }">
            <el-tag :type="row.iNI ? 'success' : 'warning'">
              {{ row.iNI ? '有缓存' : '空缓存' }}
            </el-tag>
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
import { onMounted, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { formatDate } from '@/utils/format'
import { exportQQCacheIniZip, getQQCacheList, resetQQCacheExtract } from '@/api/qqCache'

defineOptions({
  name: 'QQCacheManage'
})

const page = ref(1)
const pageSize = ref(10)
const total = ref(0)
const tableData = ref([])
const selectedRows = ref([])
const searchInfo = ref({
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

const onExportIniZip = async () => {
  const ids = (selectedRows.value || []).map((r) => r.ID).filter(Boolean)
  if (!ids.length) {
    ElMessage.warning('请先勾选要导出的记录')
    return
  }
  try {
    const res = await exportQQCacheIniZip(ids)
    const ct = String(res?.headers?.['content-type'] || '').toLowerCase()
    const blob = res?.data instanceof Blob ? res.data : null
    if (!blob) {
      ElMessage.error('导出失败')
      return
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
      return
    }
    const name = pickZipFilename(res.headers?.['content-disposition']) || `qq_cache_ini_${Date.now()}.zip`
    const url = window.URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = name.endsWith('.zip') ? name : `${name}.zip`
    a.click()
    window.URL.revokeObjectURL(url)
    ElMessage.success('已开始下载')
  } catch (e) {
    ElMessage.error(e?.message || '导出失败')
  }
}

const fetchList = async () => {
  try {
    const { data } = await getQQCacheList({
      page: page.value,
      pageSize: pageSize.value,
      qqNum: searchInfo.value.qqNum || undefined,
      deviceId: searchInfo.value.deviceId || undefined,
      extracted: searchInfo.value.extracted
    })
    tableData.value = data?.list || []
    total.value = data?.total || 0
  } catch (e) {
    ElMessage.error(e?.message || '加载失败')
  }
}

const resetSearch = () => {
  searchInfo.value = {
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
