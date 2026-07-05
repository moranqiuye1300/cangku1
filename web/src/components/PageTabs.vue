<script setup>
const model = defineModel({ type: String, required: true })

defineProps({
  tabs: {
    type: Array,
    required: true
  }
})

function select(key) {
  model.value = key
}
</script>

<template>
  <nav class="page-tabs">
    <button
      v-for="tab in tabs"
      :key="tab.key"
      type="button"
      class="page-tab"
      :class="{ active: model === tab.key }"
      @click="select(tab.key)"
    >
      {{ tab.label }}
    </button>
  </nav>
</template>

<style scoped>
.page-tabs {
  display: flex;
  gap: 0;
  position: relative;
  overflow-x: auto;
  -webkit-overflow-scrolling: touch;
  scrollbar-width: none;
}

.page-tabs::-webkit-scrollbar {
  display: none;
}

.page-tab {
  position: relative;
  padding: 12px 20px;
  border: none;
  background: none;
  color: var(--color-text-muted);
  font-size: 15px;
  font-weight: 500;
  cursor: pointer;
  white-space: nowrap;
  transition: color 0.15s ease;
  -webkit-tap-highlight-color: transparent;
}

.page-tab:hover {
  color: var(--color-text-secondary);
}

.page-tab.active {
  color: var(--color-text);
  font-weight: 600;
}

.page-tab.active::after {
  content: '';
  position: absolute;
  bottom: 0;
  left: 50%;
  transform: translateX(-50%);
  width: 24px;
  height: 3px;
  border-radius: 2px;
  background: var(--color-primary-gradient);
}
</style>
