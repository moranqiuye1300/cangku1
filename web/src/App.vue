<script setup>
import { computed } from 'vue'
import { RouterView, RouterLink, useRoute } from 'vue-router'
import AppHeader from './components/AppHeader.vue'
import { useAuthStore } from './stores/auth'

const route = useRoute()
const auth = useAuthStore()

function routeKey(r) {
  if (r.name === 'feed' || r.name === 'discover') {
    return String(r.name)
  }
  return r.fullPath
}

const myProfilePath = computed(() => {
  return auth.isLoggedIn ? `/users/${auth.user?.id}` : '/login'
})
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

    <!-- Mobile bottom navigation -->
    <nav v-if="!$route.meta.hidden" class="bottom-nav" aria-label="底部导航">
      <RouterLink to="/" class="nav-item" :class="{ active: route.path === '/' }">
        <span class="nav-icon">♡</span>
        <span class="nav-label">推荐</span>
      </RouterLink>
      <RouterLink to="/discover" class="nav-item" :class="{ active: route.path === '/discover' }">
        <span class="nav-icon">⌕</span>
        <span class="nav-label">发现</span>
      </RouterLink>
      <RouterLink to="/upload" class="nav-item nav-item--upload">
        <span class="upload-btn">+</span>
      </RouterLink>
      <RouterLink :to="myProfilePath" class="nav-item" :class="{ active: route.path.startsWith('/users') }">
        <span class="nav-icon">◎</span>
        <span class="nav-label">我</span>
      </RouterLink>
    </nav>
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

/* ===== Mobile Bottom Navigation ===== */
.bottom-nav {
  display: none;
  position: fixed;
  bottom: 0;
  left: 0;
  right: 0;
  height: var(--bottom-nav-height);
  background: var(--color-surface);
  border-top: 1px solid var(--color-border);
  z-index: var(--z-bottom-nav);
  padding-bottom: env(safe-area-inset-bottom, 0px);
  justify-content: space-around;
  align-items: center;
}

.nav-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 2px;
  padding: 4px 12px;
  text-decoration: none;
  color: var(--color-text-muted);
  transition: color 0.15s ease;
  min-width: 56px;
}

.nav-item.active {
  color: var(--color-primary);
}

.nav-icon {
  font-size: 22px;
  line-height: 1;
}

.nav-label {
  font-size: 10px;
  font-weight: 500;
}

.nav-item--upload {
  position: relative;
  top: -8px;
}

.upload-btn {
  width: 42px;
  height: 42px;
  display: grid;
  place-items: center;
  border-radius: 50%;
  background: var(--color-primary-gradient);
  color: #fff;
  font-size: 24px;
  font-weight: 300;
  box-shadow: 0 4px 16px rgba(254, 44, 85, 0.4);
}

@media (max-width: 768px) {
  .bottom-nav {
    display: flex;
  }
}
</style>
