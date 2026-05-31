<script setup>
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import VideoCard from '../components/VideoCard.vue'
import UserAvatar from '../components/UserAvatar.vue'
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
    { key: 'uploads', label: '我上传的' },
    { key: 'likes', label: '我点赞的' },
    { key: 'favorites', label: '我收藏的' }
  ]
})

const sectionTitle = computed(() => {
  if (!isSelf.value) return 'Ta 的视频'
  if (activeTab.value === 'likes') return '我点赞的视频'
  if (activeTab.value === 'favorites') return '我收藏的视频'
  return '我上传的视频'
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

function switchTab(key) {
  if (activeTab.value === key) return
  activeTab.value = key
  loadProfile()
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
</script>

<template>
  <div class="page-wrap">
    <section v-if="loading && !user" class="card card-body text-muted">加载中...</section>
    <section v-else-if="error && !user" class="card card-body text-error">{{ error }}</section>
    <template v-else-if="user">
      <section class="profile card card-body">
        <div class="avatar-wrap">
          <UserAvatar :user="user" :size="72" />
          <button
            v-if="isSelf"
            type="button"
            class="avatar-edit"
            :disabled="uploading"
            @click="pickAvatar"
          >
            {{ uploading ? '上传中...' : '更换头像' }}
          </button>
          <input
            ref="fileInput"
            type="file"
            accept="image/*"
            class="hidden-input"
            @change="onAvatarChange"
          />
        </div>
        <div>
          <h1>{{ user.nickname || user.username }}</h1>
          <p>@{{ user.username }} · 加入于 {{ formatDate(user.created_at) }}</p>
          <p v-if="isSelf" class="badge badge-blue">我的主页</p>
        </div>
      </section>

      <nav v-if="tabs.length" class="tabs">
        <button
          v-for="tab in tabs"
          :key="tab.key"
          type="button"
          class="tab"
          :class="{ active: activeTab === tab.key }"
          @click="switchTab(tab.key)"
        >
          {{ tab.label }}
        </button>
      </nav>

      <section class="section-head">
        <h2>{{ sectionTitle }}</h2>
        <span>{{ total }} 个</span>
      </section>

      <section v-if="loading" class="card card-body text-muted">加载中...</section>
      <section v-else-if="error" class="card card-body text-error">{{ error }}</section>
      <div v-else-if="videos.length" class="grid">
        <VideoCard v-for="item in videos" :key="item.id" :video="item" />
      </div>
      <div v-else class="card card-body text-muted">{{ emptyText }}</div>
    </template>
  </div>
</template>

<style scoped>
.profile {
  display: flex;
  align-items: center;
  gap: 16px;
  margin-bottom: 20px;
}

.avatar-wrap {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
}

.avatar-edit {
  padding: 4px 12px;
  border: 1px solid var(--color-border-strong);
  border-radius: var(--radius-full);
  background: var(--color-surface);
  color: var(--color-primary);
  font-size: 12px;
  cursor: pointer;
}

.avatar-edit:disabled {
  opacity: 0.6;
  cursor: wait;
}

.hidden-input {
  display: none;
}

h1 {
  margin: 0 0 6px;
  font-size: 22px;
}

.profile p {
  margin: 0;
  color: var(--color-text-secondary);
}

.badge-blue {
  margin-top: 8px;
}

.tabs {
  display: flex;
  gap: 8px;
  margin-bottom: 18px;
  flex-wrap: wrap;
}

.tab {
  padding: 8px 14px;
  border: 1px solid var(--color-border-strong);
  border-radius: var(--radius-full);
  background: var(--color-surface);
  color: var(--color-text-secondary);
  font-size: 14px;
  cursor: pointer;
}

.tab.active {
  background: var(--color-primary-soft);
  border-color: var(--color-primary);
  color: var(--color-primary);
  font-weight: 500;
}

.section-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}

.section-head h2 {
  margin: 0;
  font-size: 18px;
  font-weight: 600;
}

.section-head span {
  color: var(--color-text-muted);
  font-size: 14px;
}

.grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(260px, 1fr));
  gap: 16px;
}
</style>
