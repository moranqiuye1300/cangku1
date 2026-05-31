<script setup>
import { ref, computed, provide, onMounted, onBeforeUnmount, watch, nextTick } from 'vue'
import { useRoute, useRouter, RouterLink } from 'vue-router'
import FeedSlide from '../components/FeedSlide.vue'
import { fetchFeed, fetchVideo } from '../api/video'
import { useAuthStore } from '../stores/auth'
import { isSoundUnlocked, markSoundUnlocked } from '../utils/soundUnlock'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()

const feedEl = ref(null)
const videos = ref([])
const currentIndex = ref(0)
const page = ref(1)
const hasMore = ref(true)
const loading = ref(false)
const initialLoading = ref(true)
const error = ref('')
const pageSize = 10

const personalized = ref(false)
const soundUnlocked = ref(isSoundUnlocked())

provide('feedSoundUnlock', { soundUnlocked })

const startVideoId = computed(() => String(route.query.v || ''))

function unlockFeedSound() {
  if (soundUnlocked.value) return
  soundUnlocked.value = true
  markSoundUnlocked()
}

function onFeedGesture() {
  unlockFeedSound()
}

function shouldRenderPlayer(index) {
  return Math.abs(index - currentIndex.value) <= 1
}

async function loadPage() {
  if (loading.value || !hasMore.value) return
  loading.value = true
  error.value = ''
  try {
    const res = await fetchFeed({ page: page.value, page_size: pageSize })
    const list = res.data.videos || []
    personalized.value = Boolean(res.data.personalized)
    if (list.length === 0) {
      hasMore.value = false
      return
    }
    const existing = new Set(videos.value.map((v) => v.id))
    for (const v of list) {
      if (!existing.has(v.id)) {
        videos.value.push(v)
      }
    }
    hasMore.value = list.length >= pageSize
    page.value += 1
  } catch (e) {
    error.value = e.message
  } finally {
    loading.value = false
    initialLoading.value = false
  }
}

function retryLoad() {
  page.value = 1
  hasMore.value = true
  videos.value = []
  initialLoading.value = true
  error.value = ''
  loadPage()
}

async function ensureStartVideo() {
  const id = startVideoId.value
  if (!id) return
  let idx = videos.value.findIndex((v) => v.id === id)
  if (idx >= 0) {
    currentIndex.value = idx
    return
  }
  try {
    const res = await fetchVideo(id)
    const v = res.data.video
    if (v) {
      videos.value.unshift(v)
      currentIndex.value = 0
    }
  } catch {
    // ignore invalid id
  }
}

async function scrollToIndex(index, behavior = 'auto') {
  await nextTick()
  const el = feedEl.value?.querySelector(`[data-index="${index}"]`)
  el?.scrollIntoView({ behavior, block: 'start' })
}

let slideObserver = null

function observeSlides() {
  slideObserver?.disconnect()
  if (!feedEl.value) return
  slideObserver = new IntersectionObserver(
    (entries) => {
      for (const entry of entries) {
        if (entry.isIntersecting && entry.intersectionRatio >= 0.55) {
          const idx = Number(entry.target.dataset.index)
          if (!Number.isNaN(idx) && idx >= 0 && idx < videos.value.length) {
            currentIndex.value = idx
          }
        }
      }
    },
    { root: feedEl.value, threshold: [0.55, 0.85] }
  )
  feedEl.value.querySelectorAll('.feed-slide').forEach((el) => slideObserver.observe(el))
}

function onKeydown(e) {
  if (e.key !== 'ArrowDown' && e.key !== 'ArrowUp') return
  e.preventDefault()
  const next =
    e.key === 'ArrowDown'
      ? Math.min(currentIndex.value + 1, videos.value.length - 1)
      : Math.max(currentIndex.value - 1, 0)
  if (next !== currentIndex.value) {
    currentIndex.value = next
    scrollToIndex(next, 'smooth')
  }
}

watch(currentIndex, (idx) => {
  if (idx >= videos.value.length - 3) {
    loadPage()
  }
  const v = videos.value[idx]
  if (v?.id && route.query.v !== v.id) {
    router.replace({ path: '/', query: { v: v.id } })
  }
})

watch(
  () => videos.value.length,
  async () => {
    await nextTick()
    observeSlides()
  }
)

