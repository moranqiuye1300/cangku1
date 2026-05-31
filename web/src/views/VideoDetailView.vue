<script setup>
import { onMounted, onBeforeUnmount, ref, watch, nextTick } from 'vue'
import { useRoute, RouterLink } from 'vue-router'
import Hls from 'hls.js'
import { fetchVideo } from '../api/video'
import { fetchBarrages, postBarrage } from '../api/barrage'
import {
  fetchEngagement,
  toggleLike,
  toggleFavorite,
  fetchComments,
  postComment
} from '../api/interaction'
import { askAI } from '../api/ai'
import { useAuthStore } from '../stores/auth'
import { formatDuration, formatDate, formatCount } from '../utils/format'

const route = useRoute()
const auth = useAuthStore()
const video = ref(null)
const engagement = ref({
  like_count: 0,
  comment_count: 0,
  favorite_count: 0,
  liked: false,
  favorited: false
})
const comments = ref([])
const commentText = ref('')
const barrages = ref([])
const barrageText = ref('')
const aiQuestion = ref('')
const aiAnswer = ref('')
const loading = ref(true)
const actionLoading = ref(false)
const error = ref('')
const videoEl = ref(null)
const commentsRef = ref(null)
const currentQuality = ref('')
let hls = null
let pollTimer = null

const QUALITY_ORDER = ['1080p', '720p', '480p', '360p']

function sortedQualities(playUrls) {
  if (!playUrls) return []
  return Object.keys(playUrls).sort((a, b) => {
    const ai = QUALITY_ORDER.indexOf(a)
    const bi = QUALITY_ORDER.indexOf(b)
    return (ai === -1 ? 999 : ai) - (bi === -1 ? 999 : bi)
  })
}

function defaultQuality(playUrls) {
  if (!playUrls) return ''
  if (playUrls['720p']) return '720p'
  if (playUrls['1080p']) return '1080p'
  const sorted = sortedQualities(playUrls)
  return sorted[0] || ''
}

function hasPlayUrls(playUrls) {
  return playUrls && Object.keys(playUrls).length > 0
}

function requireLogin(message) {
  if (auth.isLoggedIn) return true
  error.value = message
  return false
}

async function mountPlayer(url, resumeTime = 0) {
  teardownPlayer()
  if (!url) return
  await nextTick()
  if (!videoEl.value) return
  if (Hls.isSupported()) {
    hls = new Hls()
    hls.loadSource(url)
    hls.attachMedia(videoEl.value)
  } else if (videoEl.value.canPlayType('application/vnd.apple.mpegurl')) {
    videoEl.value.src = url
  }
  if (resumeTime > 0) {
    videoEl.value.currentTime = resumeTime
    videoEl.value.play().catch(() => {})
  }
}

async function playQuality(quality, resumeTime = 0) {
  if (!video.value?.play_urls?.[quality]) return
  currentQuality.value = quality
  await mountPlayer(video.value.play_urls[quality], resumeTime)
}

async function switchQuality(quality) {
  if (!video.value?.play_urls?.[quality] || currentQuality.value === quality) return
  const resumeTime = videoEl.value?.currentTime ?? 0
  await playQuality(quality, resumeTime)
}

function teardownPlayer() {
  if (hls) {
    hls.destroy()
    hls = null
  }
}

async function loadEngagement() {
  try {
    const res = await fetchEngagement(route.params.id)
    engagement.value = res.data.engagement || engagement.value
  } catch {
    // ignore
  }
}

async function loadComments() {
  try {
    const res = await fetchComments(route.params.id, { page: 1, page_size: 50 })
    comments.value = res.data.comments || []
  } catch {
    comments.value = []
  }
}

async function loadVideo() {
  loading.value = true
  error.value = ''
  try {
    const res = await fetchVideo(route.params.id)
    video.value = res.data.video
    const q = defaultQuality(video.value.play_urls)
    loading.value = false
    if (q) {
      await playQuality(q)
    }
    await Promise.all([loadEngagement(), loadComments(), loadBarrages()])
    setupPolling()
  } catch (e) {
    error.value = e.message
    video.value = null
    loading.value = false
  }
}

