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
        </el-form-item>
      </el-form>
    </div>

    <div class="gva-table-box">
      <el-table :data="tableData" row-key="ID">
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
import { getQQCacheList, resetQQCacheExtract } from '@/api/qqCache'

defineOptions({
  name: 'QQCacheManage'
})

const page = ref(1)
const pageSize = ref(10)
const total = ref(0)
const tableData = ref([])
const searchInfo = ref({
  qqNum: '',
  deviceId: '',
  extracted: undefined
})

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