watch(
  () => route.query.v,
  async (id) => {
    if (!id || videos.value.length === 0) return
    const idx = videos.value.findIndex((v) => v.id === id)
    if (idx >= 0 && idx !== currentIndex.value) {
      currentIndex.value = idx
      await scrollToIndex(idx, 'smooth')
    }
  }
)

onMounted(async () => {
  await loadPage()
  await ensureStartVideo()
  await scrollToIndex(currentIndex.value)
  observeSlides()
  window.addEventListener('keydown', onKeydown)
  window.addEventListener('pointerdown', onFeedGesture, { once: true, capture: true })
  window.addEventListener('wheel', onFeedGesture, { once: true, passive: true })
})

onBeforeUnmount(() => {
  slideObserver?.disconnect()
  window.removeEventListener('keydown', onKeydown)
  window.removeEventListener('pointerdown', onFeedGesture, { capture: true })
  window.removeEventListener('wheel', onFeedGesture)
})
</script>

<template>
  <div class="feed-page">
    <header class="feed-overlay">
      <RouterLink to="/discover" class="overlay-link">发现</RouterLink>
      <span class="overlay-title">{{ personalized ? '为你推荐' : '推荐' }}</span>
      <RouterLink v-if="auth.isLoggedIn" to="/upload" class="overlay-link">投稿</RouterLink>
      <RouterLink v-else to="/login" class="overlay-link">登录</RouterLink>
    </header>

    <section v-if="initialLoading" class="empty">
      <p>加载推荐中...</p>
    </section>

    <section v-else-if="error && !videos.length" class="empty">
      <p>{{ error }}</p>
      <button type="button" class="retry-btn" @click="retryLoad">重试</button>
    </section>

    <section v-else-if="!loading && !videos.length" class="empty">
      <p>暂无视频，先去投稿吧</p>
      <RouterLink to="/upload" class="retry-btn">上传视频</RouterLink>
    </section>

    <div v-else ref="feedEl" class="feed-scroller" @click="unlockFeedSound">
      <article
        v-for="(video, index) in videos"
        :key="video.id"
        class="feed-slide"
        :data-index="index"
      >
        <FeedSlide
          :video="video"
          :active="index === currentIndex"
          :render-player="shouldRenderPlayer(index)"
        />
      </article>
    </div>

    <div v-if="loading && videos.length" class="loading-hint">加载更多...</div>
  </div>
</template>

<style scoped>
.feed-page {
  position: fixed;
  inset: 0;
  background: #000;
  color: #fff;
  overflow: hidden;
}

.feed-overlay {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  z-index: 30;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 14px 20px;
  padding-top: max(14px, env(safe-area-inset-top));
  background: rgba(0, 0, 0, 0.35);
  pointer-events: none;
}

.overlay-link {
  pointer-events: auto;
  color: #fff;
  text-decoration: none;
  font-size: 14px;
  font-weight: 500;
  padding: 6px 12px;
  border-radius: 8px;
  background: rgba(255, 255, 255, 0.12);
}

.overlay-link:hover {
  background: rgba(255, 255, 255, 0.2);
}

.overlay-title {
  font-size: 16px;
  font-weight: 600;
  letter-spacing: 0.04em;
}

.feed-scroller {
  height: 100%;
  height: 100dvh;
  overflow-y: auto;
  scroll-snap-type: y mandatory;
  scroll-behavior: smooth;
  overscroll-behavior-y: contain;
}

.feed-scroller::-webkit-scrollbar {
  display: none;
}

.feed-slide {
  height: 100%;
  height: 100dvh;
  scroll-snap-align: start;
  scroll-snap-stop: always;
}

.loading-hint {
  position: absolute;
  bottom: 20px;
  left: 50%;
  transform: translateX(-50%);
  z-index: 25;
  padding: 6px 14px;
  border-radius: 999px;
  background: rgba(0, 0, 0, 0.55);
  font-size: 12px;
  color: rgba(255, 255, 255, 0.7);
}

.empty {
  height: 100dvh;
  display: grid;
  place-content: center;
  gap: 14px;
  text-align: center;
  padding: 24px;
  color: rgba(255, 255, 255, 0.85);
}

.retry-btn {
  justify-self: center;
  padding: 10px 20px;
  border-radius: 8px;
  background: #2563eb;
  color: #fff;
  text-decoration: none;
  border: none;
  cursor: pointer;
  font-size: 14px;
}
</style>
