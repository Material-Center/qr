<template>
  <div class="account-manage-page">
    <div class="gva-search-box">
      <el-form :inline="true" :model="searchInfo">
        <el-form-item label="用户名">
          <el-input v-model="searchInfo.username" placeholder="用户名" />
        </el-form-item>
        <el-form-item label="昵称">
          <el-input v-model="searchInfo.nickname" placeholder="昵称" />
        </el-form-item>
        <el-form-item label="手机号">
          <el-input v-model="searchInfo.phone" placeholder="手机号" />
        </el-form-item>
        <el-form-item label="邮箱">
          <el-input v-model="searchInfo.email" placeholder="邮箱" />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" icon="search" @click="onSubmit">查询</el-button>
          <el-button icon="refresh" @click="onReset">重置</el-button>
        </el-form-item>
      </el-form>
    </div>

    <div class="gva-table-box">
      <div class="gva-btn-list">
        <el-button v-if="canManage" type="primary" icon="plus" @click="openAdd">
          新增账号
        </el-button>
      </div>

      <el-table
        :data="tableData"
        :row-key="accountRowKey"
        class="account-table"
        :tree-props="{ children: 'children' }"
        :row-class-name="accountRowClassName"
      >
        <el-table-column v-if="useLeaderTree" width="44" />
        <el-table-column align="left" label="头像" min-width="75">
          <template #default="scope">
            <CustomPic style="margin-top: 8px" :pic-src="scope.row.headerImg" />
          </template>
        </el-table-column>
        <el-table-column align="left" label="ID" min-width="60" prop="ID" />
        <el-table-column align="left" label="用户名" min-width="140" prop="userName">
          <template #default="scope">
            <div :class="['account-name-cell', { 'is-child': scope.row._relationChild }]">
              <span v-if="scope.row._relationChild" class="child-branch" />
              <span>{{ scope.row.userName }}</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column align="left" label="昵称" min-width="140" prop="nickName" />
        <!-- <el-table-column align="left" label="手机号" min-width="160" prop="phone" /> -->
        <!-- <el-table-column align="left" label="邮箱" min-width="180" prop="email" /> -->
        <el-table-column align="left" label="最近登录IP" min-width="150" prop="lastLoginIp" />
        <el-table-column align="left" label="最近登录时间" min-width="190">
          <template #default="scope">
            <span>{{ formatDateText(scope.row.lastLoginAt) }}</span>
          </template>
        </el-table-column>
        <el-table-column align="left" label="角色" min-width="120">
          <template #default="scope">
            <el-tag>{{ roleText(scope.row.authorityId) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column align="left" label="从属关系" min-width="170">
          <template #default="scope">
            <span>{{ relationText(scope.row) }}</span>
          </template>
        </el-table-column>
        <el-table-column v-if="useLeaderTree" align="left" label="风控比例" min-width="190">
          <template #default="scope">
            <div v-if="canConfigureCacheSample(scope.row)" class="cache-sample-cell">
              <span>{{ cacheSampleText(scope.row) }}</span>
              <el-button type="primary" link @click="openCacheSampleDialog(scope.row)">
                配置
              </el-button>
            </div>
            <span v-else>-</span>
          </template>
        </el-table-column>
        <el-table-column align="left" label="启用" min-width="110">
          <template #default="scope">
            <el-switch
              v-model="scope.row.enable"
              inline-prompt
              :active-value="1"
              :inactive-value="2"
              @change="() => switchEnable(scope.row)"
            />
          </template>
        </el-table-column>
        <el-table-column align="left" label="禁用创建任务" min-width="130">
          <template #default="scope">
            <el-switch
              v-model="scope.row.phoneRegisterTaskDisabled"
              inline-prompt
              active-text="禁用"
              inactive-text="正常"
              :disabled="scope.row.authorityId !== ROLE_PROMOTER"
              @change="() => switchTaskCreate(scope.row)"
            />
          </template>
        </el-table-column>
        <el-table-column label="操作" min-width="300" fixed="right">
          <template #default="scope">
            <el-button type="primary" link icon="edit" @click="openEdit(scope.row)">
              编辑
            </el-button>
            <el-button type="primary" link icon="delete" @click="deleteUserFunc(scope.row)">
              删除
            </el-button>
            <el-button type="primary" link icon="magic-stick" @click="openResetPwd(scope.row)">
              重置密码
            </el-button>
            <el-button
              v-if="canCreatePromoterToken(scope.row)"
              type="primary"
              link
              icon="key"
              @click="createPromoterToken(scope.row)"
            >
              生成Token
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

    <el-dialog
      v-model="resetPwdDialog"
      title="重置密码"
      width="500px"
      destroy-on-close
      :close-on-click-modal="false"
      :close-on-press-escape="false"
    >
      <el-form :model="resetPwdInfo" label-width="90px">
        <el-form-item label="用户账号">
          <el-input v-model="resetPwdInfo.userName" disabled />
        </el-form-item>
        <el-form-item label="用户昵称">
          <el-input v-model="resetPwdInfo.nickName" disabled />
        </el-form-item>
        <el-form-item label="新密码">
          <el-input v-model="resetPwdInfo.password" show-password />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="resetPwdDialog = false">取消</el-button>
        <el-button type="primary" @click="confirmResetPassword">确定</el-button>
      </template>
    </el-dialog>

    <el-dialog
      v-model="cacheSampleDialog"
      title="风控比例"
      width="420px"
      destroy-on-close
      :close-on-click-modal="false"
    >
      <el-form label-width="90px">
        <el-form-item label="账号">
          <span>{{ cacheSampleForm.nickName || cacheSampleForm.userName }}</span>
        </el-form-item>
        <el-form-item label="角色">
          <el-tag>{{ roleText(cacheSampleForm.authorityId) }}</el-tag>
        </el-form-item>
        <el-form-item label="配置方式">
          <el-radio-group v-model="cacheSampleForm.configured">
            <el-radio-button :label="false">
              不配置
            </el-radio-button>
            <el-radio-button :label="true">
              自定义
            </el-radio-button>
          </el-radio-group>
        </el-form-item>
        <el-form-item v-if="cacheSampleForm.configured" label="比例">
          <el-input-number
            v-model="cacheSampleForm.ratio"
            :min="0"
            :max="80"
            :step="1"
            controls-position="right"
          />
          <span class="cache-sample-percent">%</span>
        </el-form-item>
        <el-form-item label="生效说明">
          <span class="cache-sample-tip">{{ cacheSampleFormTip }}</span>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="cacheSampleDialog = false">取消</el-button>
        <el-button type="primary" @click="submitCacheSampleConfig">确定</el-button>
      </template>
    </el-dialog>

    <el-dialog
      v-model="tokenDialogVisible"
      title="OpenAPI Token"
      width="720px"
      :close-on-click-modal="false"
    >
      <el-input
        v-model="tokenResult"
        type="textarea"
        :rows="6"
        readonly
      />
      <template #footer>
        <el-button @click="tokenDialogVisible = false">关闭</el-button>
        <el-button type="primary" @click="copyToken">复制Token</el-button>
      </template>
    </el-dialog>

    <el-drawer
      v-model="showDrawer"
      :size="appStore.drawerSize"
      destroy-on-close
      :show-close="false"
    >
      <template #header>
        <div class="flex justify-between items-center">
          <span class="text-lg">{{ drawerMode === 'add' ? '新增账号' : '编辑账号' }}</span>
          <div>
            <el-button @click="showDrawer = false">取消</el-button>
            <el-button type="primary" @click="submitDrawer">确定</el-button>
          </div>
        </div>
      </template>

      <el-form ref="userFormRef" :model="userForm" :rules="rules" label-width="80px">
        <el-form-item v-if="drawerMode === 'add'" label="用户名" prop="userName">
          <el-input v-model="userForm.userName" />
        </el-form-item>
        <el-form-item v-if="drawerMode === 'add'" label="密码" prop="password">
          <el-input v-model="userForm.password" show-password />
        </el-form-item>
        <el-form-item label="昵称" prop="nickName">
          <el-input v-model="userForm.nickName" />
        </el-form-item>
        <el-form-item label="手机号" prop="phone">
          <el-input v-model="userForm.phone" />
        </el-form-item>
        <el-form-item label="邮箱" prop="email">
          <el-input v-model="userForm.email" />
        </el-form-item>
        <el-form-item label="角色" prop="authorityId">
          <el-select v-model="userForm.authorityId" :disabled="drawerMode === 'edit'" style="width: 100%">
            <el-option v-for="item in roleOptions" :key="item.value" :label="item.label" :value="item.value" />
          </el-select>
        </el-form-item>
        <el-form-item
          v-if="[ROLE_SUPER, ROLE_ADMIN].includes(currentRoleId) && userForm.authorityId === ROLE_PROMOTER && drawerMode === 'add'"
          label="所属团长"
        >
          <el-select v-model="userForm.leaderId" style="width: 100%" placeholder="请选择团长">
            <el-option
              v-for="leader in leaderOptions"
              :key="leader.ID"
              :label="`${leader.nickName}(${leader.userName})`"
              :value="leader.ID"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="启用">
          <el-switch v-model="userForm.enable" inline-prompt :active-value="1" :inactive-value="2" />
        </el-form-item>
        <el-form-item v-if="userForm.authorityId === ROLE_PROMOTER" label="禁用创建">
          <el-switch
            v-model="userForm.phoneRegisterTaskDisabled"
            active-text="禁用"
            inactive-text="正常"
          />
        </el-form-item>
        <el-form-item label="头像">
          <SelectImage v-model="userForm.headerImg" />
        </el-form-item>
      </el-form>
    </el-drawer>
  </div>
</template>

<script setup>
import { computed, nextTick, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useAppStore } from '@/pinia'
import { useUserStore } from '@/pinia/modules/user'
import { deleteUser, getUserList, register, resetPassword, setUserInfo } from '@/api/user'
import { createApiToken } from '@/api/sysApiToken'
import CustomPic from '@/components/customPic/index.vue'
import SelectImage from '@/components/selectImage/selectImage.vue'
import { formatDate } from '@/utils/format'

defineOptions({
  name: 'AccountManage'
})

const ROLE_SUPER = 888
const ROLE_ADMIN = 100
const ROLE_LEADER = 200
const ROLE_PROMOTER = 300
const ROLE_APP_EXTRACT = 400
const ROLE_APP_UPLOAD = 500

const appStore = useAppStore()
const userStore = useUserStore()

const currentRoleId = computed(() => userStore.userInfo?.authority?.authorityId)
const canManage = computed(() => [ROLE_SUPER, ROLE_ADMIN, ROLE_LEADER].includes(currentRoleId.value))
const useLeaderTree = computed(() => [ROLE_SUPER, ROLE_ADMIN].includes(currentRoleId.value))

const roleOptions = computed(() => {
  if (currentRoleId.value === ROLE_SUPER) {
    return [
      { label: '超级管理员', value: ROLE_SUPER },
      { label: '管理员', value: ROLE_ADMIN },
      { label: '团长', value: ROLE_LEADER },
      { label: '地推', value: ROLE_PROMOTER },
      { label: 'App提取', value: ROLE_APP_EXTRACT },
      { label: 'App上传', value: ROLE_APP_UPLOAD }
    ]
  }
  if (currentRoleId.value === ROLE_ADMIN) {
    return [
      { label: '团长', value: ROLE_LEADER },
      { label: '地推', value: ROLE_PROMOTER },
      { label: 'App提取', value: ROLE_APP_EXTRACT },
      { label: 'App上传', value: ROLE_APP_UPLOAD }
    ]
  }
  if (currentRoleId.value === ROLE_LEADER) {
    return [{ label: '地推', value: ROLE_PROMOTER }]
  }
  return []
})

const roleText = (authorityId) => {
  if (authorityId === ROLE_SUPER) return '超级管理员'
  if (authorityId === ROLE_ADMIN) return '管理员'
  if (authorityId === ROLE_LEADER) return '团长'
  if (authorityId === ROLE_PROMOTER) return '地推'
  if (authorityId === ROLE_APP_EXTRACT) return 'App提取'
  if (authorityId === ROLE_APP_UPLOAD) return 'App上传'
  return '未知'
}

const leaderDisplayText = (row) => {
  const name = row.leaderName || ''
  const username = row.leaderUserName || ''
  if (name && username && name !== username) return `${name}(${username})`
  return name || username || (row.leaderId ? `ID ${row.leaderId}` : '')
}

const relationText = (row) => {
  if (row.authorityId === ROLE_PROMOTER) {
    return leaderDisplayText(row) ? `所属团长：${leaderDisplayText(row)}` : '未分配团长'
  }
  if (row.authorityId === ROLE_LEADER) {
    return `下属地推：${row.promoterCount || 0}`
  }
  return '-'
}

const canConfigureCacheSample = (row) => {
  return [ROLE_LEADER, ROLE_PROMOTER].includes(row.authorityId)
}

const canCreatePromoterToken = (row) => {
  return [ROLE_SUPER, ROLE_ADMIN].includes(currentRoleId.value) && row.authorityId === ROLE_PROMOTER
}

const cacheSampleText = (row) => {
  if (row.cacheSampleRatio !== undefined && row.cacheSampleRatio !== null) {
    return `自定义 ${row.cacheSampleRatio}%`
  }
  if (row.cacheSampleRatioInherited) {
    return `继承 ${row.effectiveCacheSampleRatio || 0}%`
  }
  return '未配置(0%)'
}

const accountRowKey = (row) => `${row.authorityId || 'role'}-${row.ID}`

const accountRowClassName = ({ row }) => {
  return row._relationChild ? 'account-relation-child' : ''
}

const clearTreeFields = (row) => {
  const rest = { ...row }
  delete rest.children
  delete rest._relationChild
  return rest
}

const textIncludes = (value, keyword) => {
  if (!keyword) return true
  return String(value || '').toLowerCase().includes(String(keyword).toLowerCase())
}

const hasSearchCondition = () => {
  return Boolean(searchInfo.value.username || searchInfo.value.nickname || searchInfo.value.phone || searchInfo.value.email)
}

const matchSearchInfo = (item) => {
  return (
    textIncludes(item.userName, searchInfo.value.username) &&
    textIncludes(item.nickName, searchInfo.value.nickname) &&
    textIncludes(item.phone, searchInfo.value.phone) &&
    textIncludes(item.email, searchInfo.value.email)
  )
}

const filterTreeBySearch = (roots) => {
  if (!hasSearchCondition()) return roots
  return roots.reduce((result, item) => {
    if (item.authorityId !== ROLE_LEADER) {
      if (matchSearchInfo(item)) result.push(item)
      return result
    }

    const children = item.children || []
    const leaderMatched = matchSearchInfo(item)
    const matchedChildren = children.filter(matchSearchInfo)
    if (!leaderMatched && matchedChildren.length === 0) {
      return result
    }

    const nextItem = {
      ...item,
      children: leaderMatched ? children : matchedChildren
    }
    if (nextItem.children.length === 0) {
      delete nextItem.children
    }
    result.push(nextItem)
    return result
  }, [])
}

const buildLeaderTree = (list) => {
  const visibleList = filterByRole(list).map((item) => ({ ...item }))
  const leaderMap = new Map()
  const roots = []
  const promoters = []

  visibleList.forEach((item) => {
    if (item.authorityId === ROLE_LEADER) {
      item.children = []
      leaderMap.set(item.ID, item)
      roots.push(item)
      return
    }
    if (item.authorityId === ROLE_PROMOTER) {
      promoters.push(item)
      return
    }
    roots.push(item)
  })

  promoters.forEach((item) => {
    const leader = item.leaderId ? leaderMap.get(item.leaderId) : null
    if (leader) {
      leader.children.push({
        ...item,
        _relationChild: true
      })
      return
    }
    roots.push(item)
  })

  roots.forEach((item) => {
    if (Array.isArray(item.children) && item.children.length === 0) {
      delete item.children
    }
  })

  return filterTreeBySearch(roots)
}

const formatDateText = (val) => {
  if (!val) return '-'
  return formatDate(val)
}

const searchInfo = ref({
  username: '',
  nickname: '',
  phone: '',
  email: ''
})

const page = ref(1)
const pageSize = ref(10)
const total = ref(0)
const tableData = ref([])
const leaderOptions = ref([])
const currentUserId = computed(() => userStore.userInfo?.ID)
const cacheSampleDialog = ref(false)
const cacheSampleForm = ref({
  row: null,
  ID: 0,
  userName: '',
  nickName: '',
  authorityId: undefined,
  configured: false,
  ratio: 0
})
const cacheSampleFormTip = computed(() => {
  if (cacheSampleForm.value.configured) {
    return '当前账号使用自定义风控比例，范围 0-80%。'
  }
  if (cacheSampleForm.value.authorityId === ROLE_PROMOTER) {
    return '地推不单独配置时，优先继承所属团长；团长未配置则按 0% 生效。'
  }
  return '团长不配置时默认按 0% 生效，下属地推可继承该配置。'
})

const filterByRole = (list) => {
  if (currentRoleId.value === ROLE_SUPER) {
    return list
  }
  if (currentRoleId.value === ROLE_ADMIN) {
    return list.filter((item) =>
      [ROLE_LEADER, ROLE_PROMOTER, ROLE_APP_EXTRACT, ROLE_APP_UPLOAD].includes(item.authorityId)
    )
  }
  if (currentRoleId.value === ROLE_LEADER) {
    return list.filter((item) => item.authorityId === ROLE_PROMOTER && item.leaderId === currentUserId.value)
  }
  return []
}

const fetchUsers = async () => {
  const query = {
    page: useLeaderTree.value ? 1 : page.value,
    pageSize: useLeaderTree.value ? 10000 : pageSize.value
  }
  if (!useLeaderTree.value) {
    Object.assign(query, searchInfo.value)
  }
  if (currentRoleId.value === ROLE_LEADER) {
    query.authorityId = ROLE_PROMOTER
    query.leaderId = currentUserId.value
  }
  const res = await getUserList(query)
  if (res.code === 0) {
    const list = (res.data.list || []).map((item) => ({
      ...item,
      phoneRegisterTaskDisabled: item.phoneRegisterTaskDisabled === true
    }))
    leaderOptions.value = list.filter((item) => item.authorityId === ROLE_LEADER)
    if (useLeaderTree.value) {
      const roots = buildLeaderTree(list)
      const start = (page.value - 1) * pageSize.value
      const pageRoots = roots.slice(start, start + pageSize.value)
      tableData.value = pageRoots
      total.value = roots.length
      return
    }
    tableData.value = filterByRole(list)
    total.value = res.data.total
  }
}

const onSubmit = async () => {
  page.value = 1
  await fetchUsers()
}

const onReset = async () => {
  searchInfo.value = { username: '', nickname: '', phone: '', email: '' }
  page.value = 1
  await fetchUsers()
}

const handleCurrentChange = async (val) => {
  page.value = val
  await fetchUsers()
}

const handleSizeChange = async (val) => {
  pageSize.value = val
  page.value = 1
  await fetchUsers()
}

const showDrawer = ref(false)
const drawerMode = ref('add')
const userFormRef = ref()
const userForm = ref({
  ID: 0,
  userName: '',
  password: '',
  nickName: '',
  phone: '',
  email: '',
  authorityId: undefined,
  leaderId: undefined,
  enable: 1,
  phoneRegisterTaskDisabled: false,
  headerImg: ''
})

const rules = {
  userName: [
    { required: true, message: '请输入用户名', trigger: 'blur' },
    { min: 5, message: '最低5位字符', trigger: 'blur' }
  ],
  password: [
    { required: true, message: '请输入密码', trigger: 'blur' },
    { min: 6, message: '最低6位字符', trigger: 'blur' }
  ],
  nickName: [{ required: true, message: '请输入昵称', trigger: 'blur' }],
  authorityId: [{ required: true, message: '请选择角色', trigger: 'change' }]
}

const openAdd = () => {
  drawerMode.value = 'add'
  userForm.value = {
    ID: 0,
    userName: '',
    password: '',
    nickName: '',
    phone: '',
    email: '',
    authorityId: roleOptions.value[0]?.value,
    leaderId: currentRoleId.value === ROLE_LEADER ? currentUserId.value : undefined,
    enable: 1,
    phoneRegisterTaskDisabled: false,
    headerImg: ''
  }
  showDrawer.value = true
}

const openEdit = (row) => {
  drawerMode.value = 'edit'
  userForm.value = clearTreeFields(JSON.parse(JSON.stringify(row)))
  userForm.value.phoneRegisterTaskDisabled = userForm.value.phoneRegisterTaskDisabled === true
  showDrawer.value = true
}

const submitDrawer = async () => {
  await userFormRef.value.validate(async (valid) => {
    if (!valid) return

    if (drawerMode.value === 'add') {
      if (
        [ROLE_SUPER, ROLE_ADMIN].includes(currentRoleId.value) &&
        userForm.value.authorityId === ROLE_PROMOTER &&
        !userForm.value.leaderId
      ) {
        ElMessage.warning('请选择所属团长')
        return
      }
      const payload = {
        ...clearTreeFields(userForm.value),
        authorityIds: [userForm.value.authorityId]
      }
      if (currentRoleId.value === ROLE_LEADER && payload.authorityId === ROLE_PROMOTER) {
        payload.leaderId = currentUserId.value
      }
      const res = await register(payload)
      if (res.code === 0) {
        ElMessage.success('创建成功')
        showDrawer.value = false
        await fetchUsers()
      }
      return
    }

    const payload = {
      ...clearTreeFields(userForm.value),
      authorityIds: [userForm.value.authorityId]
    }
    const res = await setUserInfo(payload)
    if (res.code === 0) {
      ElMessage.success('编辑成功')
      showDrawer.value = false
      await fetchUsers()
    }
  })
}

const deleteUserFunc = async (row) => {
  await ElMessageBox.confirm('确定删除该账号吗?', '提示', {
    confirmButtonText: '确定',
    cancelButtonText: '取消',
    type: 'warning'
  })
  const res = await deleteUser({ id: row.ID })
  if (res.code === 0) {
    ElMessage.success('删除成功')
    await fetchUsers()
  }
}

const switchEnable = async (row) => {
  const res = await setUserInfo({
    ...clearTreeFields(row),
    authorityIds: [row.authorityId]
  })
  if (res.code === 0) {
    ElMessage.success(`${row.enable === 1 ? '启用' : '禁用'}成功`)
    await nextTick()
    await fetchUsers()
  }
}

const switchTaskCreate = async (row) => {
  const res = await setUserInfo({
    ...clearTreeFields(row),
    authorityIds: [row.authorityId]
  })
  if (res.code === 0) {
    ElMessage.success(`${row.phoneRegisterTaskDisabled ? '禁用' : '恢复'}成功`)
    await nextTick()
    await fetchUsers()
  }
}

const openCacheSampleDialog = (row) => {
  const hasCustomRatio = row.cacheSampleRatio !== undefined && row.cacheSampleRatio !== null
  cacheSampleForm.value = {
    row,
    ID: row.ID,
    userName: row.userName,
    nickName: row.nickName,
    authorityId: row.authorityId,
    configured: hasCustomRatio,
    ratio: hasCustomRatio ? row.cacheSampleRatio : (row.effectiveCacheSampleRatio || 0)
  }
  cacheSampleDialog.value = true
}

const submitCacheSampleConfig = async () => {
  const row = cacheSampleForm.value.row
  if (!row) return
  const ratio = Number(cacheSampleForm.value.ratio || 0)
  if (cacheSampleForm.value.configured && (ratio < 0 || ratio > 80)) {
    ElMessage.warning('风控比例必须在0-80之间')
    return
  }
  const res = await setUserInfo({
    ...clearTreeFields(row),
    authorityIds: [row.authorityId],
    cacheSampleRatioConfigured: cacheSampleForm.value.configured,
    cacheSampleRatio: cacheSampleForm.value.configured ? ratio : null
  })
  if (res.code === 0) {
    ElMessage.success('风控比例已保存')
    cacheSampleDialog.value = false
    await fetchUsers()
  }
}

const tokenDialogVisible = ref(false)
const tokenResult = ref('')

const createPromoterToken = async(row) => {
  if (!canCreatePromoterToken(row)) {
    ElMessage.warning('仅支持为地推账号生成Token')
    return
  }
  const res = await createApiToken({
    userId: row.ID,
    authorityId: ROLE_PROMOTER,
    days: 90,
    remark: `phoneworker:${row.userName || row.ID}`
  })
  if (res.code === 0) {
    tokenResult.value = res.data.token
    tokenDialogVisible.value = true
    ElMessage.success('Token已生成')
  }
}

const copyToken = () => {
  navigator.clipboard.writeText(tokenResult.value).then(() => {
    ElMessage.success('Token已复制')
  }).catch(() => {
    ElMessage.error('复制失败，请手动复制')
  })
}

const resetPwdDialog = ref(false)
const resetPwdInfo = ref({
  ID: 0,
  userName: '',
  nickName: '',
  password: ''
})

const openResetPwd = (row) => {
  resetPwdInfo.value = {
    ID: row.ID,
    userName: row.userName,
    nickName: row.nickName,
    password: ''
  }
  resetPwdDialog.value = true
}

const confirmResetPassword = async () => {
  if (!resetPwdInfo.value.password) {
    ElMessage.warning('请输入新密码')
    return
  }
  const res = await resetPassword({
    ID: resetPwdInfo.value.ID,
    password: resetPwdInfo.value.password
  })
  if (res.code === 0) {
    ElMessage.success('重置成功')
    resetPwdDialog.value = false
  }
}

fetchUsers()
</script>

<style scoped>
.account-name-cell {
  display: flex;
  align-items: center;
  min-width: 0;
  gap: 6px;
}

.account-name-cell.is-child {
  padding-left: 18px;
  color: #475569;
}

.child-branch {
  position: relative;
  display: inline-block;
  flex: none;
  width: 14px;
  height: 1px;
  background: #94a3b8;
}

.child-branch::before {
  position: absolute;
  left: 0;
  bottom: 0;
  width: 1px;
  height: 14px;
  content: '';
  background: #cbd5e1;
}

.cache-sample-cell {
  display: flex;
  align-items: center;
  gap: 8px;
  white-space: nowrap;
}

.cache-sample-percent {
  margin-left: 6px;
}

.cache-sample-tip {
  line-height: 1.4;
  color: #64748b;
}

@media (max-width: 768px) {
  .account-manage-page :deep(.gva-search-box .el-form) {
    display: block;
  }

  .account-manage-page :deep(.gva-search-box .el-form-item) {
    display: flex;
    margin-right: 0;
    margin-bottom: 8px;
  }

  .account-manage-page :deep(.gva-search-box .el-form-item__label) {
    width: 64px;
    justify-content: flex-start;
    padding-right: 8px;
  }

  .account-manage-page :deep(.gva-search-box .el-form-item__content) {
    flex: 1;
    min-width: 0;
  }

  .account-manage-page :deep(.gva-search-box .el-input) {
    width: 100%;
  }

  .account-manage-page :deep(.gva-btn-list) {
    margin-bottom: 8px;
  }

  .account-table :deep(.el-table__cell) {
    padding: 6px 0;
  }

  .account-table :deep(.cell) {
    padding: 0 6px;
    line-height: 1.25;
  }

  .account-manage-page :deep(.gva-pagination) {
    overflow-x: auto;
    padding-bottom: 4px;
  }

  .account-manage-page :deep(.el-pagination) {
    min-width: max-content;
  }

  .account-manage-page :deep(.el-dialog) {
    width: 94vw !important;
    max-width: 94vw;
  }

  .account-manage-page :deep(.el-drawer) {
    width: 100vw !important;
  }

  .account-manage-page :deep(.el-drawer__header) {
    margin-bottom: 8px;
    padding: 12px;
  }

  .account-manage-page :deep(.el-drawer__body) {
    padding: 12px;
  }

  .account-manage-page :deep(.el-drawer__header .flex) {
    align-items: flex-start;
    gap: 8px;
  }
}
</style>
