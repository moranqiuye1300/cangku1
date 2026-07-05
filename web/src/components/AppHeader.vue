<script setup>
import { ref, onMounted, onBeforeUnmount } from 'vue'
import { RouterLink, useRouter } from 'vue-router'
import { useAuthStore } from '../stores/auth'
import UserAvatar from './UserAvatar.vue'

const auth = useAuthStore()
const router = useRouter()
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
          <span class="brand-slogan">发现 · 创作 · 分享</span>
        </span>
      </RouterLink>
      <nav class="nav" aria-label="主导航">
        <RouterLink to="/">推荐</RouterLink>
        <RouterLink to="/discover">发现</RouterLink>
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
            <RouterLink v-if="auth.isAdmin" to="/console" @click="menuOpen = false">管理后台</RouterLink>
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
  background: rgba(255, 255, 255, 0.92);
  border-bottom: 1px solid var(--color-border);
  backdrop-filter: blur(10px);
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
  background: linear-gradient(135deg, #3b82f6, #2563eb);
  color: #fff;
  font-size: 12px;
  font-weight: 700;
}

.brand-copy {
  display: flex;
  flex-direction: column;
  gap: 1px;
}

.brand-text {
  font-size: 15px;
  font-weight: 600;
  letter-spacing: -0.02em;
  line-height: 1.2;
}

.brand-slogan {
  font-size: 11px;
  color: var(--color-text-muted);
  font-weight: 400;
}

.nav {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  flex: 1;
}

.nav a {
  padding: 8px 14px;
  border: 1px solid transparent;
  border-radius: var(--radius-full);
  color: var(--color-text-secondary);
  text-decoration: none;
  font-size: var(--text-base);
  font-weight: 500;
  transition: background 0.15s ease, border-color 0.15s ease, color 0.15s ease;
}

.nav a:hover {
  border-color: var(--color-border-strong);
  color: var(--color-primary);
}

.nav a.router-link-active {
  background: var(--color-primary-soft);
  border-color: var(--color-primary);
  color: var(--color-primary);
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
  background: var(--color-surface);
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

@media (max-width: 640px) {
  .brand-slogan,
  .user-menu-name {
    display: none;
  }
}
</style>
