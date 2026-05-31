<script setup>
import { RouterLink } from 'vue-router'
import { useAuthStore } from '../stores/auth'
import { useRouter } from 'vue-router'

const auth = useAuthStore()
const router = useRouter()

function handleLogout() {
  auth.logout()
  router.push('/login')
}
</script>

<template>
  <header class="header">
    <div class="header-inner">
      <RouterLink to="/" class="brand">
        <span class="brand-mark">SV</span>
        <span class="brand-text">短视频</span>
      </RouterLink>
      <nav class="nav">
        <RouterLink to="/">推荐</RouterLink>
        <RouterLink to="/discover">发现</RouterLink>
        <RouterLink v-if="auth.isLoggedIn" to="/upload">上传</RouterLink>
        <RouterLink v-if="auth.user" :to="`/users/${auth.user.id}`">我的</RouterLink>
      </nav>
      <div class="actions">
        <RouterLink v-if="!auth.isLoggedIn" to="/login" class="btn btn-ghost btn-sm">登录</RouterLink>
        <RouterLink v-if="!auth.isLoggedIn" to="/register" class="btn btn-primary btn-sm">注册</RouterLink>
        <button v-else type="button" class="btn btn-ghost btn-sm" @click="handleLogout">退出</button>
      </div>
    </div>
  </header>
</template>

<style scoped>
.header {
  position: sticky;
  top: 0;
  z-index: 100;
  height: var(--header-height);
  background: var(--color-surface);
  border-bottom: 1px solid var(--color-border);
}

.header-inner {
  max-width: var(--page-max);
  height: 100%;
  margin: 0 auto;
  padding: 0 24px;
  display: flex;
  align-items: center;
  gap: 24px;
}

.brand {
  display: flex;
  align-items: center;
  gap: 8px;
  text-decoration: none;
  color: var(--color-text);
  flex-shrink: 0;
}

.brand-mark {
  width: 32px;
  height: 32px;
  display: grid;
  place-items: center;
  border-radius: var(--radius-sm);
  background: var(--color-primary);
  color: #fff;
  font-size: 12px;
  font-weight: 700;
}

.brand-text {
  font-size: 16px;
  font-weight: 600;
  letter-spacing: -0.02em;
}

.nav {
  display: flex;
  align-items: center;
  gap: 4px;
  flex: 1;
}

.nav a {
  padding: 8px 14px;
  border-radius: var(--radius-sm);
  color: var(--color-text-secondary);
  text-decoration: none;
  font-size: 14px;
  font-weight: 500;
}

.nav a:hover {
  color: var(--color-primary);
  background: var(--color-primary-soft);
}

.nav a.router-link-active {
  color: var(--color-primary);
  background: var(--color-primary-soft);
}

.actions {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-shrink: 0;
}

.btn-sm {
  padding: 7px 14px;
  font-size: 13px;
}
</style>
