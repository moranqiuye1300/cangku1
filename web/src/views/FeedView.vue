<script setup>
import { ref, computed, provide, onMounted, onBeforeUnmount, watch, nextTick } from 'vue'
import { useRoute, useRouter, RouterLink } from 'vue-router'
import FeedSlide from '../components/FeedSlide.vue'
import UiState from '../components/UiState.vue'
import { fetchFeed, fetchVideo, askAI } from '../api/video'
import { isSoundUnlocked, markSoundUnlocked } from '../utils/soundUnlock'

defineOptions({ name: 'FeedView' })

const route = useRoute()
const router = useRouter()

const feedEl = ref(null)
const slideRefs = ref({})
const videos = ref([])
const currentIndex = ref(0)
const page = ref(1)
const hasMore = ref(true)
const loading = ref(false)
const initialLoading = ref(true)
const error = ref('')
const pageSize = 10

const personalized = ref(false)
const showPersonalizedTip = ref(false)
const soundUnlocked = ref(isSoundUnlocked())

// AI Chat for recommended Feed videos
const showAIChat = ref(false)
const aiMessages = ref([])
const aiInput = ref('')
const aiLoading = ref(false)

// Draggable AI button position
const aiBtnPos = ref({ bottom: 80, right: 20 })
let aiDragging = false
let aiDragStart = { x: 0, y: 0, bottom: 0, right: 0 }

function onAiBtnPointerDown(e) {
  aiDragging = true
  const touch = e.touches?.[0] || e
  aiDragStart = {
    x: touch.clientX,
    y: touch.clientY,
    bottom: aiBtnPos.value.bottom,
    right: aiBtnPos.value.right
  }
  document.addEventListener('pointermove', onAiBtnPointerMove)
  document.addEventListener('pointerup', onAiBtnPointerUp)
  document.addEventListener('touchmove', onAiBtnTouchMove, { passive: false })
  document.addEventListener('touchend', onAiBtnTouchEnd)
}

function onAiBtnPointerMove(e) {
  if (!aiDragging) return
  const dx = (e.clientX || 0) - aiDragStart.x
  const dy = (e.clientY || 0) - aiDragStart.y
  aiBtnPos.value = {
    bottom: Math.max(8, aiDragStart.bottom - dy),
    right: Math.max(8, aiDragStart.right - dx)
  }
}

function onAiBtnPointerUp() {
  if (!aiDragging) return
  aiDragging = false
  document.removeEventListener('pointermove', onAiBtnPointerMove)
  document.removeEventListener('pointerup', onAiBtnPointerUp)
  document.removeEventListener('touchmove', onAiBtnTouchMove)
  document.removeEventListener('touchend', onAiBtnTouchEnd)
}

function onAiBtnTouchMove(e) {
  if (!aiDragging) return
  e.preventDefault()
  const touch = e.touches[0]
  const dx = touch.clientX - aiDragStart.x
  const dy = touch.clientY - aiDragStart.y
  aiBtnPos.value = {
    bottom: Math.max(8, aiDragStart.bottom - dy),
    right: Math.max(8, aiDragStart.right - dx)
  }
}

function onAiBtnTouchEnd() {
  onAiBtnPointerUp()
}

provide('feedSoundUnlock', { soundUnlocked })

const startVideoId = computed(() => String(route.query.v || ''))
const activeSlide = computed(() => slideRefs.value[currentIndex.value])

function setSlideRef(index, el) {
  if (el) slideRefs.value[index] = el
  else delete slideRefs.value[index]
}

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
    if (personalized.value && page.value === 1) {
      showPersonalizedTip.value = true
      setTimeout(() => {
        showPersonalizedTip.value = false
      }, 4000)
    }
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

// AI Chat helpers (RAG over current recommended videos)
function getCurrentFeedVideoIDs() {
  return videos.value.map(v => v.id)
}

async function sendAIMessage() {
  const q = aiInput.value.trim()
  if (!q || aiLoading.value) return

  aiMessages.value.push({ role: 'user', content: q })
  aiInput.value = ''
  aiLoading.value = true

  try {
    const ids = getCurrentFeedVideoIDs()
    const res = await askAI(q, ids)
    const answer = res.data.answer || res.data.raw || 'AI 暂时无法回答'
    aiMessages.value.push({ role: 'assistant', content: answer })
  } catch (e) {
    aiMessages.value.push({ role: 'assistant', content: 'AI 服务暂时不可用：' + e.message })
  } finally {
    aiLoading.value = false
  }
}

