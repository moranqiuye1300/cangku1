<script setup>
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, RouterLink } from 'vue-router'
import VideoCard from '../components/VideoCard.vue'
import UserAvatar from '../components/UserAvatar.vue'
import PageTabs from '../components/PageTabs.vue'
import VideoGrid from '../components/VideoGrid.vue'
import UiState from '../components/UiState.vue'
import {
  fetchUser,
  fetchUserVideos,
  fetchUserLikedVideos,
  fetchUserFavoriteVideos,
  uploadAvatar
} from '../api/user'
import { useAuthStore } from '../stores/auth'
import { formatDate } from '../utils/format'

const route = useRoute()
const auth = useAuthStore()

const user = ref(null)
const videos = ref([])
const total = ref(0)
const loading = ref(true)
const uploading = ref(false)
const error = ref('')
const activeTab = ref('uploads')
const fileInput = ref(null)

const isSelf = computed(() => auth.user?.id === route.params.id)

const tabs = computed(() => {
  if (!isSelf.value) return []
  return [
    { key: 'uploads', label: '作品' },
    { key: 'likes', label: '点赞' },
    { key: 'favorites', label: '收藏' }
  ]
})

const emptyText = computed(() => {
  if (!isSelf.value) return '该用户还没有发布视频'
  if (activeTab.value === 'likes') return '还没有点赞过视频'
  if (activeTab.value === 'favorites') return '还没有收藏过视频'
  return '你还没有发布视频，去投稿吧'
})

async function loadVideos() {
  const id = route.params.id
  const params = { page: 1, page_size: 24 }
  if (activeTab.value === 'likes') {
    return fetchUserLikedVideos(id, params)
  }
  if (activeTab.value === 'favorites') {
    return fetchUserFavoriteVideos(id, params)
  }
  return fetchUserVideos(id, params)
}

async function loadProfile() {
  loading.value = true
  error.value = ''
  try {
    const userRes = await fetchUser(route.params.id)
    user.value = userRes.data.user
    const videoRes = await loadVideos()
    videos.value = videoRes.data.videos || []
    total.value = videoRes.data.total || 0
  } catch (e) {
    error.value = e.message
    user.value = user.value || null
    videos.value = []
    total.value = 0
  } finally {
    loading.value = false
  }
}

async function reloadVideos() {
  loading.value = true
  error.value = ''
  try {
    const videoRes = await loadVideos()
    videos.value = videoRes.data.videos || []
    total.value = videoRes.data.total || 0
  } catch (e) {
    error.value = e.message
    videos.value = []
    total.value = 0
  } finally {
    loading.value = false
  }
}

function pickAvatar() {
  fileInput.value?.click()
}

async function onAvatarChange(e) {
  const file = e.target.files?.[0]
  e.target.value = ''
  if (!file) return
  if (!file.type.startsWith('image/')) {
    error.value = '请选择图片文件'
    return
  }
  if (file.size > 2 * 1024 * 1024) {
    error.value = '图片不能超过 2MB'
    return
  }
  uploading.value = true
  error.value = ''
  try {
    const res = await uploadAvatar(file)
    user.value = res.data.user
    if (isSelf.value) {
      auth.updateUser({ avatar: res.data.user.avatar })
    }
  } catch (err) {
    error.value = err.message
  } finally {
    uploading.value = false
  }
}

onMounted(loadProfile)

watch(() => route.params.id, () => {
  activeTab.value = 'uploads'
  loadProfile()
})

watch(activeTab, (val, oldVal) => {
  if (oldVal && val !== oldVal && user.value) {
    reloadVideos()
  }
})
</script>

