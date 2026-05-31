<script setup>
import { computed, inject, ref, watch, onBeforeUnmount, nextTick } from 'vue'
import { RouterLink } from 'vue-router'
import { fetchVideo } from '../api/video'
import { fetchBarrages, postBarrage } from '../api/barrage'
import {
  fetchEngagement,
  toggleLike,
  toggleFavorite,
  fetchComments,
  postComment
} from '../api/interaction'
import { useAuthStore } from '../stores/auth'
import { formatCount } from '../utils/format'
import {
  destroyHls,
  defaultQuality,
  hasPlayUrls,
  mountHls,
  remountAtTime,
  sortedQualities
} from '../utils/hlsPlayer'
import { isSoundUnlocked, markSoundUnlocked } from '../utils/soundUnlock'

const SPEED_OPTIONS = [0.75, 1, 1.25, 1.5, 2]
const VOLUME_KEY = 'svp_volume'

function loadStoredVolume() {
  const raw = sessionStorage.getItem(VOLUME_KEY)
  if (raw == null) return 0.85
  const n = Number(raw)
  return Number.isFinite(n) ? Math.min(1, Math.max(0, n)) : 0.85
}

const props = defineProps({
  video: { type: Object, required: true },
  active: { type: Boolean, default: false },
  renderPlayer: { type: Boolean, default: false }
})

const auth = useAuthStore()
const feedSound = inject('feedSoundUnlock', null)
const videoEl = ref(null)
const localVideo = ref(props.video)
const paused = ref(false)
const showPlayHint = ref(false)
const commentsOpen = ref(false)
const barrageOpen = ref(false)
const actionLoading = ref(false)
const toast = ref('')

const engagement = ref({
  like_count: 0,
  comment_count: 0,
  favorite_count: 0,
  liked: false,
  favorited: false
})
const comments = ref([])
const commentText = ref('')
const barrageText = ref('')
const barrages = ref([])
const flyingDanmaku = ref([])
let hls = null
let pollTimer = null
let toastTimer = null
let danmakuSeq = 0
const shownBarrageKeys = new Set()
let lastPlaybackTime = 0

const volume = ref(loadStoredVolume())
const muted = ref(false)
const playbackRate = ref(1)
const currentQuality = ref('')
const showQualityMenu = ref(false)
const showSpeedMenu = ref(false)
const showVolumePanel = ref(false)
const needSoundUnlock = ref(false)

const availableQualities = computed(() => sortedQualities(localVideo.value?.play_urls || {}))

function qualityLabel(q) {
  const map = { '1080p': '1080P', '720p': '720P', '480p': '480P', '360p': '360P' }
  return map[q] || q
}

function speedLabel(rate) {
  return rate === 1 ? '1.0x' : `${rate}x`
}

function closeControlMenus() {
  showQualityMenu.value = false
  showSpeedMenu.value = false
  showVolumePanel.value = false
}

function applyVideoSettings() {
  if (!videoEl.value) return
  videoEl.value.volume = volume.value
  videoEl.value.muted = muted.value
  videoEl.value.playbackRate = playbackRate.value
}

function persistVolume() {
  sessionStorage.setItem(VOLUME_KEY, String(volume.value))
}

function unlockSound() {
  if (!videoEl.value) return
  muted.value = false
  needSoundUnlock.value = false
  markSoundUnlocked()
  if (feedSound?.soundUnlocked) feedSound.soundUnlocked.value = true
  if (volume.value === 0) volume.value = loadStoredVolume() || 0.85
  applyVideoSettings()
  videoEl.value.play().catch(() => {})
}

function toggleMute() {
  if (muted.value || volume.value === 0) {
    muted.value = false
    if (volume.value === 0) volume.value = 0.85
    needSoundUnlock.value = false
  } else {
    muted.value = true
  }
  persistVolume()
  applyVideoSettings()
}

function onVolumeInput(e) {
  volume.value = Number(e.target.value)
  muted.value = volume.value === 0
  needSoundUnlock.value = false
  persistVolume()
  applyVideoSettings()
}

function setPlaybackRate(rate) {
  playbackRate.value = rate
  applyVideoSettings()
  showSpeedMenu.value = false
}

