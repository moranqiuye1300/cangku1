<script setup>
import { onMounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import PageSectionHead from '../components/PageSectionHead.vue'
import PageTabs from '../components/PageTabs.vue'
import UiState from '../components/UiState.vue'
import { useAuthStore } from '../stores/auth'
import {
  reviewerListPending,
  approveSource,
  rejectSource,
  approvePublish,
  rejectPublish
} from '../api/admin'
import { videoStatusLabel } from '../utils/videoStatus'

const router = useRouter()
const auth = useAuthStore()
const activeTab = ref('source')
const tabs = [
  { key: 'source', label: '源片快审' },
  { key: 'final', label: '精审发布' }
]
const videos = ref([])
const total = ref(0)
const loading = ref(false)
const error = ref('')
const actionId = ref('')
const actionType = ref('')
const rejectReason = ref('')

function videoTitle(item) {
  return item.video?.title || item.title || '-'
}

function videoId(item) {
  return item.video?.id || item.id
}

function sourcePath(item) {
  return item.source_path || item.video?.source_path || ''
}

function coverUrl(item) {
  return item.video?.cover_url || item.cover_url || ''
}

async function previewSource(item) {
  const rel = sourcePath(item)
  if (!rel) return
  try {
    const url = `/api/v1/media/uploads/${rel.replace(/^uploads\//, '')}`
    const res = await fetch(url, {
      headers: { Authorization: `Bearer ${auth.token}` }
    })
    if (!res.ok) throw new Error('无法预览源片')
    const blob = await res.blob()
    window.open(URL.createObjectURL(blob), '_blank', 'noopener')
  } catch (e) {
    error.value = e.message || '预览失败'
  }
}

async function loadVideos() {
  loading.value = true
  error.value = ''
  try {
    const res = await reviewerListPending(activeTab.value, { page: 1, page_size: 50 })
    videos.value = res.data.videos || []
    total.value = res.data.total || 0
  } catch (e) {
    error.value = e.message
  } finally {
    loading.value = false
  }
}

function openReject(id, type) {
  actionId.value = id
  actionType.value = type
  rejectReason.value = ''
}

function closeModal() {
  actionId.value = ''
  actionType.value = ''
  rejectReason.value = ''
}

async function submitApprove(id) {
  error.value = ''
  try {
    if (activeTab.value === 'source') {
      await approveSource(id)
    } else {
      await approvePublish(id)
    }
    await loadVideos()
  } catch (e) {
    error.value = e.message
  }
}

async function submitReject() {
  if (!actionId.value || !rejectReason.value.trim()) {
    error.value = '请填写驳回原因'
    return
  }
  error.value = ''
  try {
    if (actionType.value === 'source') {
      await rejectSource(actionId.value, rejectReason.value.trim())
    } else {
      await rejectPublish(actionId.value, rejectReason.value.trim())
    }
    closeModal()
    await loadVideos()
  } catch (e) {
    error.value = e.message
  }
}

watch(activeTab, () => {
  loadVideos()
})

onMounted(() => {
  if (!auth.isReviewer) {
    router.replace('/console/auth')
    return
  }
  loadVideos()
})
</script>

<template>
  <div class="review-page">
    <div class="review-header">
      <div class="review-header-text">
        <h1>审核员工作台</h1>
        <p>源片快审通过后进入转码；精审通过后发布到推荐/发现/搜索。驳回后前台不可见并进入回收站。</p>
      </div>
      <div class="review-header-links">
        <RouterLink v-if="auth.isAdmin" to="/console" class="btn btn-ghost btn-sm">管理后台</RouterLink>
        <RouterLink to="/" class="btn btn-ghost btn-sm">返回前台</RouterLink>
      </div>
    </div>

    <div class="review-body">
      <PageTabs v-model="activeTab" :tabs="tabs" />

      <p v-if="error" class="text-error review-error">{{ error }}</p>
      <UiState v-if="loading" type="loading" message="加载中..." />

      <section v-else>
        <PageSectionHead
          :title="activeTab === 'source' ? '待源审视频' : '待精审视频'"
          :count="total"
        />
        <div class="table-wrap">
          <table class="data-table">
            <thead>
              <tr>
                <th>ID</th>
                <th>标题</th>
                <th>作者</th>
                <th>状态</th>
                <th>预览</th>
                <th>操作</th>
              </tr>
            </thead>
            <tbody>
              <tr v-if="!videos.length">
                <td colspan="6" class="empty-cell">暂无待审视频</td>
              </tr>
              <tr v-for="item in videos" :key="videoId(item)">
                <td>{{ videoId(item) }}</td>
                <td>{{ videoTitle(item) }}</td>
                <td>{{ item.video?.user_id || item.user_id }}</td>
                <td>{{ videoStatusLabel(item.video?.status || item.status) }}</td>
                <td>
                  <button
                    v-if="activeTab === 'source' && sourcePath(item)"
                    type="button"
                    class="btn btn-sm btn-ghost preview-btn"
                    @click="previewSource(item)"
                  >
                    预览原片
                  </button>
                  <template v-else-if="activeTab === 'final'">
                    <img
                      v-if="coverUrl(item)"
                      :src="coverUrl(item)"
                      alt=""
                      class="cover-thumb"
                    />
                    <RouterLink :to="{ path: '/', query: { v: videoId(item) } }" target="_blank" class="hls-link">
                      预览 HLS
                    </RouterLink>
                  </template>
                  <span v-else class="text-muted">-</span>
                </td>
                <td class="action-cell">
                  <button
                    type="button"
                    class="btn btn-sm btn-primary"
                    @click="submitApprove(videoId(item))"
                  >
                    {{ activeTab === 'source' ? '通过' : '发布' }}
                  </button>
                  <button
                    type="button"
                    class="btn btn-sm btn-danger"
                    @click="openReject(videoId(item), activeTab)"
                  >
                    驳回
                  </button>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>
    </div>

    <div v-if="actionId" class="modal-backdrop">
      <div class="modal-card">
        <h3>驳回视频 #{{ actionId }}</h3>
        <textarea v-model="rejectReason" placeholder="驳回原因（必填）" rows="4" />
        <div class="modal-actions">
          <button type="button" class="btn btn-ghost" @click="closeModal">取消</button>
          <button type="button" class="btn btn-danger" @click="submitReject">确认驳回</button>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.review-page {
  max-width: var(--content-max);
  margin: 0 auto;
  padding: var(--space-5);
}

.review-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: var(--space-4);
  margin-bottom: var(--space-5);
}

