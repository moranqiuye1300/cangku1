<script setup>
import { computed } from 'vue'
import { RouterLink } from 'vue-router'
import { formatDuration } from '../utils/format'
import { videoStatusLabel } from '../utils/videoStatus'

const props = defineProps({
  video: {
    type: Object,
    required: true
  },
  showStatus: {
    type: Boolean,
    default: false
  }
})

const statusLabel = computed(() => videoStatusLabel(props.video.status))
const linkTo = computed(() => ({ path: '/', query: { v: props.video.id } }))

const placeholderInitial = computed(() => {
  const t = props.video.title || '?'
  return t.trim()[0] || '?'
})

const statusBadgeClass = computed(() => {
  if (props.video.status === 'ready') return 'status-badge--ready'
  if (props.video.status === 'failed') return 'status-badge--failed'
  return 'status-badge--pending'
})
</script>

<template>
  <RouterLink :to="linkTo" class="video-card">
    <div class="cover-wrap">
      <img
        v-if="video.cover_url"
        :src="video.cover_url"
        :alt="video.title"
        class="cover"
        loading="lazy"
        decoding="async"
      />
      <div v-else class="cover placeholder">
        <span class="placeholder-letter">{{ placeholderInitial }}</span>
      </div>
      <div class="cover-overlay" />
      <div class="play-icon-overlay" aria-hidden="true">
        <span class="play-triangle">▶</span>
      </div>
      <span
        v-if="showStatus && video.status"
        class="status-badge"
        :class="statusBadgeClass"
      >
        {{ statusLabel }}
      </span>
      <span v-if="video.duration" class="duration">{{ formatDuration(video.duration) }}</span>
      <div class="card-title-overlay">
        <h3 class="title-text">{{ video.title }}</h3>
      </div>
    </div>
  </RouterLink>
</template>

<style scoped>
.video-card {
  display: block;
  text-decoration: none;
  color: inherit;
  overflow: hidden;
  border-radius: var(--radius-md);
  transition: transform 0.15s ease;
}

.video-card:hover {
  transform: scale(1.02);
}

.video-card:active {
  transform: scale(0.98);
}

.cover-wrap {
  position: relative;
  aspect-ratio: 9 / 16;
  background: var(--color-surface-elevated);
  overflow: hidden;
  border-radius: var(--radius-md);
}

.cover {
  width: 100%;
  height: 100%;
  object-fit: cover;
  transition: transform 0.25s ease;
}

.video-card:hover .cover {
  transform: scale(1.05);
}

.placeholder {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 100%;
  height: 100%;
  background: linear-gradient(135deg, #1a1a2e, #16213e);
}

.placeholder-letter {
  font-size: 36px;
  font-weight: 700;
  color: rgba(255, 255, 255, 0.4);
}

.cover-overlay {
  position: absolute;
  inset: 0;
  background: linear-gradient(transparent 60%, rgba(0, 0, 0, 0.7));
  pointer-events: none;
}

.play-icon-overlay {
  position: absolute;
  inset: 0;
  display: grid;
  place-items: center;
  opacity: 0;
  transition: opacity 0.2s ease;
  background: rgba(0, 0, 0, 0.15);
}

.video-card:hover .play-icon-overlay {
  opacity: 1;
}

.play-triangle {
  width: 44px;
  height: 44px;
  display: grid;
  place-items: center;
  border-radius: 50%;
  background: rgba(0, 0, 0, 0.55);
  color: #fff;
  font-size: 18px;
  padding-left: 3px;
}

.status-badge {
  position: absolute;
  left: 6px;
  top: 6px;
  padding: 2px 8px;
  border-radius: var(--radius-sm);
  font-size: 11px;
  font-weight: 500;
  color: #fff;
  z-index: 1;
}

.status-badge--ready {
  background: rgba(37, 244, 238, 0.85);
}

.status-badge--failed {
  background: rgba(254, 44, 85, 0.85);
}

.status-badge--pending {
  background: rgba(255, 193, 7, 0.85);
}

.duration {
  position: absolute;
  right: 6px;
  bottom: 34px;
  padding: 1px 6px;
  border-radius: 4px;
  background: rgba(0, 0, 0, 0.65);
  color: #fff;
  font-size: 11px;
  z-index: 1;
}

.card-title-overlay {
  position: absolute;
  left: 8px;
  right: 8px;
  bottom: 6px;
  z-index: 1;
}

.title-text {
  margin: 0;
  font-size: 13px;
  font-weight: 600;
  line-height: 1.3;
  display: -webkit-box;
  -webkit-line-clamp: 1;
  -webkit-box-orient: vertical;
  overflow: hidden;
  color: #fff;
  text-shadow: 0 1px 3px rgba(0, 0, 0, 0.6);
}
</style>
