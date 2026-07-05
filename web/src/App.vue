<script setup>
import { RouterView } from 'vue-router'
import AppHeader from './components/AppHeader.vue'

function routeKey(r) {
  if (r.name === 'feed' || r.name === 'discover') {
    return String(r.name)
  }
  return r.fullPath
}
</script>

<template>
  <div class="app-shell">
    <AppHeader v-if="!$route.meta.hidden" />
    <main class="app-main" :class="{ 'app-main--full': $route.meta.hidden }">
      <RouterView v-slot="{ Component, route: r }">
        <KeepAlive include="FeedView,HomeView">
          <component :is="Component" :key="routeKey(r)" />
        </KeepAlive>
      </RouterView>
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