async function switchQuality(quality) {
  if (!videoEl.value || !localVideo.value?.play_urls?.[quality]) return
  const currentTime = videoEl.value.currentTime || 0
  const wasPaused = videoEl.value.paused
  currentQuality.value = quality
  const url = localVideo.value.play_urls[quality]
  hls = remountAtTime(videoEl.value, url, currentTime, hls)
  applyVideoSettings()
  closeControlMenus()
  if (!wasPaused) {
    try {
      await videoEl.value.play()
      paused.value = false
    } catch {
      paused.value = true
    }
  }
}

async function tryAutoplay() {
  if (!videoEl.value) return
  const canPlayWithSound =
    isSoundUnlocked() || Boolean(feedSound?.soundUnlocked?.value)
  applyVideoSettings()
  if (canPlayWithSound) {
    videoEl.value.muted = false
    muted.value = false
    needSoundUnlock.value = false
  }
  try {
    await videoEl.value.play()
    paused.value = false
    showPlayHint.value = false
    if (!videoEl.value.muted) needSoundUnlock.value = false
    return
  } catch {
    if (canPlayWithSound) {
      paused.value = true
      showPlayHint.value = true
      return
    }
  }
  videoEl.value.muted = true
  muted.value = true
  needSoundUnlock.value = true
  try {
    await videoEl.value.play()
    paused.value = false
    showPlayHint.value = false
  } catch {
    paused.value = true
    showPlayHint.value = true
  }
}

function showToast(msg) {
  toast.value = msg
  clearTimeout(toastTimer)
  toastTimer = setTimeout(() => {
    toast.value = ''
  }, 2200)
}

function requireLogin(msg) {
  if (auth.isLoggedIn) return true
  showToast(msg)
  return false
}

async function loadEngagement() {
  try {
    const res = await fetchEngagement(localVideo.value.id)
    engagement.value = res.data.engagement || engagement.value
  } catch {
    // ignore
  }
}

async function loadComments() {
  try {
    const res = await fetchComments(localVideo.value.id, { page: 1, page_size: 40 })
    comments.value = res.data.comments || []
  } catch {
    comments.value = []
  }
}

async function loadBarrages() {
  try {
    const res = await fetchBarrages(localVideo.value.id)
    barrages.value = res.data.barrages || []
    shownBarrageKeys.clear()
    lastPlaybackTime = 0
    await nextTick()
    syncBarragesAtCurrentTime()
  } catch {
    barrages.value = []
  }
}

function spawnDanmaku(content, topPercent) {
  const top = topPercent ?? 10 + Math.random() * 55
  const key = `dm-${++danmakuSeq}`
  const hue = Math.floor(Math.random() * 360)
  flyingDanmaku.value.push({ key, content, top, color: `hsl(${hue} 95% 72%)` })
  setTimeout(() => {
    flyingDanmaku.value = flyingDanmaku.value.filter((d) => d.key !== key)
  }, 9000)
}

function barrageTimeMs(b) {
  const ms = Number(b?.time_ms)
  return Number.isFinite(ms) && ms >= 0 ? ms : 0
}

function onVideoTimeUpdate() {
  if (!videoEl.value || !props.active) return
  const current = videoEl.value.currentTime
  if (current < lastPlaybackTime - 0.4) {
    shownBarrageKeys.clear()
  }
  lastPlaybackTime = current
  const currentMs = current * 1000
  for (const b of barrages.value) {
    const timeMs = barrageTimeMs(b)
    const loop = Math.floor(current / Math.max(videoEl.value.duration || 1, 1))
    const trackKey = `${b.id}-${loop}`
    if (shownBarrageKeys.has(trackKey)) continue
    if (currentMs >= timeMs && currentMs < timeMs + 2500) {
      shownBarrageKeys.add(trackKey)
      spawnDanmaku(b.content)
    }
  }
}

function syncBarragesAtCurrentTime() {
  if (!videoEl.value) return
  const currentMs = videoEl.value.currentTime * 1000
  for (const b of barrages.value) {
    const timeMs = barrageTimeMs(b)
    const trackKey = `${b.id}-sync`
    if (shownBarrageKeys.has(trackKey)) continue
    if (Math.abs(currentMs - timeMs) <= 1500 || (timeMs === 0 && currentMs < 2000)) {
      shownBarrageKeys.add(trackKey)
      spawnDanmaku(b.content)
    }
  }
}

function bindVideoEvents() {
  nextTick(() => {
    if (!videoEl.value) return
    videoEl.value.removeEventListener('timeupdate', onVideoTimeUpdate)
    videoEl.value.addEventListener('timeupdate', onVideoTimeUpdate)
  })
}

