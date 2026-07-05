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
    { key: 'uploads', label: '我上传的' },
    { key: 'likes', label: '我点赞的' },
    { key: 'favorites', label: '我收藏的' }
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
  <div class="page-wrap">
    <UiState v-if="loading && !user" type="loading" message="加载中..." />
    <UiState v-else-if="error && !user" type="empty" :message="error" />
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
        <div class="profile-meta">
          <h1>{{ user.nickname || user.username }}</h1>
          <p>@{{ user.username }} · 加入于 {{ formatDate(user.created_at) }}</p>
          <div class="stats-row">
            <div class="stat-item">
              <strong>{{ total }}</strong>
              <span>{{ isSelf && activeTab === 'uploads' ? '作品' : '视频' }}</span>
            </div>
          </div>
        </div>
      </section>

      <PageTabs v-if="tabs.length" v-model="activeTab" :tabs="tabs" />

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

.profile-meta {
  flex: 1;
  min-width: 0;
}

.profile-meta h1 {
  margin: 0 0 6px;
  font-size: 22px;
}

.profile-meta p {
  margin: 0;
  color: var(--color-text-secondary);
}

</style>
