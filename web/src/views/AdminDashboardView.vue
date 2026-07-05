<script setup>
import { onMounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import PageTabs from '../components/PageTabs.vue'
import PageSectionHead from '../components/PageSectionHead.vue'
import UiState from '../components/UiState.vue'
import { useAuthStore } from '../stores/auth'
import {
  listUsers,
  setUserRole,
  adminListVideos,
  adminDeleteVideo,
  adminRestoreVideo,
  adminPermanentDeleteVideo,
  listRecycleBin,
  listAuditLogs
} from '../api/admin'
import { formatTime } from '../utils/format'

const router = useRouter()
const auth = useAuthStore()
const tab = ref('users')
const loading = ref(false)
const error = ref('')

const users = ref([])
const usersTotal = ref(0)
const videos = ref([])
const videosTotal = ref(0)
const recycleVideos = ref([])
const recycleTotal = ref(0)
const auditLogs = ref([])
const auditTotal = ref(0)

const deleteReason = ref('')
const pendingDeleteId = ref('')

const adminTabs = [
  { key: 'users', label: '用户管理' },
  { key: 'videos', label: '视频管理' },
  { key: 'recycle', label: '回收站' },
  { key: 'audit', label: '审计日志' }
]

const roleOptions = [
  { value: 'user', label: '普通用户' },
  { value: 'reviewer', label: '审核员' },
  { value: 'admin', label: '管理员' }
]

async function loadUsers() {
  loading.value = true
  error.value = ''
  try {
    const res = await listUsers({ page: 1, page_size: 50 })
    users.value = res.data.users || []
    usersTotal.value = res.data.total || 0
  } catch (e) {
    error.value = e.message
  } finally {
    loading.value = false
  }
}

async function loadVideos() {
  loading.value = true
  error.value = ''
  try {
    const res = await adminListVideos({ page: 1, page_size: 50, include_deleted: 0 })
    videos.value = res.data.videos || []
    videosTotal.value = res.data.total || 0
  } catch (e) {
    error.value = e.message
  } finally {
    loading.value = false
  }
}

async function loadRecycle() {
  loading.value = true
  error.value = ''
  try {
    const res = await listRecycleBin({ page: 1, page_size: 50 })
    recycleVideos.value = res.data.videos || []
    recycleTotal.value = res.data.total || 0
  } catch (e) {
    error.value = e.message
  } finally {
    loading.value = false
  }
}

async function loadAudit() {
  loading.value = true
  error.value = ''
  try {
    const res = await listAuditLogs({ page: 1, page_size: 50 })
    auditLogs.value = res.data.logs || []
    auditTotal.value = res.data.total || 0
  } catch (e) {
    error.value = e.message
  } finally {
    loading.value = false
  }
}

async function changeRole(user, role) {
  if (user.role === role) return
  try {
    await setUserRole(user.id, role)
    user.role = role
  } catch (e) {
    error.value = e.message
  }
}

async function confirmDelete() {
  if (!pendingDeleteId.value) return
  try {
    await adminDeleteVideo(pendingDeleteId.value, deleteReason.value)
    pendingDeleteId.value = ''
    deleteReason.value = ''
    await loadVideos()
    await loadRecycle()
  } catch (e) {
    error.value = e.message
  }
}

async function restoreVideo(id) {
  try {
    await adminRestoreVideo(id)
    await loadRecycle()
    await loadVideos()
  } catch (e) {
    error.value = e.message
  }
}

async function permanentDelete(id) {
  if (!confirm('永久删除后前端无法恢复，合规备份仍保留。确认？')) return
  try {
    await adminPermanentDeleteVideo(id)
    await loadRecycle()
  } catch (e) {
    error.value = e.message
  }
}

function loadTabData(name) {
  if (name === 'users') loadUsers()
  if (name === 'videos') loadVideos()
  if (name === 'recycle') loadRecycle()
  if (name === 'audit') loadAudit()
}

function videoTitle(item) {
  return item.video?.title || item.title || item.video?.id || '-'
}

function videoId(item) {
  return item.video?.id || item.id
}

watch(tab, loadTabData)

onMounted(() => {
  if (!auth.isAdmin) {
    router.replace('/console/auth')
    return
  }
  loadUsers()
})
</script>

<template>
  <div class="page-wrap">
    <header class="page-top">
      <div>
        <h1>管理后台</h1>
        <p>用户管理 · 视频管理 · 回收站（30 天可恢复）· 审计日志（合规留存）</p>
      </div>
      <div class="page-top-actions">
        <RouterLink to="/console/review">审核台</RouterLink>
        <RouterLink to="/">返回前台</RouterLink>
      </div>
    </header>

    <PageTabs v-model="tab" :tabs="adminTabs" />

    <p v-if="error" class="text-error">{{ error }}</p>
    <UiState v-if="loading" type="loading" message="加载中..." />

    <section v-else-if="tab === 'users'" class="card card-body">
      <PageSectionHead title="用户列表" :count="usersTotal" />
      <table class="data-table">
        <thead>
          <tr>
            <th>ID</th>
            <th>用户名</th>
            <th>昵称</th>
            <th>角色</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="u in users" :key="u.id">
            <td>{{ u.id }}</td>
            <td>{{ u.username }}</td>
            <td>{{ u.nickname }}</td>
            <td>
              <select :value="u.role || 'user'" @change="changeRole(u, $event.target.value)">
                <option v-for="opt in roleOptions" :key="opt.value" :value="opt.value">{{ opt.label }}</option>
              </select>
            </td>
          </tr>
        </tbody>
      </table>
      <p class="panel-hint">管理员可将普通用户设为「审核员」，审核员可进入审核台下架违规视频。</p>
    </section>

    <section v-else-if="tab === 'videos'" class="card card-body">
      <PageSectionHead title="在线视频" :count="videosTotal" />
      <table class="data-table">
        <thead>
          <tr>
            <th>ID</th>
            <th>标题</th>
            <th>作者</th>
            <th>状态</th>
            <th>操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="item in videos" :key="videoId(item)">
            <td>{{ videoId(item) }}</td>
            <td>{{ videoTitle(item) }}</td>
            <td>{{ item.video?.user_id || item.user_id }}</td>
            <td>{{ item.video?.status || item.status }}</td>
            <td>
              <button type="button" class="btn btn-sm btn-danger" @click="pendingDeleteId = videoId(item)">
                删除到回收站
              </button>
            </td>
          </tr>
        </tbody>
      </table>
      <p class="panel-hint">
        删除后：前台 Feed/搜索/个人页立即不可见，点赞评论收藏弹幕一并清空；后台保留 30 天回收站 + 2 年合规备份 + 审计日志。
      </p>

      <div v-if="pendingDeleteId" class="modal-backdrop">
        <div class="modal-card">
          <h3>删除视频 {{ pendingDeleteId }}</h3>
          <textarea v-model="deleteReason" placeholder="删除原因（可选）" rows="3" />
          <div class="modal-actions">
            <button type="button" class="btn btn-ghost" @click="pendingDeleteId = ''">取消</button>
            <button type="button" class="btn btn-danger" @click="confirmDelete">确认删除</button>
          </div>
        </div>
      </div>
    </section>

    <section v-else-if="tab === 'recycle'" class="card card-body">
      <PageSectionHead title="回收站" :count="recycleTotal" />
      <table class="data-table">
        <thead>
          <tr>
            <th>ID</th>
            <th>标题</th>
            <th>删除时间</th>
            <th>过期时间</th>
            <th>原因</th>
            <th>操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="item in recycleVideos" :key="videoId(item)">
            <td>{{ videoId(item) }}</td>
            <td>{{ videoTitle(item) }}</td>
            <td>{{ formatTime(item.deleted_at) }}</td>
            <td>{{ formatTime(item.purge_at) }}</td>
            <td>{{ item.delete_reason || '-' }}</td>
            <td class="row-actions">
              <button type="button" class="btn btn-sm" @click="restoreVideo(videoId(item))">恢复</button>
              <button type="button" class="btn btn-sm btn-danger" @click="permanentDelete(videoId(item))">
                永久删除
              </button>
            </td>
          </tr>
        </tbody>
      </table>
    </section>

    <section v-else-if="tab === 'audit'" class="card card-body">
      <PageSectionHead title="操作审计" :count="auditTotal" />
      <table class="data-table">
        <thead>
          <tr>
            <th>时间</th>
            <th>操作</th>
            <th>操作人</th>
            <th>目标</th>
            <th>IP</th>
            <th>详情</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="log in auditLogs" :key="log.id">
            <td>{{ formatTime(log.created_at) }}</td>
            <td>{{ log.action }}</td>
            <td>{{ log.actor_username }} ({{ log.actor_id }})</td>
            <td>{{ log.target_type }} / {{ log.target_id }}</td>
            <td>{{ log.ip }}</td>
            <td class="detail">{{ log.detail || log.user_agent }}</td>
          </tr>
        </tbody>
      </table>
      <p class="panel-hint">审计日志按合规要求长期留存，记录操作者、时间、IP 与设备信息。</p>
    </section>
  </div>
</template>

<style scoped>
.row-actions {
  display: flex;
  gap: 8px;
}

.detail {
  max-width: 260px;
  word-break: break-all;
}
</style>
