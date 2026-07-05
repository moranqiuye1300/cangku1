<script setup>
defineProps({
  type: { type: String, default: 'empty' },
  message: { type: String, default: '' },
  retry: { type: Boolean, default: false },
  dark: { type: Boolean, default: false },
  showIllustration: { type: Boolean, default: true }
})
defineEmits(['retry'])
</script>

<template>
  <div class="ui-state" :class="[type, { 'ui-state--dark': dark }]">
    <div v-if="type === 'loading'" class="ui-spinner" aria-hidden="true" />
    <div v-else-if="showIllustration && type === 'empty'" class="empty-illustration" aria-hidden="true" />
    <p class="ui-state-msg">{{ message }}</p>
    <slot />
    <button v-if="retry" type="button" class="btn btn-primary ui-state-retry" @click="$emit('retry')">
      重试
    </button>
  </div>
</template>

<style scoped>
.ui-state {
  display: grid;
  place-content: center;
  gap: 14px;
  text-align: center;
  padding: 32px 24px;
}

.ui-state--dark {
  color: rgba(255, 255, 255, 0.85);
}

.ui-state-msg {
  margin: 0;
  font-size: 14px;
  max-width: 36ch;
  justify-self: center;
}

.ui-spinner {
  width: 32px;
  height: 32px;
  margin: 0 auto;
  border: 3px solid rgba(254, 44, 85, 0.2);
  border-top-color: var(--color-primary);
  border-radius: 50%;
  animation: ui-spin 0.7s linear infinite;
}

.ui-state--dark .ui-spinner {
  border-color: rgba(255, 255, 255, 0.15);
  border-top-color: #fff;
}

.ui-state-retry {
  justify-self: center;
}

@keyframes ui-spin {
  to {
    transform: rotate(360deg);
  }
}
</style>
