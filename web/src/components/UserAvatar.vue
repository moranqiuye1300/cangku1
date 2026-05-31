<script setup>
import { computed } from 'vue'

const props = defineProps({
  user: { type: Object, default: null },
  userId: { type: String, default: '' },
  size: { type: Number, default: 64 }
})

const label = computed(() => {
  const name = props.user?.nickname || props.user?.username || props.userId || 'U'
  return name[0]?.toUpperCase() || 'U'
})

const avatarUrl = computed(() => props.user?.avatar || '')
</script>

<template>
  <div class="user-avatar" :style="{ width: `${size}px`, height: `${size}px` }">
    <img v-if="avatarUrl" :src="avatarUrl" :alt="label" class="avatar-img" />
    <span v-else class="avatar-fallback" :style="{ fontSize: `${Math.round(size * 0.38)}px` }">{{ label }}</span>
  </div>
</template>

<style scoped>
.user-avatar {
  border-radius: 50%;
  overflow: hidden;
  background: #334155;
  display: grid;
  place-items: center;
  flex-shrink: 0;
}

.avatar-img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  display: block;
}

.avatar-fallback {
  color: #fff;
  font-weight: 700;
}
</style>
