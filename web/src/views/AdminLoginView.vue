<script setup>
import { ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '../stores/auth'

const router = useRouter()
const route = useRoute()
const auth = useAuthStore()

const form = ref({ username: '', password: '' })
const loading = ref(false)
const error = ref('')

async function handleSubmit() {
  loading.value = true
  error.value = ''
  try {
    await auth.loginAdmin(form.value)
    const redirect = route.query.redirect
    if (typeof redirect === 'string' && redirect.startsWith('/console')) {
      router.push(redirect)
      return
    }
    if (auth.isAdmin) {
      router.push('/console')
    } else {
      router.push('/console/review')
    }
  } catch (e) {
    error.value = e.message
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="admin-login-page">
    <form class="admin-login-form" @submit.prevent="handleSubmit">
      <div class="admin-brand">
        <span class="admin-brand-mark">SV</span>
        <h1>管理后台</h1>
        <p>仅限管理员与审核员访问</p>
      </div>

      <div class="admin-form-body">
        <label class="field">
          用户名
          <input v-model="form.username" type="text" required autocomplete="username" />
        </label>
        <label class="field">
          密码
          <input v-model="form.password" type="password" required autocomplete="current-password" />
        </label>

        <p v-if="error" class="text-error">{{ error }}</p>

        <button type="submit" class="btn btn-primary btn-block" :disabled="loading">
          {{ loading ? '验证中...' : '进入系统' }}
        </button>
      </div>
    </form>
  </div>
</template>

<style scoped>
.admin-login-page {
  min-height: 100vh;
  display: grid;
  place-items: center;
  padding: 24px;
  background: linear-gradient(145deg, #0f0f0f 0%, #0a0f1a 50%, #1a0a0f 100%);
}

.admin-login-form {
  width: min(400px, 100%);
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  overflow: hidden;
}

.admin-brand {
  text-align: center;
  padding: 32px 24px 20px;
}

.admin-brand-mark {
  width: 48px;
  height: 48px;
  margin: 0 auto 12px;
  display: grid;
  place-items: center;
  border-radius: var(--radius-md);
  background: var(--color-primary-gradient);
  color: #fff;
  font-weight: 700;
  font-size: 16px;
  box-shadow: 0 4px 16px rgba(254, 44, 85, 0.3);
}

.admin-brand h1 {
  margin: 0 0 6px;
  font-size: 22px;
  font-weight: 700;
  color: var(--color-text);
}

.admin-brand p {
  margin: 0;
  color: var(--color-text-muted);
  font-size: 14px;
}

.admin-form-body {
  padding: 0 24px 24px;
}
</style>