function setupPolling() {
  clearInterval(pollTimer)
  if (!video.value || video.value.status === 'ready' || video.value.status === 'failed') {
    return
  }
  pollTimer = setInterval(async () => {
    try {
      const res = await fetchVideo(route.params.id)
      video.value = res.data.video
      const q =
        currentQuality.value && video.value.play_urls?.[currentQuality.value]
          ? currentQuality.value
          : defaultQuality(video.value.play_urls)
      if (q) {
        await playQuality(q)
      }
      if (video.value.status === 'ready' || video.value.status === 'failed') {
        clearInterval(pollTimer)
      }
    } catch {
      // ignore polling errors
    }
  }, 3000)
}

async function loadBarrages() {
  try {
    const res = await fetchBarrages(route.params.id)
    barrages.value = res.data.barrages || []
  } catch {
    barrages.value = []
  }
}

async function handleLike() {
  if (!requireLogin('请先登录再点赞')) return
  if (actionLoading.value) return
  actionLoading.value = true
  try {
    const res = await toggleLike(route.params.id)
    engagement.value.liked = res.data.liked
    engagement.value.like_count = res.data.like_count
  } catch (e) {
    error.value = e.message
  } finally {
    actionLoading.value = false
  }
}

async function handleFavorite() {
  if (!requireLogin('请先登录再收藏')) return
  if (actionLoading.value) return
  actionLoading.value = true
  try {
    const res = await toggleFavorite(route.params.id)
    engagement.value.favorited = res.data.favorited
    engagement.value.favorite_count = res.data.favorite_count
  } catch (e) {
    error.value = e.message
  } finally {
    actionLoading.value = false
  }
}

function scrollToComments() {
  commentsRef.value?.scrollIntoView({ behavior: 'smooth', block: 'start' })
}

async function submitComment() {
  if (!requireLogin('请先登录再评论')) return
  const content = commentText.value.trim()
  if (!content) return
  try {
    const res = await postComment(route.params.id, { content })
    comments.value.unshift(res.data.comment)
    commentText.value = ''
    engagement.value.comment_count += 1
  } catch (e) {
    error.value = e.message
  }
}

async function sendBarrage() {
  if (!requireLogin('请先登录再发弹幕')) return
  const content = barrageText.value.trim()
  if (!content) return
  const timeMs = Math.floor((videoEl.value?.currentTime || 0) * 1000)
  await postBarrage(route.params.id, { content, time_ms: timeMs })
  barrageText.value = ''
  await loadBarrages()
}

async function doAskAI() {
  const q = aiQuestion.value.trim()
  if (!q) return
  const res = await askAI(q)
  aiAnswer.value = res.data.answer || JSON.stringify(res.data)
}

onMounted(loadVideo)
watch(() => route.params.id, loadVideo)
onBeforeUnmount(() => {
  teardownPlayer()
  clearInterval(pollTimer)
})
</script>

