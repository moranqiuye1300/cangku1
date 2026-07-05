<script setup>
defineProps({
  empty: { type: Boolean, default: false },
  emptyMessage: { type: String, default: '暂无内容' }
})
</script>

<template>
  <div v-if="empty" class="empty-state">{{ emptyMessage }}</div>
  <div v-else class="video-grid">
    <slot />
  </div>
</template>

<style scoped>
.video-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(180px, 1fr));
  gap: 6px;
}

.empty-state {
  text-align: center;
  padding: 32px 16px;
  color: var(--color-text-muted);
  font-size: 14px;
}

@media (min-width: 768px) {
  .video-grid {
    grid-template-columns: repeat(auto-fill, minmax(220px, 1fr));
    gap: 8px;
  }
}
</style>
