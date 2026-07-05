<script setup>
defineProps({
  empty: { type: Boolean, default: false },
  emptyMessage: { type: String, default: '暂无内容' }
})
</script>

<template>
  <div v-if="empty" class="card card-body text-muted">{{ emptyMessage }}</div>
  <div v-else class="video-grid">
    <slot />
  </div>
</template>