.review-header-text h1 {
  margin: 0 0 6px;
  font-size: 22px;
  font-weight: 700;
  color: var(--color-text);
}

.review-header-text p {
  margin: 0;
  color: var(--color-text-secondary);
  font-size: var(--text-base);
  max-width: 52ch;
}

.review-header-links {
  display: flex;
  gap: var(--space-3);
  flex-shrink: 0;
}

.review-body {
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  padding: var(--space-4);
  box-shadow: var(--shadow-card);
}

.review-error {
  margin: var(--space-3) 0;
}

.table-wrap {
  overflow-x: auto;
  -webkit-overflow-scrolling: touch;
}

.data-table th {
  white-space: nowrap;
}

.cover-thumb {
  width: 72px;
  height: 40px;
  object-fit: cover;
  border-radius: var(--radius-sm);
  vertical-align: middle;
  margin-right: 8px;
}

.hls-link {
  font-size: 13px;
}

.action-cell {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.empty-cell {
  text-align: center;
  color: var(--color-text-muted);
  padding: 24px;
  font-size: 14px;
}

@media (max-width: 768px) {
  .review-page {
    padding: var(--space-3);
  }

  .review-header {
    flex-direction: column;
    gap: var(--space-3);
  }

  .review-body {
    padding: var(--space-3);
  }
}
</style>
