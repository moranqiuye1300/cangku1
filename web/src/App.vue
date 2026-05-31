<script setup>
import { computed } from 'vue'
import { RouterView, useRoute } from 'vue-router'
import AppHeader from './components/AppHeader.vue'

const route = useRoute()
const showHeader = computed(() => !route.meta.hidden && !route.meta.immersive)
</script>

<template>
  <div class="app-shell">
    <AppHeader v-if="showHeader" />
    <main class="app-main" :class="{ 'app-main--full': route.meta.hidden || route.meta.immersive }">
      <RouterView />
    </main>
  </div>
</template>

<style scoped>
.app-shell {
  min-height: 100vh;
  background: var(--color-bg);
}

.app-main {
  min-height: calc(100vh - var(--header-height));
}

.app-main--full {
  min-height: 100vh;
}
</style>
