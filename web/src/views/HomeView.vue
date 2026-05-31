<script setup>
import { onMounted, ref } from 'vue'
import VideoCard from '../components/VideoCard.vue'
import { fetchVideos, searchVideos } from '../api/video'
import { fetchHealth } from '../api/health'

const videos = ref([])
const total = ref(0)
const loading = ref(true)
const error = ref('')
const backendOk = ref(false)
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

async function checkBackend() {
  try {
    await fetchHealth()
    backendOk.value = true
  } catch {
    backendOk.value = false
  }
}

onMounted(async () => {
  await Promise.all([checkBackend(), loadVideos()])
})
</script>

<template>
  <div class="page-wrap">
    <section class="hero card card-body">
      <div>
        <span class="badge badge-blue">发现</span>
        <h1 class="page-title">搜索与浏览</h1>
        <p class="page-desc">网格浏览视频，支持关键词搜索，点击进入推荐流观看</p>
      </div>
      <span class="badge" :class="backendOk ? 'badge-green' : 'badge-warn'">
        后端 {{ backendOk ? '已连接' : '未连接' }}
      </span>
    </section>

    <section class="search-bar card card-body">
      <input
        v-model="keyword"
        class="input"
        type="search"
        placeholder="搜索标题或简介，如：Go、Gin、gRPC"
        @keyup.enter="doSearch"
      />
      <button type="button" class="btn btn-primary" @click="doSearch">搜索</button>
      <button type="button" class="btn btn-ghost" @click="keyword = ''; loadVideos()">重置</button>
    </section>

    <section v-if="loading" class="hint card card-body text-muted">加载视频中...</section>
    <section v-else-if="error" class="hint card card-body">
      <p class="text-error">{{ error }}</p>
      <button type="button" class="btn btn-primary" @click="searching ? doSearch() : loadVideos()">重试</button>
    </section>
    <section v-else>
      <div class="toolbar">
        <span v-if="searching">搜索「{{ keyword }}」共 {{ total }} 个结果</span>
        <span v-else>共 {{ total }} 个视频</span>
      </div>
      <div v-if="videos.length" class="grid">
        <VideoCard v-for="item in videos" :key="item.id" :video="item" />
      </div>
      <div v-else class="hint card card-body text-muted">暂无视频</div>
    </section>
  </div>
</template>

<style scoped>
.hero {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 16px;
  margin-bottom: 16px;
}

.hero .badge-blue {
  margin-bottom: 10px;
}

.search-bar {
  display: flex;
  gap: 10px;
  margin-bottom: 20px;
  align-items: center;
}

.search-bar .input {
  flex: 1;
  margin: 0;
}

.toolbar {
  margin-bottom: 16px;
  color: var(--color-text-secondary);
  font-size: 14px;
}

.grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(260px, 1fr));
  gap: 16px;
}

.hint {
  text-align: center;
}

.hint .btn {
  margin-top: 12px;
}
</style>
