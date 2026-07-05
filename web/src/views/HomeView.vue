<script setup>
import { onMounted, ref, watch } from 'vue'
import { RouterLink } from 'vue-router'
import VideoCard from '../components/VideoCard.vue'
import PageSectionHead from '../components/PageSectionHead.vue'
import VideoGrid from '../components/VideoGrid.vue'
import UiState from '../components/UiState.vue'
import { fetchVideos, searchVideos } from '../api/video'
import { throttle } from '../utils/throttle'

defineOptions({ name: 'HomeView' })

const videos = ref([])
const total = ref(0)
const loading = ref(true)
const error = ref('')
const keyword = ref('')
const searching = ref(false)

async function loadVideos() {
  loading.value = true
  error.value = ''
  searching.value = false
  try {
    const res = await fetchVideos({ page: 1, page_size: 24 })
    videos.value = res.data.videos || []
    total.value = res.data.total || 0
  } catch (e) {
    error.value = e.message
  } finally {
    loading.value = false
  }
}

async function doSearch() {
  const q = keyword.value.trim()
  if (!q) {
    await loadVideos()
    return
  }
  loading.value = true
  error.value = ''
  searching.value = true
  try {
    const res = await searchVideos({ q, page: 1, page_size: 24 })
    videos.value = res.data.videos || []
    total.value = res.data.total || 0
  } catch (e) {
    error.value = e.message
    videos.value = []
    total.value = 0
  } finally {
    loading.value = false
  }
}

const debouncedSearch = throttle(() => {
  if (keyword.value.trim()) doSearch()
}, 400)

function clearSearch() {
  keyword.value = ''
  loadVideos()
}

watch(keyword, (val) => {
  if (!val.trim() && searching.value) {
    loadVideos()
  }
})

onMounted(loadVideos)
</script>

<template>
  <div class="discover-page">
    <!-- Search bar -->
    <div class="search-section">
      <div class="search-box">
        <span class="search-icon">⌕</span>
        <input
          v-model="keyword"
          type="search"
          placeholder="搜索视频标题..."
          @keyup.enter="doSearch"
          @input="debouncedSearch"
        />
        <button
          v-if="keyword"
          type="button"
          class="clear-btn"
          @click="clearSearch"
        >
          ✕
        </button>
      </div>
      <button type="button" class="search-btn" @click="doSearch">搜索</button>
    </div>

    <!-- Section header -->
    <div class="discover-header">
      <h1>发现</h1>
      <span v-if="!loading && !searching" class="video-count">{{ total }} 个视频</span>
    </div>

    <!-- Loading skeleton -->
    <section v-if="loading" class="skeleton-grid" aria-busy="true">
      <div v-for="n in 12" :key="n" class="skeleton-card">
        <div class="skeleton skeleton-cover" />
      </div>
    </section>

    <!-- Error -->
    <UiState
      v-else-if="error"
      type="empty"
      :message="error"
      retry
      @retry="searching ? doSearch() : loadVideos()"
    />

    <!-- Results -->
    <section v-else>
      <PageSectionHead
        v-if="searching"
        :title="`「${keyword}」`"
        :count="total"
        suffix="个结果"
      />
      <VideoGrid v-if="videos.length">
        <VideoCard v-for="item in videos" :key="item.id" :video="item" />
      </VideoGrid>
      <UiState
        v-else
        :show-illustration="true"
        message="暂无匹配视频，试试其他关键词"
      >
        <RouterLink to="/" class="btn btn-primary">去推荐看看</RouterLink>
        <RouterLink to="/upload" class="btn btn-ghost">上传视频</RouterLink>
      </UiState>
    </section>
  </div>
</template>

<style scoped>
.discover-page {
  max-width: var(--content-max);
  margin: 0 auto;
  padding: var(--space-4) var(--space-3);
}

/* ===== Search ===== */
.search-section {
  display: flex;
  gap: 8px;
  align-items: center;
  margin-bottom: var(--space-4);
}

.search-box {
  position: relative;
  flex: 1;
  min-width: 0;
}

.search-icon {
  position: absolute;
  left: 14px;
  top: 50%;
  transform: translateY(-50%);
  color: var(--color-text-muted);
  font-size: 16px;
  pointer-events: none;
}

.search-box input {
  width: 100%;
  margin: 0;
  padding: 12px 40px 12px 42px;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-full);
  background: var(--color-surface);
  color: var(--color-text);
  font-size: 15px;
  outline: none;
  transition: border-color 0.15s ease;
}

.search-box input:focus {
  border-color: var(--color-primary);
}

.search-box input::placeholder {
  color: var(--color-text-muted);
}

.clear-btn {
  position: absolute;
  right: 12px;
  top: 50%;
  transform: translateY(-50%);
  border: none;
  background: none;
  color: var(--color-text-muted);
  cursor: pointer;
  padding: 4px;
  font-size: 14px;
}

.search-btn {
  padding: 10px 20px;
  border: none;
  border-radius: var(--radius-full);
  background: var(--color-primary-gradient);
  color: #fff;
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
  flex-shrink: 0;
  transition: transform 0.12s ease;
}

.search-btn:active {
  transform: scale(0.96);
}

/* ===== Header ===== */
.discover-header {
  display: flex;
  align-items: baseline;
  gap: 12px;
  margin-bottom: var(--space-3);
  padding: 0 4px;
}

.discover-header h1 {
  margin: 0;
  font-size: 22px;
  font-weight: 700;
  color: var(--color-text);
}

.video-count {
  font-size: 13px;
  color: var(--color-text-muted);
}

/* ===== Skeleton ===== */
.skeleton-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(180px, 1fr));
  gap: 6px;
}

.skeleton-card {
  overflow: hidden;
  border-radius: var(--radius-md);
  background: var(--color-surface);
}

.skeleton-cover {
  aspect-ratio: 9 / 16;
}

@media (min-width: 768px) {
  .discover-page {
    padding: var(--space-5);
  }

  .skeleton-grid {
    grid-template-columns: repeat(auto-fill, minmax(220px, 1fr));
    gap: 8px;
  }
}
</style>