function unbindVideoEvents() {
  if (videoEl.value) {
    videoEl.value.removeEventListener('timeupdate', onVideoTimeUpdate)
  }
}

async function setupPlayer() {
  teardownPlayer()
  if (!props.renderPlayer || !props.active) return
  const urls = localVideo.value?.play_urls
  if (!hasPlayUrls(urls)) return
  if (!currentQuality.value || !urls[currentQuality.value]) {
    currentQuality.value = defaultQuality(urls)
  }
  const url = urls[currentQuality.value]
  if (!url) return
  await nextTick()
  if (!videoEl.value) return
  hls = mountHls(videoEl.value, url)
  paused.value = false
  bindVideoEvents()
  await tryAutoplay()
}

function teardownPlayer() {
  unbindVideoEvents()
  destroyHls(hls)
  hls = null
  if (videoEl.value) {
    videoEl.value.pause()
    videoEl.value.removeAttribute('src')
    videoEl.value.load()
  }
}

async function togglePlay() {
  if (commentsOpen.value || barrageOpen.value) return
  closeControlMenus()
  if (!hasPlayUrls(localVideo.value?.play_urls) || !videoEl.value) return
  if (needSoundUnlock.value) {
    unlockSound()
    return
  }
  if (videoEl.value.paused) {
    try {
      await videoEl.value.play()
      paused.value = false
      showPlayHint.value = false
    } catch {
      showPlayHint.value = true
    }
  } else {
    videoEl.value.pause()
    paused.value = true
    showPlayHint.value = true
  }
}

function setupPolling() {
  clearInterval(pollTimer)
  const v = localVideo.value
  if (!props.active || !v || v.status === 'ready' || v.status === 'failed') return
  pollTimer = setInterval(async () => {
    try {
      const res = await fetchVideo(v.id)
      localVideo.value = res.data.video
      if (hasPlayUrls(localVideo.value.play_urls)) {
        clearInterval(pollTimer)
        await setupPlayer()
      } else if (localVideo.value.status === 'failed') {
        clearInterval(pollTimer)
      }
    } catch {
      // ignore
    }
  }, 3000)
}

async function handleLike() {
  if (!requireLogin('请先登录再点赞')) return
  if (actionLoading.value) return
  actionLoading.value = true
  try {
    const res = await toggleLike(localVideo.value.id)
    engagement.value.liked = res.data.liked
    engagement.value.like_count = res.data.like_count
  } catch (e) {
    showToast(e.message)
  } finally {
    actionLoading.value = false
  }
}

async function handleFavorite() {
  if (!requireLogin('请先登录再收藏')) return
  if (actionLoading.value) return
  actionLoading.value = true
  try {
    const res = await toggleFavorite(localVideo.value.id)
    engagement.value.favorited = res.data.favorited
    engagement.value.favorite_count = res.data.favorite_count
  } catch (e) {
    showToast(e.message)
  } finally {
    actionLoading.value = false
  }
}

async function openComments() {
  barrageOpen.value = false
  if (commentsOpen.value) {
    commentsOpen.value = false
    return
  }
  commentsOpen.value = true
  await loadComments()
}

function openBarrage() {
  commentsOpen.value = false
  barrageOpen.value = !barrageOpen.value
}

async function submitComment() {
  if (!requireLogin('请先登录再评论')) return
  const content = commentText.value.trim()
  if (!content) return
  try {
    const res = await postComment(localVideo.value.id, { content })
    comments.value.unshift(res.data.comment)
    commentText.value = ''
    engagement.value.comment_count += 1
  } catch (e) {
    showToast(e.message)
  }
}

async function sendBarrage() {
  if (!requireLogin('请先登录再发弹幕')) return
  const content = barrageText.value.trim()
  if (!content) return
  const timeMs = Math.floor((videoEl.value?.currentTime || 0) * 1000)
  try {
    const res = await postBarrage(localVideo.value.id, { content, time_ms: timeMs })
    const b = res.data.barrage
    if (b) {
      barrages.value.push(b)
      spawnDanmaku(b.content, 15 + Math.random() * 45)
    }
    barrageText.value = ''
    barrageOpen.value = false
    showToast('弹幕已发送')
  } catch (e) {
    showToast(e.message)
  }
}

function resetPanels() {
  commentsOpen.value = false
  barrageOpen.value = false
  flyingDanmaku.value = []
  shownBarrageKeys.clear()
  lastPlaybackTime = 0
}

