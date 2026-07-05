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
    const res = await fetchVideos({ page: 1, page_size: 12 })
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
    const res = await searchVideos({ q, page: 1, page_size: 12 })
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
  <div class="page-wrap">
    <section class="page-intro">
      <h1>发现精彩视频</h1>
      <p>浏览已发布的作品，搜索你感兴趣的主题，点击进入推荐流观看完整内容。</p>
      <p v-if="!loading && !searching" class="page-intro-meta">共 {{ total }} 个已发布视频</p>
    </section>

    <section class="search-toolbar card card-body">
      <div class="search-input-wrap">
        <span class="search-icon" aria-hidden="true">⌕</span>
        <input
          v-model="keyword"
          class="input"
          type="search"
          placeholder="搜索标题或简介"
          @keyup.enter="doSearch"
          @input="debouncedSearch"
        />
        <button
          v-if="keyword"
          type="button"
          class="search-clear"
          aria-label="清除搜索"
          @click="clearSearch"
        >
          ✕
        </button>
      </div>
      <button type="button" class="btn btn-primary" @click="doSearch">搜索</button>
    </section>

    <section v-if="loading" class="skeleton-grid" aria-busy="true">
      <div v-for="n in 6" :key="n" class="skeleton-card card">
        <div class="skeleton skeleton-cover" />
        <div class="skeleton skeleton-line" />
        <div class="skeleton skeleton-line short" />
      </div>
    </section>

    <UiState
      v-else-if="error"
      type="empty"
      :message="error"
      retry
      @retry="searching ? doSearch() : loadVideos()"
    />

    <section v-else>
      <PageSectionHead
        v-if="searching"
        :title="`搜索「${keyword}」`"
        :count="total"
      />
      <VideoGrid v-if="videos.length">
        <VideoCard v-for="item in videos" :key="item.id" :video="item" />
      </VideoGrid>
      <UiState v-else :show-illustration="true" message="暂无匹配视频，试试其他关键词">
        <RouterLink to="/" class="btn btn-ghost">去推荐看看</RouterLink>
        <RouterLink to="/upload" class="btn btn-primary">上传视频</RouterLink>
      </UiState>
    </section>
  </div>
</template>
