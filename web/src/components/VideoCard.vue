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
  <RouterLink :to="linkTo" class="card video-card">
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
      <span class="play-overlay" aria-hidden="true">▶</span>
      <span
        v-if="showStatus && video.status"
        class="status-badge"
        :class="statusBadgeClass"
      >
        {{ statusLabel }}
      </span>
      <span v-if="video.duration" class="duration">{{ formatDuration(video.duration) }}</span>
    </div>
    <div class="info">
      <h3 class="line-clamp-2">{{ video.title }}</h3>
      <p v-if="video.description" class="line-clamp-2">{{ video.description }}</p>
      <div class="meta">
        <span>@{{ video.user_id }}</span>
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
  transition: border-color 0.15s ease, box-shadow 0.15s ease, transform 0.12s ease;
}

.video-card:hover {
  border-color: var(--color-primary);
  box-shadow: var(--shadow-md);
  transform: translateY(-2px);
}

.video-card:active {
  transform: translateY(0);
}

.cover-wrap {
  position: relative;
  aspect-ratio: 16 / 9;
  background: linear-gradient(145deg, #e8eef5, #dbeafe);
  border-bottom: 1px solid var(--color-border);
  overflow: hidden;
}

.cover {
  width: 100%;
  height: 100%;
  object-fit: cover;
  transition: transform 0.25s ease;
}

.video-card:hover .cover {
  transform: scale(1.03);
}

.placeholder {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 100%;
  height: 100%;
  background: linear-gradient(135deg, #c7d2fe, #93c5fd);
}

.placeholder-letter {
  font-size: 36px;
  font-weight: 700;
  color: rgba(255, 255, 255, 0.9);
  text-shadow: 0 1px 2px rgba(37, 99, 235, 0.3);
}

.play-overlay {
  position: absolute;
  inset: 0;
  display: grid;
  place-items: center;
  font-size: 28px;
  color: #fff;
  background: rgba(26, 35, 50, 0.25);
  opacity: 0;
  transition: opacity 0.2s ease;
}

.video-card:hover .play-overlay {
  opacity: 1;
}

.status-badge {
  position: absolute;
  left: 8px;
  top: 8px;
  padding: 3px 8px;
  border-radius: var(--radius-sm);
  font-size: 11px;
  font-weight: 500;
  color: #fff;
  z-index: 1;
}

.status-badge--ready {
  background: rgba(5, 150, 105, 0.88);
}

.status-badge--failed {
  background: rgba(220, 38, 38, 0.88);
}

.status-badge--pending {
  background: rgba(180, 83, 9, 0.88);
}

.duration {
  position: absolute;
  right: 8px;
  bottom: 8px;
  padding: 2px 8px;
  border-radius: var(--radius-sm);
  background: rgba(26, 35, 50, 0.72);
  color: #fff;
  font-size: 12px;
  z-index: 1;
}

.info {
  padding: 14px 16px;
}

h3 {
  margin: 0 0 6px;
  font-size: 15px;
  font-weight: 600;
  line-height: 1.4;
  color: var(--color-text);
}

p {
  margin: 0 0 10px;
  color: var(--color-text-secondary);
  font-size: 13px;
  line-height: 1.5;
}

.meta {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 8px;
  font-size: 12px;
  color: var(--color-text-muted);
}
</style>