<template>
  <div class="page">
    <RouterLink :to="{ path: '/', query: { v: route.params.id } }" class="back">← 返回推荐流</RouterLink>

    <section v-if="loading" class="panel">加载中...</section>
    <section v-else-if="error && !video" class="panel error">{{ error }}</section>
    <section v-else-if="video" class="layout">
      <div class="player-row">
        <div class="player">
          <video v-if="hasPlayUrls(video.play_urls)" ref="videoEl" controls playsinline class="video" />
          <template v-else>
            <img v-if="video.cover_url" :src="video.cover_url" :alt="video.title" />
            <div class="overlay">
              <p v-if="video.status === 'transcoding' || video.status === 'pending'">转码中，请稍候...</p>
              <p v-else-if="video.status === 'failed'">转码失败</p>
              <p v-else>暂无可播放地址</p>
            </div>
          </template>
        </div>

        <aside class="sidebar">
          <RouterLink :to="`/users/${video.user_id}`" class="sidebar-avatar">
            {{ (video.user_id || 'U')[0].toUpperCase() }}
          </RouterLink>

          <button
            type="button"
            class="sidebar-btn"
            :class="{ active: engagement.liked }"
            :disabled="actionLoading"
            @click="handleLike"
          >
            <span class="icon">{{ engagement.liked ? '❤️' : '🤍' }}</span>
            <span class="count">{{ formatCount(engagement.like_count) }}</span>
          </button>

          <button type="button" class="sidebar-btn" @click="scrollToComments">
            <span class="icon">💬</span>
            <span class="count">{{ formatCount(engagement.comment_count) }}</span>
          </button>

          <button
            type="button"
            class="sidebar-btn"
            :class="{ active: engagement.favorited }"
            :disabled="actionLoading"
            @click="handleFavorite"
          >
            <span class="icon">{{ engagement.favorited ? '⭐' : '☆' }}</span>
            <span class="count">{{ formatCount(engagement.favorite_count) }}</span>
          </button>
        </aside>
      </div>

      <p v-if="error" class="inline-error">{{ error }}</p>

      <div class="panel">
        <h1>{{ video.title }}</h1>
        <div class="meta">
          <RouterLink :to="`/users/${video.user_id}`">@{{ video.user_id }}</RouterLink>
          <span>{{ formatDate(video.created_at) }}</span>
          <span class="badge">{{ video.status }}</span>
          <span v-if="video.duration">{{ formatDuration(video.duration) }}</span>
        </div>
        <p class="desc">{{ video.description }}</p>
        <div v-if="hasPlayUrls(video.play_urls)" class="qualities">
          <span class="qualities-label">清晰度</span>
          <button
            v-for="q in sortedQualities(video.play_urls)"
            :key="q"
            type="button"
            class="quality-btn"
            :class="{ active: currentQuality === q }"
            @click="switchQuality(q)"
          >
            {{ q }}
          </button>
        </div>

        <div ref="commentsRef" class="comments-box">
          <h3>评论 ({{ engagement.comment_count }})</h3>
          <div v-if="comments.length" class="comment-list">
            <div v-for="c in comments" :key="c.id" class="comment-item">
              <strong>{{ c.username || c.user_id }}</strong>
              <span class="time">{{ formatDate(c.created_at) }}</span>
              <p>{{ c.content }}</p>
            </div>
          </div>
          <p v-else class="muted">暂无评论，来说两句吧</p>
          <div class="comment-form">
            <input
              v-model="commentText"
              placeholder="写下你的评论..."
              maxlength="500"
              @keyup.enter="submitComment"
            />
            <button type="button" @click="submitComment">发送</button>
          </div>
        </div>

        <div class="barrage-box">
          <h3>弹幕</h3>
          <div v-if="barrages.length" class="barrage-list">
            <div v-for="b in barrages" :key="b.id" class="barrage-item">
              <strong>{{ b.username || b.user_id }}</strong>
              <span class="time">{{ (b.time_ms / 1000).toFixed(1) }}s</span>
              {{ b.content }}
            </div>
          </div>
          <p v-else class="muted">暂无弹幕</p>
          <div class="barrage-form">
            <input v-model="barrageText" placeholder="发送弹幕..." @keyup.enter="sendBarrage" />
            <button type="button" @click="sendBarrage">发送</button>
          </div>
        </div>

        <div class="ai-box">
          <h3>AI 问答（LangChain MVP）</h3>
          <div class="barrage-form">
            <input v-model="aiQuestion" placeholder="问：Go 并发相关视频有哪些？" @keyup.enter="doAskAI" />
            <button type="button" @click="doAskAI">提问</button>
          </div>
          <p v-if="aiAnswer" class="ai-answer">{{ aiAnswer }}</p>
        </div>
      </div>
    </section>
  </div>
</template>

<style scoped>
.page {
  max-width: 960px;
  margin: 0 auto;
  padding: 24px;
}

.back {
  display: inline-block;
  margin-bottom: 16px;
  color: #94a3b8;
  text-decoration: none;
}

.layout {
  display: grid;
  gap: 18px;
}

.player-row {
  display: grid;
  grid-template-columns: 1fr 88px;
  gap: 12px;
  align-items: stretch;
}

.player {
  position: relative;
  border-radius: 16px;
  overflow: hidden;
  background: #000;
  aspect-ratio: 16 / 9;
}

.video,
.player img {
  width: 100%;
  height: 100%;
  object-fit: contain;
  background: #000;
}

.overlay {
  position: absolute;
  inset: 0;
  display: grid;
  place-items: center;
  background: rgba(0, 0, 0, 0.45);
  text-align: center;
  padding: 16px;
}