watch(
  () => props.video,
  (v) => {
    localVideo.value = v
    currentQuality.value = defaultQuality(v?.play_urls)
    resetPanels()
  }
)

watch(
  () => props.active,
  async (active) => {
    if (active) {
      await Promise.all([loadEngagement(), loadBarrages()])
      if (feedSound?.soundUnlocked?.value && needSoundUnlock.value) {
        unlockSound()
      }
    } else {
      resetPanels()
      closeControlMenus()
      clearInterval(pollTimer)
      teardownPlayer()
    }
  }
)

watch(
  () => feedSound?.soundUnlocked?.value,
  (unlocked) => {
    if (unlocked && props.active && needSoundUnlock.value) {
      unlockSound()
    }
  }
)

watch(
  () => [props.active, props.renderPlayer, localVideo.value?.play_urls],
  async ([active, render]) => {
    if (active && render) {
      await setupPlayer()
      setupPolling()
      if (active) await Promise.all([loadEngagement(), loadBarrages()])
    } else if (!active) {
      clearInterval(pollTimer)
      teardownPlayer()
    }
  },
  { immediate: true, deep: true }
)

onBeforeUnmount(() => {
  clearInterval(pollTimer)
  clearTimeout(toastTimer)
  teardownPlayer()
})
</script>

