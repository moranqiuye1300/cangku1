<script setup>
import { ref, onMounted, onBeforeUnmount } from 'vue'
import { RouterLink, useRouter, useRoute } from 'vue-router'
import { useAuthStore } from '../stores/auth'
import UserAvatar from './UserAvatar.vue'

const auth = useAuthStore()
const router = useRouter()
const route = useRoute()
const menuOpen = ref(false)

function handleLogout() {
  menuOpen.value = false
  auth.logout()
  router.push('/login')
}

function onDocClick(e) {
  if (!e.target.closest?.('.user-menu')) {
    menuOpen.value = false
  }
}

onMounted(() => document.addEventListener('click', onDocClick))
onBeforeUnmount(() => document.removeEventListener('click', onDocClick))
</script>

<template>
  <header class="header">
    <div class="header-inner">
      <RouterLink to="/" class="brand">
        <span class="brand-mark">SV</span>
        <span class="brand-copy">
          <span class="brand-text">短视频</span>
        </span>
      </RouterLink>
      <nav class="nav" aria-label="主导航">
        <RouterLink to="/" :class="{ active: route.path === '/' }">推荐</RouterLink>
        <RouterLink to="/discover" :class="{ active: route.path === '/discover' }">发现</RouterLink>
        <RouterLink v-if="auth.isLoggedIn" to="/upload">上传</RouterLink>
      </nav>
      <div class="actions">
        <RouterLink
          v-if="auth.isReviewer"
          to="/console/review"
          class="btn btn-ghost btn-sm console-link"
        >
          审核台
        </RouterLink>
        <RouterLink
          v-if="auth.isAdmin"
          to="/console"
          class="btn btn-ghost btn-sm console-link"
        >
          管理
        </RouterLink>
        <template v-if="!auth.isLoggedIn">
          <RouterLink to="/login" class="btn btn-ghost btn-sm">登录</RouterLink>
          <RouterLink to="/register" class="btn btn-primary btn-sm">注册</RouterLink>
        </template>
        <div v-else class="user-menu">
          <button
            type="button"
            class="user-menu-trigger"
            aria-haspopup="true"
            :aria-expanded="menuOpen"
            @click.stop="menuOpen = !menuOpen"
          >
            <UserAvatar :user="auth.user" :size="32" />
            <span class="user-menu-name">{{ auth.user?.nickname || auth.user?.username }}</span>
          </button>
          <div v-if="menuOpen" class="user-menu-panel">
            <RouterLink :to="`/users/${auth.user.id}`" @click="menuOpen = false">我的主页</RouterLink>
            <button type="button" @click="handleLogout">退出登录</button>
          </div>
        </div>
      </div>
    </div>
  </header>
</template>

<style scoped>
.header {
  position: sticky;
  top: 0;
  z-index: var(--z-header);
  height: var(--header-height);
  background: rgba(15, 15, 15, 0.88);
  border-bottom: 1px solid var(--color-border);
  backdrop-filter: blur(12px);
  -webkit-backdrop-filter: blur(12px);
}

.header-inner {
  max-width: var(--page-max);
  height: 100%;
  margin: 0 auto;
  padding: 0 24px;
  display: flex;
  align-items: center;
  gap: 20px;
}

.brand {
  display: flex;
  align-items: center;
  gap: 10px;
  text-decoration: none;
  color: var(--color-text);
  flex-shrink: 0;
}

.brand-mark {
  width: 34px;
  height: 34px;
  display: grid;
  place-items: center;
  border-radius: var(--radius-sm);
  background: var(--color-primary-gradient);
  color: #fff;
  font-size: 12px;
  font-weight: 700;
}

.brand-copy {
  display: flex;
  flex-direction: column;
}

.brand-text {
  font-size: 16px;
  font-weight: 700;
  letter-spacing: -0.01em;
  line-height: 1.2;
  background: var(--color-primary-gradient);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
}

.nav {
  display: flex;
  align-items: center;
  gap: 4px;
  flex: 1;
}

.nav a {
  padding: 8px 16px;
  border-radius: var(--radius-full);
  color: var(--color-text-secondary);
  text-decoration: none;
  font-size: 15px;
  font-weight: 500;
  transition: background 0.15s ease, color 0.15s ease;
}

.nav a:hover {
  background: var(--glass-bg);
  color: var(--color-text);
}

.nav a.active {
  color: var(--color-text);
  font-weight: 600;
}

.nav a.active::after {
  content: '';
  display: block;
  height: 3px;
  width: 20px;
  margin: 2px auto 0;
  border-radius: 2px;
  background: var(--color-primary-gradient);
}

.actions {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-shrink: 0;
}

.console-link {
  display: none;
}

@media (min-width: 720px) {
  .console-link {
    display: inline-flex;
  }
}

.user-menu {
  position: relative;
}

.user-menu-trigger {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 4px 10px 4px 4px;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-full);
  background: var(--color-surface);
  cursor: pointer;
  font: inherit;
  color: var(--color-text);
  transition: border-color 0.15s ease;
}

.user-menu-trigger:hover {
  border-color: var(--color-primary);
}

.user-menu-name {
  max-width: 88px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 13px;
  font-weight: 500;
}

.user-menu-panel {
  position: absolute;
  right: 0;
  top: calc(100% + 8px);
  min-width: 160px;
  padding: 6px;
  background: var(--color-surface-elevated);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  box-shadow: var(--shadow-float);
  z-index: calc(var(--z-header) + 1);
}

.user-menu-panel a,
.user-menu-panel button {
  display: block;
  width: 100%;
  padding: 10px 12px;
  border: none;
  border-radius: var(--radius-sm);
  background: none;
  color: var(--color-text);
  text-align: left;
  text-decoration: none;
  font: inherit;
  font-size: 14px;
  cursor: pointer;
}

.user-menu-panel a:hover,
.user-menu-panel button:hover {
  background: var(--color-primary-soft);
  color: var(--color-primary);
}

/* Hide header on mobile */
@media (max-width: 768px) {
  .header {
    display: none;
  }
}
</style>