.sidebar {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 18px;
  padding: 12px 0;
  border-radius: 16px;
  background: #1e293b;
}

.sidebar-avatar {
  width: 52px;
  height: 52px;
  border-radius: 50%;
  border: 2px solid #38bdf8;
  display: grid;
  place-items: center;
  background: #334155;
  color: #fff;
  font-weight: 700;
  font-size: 20px;
  text-decoration: none;
}

.sidebar-btn {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;
  border: none;
  background: none;
  color: #e2e8f0;
  cursor: pointer;
  padding: 4px;
  min-width: 56px;
}

.sidebar-btn:disabled {
  opacity: 0.6;
  cursor: wait;
}

.sidebar-btn.active .count {
  color: #f472b6;
}

.sidebar-btn .icon {
  font-size: 28px;
  line-height: 1;
}

.sidebar-btn .count {
  font-size: 12px;
  color: #94a3b8;
}

.inline-error {
  margin: 0;
  padding: 10px 14px;
  border-radius: 10px;
  background: #7f1d1d;
  color: #fecaca;
  font-size: 13px;
}

.panel {
  padding: 20px;
  border-radius: 16px;
  background: #1e293b;
}

.panel.error {
  color: #fecaca;
}

h1 {
  margin: 0 0 12px;
  font-size: 24px;
}

.meta {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  margin-bottom: 16px;
  font-size: 14px;
  color: #94a3b8;
}

.meta a {
  color: #38bdf8;
  text-decoration: none;
}

.badge {
  padding: 2px 8px;
  border-radius: 999px;
  background: #334155;
  text-transform: uppercase;
  font-size: 12px;
}

.desc {
  margin: 0;
  line-height: 1.7;
  color: #cbd5e1;
}

.qualities {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
  margin-top: 12px;
}

.qualities-label {
  font-size: 13px;
  color: #94a3b8;
  margin-right: 4px;
}

.quality-btn {
  padding: 6px 12px;
  border: 1px solid #475569;
  border-radius: 8px;
  background: #334155;
  color: #e2e8f0;
  font-size: 12px;
  cursor: pointer;
}

.quality-btn.active {
  background: #0ea5e9;
  border-color: #38bdf8;
  color: #fff;
}

.comments-box,
.barrage-box,
.ai-box {
  margin-top: 20px;
  padding-top: 16px;
  border-top: 1px solid #334155;
}

.comments-box h3,
.barrage-box h3,
.ai-box h3 {
  margin: 0 0 12px;
  font-size: 16px;
}

.comment-list {
  max-height: 280px;
  overflow-y: auto;
  margin-bottom: 12px;
}

.comment-item {
  padding: 10px 0;
  border-bottom: 1px solid #334155;
}

.comment-item:last-child {
  border-bottom: none;
}

.comment-item strong {
  color: #f8fafc;
  font-size: 14px;
}

.comment-item .time {
  margin-left: 8px;
  color: #64748b;
  font-size: 12px;
}

.comment-item p {
  margin: 6px 0 0;
  color: #cbd5e1;
  font-size: 14px;
  line-height: 1.5;
}

.comment-form,
.barrage-form {
  display: flex;
  gap: 8px;
}

.comment-form input,
.barrage-form input {
  flex: 1;
  padding: 8px 10px;
  border-radius: 8px;
  border: 1px solid #475569;
  background: #0f172a;
  color: #f8fafc;
}

.comment-form button,
.barrage-form button {
  padding: 8px 12px;
  border: none;
  border-radius: 8px;
  background: #0ea5e9;
  color: #fff;
  cursor: pointer;
}

.barrage-list {
  max-height: 160px;
  overflow-y: auto;
  margin-bottom: 12px;
}

.barrage-item {
  font-size: 13px;
  padding: 6px 0;
  color: #cbd5e1;
}

.barrage-item .time {
  color: #64748b;
  margin: 0 6px;
}

.muted {
  color: #64748b;
  font-size: 13px;
}

.ai-answer {
  margin: 12px 0 0;
  line-height: 1.6;
  color: #a5f3fc;
  font-size: 14px;
}

@media (max-width: 720px) {
  .player-row {
    grid-template-columns: 1fr;
  }

  .sidebar {
    flex-direction: row;
    justify-content: space-around;
    padding: 14px;
  }
}
</style>
