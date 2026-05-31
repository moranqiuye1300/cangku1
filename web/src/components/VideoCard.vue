<script setup>
import { RouterLink } from 'vue-router'
import { formatDuration } from '../utils/format'

defineProps({
  video: {
    type: Object,
    required: true
  }
})
</script>

<template>
  <RouterLink :to="{ path: '/', query: { v: video.id } }" class="card video-card">
    <div class="cover-wrap">
      <img :src="video.cover_url" :alt="video.title" class="cover" loading="lazy" />
      <span class="duration">{{ formatDuration(video.duration) }}</span>
    </div>
    <div class="info">
      <h3>{{ video.title }}</h3>
      <p>{{ video.description }}</p>
      <div class="meta">
        <span>@{{ video.user_id }}</span>
        <span class="status">{{ video.status }}</span>
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
  transition: border-color 0.15s ease, box-shadow 0.15s ease;
}

.video-card:hover {
  border-color: var(--color-primary);
  box-shadow: 0 4px 12px rgba(37, 99, 235, 0.08);
}

.cover-wrap {
  position: relative;
  aspect-ratio: 16 / 9;
  background: #eef2f7;
  border-bottom: 1px solid var(--color-border);
}

.cover {
  width: 100%;
  height: 100%;
  object-fit: cover;
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
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.meta {
  display: flex;
  justify-content: space-between;
  font-size: 12px;
  color: var(--color-text-muted);
}

.status {
  text-transform: uppercase;
  color: var(--color-primary);
}
</style>
