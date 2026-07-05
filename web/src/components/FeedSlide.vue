<script setup>
import { computed, inject, ref, watch, onMounted, onBeforeUnmount, nextTick } from 'vue'
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
  seekWhenReady,
  sortedQualities
} from '../utils/hlsPlayer'
import { isSoundUnlocked, markSoundUnlocked } from '../utils/soundUnlock'
import { saveProgress, loadProgress, formatPlaybackTime } from '../utils/playbackMemory'
import { throttle } from '../utils/throttle'
import { videoStatusLabel } from '../utils/videoStatus'

defineOptions({ name: 'FeedSlide' })

const SPEED_OPTIONS = [0.75, 1, 1.25, 1.5, 2]
const VOLUME_KEY = 'svp_volume'
const RATE_KEY = 'svp_rate'
const MAX_FLYING_DANMAKU = 8

function loadStoredVolume() {
  const raw = localStorage.getItem(VOLUME_KEY)
  if (raw == null) return 0.85
  const n = Number(raw)
  return Number.isFinite(n) ? Math.min(1, Math.max(0, n)) : 0.85
}

function loadStoredRate() {
  const n = Number(localStorage.getItem(RATE_KEY))
  return SPEED_OPTIONS.includes(n) ? n : 1
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
const showSwipeHint = ref(false)
let swipeHintTimer = null
let hls = null
const SWIPE_HINT_KEY = 'svp_feed_swipe_hint'
let pollTimer = null
let toastTimer = null
let danmakuSeq = 0
const shownBarrageKeys = new Set()
let lastPlaybackTime = 0

const volume = ref(loadStoredVolume())
const muted = ref(false)
const playbackRate = ref(loadStoredRate())
const currentQuality = ref('')
const showQualityMenu = ref(false)
const showSpeedMenu = ref(false)
const showVolumePanel = ref(false)
const needSoundUnlock = ref(false)
const playbackTime = ref(0)
const videoDuration = ref(0)

const availableQualities = computed(() => sortedQualities(localVideo.value?.play_urls || {}))

const progressPercent = computed(() => {
  if (!videoDuration.value) return 0
  return Math.min(100, (playbackTime.value / videoDuration.value) * 100)
})

const showPiP = computed(
  () => typeof document !== 'undefined' && Boolean(document.pictureInPictureEnabled)
)

const throttledSaveProgress = throttle((id, time) => {
  saveProgress(id, time)
}, 2000)

function qualityLabel(q) {
  const map = { '1080p': '1080P', '720p': '720P', '480p': '480P', '360p': '360P' }
  return map[q] || q
}

function speedLabel(rate) {
  return rate === 1 ? '1.0x' : `${rate}x`
}

function onVolumeBtnClick() {
  if (needSoundUnlock.value) {
    unlockSound()
    return
  }
  showVolumePanel.value = !showVolumePanel.value
  showQualityMenu.value = false
  showSpeedMenu.value = false
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
  localStorage.setItem(VOLUME_KEY, String(volume.value))
}

function persistRate() {
  localStorage.setItem(RATE_KEY, String(playbackRate.value))
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
  persistRate()
  applyVideoSettings()
  showSpeedMenu.value = false
}

async function toggleFullscreen() {
  closeControlMenus()
  const el = videoEl.value?.closest('.media-wrap')
  if (!el) return
  try {
    if (document.fullscreenElement) {
      await document.exitFullscreen()
    } else {
      await el.requestFullscreen?.()
    }
  } catch {
    showToast('全屏不可用')
  }
}

async function togglePiP() {
  closeControlMenus()
  if (!videoEl.value) return
  try {
    if (document.pictureInPictureElement) {
      await document.exitPictureInPicture()
    } else if (document.pictureInPictureEnabled) {
      await videoEl.value.requestPictureInPicture()
    } else {
      showToast('小窗播放不可用')
    }
  } catch {
    showToast('小窗播放不可用')
  }
}

function onDocumentClick() {
  closeControlMenus()
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
  if (flyingDanmaku.value.length >= MAX_FLYING_DANMAKU) return
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
  playbackTime.value = current
  if (videoEl.value.duration && Number.isFinite(videoEl.value.duration)) {
    videoDuration.value = videoEl.value.duration
  }
  throttledSaveProgress(localVideo.value.id, current)
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
  const saved = loadProgress(localVideo.value.id)
  if (saved > 3) {
    seekWhenReady(videoEl.value, saved, () => {
      showToast(`已从 ${formatPlaybackTime(saved)} 继续播放`)
    })
  }
  applyVideoSettings()
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
      if (localVideo.value.status === 'ready' && hasPlayUrls(localVideo.value.play_urls)) {
        clearInterval(pollTimer)
        await setupPlayer()
      } else if (localVideo.value.status === 'failed') {
        clearInterval(pollTimer)
      } else if (
        localVideo.value.status === 'pending_final_review' &&
        hasPlayUrls(localVideo.value.play_urls)
      ) {
        clearInterval(pollTimer)
        await setupPlayer()
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

function maybeShowSwipeHint() {
  if (!props.active || localStorage.getItem(SWIPE_HINT_KEY)) return
  showSwipeHint.value = true
  clearTimeout(swipeHintTimer)
  swipeHintTimer = setTimeout(() => {
    showSwipeHint.value = false
    localStorage.setItem(SWIPE_HINT_KEY, '1')
  }, 3200)
}

watch(
  () => [props.active, props.renderPlayer, localVideo.value?.play_urls],
  async ([active, render]) => {
    if (active && render) {
      maybeShowSwipeHint()
      await setupPlayer()
      setupPolling()
      if (active) await Promise.all([loadEngagement(), loadBarrages()])
    } else if (!active) {
      showSwipeHint.value = false
      clearInterval(pollTimer)
      teardownPlayer()
    }
  },
  { immediate: true, deep: true }
)

onMounted(() => {
  document.addEventListener('click', onDocumentClick)
})

onBeforeUnmount(() => {
  document.removeEventListener('click', onDocumentClick)
  clearInterval(pollTimer)
  clearTimeout(toastTimer)
  clearTimeout(swipeHintTimer)
  teardownPlayer()
})

defineExpose({ togglePlay, toggleMute, toggleFullscreen })
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
            :loading="active ? 'eager' : 'lazy'"
            decoding="async"
          />
          <div v-else class="cover placeholder skeleton skeleton--dark" />
        </template>

        <div v-if="showSwipeHint" class="swipe-hint" @click.stop>
          上下滑动 · 切换视频
        </div>

        <div v-if="showPlayHint && paused" class="play-hint" aria-hidden="true">
          <span class="play-icon">▶</span>
        </div>

        <div
          v-if="localVideo.status !== 'ready' && !hasPlayUrls(localVideo.play_urls)"
          class="status-banner"
          @click.stop
        >
          <p v-if="localVideo.status === 'pending_source_review'">
            {{ videoStatusLabel('pending_source_review') }}，等待审核
          </p>
          <p v-else-if="localVideo.status === 'transcoding' || localVideo.status === 'pending'">
            转码中，上滑看下一条
          </p>
          <p v-else-if="localVideo.status === 'failed'">转码失败</p>
          <p v-else>{{ videoStatusLabel(localVideo.status) }}</p>
        </div>

        <div
          v-else-if="localVideo.status === 'pending_final_review'"
          class="status-banner status-banner--info"
          @click.stop
        >
          <p>{{ videoStatusLabel('pending_final_review') }}，可预览，尚未公开发布</p>
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

        <footer class="meta" @click.stop>
          <h2 class="title">{{ localVideo.title }}</h2>
          <p v-if="localVideo.tags?.length" class="tags">
            <span v-for="tag in localVideo.tags" :key="tag" class="tag">#{{ tag }}</span>
          </p>
        </footer>

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

          <button type="button" class="action-btn" :class="{ active: barrageOpen }" @click="openBarrage">
            <span class="icon">弹</span>
            <span class="label">弹幕</span>
          </button>
        </aside>

        <div v-if="barrageOpen" class="barrage-sheet" @click.stop>
          <input
            v-model="barrageText"
            placeholder="输入弹幕..."
            maxlength="100"
            @keyup.enter="sendBarrage"
          />
          <button type="button" class="btn btn-primary btn-sm" @click="sendBarrage">发送</button>
          <button type="button" class="icon-btn icon-btn--ghost" aria-label="关闭弹幕" @click="barrageOpen = false">
            ✕
          </button>
        </div>

        <div
          v-if="renderPlayer && hasPlayUrls(localVideo.play_urls)"
          class="playback-bar"
          @click.stop
        >
          <div v-if="videoDuration > 0" class="progress-track" aria-hidden="true">
            <div class="progress-fill" :style="{ width: `${progressPercent}%` }" />
          </div>
          <div class="playback-bar-inner">
            <div class="playback-tools">
              <div class="ctrl-wrap">
                <button
                  type="button"
                  class="icon-btn"
                  :class="{ highlight: needSoundUnlock || muted }"
                  aria-label="音量"
                  :aria-expanded="showVolumePanel"
                  @click="onVolumeBtnClick"
                >
                  {{ muted || volume === 0 ? '🔇' : '🔊' }}
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
                  class="icon-btn"
                  aria-label="播放倍速"
                  :aria-expanded="showSpeedMenu"
                  @click.stop="showSpeedMenu = !showSpeedMenu; showQualityMenu = false; showVolumePanel = false"
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

              <div v-if="availableQualities.length" class="ctrl-wrap">
                <button
                  type="button"
                  class="icon-btn"
                  aria-label="清晰度"
                  :aria-expanded="showQualityMenu"
                  @click.stop="showQualityMenu = !showQualityMenu; showSpeedMenu = false; showVolumePanel = false"
                >
                  {{ qualityLabel(currentQuality) || 'HD' }}
                </button>
                <div v-if="showQualityMenu" class="ctrl-menu" role="menu">
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
            </div>

            <div class="playback-tools">
              <button
                v-if="showPiP"
                type="button"
                class="icon-btn"
                aria-label="画中画"
                @click="togglePiP"
              >
                ⧉
              </button>
              <button type="button" class="icon-btn" aria-label="全屏" @click="toggleFullscreen">
                ⛶
              </button>
            </div>
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
  background: var(--color-surface);
  overflow: hidden;
}

.stage {
  flex: 1;
  min-width: 0;
  height: 100%;
  transition: flex 0.28s ease;
  cursor: pointer;
}

.comments-open .stage {
  flex: 0 0 72%;
}

.media-wrap {
  position: relative;
  width: calc(100% - 16px);
  height: calc(100% - 16px);
  margin: 8px;
  overflow: hidden;
  border-radius: var(--radius-md);
  background: #0f172a;
  box-shadow:
    inset 0 0 0 1px rgba(255, 255, 255, 0.08),
    0 12px 40px rgba(15, 23, 42, 0.28);
}

.video,
.cover {
  width: 100%;
  height: 100%;
  object-fit: cover;
  object-position: center center;
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

.status-banner--info {
  top: auto;
  bottom: 72px;
  transform: translateX(-50%);
  background: rgba(37, 99, 235, 0.72);
}

.swipe-hint {
  position: absolute;
  top: 16px;
  left: 50%;
  transform: translateX(-50%);
  z-index: 20;
  padding: 8px 16px;
  border-radius: var(--radius-full);
  background: rgba(0, 0, 0, 0.65);
  color: #fff;
  font-size: 13px;
  font-weight: 500;
  pointer-events: none;
  animation: swipe-fade 3.2s ease forwards;
}

@keyframes swipe-fade {
  0%,
  70% {
    opacity: 1;
  }
  100% {
    opacity: 0;
  }
}

.gradient {
  position: absolute;
  left: 0;
  right: 0;
  bottom: 0;
  height: 48%;
  background: linear-gradient(transparent, rgba(0, 0, 0, 0.55));
  pointer-events: none;
}

.progress-track {
  position: absolute;
  left: 0;
  right: 0;
  top: 0;
  height: 3px;
  background: rgba(255, 255, 255, 0.15);
  z-index: 1;
}

.progress-fill {
  height: 100%;
  background: var(--color-primary);
  transition: width 0.12s linear;
}

.playback-bar {
  --playback-bar-height: 52px;
  position: absolute;
  left: 0;
  right: 0;
  bottom: 0;
  z-index: 7;
  padding-bottom: env(safe-area-inset-bottom, 0px);
  background: linear-gradient(transparent, rgba(0, 0, 0, 0.72));
}

.playback-bar-inner {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  min-height: var(--playback-bar-height);
  padding: 8px 12px 10px;
}

.playback-tools {
  display: flex;
  align-items: center;
  gap: 6px;
}

.playback-bar .icon-btn {
  min-width: 36px;
  height: 36px;
  padding: 0 8px;
  border-radius: var(--radius-sm);
  border: 1px solid rgba(255, 255, 255, 0.2);
  background: rgba(255, 255, 255, 0.12);
  color: #fff;
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
  backdrop-filter: blur(6px);
}

.playback-bar .icon-btn.highlight {
  border-color: var(--color-primary);
  background: rgba(37, 99, 235, 0.35);
}

.playback-bar .icon-btn--ghost {
  background: transparent;
  border-color: transparent;
}

.playback-bar .ctrl-wrap {
  position: relative;
}

.playback-bar .ctrl-menu {
  bottom: calc(100% + 8px);
  left: 0;
  background: rgba(15, 23, 42, 0.92);
  border-color: rgba(255, 255, 255, 0.12);
}

.playback-bar .ctrl-menu-item {
  color: #e2e8f0;
}

.playback-bar .ctrl-menu-item:hover,
.playback-bar .ctrl-menu-item.active {
  background: rgba(37, 99, 235, 0.35);
  color: #fff;
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
  -webkit-text-stroke: 0.4px rgba(0, 0, 0, 0.5);
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
  bottom: calc(52px + env(safe-area-inset-bottom, 0px) + 16px);
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 14px;
  z-index: 4;
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
  padding: 6px;
  min-width: 48px;
  min-height: 44px;
  border-radius: var(--radius-md);
  -webkit-tap-highlight-color: transparent;
  transition: transform 0.12s ease, background 0.15s ease;
}

.action-btn:hover {
  background: rgba(255, 255, 255, 0.12);
}

.action-btn:active:not(:disabled) {
  transform: scale(0.94);
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
  right: 72px;
  bottom: calc(52px + env(safe-area-inset-bottom, 0px));
  padding: 12px 14px 10px;
  z-index: 3;
  pointer-events: none;
}

.meta .tag {
  pointer-events: auto;
}

.barrage-sheet {
  position: absolute;
  left: 12px;
  right: 72px;
  bottom: calc(52px + env(safe-area-inset-bottom, 0px) + 8px);
  z-index: 6;
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 10px;
  border-radius: var(--radius-md);
  background: rgba(15, 23, 42, 0.88);
  border: 1px solid rgba(255, 255, 255, 0.12);
  backdrop-filter: blur(8px);
}

.barrage-sheet input {
  flex: 1;
  min-width: 0;
  padding: 8px 12px;
  border-radius: var(--radius-full);
  border: 1px solid rgba(255, 255, 255, 0.2);
  background: rgba(255, 255, 255, 0.1);
  color: #fff;
  font-size: 16px;
}

.barrage-sheet input::placeholder {
  color: rgba(255, 255, 255, 0.55);
}

.barrage-sheet .btn-sm {
  padding: 7px 14px;
  font-size: 13px;
  white-space: nowrap;
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
  margin: 0;
}

.tag {
  font-size: 12px;
  color: #fff;
  background: rgba(37, 99, 235, 0.75);
  padding: 2px 8px;
  border-radius: var(--radius-full);
}

.ctrl-wrap {
  position: relative;
}

.ctrl-menu {
  position: absolute;
  bottom: calc(100% + 6px);
  left: 0;
  min-width: 88px;
  padding: 6px;
  border-radius: var(--radius-md);
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  box-shadow: var(--shadow-float);
  display: flex;
  flex-direction: column;
  gap: 4px;
  transform-origin: bottom left;
  animation: menu-in 0.18s ease;
}

@keyframes menu-in {
  from {
    opacity: 0;
    transform: translateY(6px) scale(0.96);
  }
  to {
    opacity: 1;
    transform: none;
  }
}

.ctrl-menu--volume {
  min-width: 140px;
  padding: 10px;
}

.ctrl-menu-item {
  border: none;
  background: transparent;
  color: var(--color-text-secondary);
  text-align: left;
  padding: 6px 8px;
  border-radius: 6px;
  cursor: pointer;
  font-size: 13px;
}

.ctrl-menu-item:hover,
.ctrl-menu-item.active {
  background: var(--color-primary-soft);
  color: var(--color-primary);
}

.volume-slider {
  width: 100%;
  margin-top: 8px;
  accent-color: #2563eb;
}

.comment-panel {
  flex: 0 0 28%;
  min-width: 0;
  height: 100%;
  display: flex;
  flex-direction: column;
  background: var(--color-surface);
  border-left: 1px solid var(--color-border);
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
  border-bottom: 1px solid var(--color-border);
}

.comment-head h3 {
  margin: 0;
  font-size: 15px;
  color: var(--color-text);
}

.close-btn {
  border: none;
  background: none;
  color: var(--color-text-muted);
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
  color: var(--color-text);
  margin-bottom: 4px;
}

.comment-item p {
  margin: 0;
  font-size: 13px;
  line-height: 1.5;
  color: var(--color-text-secondary);
  word-break: break-word;
}

.comment-empty {
  margin: 24px 0;
  text-align: center;
  color: var(--color-text-muted);
  font-size: 13px;
}

.comment-input {
  display: flex;
  gap: 8px;
  padding: 10px 12px 16px;
  border-top: 1px solid var(--color-border);
}

.comment-input input {
  flex: 1;
  padding: 8px 10px;
  border-radius: var(--radius-sm);
  border: 1px solid var(--color-border-strong);
  background: var(--color-surface);
  color: var(--color-text);
  font-size: 16px;
}

.comment-input button {
  padding: 8px 12px;
  border: none;
  border-radius: var(--radius-sm);
  background: var(--color-primary);
  color: #fff;
  font-size: 13px;
  cursor: pointer;
}

.toast {
  position: absolute;
  left: 50%;
  top: 72px;
  transform: translateX(-50%);
  z-index: var(--z-toast);
  padding: 8px 16px;
  border-radius: var(--radius-full);
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  color: var(--color-text);
  font-size: 13px;
  pointer-events: none;
  white-space: nowrap;
  box-shadow: var(--shadow-float);
}

@media (max-width: 640px) {
  .player-controls {
    flex-wrap: wrap;
    max-width: calc(100vw - 100px);
  }

  .comments-open .stage {
    flex: 0 0 62%;
  }

  .comment-panel {
    flex: 0 0 38%;
  }
}
</style>