<template>
  <div class="slide-root" :class="{ 'comments-open': commentsOpen }">
    <div class="stage" @click="togglePlay">
      <div class="media-wrap">
        <template v-if="renderPlayer && hasPlayUrls(localVideo.play_urls)">
          <video
            ref="videoEl"
            class="video"
            playsinline
            webkit-playsinline
            loop
            @timeupdate="onVideoTimeUpdate"
          />
        </template>
        <template v-else>
          <img
            v-if="localVideo.cover_url"
            :src="localVideo.cover_url"
            :alt="localVideo.title"
            class="cover"
          />
          <div v-else class="cover placeholder" />
        </template>

        <div v-if="showPlayHint && paused" class="play-hint" aria-hidden="true">
          <span class="play-icon">▶</span>
        </div>

        <div
          v-if="!hasPlayUrls(localVideo.play_urls)"
          class="status-banner"
          @click.stop
        >
          <p v-if="localVideo.status === 'transcoding' || localVideo.status === 'pending'">
            转码中，上滑看下一条
          </p>
          <p v-else-if="localVideo.status === 'failed'">转码失败</p>
          <p v-else>暂无可播放地址</p>
        </div>

        <div class="gradient" />

        <div class="danmaku-layer" aria-hidden="true">
          <span
            v-for="d in flyingDanmaku"
            :key="d.key"
            class="danmaku-item"
            :style="{ top: `${d.top}%`, color: d.color }"
          >
            {{ d.content }}
          </span>
        </div>

        <aside class="actions" @click.stop>
          <RouterLink :to="`/users/${localVideo.user_id}`" class="avatar">
            {{ (localVideo.user_id || 'U')[0].toUpperCase() }}
          </RouterLink>

          <button
            type="button"
            class="action-btn"
            :class="{ active: engagement.liked }"
            :disabled="actionLoading"
            @click="handleLike"
          >
            <span class="icon">{{ engagement.liked ? '❤️' : '🤍' }}</span>
            <span class="label">{{ formatCount(engagement.like_count) }}</span>
          </button>

          <button type="button" class="action-btn" :class="{ active: commentsOpen }" @click="openComments">
            <span class="icon">💬</span>
            <span class="label">{{ formatCount(engagement.comment_count) }}</span>
          </button>

          <button
            type="button"
            class="action-btn"
            :class="{ active: engagement.favorited }"
            :disabled="actionLoading"
            @click="handleFavorite"
          >
            <span class="icon">{{ engagement.favorited ? '⭐' : '☆' }}</span>
            <span class="label">{{ formatCount(engagement.favorite_count) }}</span>
          </button>
        </aside>

        <footer class="meta" @click.stop>
          <RouterLink :to="`/users/${localVideo.user_id}`" class="author">
            @{{ localVideo.user_id }}
          </RouterLink>
          <h2 class="title">{{ localVideo.title }}</h2>
          <p v-if="localVideo.tags?.length" class="tags">
            <span v-for="tag in localVideo.tags" :key="tag" class="tag">#{{ tag }}</span>
          </p>
          <p class="desc">{{ localVideo.description }}</p>
        </footer>

        <div
          v-if="renderPlayer && hasPlayUrls(localVideo.play_urls)"
          class="player-controls"
          @click.stop
        >
          <div v-if="availableQualities.length" class="ctrl-wrap">
            <button
              type="button"
              class="ctrl-btn"
              :disabled="!hasPlayUrls(localVideo.play_urls)"
              @click="showQualityMenu = !showQualityMenu; showSpeedMenu = false; showVolumePanel = false"
            >
              {{ qualityLabel(currentQuality) || '清晰度' }}
            </button>
            <div v-if="showQualityMenu" class="ctrl-menu">
              <button
                v-for="q in availableQualities"
                :key="q"
                type="button"
                class="ctrl-menu-item"
                :class="{ active: q === currentQuality }"
                @click="switchQuality(q)"
              >
                {{ qualityLabel(q) }}
              </button>
            </div>
          </div>

          <div class="ctrl-wrap">
            <button
              type="button"
              class="ctrl-btn"
              :class="{ highlight: needSoundUnlock || muted }"
              @click="showVolumePanel = !showVolumePanel; showQualityMenu = false; showSpeedMenu = false"
            >
              {{ muted || volume === 0 ? '静音' : '音量' }}
            </button>
            <div v-if="showVolumePanel" class="ctrl-menu ctrl-menu--volume">
              <button type="button" class="ctrl-menu-item" @click="toggleMute">
                {{ muted || volume === 0 ? '开启声音' : '静音' }}
              </button>
              <input
                type="range"
                min="0"
                max="1"
                step="0.05"
                :value="volume"
                class="volume-slider"
                @input="onVolumeInput"
              />
            </div>
          </div>

          <div class="ctrl-wrap">
            <button
              type="button"
              class="ctrl-btn"
              @click="showSpeedMenu = !showSpeedMenu; showQualityMenu = false; showVolumePanel = false"
            >
              {{ speedLabel(playbackRate) }}
            </button>
            <div v-if="showSpeedMenu" class="ctrl-menu">
              <button
                v-for="rate in SPEED_OPTIONS"
                :key="rate"
                type="button"
                class="ctrl-menu-item"
                :class="{ active: rate === playbackRate }"
                @click="setPlaybackRate(rate)"
              >
                {{ speedLabel(rate) }}
              </button>
            </div>
          </div>
        </div>

        <div v-if="needSoundUnlock" class="sound-hint" @click.stop="unlockSound">
          点击开启声音
        </div>

        <div class="barrage-corner" @click.stop>
          <button type="button" class="barrage-toggle" @click="openBarrage">
            {{ barrageOpen ? '收起弹幕' : '发弹幕' }}
          </button>
          <div v-if="barrageOpen" class="barrage-form">
            <input
              v-model="barrageText"
              placeholder="输入弹幕..."
              maxlength="100"
              @keyup.enter="sendBarrage"
            />
            <button type="button" @click="sendBarrage">发送</button>
          </div>
        </div>
      </div>
    </div>

    <aside v-if="commentsOpen" class="comment-panel" @click.stop>
      <header class="comment-head">
        <h3>评论 {{ formatCount(engagement.comment_count) }}</h3>
        <button type="button" class="close-btn" @click="commentsOpen = false">✕</button>
      </header>
      <div class="comment-list">
        <div v-if="comments.length" class="comment-items">
          <div v-for="c in comments" :key="c.id" class="comment-item">
            <strong>{{ c.username || c.user_id }}</strong>
            <p>{{ c.content }}</p>
          </div>
        </div>
        <p v-else class="comment-empty">暂无评论，来说两句吧</p>
      </div>
      <div class="comment-input">
        <input
          v-model="commentText"
          placeholder="写下你的评论..."
          maxlength="500"
          @keyup.enter="submitComment"
        />
        <button type="button" @click="submitComment">发送</button>
      </div>
    </aside>

    <div v-if="toast" class="toast">{{ toast }}</div>
  </div>
</template>

<style scoped>
.slide-root {
  display: flex;
  width: 100%;
  height: 100%;
  background: #000;
  overflow: hidden;
}

.stage {
  flex: 1;
  min-width: 0;
  height: 100%;
  transition: flex 0.28s ease;
}

.comments-open .stage {
  flex: 0 0 72%;
}

.media-wrap {
  position: relative;
  width: 100%;
  height: 100%;
  overflow: hidden;
  background: #000;
}

