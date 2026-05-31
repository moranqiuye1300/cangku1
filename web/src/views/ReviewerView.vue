<script setup>
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '../stores/auth'
import { reviewerListVideos, reviewerRejectVideo } from '../api/admin'

const router = useRouter()
const auth = useAuthStore()
const videos = ref([])
const total = ref(0)
const loading = ref(false)
const error = ref('')
const rejectId = ref('')
const rejectReason = ref('')

function videoTitle(item) {
  return item.video?.title || item.title || '-'
}

function videoId(item) {
  return item.video?.id || item.id
}

async function loadVideos() {
  loading.value = true
  error.value = ''
  try {
    const res = await reviewerListVideos({ page: 1, page_size: 50 })
    videos.value = res.data.videos || []
    total.value = res.data.total || 0
  } catch (e) {
    error.value = e.message
  } finally {
    loading.value = false
  }
}

async function submitReject() {
  if (!rejectId.value || !rejectReason.value.trim()) {
    error.value = '请填写下架原因'
    return
  }
  try {
    await reviewerRejectVideo(rejectId.value, rejectReason.value.trim())
    rejectId.value = ''
    rejectReason.value = ''
    await loadVideos()
  } catch (e) {
    error.value = e.message
  }
}

onMounted(() => {
  if (!auth.isReviewer) {
    router.replace('/console/auth')
    return
  }
  loadVideos()
})
</script>

<template>
  <div class="page page-wrap">
    <header class="top">
      <div>
        <h1>审核员工作台</h1>
        <p>审核并下架违规视频。下架后前台立即不可见，互动数据清空，视频进入回收站。</p>
      </div>
      <div class="actions">
        <RouterLink v-if="auth.isAdmin" to="/console">管理后台</RouterLink>
        <RouterLink to="/">返回前台</RouterLink>
      </div>
    </header>

    <p v-if="error" class="error">{{ error }}</p>
    <p v-if="loading" class="hint">加载中...</p>

    <section class="panel card card-body">
      <h2>待审核视频（{{ total }}）</h2>
      <table>
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
              <button class="danger" @click="rejectId = videoId(item)">违规下架</button>
            </td>
          </tr>
        </tbody>
      </table>
    </section>

    <div v-if="rejectId" class="modal">
      <div class="modal-body">
        <h3>下架视频 {{ rejectId }}</h3>
        <textarea v-model="rejectReason" placeholder="违规原因（必填）" rows="4" />
        <div class="modal-actions">
          <button @click="rejectId = ''">取消</button>
          <button class="danger" @click="submitReject">确认下架</button>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.page {
  max-width: 1000px;
}

.top {
  display: flex;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 20px;
}

.top h1 {
  margin: 0 0 6px;
  font-size: 22px;
  font-weight: 600;
}

.top p {
  margin: 0;
  color: var(--color-text-secondary);
  font-size: 14px;
}

.actions {
  display: flex;
  gap: 12px;
}

.actions a {
  color: var(--color-primary);
  text-decoration: none;
  font-size: 14px;
}

.panel h2 {
  margin: 0 0 16px;
  font-size: 18px;
}

table {
  width: 100%;
  border-collapse: collapse;
  font-size: 14px;
}

th,
td {
  padding: 10px 8px;
  border-bottom: 1px solid var(--color-border);
  text-align: left;
}

th {
  color: var(--color-text-secondary);
  font-weight: 500;
}

button {
  border: 1px solid var(--color-border-strong);
  background: var(--color-surface);
  color: var(--color-text-secondary);
  padding: 6px 10px;
  border-radius: var(--radius-sm);
  cursor: pointer;
  font: inherit;
}

button.danger {
  background: var(--color-danger);
  border-color: var(--color-danger);
  color: #fff;
}

.error {
  color: var(--color-danger);
}

.hint {
  color: var(--color-text-muted);
}

.modal {
  position: fixed;
  inset: 0;
  background: rgba(26, 35, 50, 0.35);
  display: grid;
  place-items: center;
  z-index: 20;
}

.modal-body {
  width: min(420px, 92vw);
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  padding: 20px;
}

.modal-body textarea {
  width: 100%;
  margin: 12px 0;
  padding: 10px;
  border-radius: var(--radius-sm);
  border: 1px solid var(--color-border-strong);
  background: var(--color-surface);
  color: var(--color-text);
  font: inherit;
}

.modal-actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}
</style>