function openAIChat() {
  showAIChat.value = true
  if (aiMessages.value.length === 0) {
    aiMessages.value.push({
      role: 'assistant',
      content: '你好！我是推荐视频 AI，可以针对你当前看到的个性化推荐视频回答问题。'
    })
  }
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
  if (e.target.matches('input, textarea, select')) return
  if (e.key === 'ArrowDown' || e.key === 'ArrowUp') {
    e.preventDefault()
    const next =
      e.key === 'ArrowDown'
        ? Math.min(currentIndex.value + 1, videos.value.length - 1)
        : Math.max(currentIndex.value - 1, 0)
    if (next !== currentIndex.value) {
      currentIndex.value = next
      scrollToIndex(next, 'smooth')
    }
    return
  }

  if (e.key === ' ') {
    e.preventDefault()
    activeSlide.value?.togglePlay?.()
    return
  }

  if (e.key === 'm' || e.key === 'M') {
    activeSlide.value?.toggleMute?.()
    return
  }

  if (e.key === 'f' || e.key === 'F') {
    activeSlide.value?.toggleFullscreen?.()
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
    <section v-if="initialLoading" class="feed-skeleton">
      <div v-for="n in 2" :key="n" class="skeleton-slide skeleton" />
    </section>

    <UiState
      v-else-if="error && !videos.length"
      type="empty"
      :message="error"
      retry
      @retry="retryLoad"
    />

    <UiState v-else-if="!loading && !videos.length" type="empty" message="暂无视频，先去投稿吧">
      <RouterLink to="/upload" class="btn btn-primary">上传视频</RouterLink>
    </UiState>

    <template v-else>
      <p v-if="showPersonalizedTip" class="feed-personalized-tip">已为你个性化推荐</p>

      <div ref="feedEl" class="feed-scroller" @click="unlockFeedSound">
        <article
          v-for="(video, index) in videos"
          :key="video.id"
          class="feed-slide"
          :data-index="index"
        >
          <FeedSlide
            :ref="(el) => setSlideRef(index, el)"
            :video="video"
            :active="index === currentIndex"
            :render-player="shouldRenderPlayer(index)"
          />
        </article>
      </div>

      <div v-if="loading && videos.length" class="loading-hint">
        <span class="loading-dot" />正在加载...
      </div>
    </template>

    <!-- Draggable AI Chat button -->
    <button
      class="ai-float-btn"
      :style="{
        bottom: aiBtnPos.bottom + 'px',
        right: aiBtnPos.right + 'px'
      }"
      @pointerdown="onAiBtnPointerDown"
      @click="openAIChat"
      title="拖动可移动 · 点击打开 AI"
    >
      AI
    </button>

    <div
      v-if="showAIChat"
      class="ai-chat-panel"
      :style="{
        bottom: (aiBtnPos.bottom + 56) + 'px',
        right: aiBtnPos.right + 'px'
      }"
    >
      <div class="ai-chat-header">
        <span>推荐视频 AI</span>
        <button class="ai-close" @click="showAIChat = false">✕</button>
      </div>
      <div class="ai-chat-body">
        <div
          v-for="(msg, idx) in aiMessages"
          :key="idx"
          :class="['ai-msg', msg.role]"
        >
          {{ msg.content }}
        </div>
        <div v-if="aiLoading" class="ai-msg assistant">思考中...</div>
      </div>
      <div class="ai-chat-input">
        <input
          v-model="aiInput"
          @keyup.enter="sendAIMessage"
          placeholder="针对当前视频提问..."
          :disabled="aiLoading"
        />
        <button @click="sendAIMessage" :disabled="aiLoading || !aiInput.trim()">发送</button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.feed-page {
  height: calc(100dvh - var(--header-height));
  background: var(--color-bg);
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.feed-personalized-tip {
  position: absolute;
  top: var(--space-3);
  left: 50%;
  transform: translateX(-50%);
  z-index: 30;
  margin: 0;
  padding: 8px 14px;
  border-radius: var(--radius-full);
  background: var(--color-primary);
  color: #fff;
  font-size: var(--text-sm);
  font-weight: 500;
  box-shadow: var(--shadow-md);
  pointer-events: none;
}

.feed-scroller {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  scroll-snap-type: y mandatory;
  scroll-behavior: smooth;
  overscroll-behavior-y: contain;
  -webkit-overflow-scrolling: touch;
  background: var(--color-bg);
}

.feed-scroller::-webkit-scrollbar {
  display: none;
}

.feed-slide {
  height: 100%;
  scroll-snap-align: start;
  scroll-snap-stop: always;
  contain: layout paint;
}

.feed-skeleton {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.skeleton-slide {
  flex: 1;
}

.loading-hint {
  position: fixed;
  bottom: 80px;
  left: 50%;
  transform: translateX(-50%);
  z-index: 25;
  display: flex;
  align-items: center;
  gap: var(--space-2);
  padding: 6px 14px;
  border-radius: var(--radius-full);
  background: var(--color-surface-elevated);
  border: 1px solid var(--color-border);
  color: var(--color-text-secondary);
  font-size: 12px;
}

/* AI Chat styles */
.ai-float-btn {
  position: fixed;
  z-index: 100;
  width: 44px;
  height: 44px;
  border-radius: 50%;
  background: var(--color-primary-gradient);
  color: white;
  font-weight: 700;
  font-size: 14px;
  box-shadow: 0 4px 16px rgba(254, 44, 85, 0.4);
  border: none;
  cursor: grab;
  display: grid;
  place-items: center;
  transition: transform 0.15s ease;
  touch-action: none;
  user-select: none;
  -webkit-user-select: none;
}

.ai-float-btn:active {
  cursor: grabbing;
  transform: scale(0.92);
}

.ai-chat-panel {
  position: fixed;
  width: 380px;
  max-height: 520px;
  background: var(--color-surface-elevated);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  box-shadow: 0 8px 30px rgba(0, 0, 0, 0.5);
  display: flex;
  flex-direction: column;
  z-index: 110;
  overflow: hidden;
}

.ai-chat-header {
  padding: 10px 14px;
  background: var(--color-surface);
  font-weight: 600;
  color: var(--color-text);
  display: flex;
  justify-content: space-between;
  align-items: center;
  border-bottom: 1px solid var(--color-border);
}

.ai-close {
  border: none;
  background: none;
  font-size: 20px;
  cursor: pointer;
  color: var(--color-text-muted);
  padding: 4px 6px;
}

.ai-chat-body {
  flex: 1;
  padding: 12px;
  overflow-y: auto;
  font-size: 14px;
  line-height: 1.5;
}

.ai-msg {
  margin-bottom: 10px;
  padding: 8px 12px;
  border-radius: 12px;
  max-width: 85%;
}

.ai-msg.user {
  background: var(--color-primary);
  color: white;
  align-self: flex-end;
  margin-left: auto;
}

.ai-msg.assistant {
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  color: var(--color-text);
}

.ai-chat-input {
  display: flex;
  padding: 10px;
  border-top: 1px solid var(--color-border);
  gap: 8px;
  background: var(--color-surface);
}

.ai-chat-input input {
  flex: 1;
  padding: 8px 14px;
  border: 1px solid var(--color-border-strong);
  border-radius: var(--radius-full);
  background: var(--color-surface-elevated);
  color: var(--color-text);
  font-size: 14px;
  outline: none;
}

.ai-chat-input input:focus {
  border-color: var(--color-primary);
}

.ai-chat-input button {
  padding: 0 18px;
  border-radius: var(--radius-full);
  background: var(--color-primary-gradient);
  color: white;
  border: none;
  font-weight: 600;
  cursor: pointer;
}

.ai-chat-input button:disabled {
  opacity: 0.5;
}

.loading-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--color-primary);
  animation: pulse 0.8s ease infinite alternate;
}

@keyframes pulse {
  from {
    opacity: 0.4;
    transform: scale(0.85);
  }
  to {
    opacity: 1;
    transform: scale(1);
  }
}

@media (max-width: 768px) {
  .feed-page {
    height: 100dvh;
  }

  .ai-chat-panel {
    width: calc(100vw - 24px);
    max-height: 60vh;
  }

  .ai-float-btn {
    width: 40px;
    height: 40px;
    font-size: 12px;
  }
}
</style>