.video,
.cover {
  width: 100%;
  height: 100%;
  object-fit: cover;
  display: block;
  transition: transform 0.28s ease;
}

.comments-open .video,
.comments-open .cover {
  transform: scale(0.96);
  transform-origin: center center;
}

.cover.placeholder {
  background: #111827;
}

.play-hint {
  position: absolute;
  inset: 0;
  display: grid;
  place-items: center;
  pointer-events: none;
}

.play-icon {
  width: 64px;
  height: 64px;
  border-radius: 50%;
  background: rgba(0, 0, 0, 0.45);
  display: grid;
  place-items: center;
  font-size: 28px;
  color: #fff;
  padding-left: 4px;
}

.status-banner {
  position: absolute;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  padding: 12px 16px;
  border-radius: 12px;
  background: rgba(0, 0, 0, 0.55);
  font-size: 14px;
  text-align: center;
  max-width: 80%;
}

.gradient {
  position: absolute;
  left: 0;
  right: 0;
  bottom: 0;
  height: 42%;
  background: rgba(0, 0, 0, 0.35);
  pointer-events: none;
}

.danmaku-layer {
  position: absolute;
  inset: 0;
  overflow: hidden;
  pointer-events: none;
  z-index: 6;
}

.danmaku-item {
  position: absolute;
  left: 100%;
  top: 0;
  padding: 3px 12px;
  border-radius: 999px;
  background: rgba(0, 0, 0, 0.42);
  font-size: 16px;
  font-weight: 700;
  line-height: 1.4;
  white-space: nowrap;
  text-shadow: 0 0 4px rgba(0, 0, 0, 0.9), 0 1px 2px #000;
  animation: danmaku-fly 9s linear forwards;
  will-change: transform;
}

@keyframes danmaku-fly {
  from {
    transform: translateX(0);
  }
  to {
    transform: translateX(calc(-100% - 100vw));
  }
}

.actions {
  position: absolute;
  right: 10px;
  bottom: 110px;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 18px;
  z-index: 3;
}

.avatar {
  width: 48px;
  height: 48px;
  border-radius: 50%;
  border: 2px solid #fff;
  display: grid;
  place-items: center;
  background: #334155;
  color: #fff;
  font-weight: 700;
  text-decoration: none;
  font-size: 18px;
}

.action-btn {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;
  border: none;
  background: none;
  color: #fff;
  cursor: pointer;
  padding: 2px;
  min-width: 48px;
}

.action-btn:disabled {
  opacity: 0.6;
}

.action-btn.active .label {
  color: #f472b6;
}

.action-btn .icon {
  font-size: 28px;
  filter: drop-shadow(0 1px 2px rgba(0, 0, 0, 0.5));
}

.action-btn .label {
  font-size: 11px;
  color: #e2e8f0;
}

.meta {
  position: absolute;
  left: 0;
  right: 68px;
  bottom: 0;
  padding: 16px 16px 72px;
  z-index: 2;
}

.author {
  display: inline-block;
  margin-bottom: 8px;
  color: #f8fafc;
  font-weight: 600;
  text-decoration: none;
  font-size: 15px;
}