<template>
  <div class="profile-page">
    <UiState v-if="loading && !user" type="loading" message="加载中..." />
    <UiState v-else-if="error && !user" type="empty" :message="error" />
    <template v-else-if="user">
      <!-- Profile header -->
      <section class="profile-header">
        <div class="profile-cover" />
        <div class="profile-info">
          <div class="avatar-section">
            <UserAvatar :user="user" :size="80" />
            <button
              v-if="isSelf"
              type="button"
              class="avatar-edit-btn"
              :disabled="uploading"
              @click="pickAvatar"
            >
              {{ uploading ? '...' : '📷' }}
            </button>
            <input
              ref="fileInput"
              type="file"
              accept="image/*"
              class="hidden-input"
              @change="onAvatarChange"
            />
          </div>
          <h1 class="profile-name">{{ user.nickname || user.username }}</h1>
          <p class="profile-username">@{{ user.username }} · {{ formatDate(user.created_at) }}</p>
          <div class="profile-stats">
            <div class="stat">
              <strong>{{ total }}</strong>
              <span>作品</span>
            </div>
          </div>
        </div>
      </section>

      <!-- Tabs -->
      <div class="profile-tabs">
        <PageTabs v-if="tabs.length" v-model="activeTab" :tabs="tabs" />
      </div>

      <!-- Content -->
      <div class="profile-content">
        <UiState v-if="loading" type="loading" message="加载中..." />
        <UiState v-else-if="error" type="empty" :message="error" retry @retry="reloadVideos" />
        <UiState
          v-else-if="!videos.length && isSelf && activeTab === 'uploads'"
          message="你还没有发布视频，上传后将进入审核流程"
        >
          <RouterLink to="/upload" class="btn btn-primary">去投稿</RouterLink>
        </UiState>
        <VideoGrid v-else-if="!videos.length" :empty="true" :empty-message="emptyText" />
        <VideoGrid v-else>
          <VideoCard
            v-for="item in videos"
            :key="item.id"
            :video="item"
            :show-status="isSelf && activeTab === 'uploads'"
          />
        </VideoGrid>
      </div>
    </template>
  </div>
</template>

<style scoped>
.profile-page {
  max-width: var(--content-max);
  margin: 0 auto;
  padding: 0;
}

.profile-header {
  position: relative;
  text-align: center;
  padding-bottom: 20px;
}

.profile-cover {
  height: 160px;
  background: linear-gradient(135deg, #1a0a0f, #0a0f1a, #0f0f0f);
  border-bottom: 1px solid var(--color-border);
}

.profile-info {
  position: relative;
  margin-top: -40px;
  padding: 0 16px;
}

.avatar-section {
  position: relative;
  display: inline-block;
}

.avatar-edit-btn {
  position: absolute;
  bottom: 0;
  right: -4px;
  width: 32px;
  height: 32px;
  border-radius: 50%;
  border: 2px solid var(--color-surface);
  background: var(--color-primary);
  color: #fff;
  font-size: 14px;
  cursor: pointer;
  display: grid;
  place-items: center;
  transition: transform 0.12s ease;
}

.avatar-edit-btn:active {
  transform: scale(0.9);
}

.avatar-edit-btn:disabled {
  opacity: 0.6;
}

.hidden-input {
  display: none;
}

.profile-name {
  margin: 12px 0 4px;
  font-size: 22px;
  font-weight: 700;
  color: var(--color-text);
}

.profile-username {
  margin: 0 0 16px;
  color: var(--color-text-muted);
  font-size: 14px;
}

.profile-stats {
  display: flex;
  justify-content: center;
  gap: 32px;
}

.stat {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 2px;
}

.stat strong {
  font-size: 20px;
  font-weight: 700;
  color: var(--color-text);
}

.stat span {
  font-size: 12px;
  color: var(--color-text-muted);
}

.profile-tabs {
  padding: 0 16px;
  border-bottom: 1px solid var(--color-border);
  margin-bottom: 8px;
}

.profile-tabs :deep(.tab-group) {
  margin-bottom: 0;
}

.profile-content {
  padding: 0 8px 16px;
}

@media (min-width: 768px) {
  .profile-page {
    padding: 0 24px;
  }

  .profile-cover {
    height: 200px;
    border-radius: 0 0 var(--radius-lg) var(--radius-lg);
  }

  .profile-content {
    padding: 0 0 24px;
  }

  .profile-tabs {
    padding: 0;
  }
}
</style>