.title {
  margin: 0 0 6px;
  font-size: 16px;
  font-weight: 600;
  line-height: 1.35;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.tags {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin: 0 0 6px;
}

.tag {
  font-size: 12px;
  color: #fff;
  background: rgba(37, 99, 235, 0.75);
  padding: 2px 8px;
  border-radius: var(--radius-full);
}

.desc {
  margin: 0;
  font-size: 13px;
  color: #cbd5e1;
  line-height: 1.45;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.player-controls {
  position: absolute;
  left: 12px;
  bottom: 72px;
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  z-index: 4;
}

.ctrl-wrap {
  position: relative;
}

.ctrl-btn {
  border: 1px solid rgba(255, 255, 255, 0.35);
  background: rgba(0, 0, 0, 0.45);
  color: #fff;
  font-size: 12px;
  padding: 6px 10px;
  border-radius: 8px;
  cursor: pointer;
  backdrop-filter: blur(4px);
}

.ctrl-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.ctrl-btn.highlight {
  border-color: #60a5fa;
  color: #bfdbfe;
}

.ctrl-menu {
  position: absolute;
  bottom: calc(100% + 6px);
  left: 0;
  min-width: 88px;
  padding: 6px;
  border-radius: 10px;
  background: rgba(15, 23, 42, 0.92);
  border: 1px solid rgba(255, 255, 255, 0.12);
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.ctrl-menu--volume {
  min-width: 140px;
  padding: 10px;
}

.ctrl-menu-item {
  border: none;
  background: transparent;
  color: #e2e8f0;
  text-align: left;
  padding: 6px 8px;
  border-radius: 6px;
  cursor: pointer;
  font-size: 13px;
}

.ctrl-menu-item:hover,
.ctrl-menu-item.active {
  background: rgba(37, 99, 235, 0.35);
  color: #fff;
}

.volume-slider {
  width: 100%;
  margin-top: 8px;
  accent-color: #2563eb;
}

.sound-hint {
  position: absolute;
  left: 12px;
  bottom: 118px;
  z-index: 5;
  padding: 6px 12px;
  border-radius: 8px;
  background: rgba(37, 99, 235, 0.85);
  color: #fff;
  font-size: 12px;
  cursor: pointer;
}

.barrage-corner {
  position: absolute;
  left: 12px;
  bottom: 20px;
  z-index: 4;
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 8px;
  max-width: calc(100% - 90px);
}

.barrage-toggle {
  padding: 8px 14px;
  border: none;
  border-radius: 999px;
  background: rgba(0, 0, 0, 0.55);
  color: #f8fafc;
  font-size: 13px;
  cursor: pointer;
  backdrop-filter: blur(6px);
}

.barrage-form {
  display: flex;
  gap: 8px;
  width: min(320px, 72vw);
}

.barrage-form input {
  flex: 1;
  padding: 8px 12px;
  border-radius: 999px;
  border: 1px solid rgba(255, 255, 255, 0.2);
  background: rgba(15, 23, 42, 0.85);
  color: #f8fafc;
  font-size: 13px;
}

.barrage-form button {
  padding: 8px 14px;
  border: none;
  border-radius: 999px;
  background: #0ea5e9;
  color: #fff;
  font-size: 13px;
  cursor: pointer;
  white-space: nowrap;
}

.comment-panel {
  flex: 0 0 28%;
  min-width: 0;
  height: 100%;
  display: flex;
  flex-direction: column;
  background: #0f172a;
  border-left: 1px solid #334155;
  z-index: 5;
  animation: slide-in 0.28s ease;
}

@keyframes slide-in {
  from {
    transform: translateX(100%);
    opacity: 0.6;
  }
  to {
    transform: translateX(0);
    opacity: 1;
  }
}

.comment-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 14px 12px 10px;
  padding-top: max(14px, env(safe-area-inset-top));
  border-bottom: 1px solid #334155;
}

.comment-head h3 {
  margin: 0;
  font-size: 15px;
}

.close-btn {
  border: none;
  background: none;
  color: #94a3b8;
  font-size: 18px;
  cursor: pointer;
  padding: 4px 8px;
}

.comment-list {
  flex: 1;
  overflow-y: auto;
  padding: 8px 12px;
}

.comment-items {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.comment-item strong {
  display: block;
  font-size: 13px;
  color: #f8fafc;
  margin-bottom: 4px;
}

.comment-item p {
  margin: 0;
  font-size: 13px;
  line-height: 1.5;
  color: #cbd5e1;
  word-break: break-word;
}

.comment-empty {
  margin: 24px 0;
  text-align: center;
  color: #64748b;
  font-size: 13px;
}

.comment-input {
  display: flex;
  gap: 8px;
  padding: 10px 12px 16px;
  border-top: 1px solid #334155;
}

.comment-input input {
  flex: 1;
  padding: 8px 10px;
  border-radius: 8px;
  border: 1px solid #475569;
  background: #1e293b;
  color: #f8fafc;
  font-size: 13px;
}

.comment-input button {
  padding: 8px 12px;
  border: none;
  border-radius: 8px;
  background: #0ea5e9;
  color: #fff;
  font-size: 13px;
  cursor: pointer;
}

.toast {
  position: absolute;
  left: 50%;
  top: 72px;
  transform: translateX(-50%);
  z-index: 30;
  padding: 8px 16px;
  border-radius: 999px;
  background: rgba(0, 0, 0, 0.75);
  color: #f8fafc;
  font-size: 13px;
  pointer-events: none;
  white-space: nowrap;
}

@media (max-width: 640px) {
  .comments-open .stage {
    flex: 0 0 62%;
  }

  .comment-panel {
    flex: 0 0 38%;
  }
}
</style>
