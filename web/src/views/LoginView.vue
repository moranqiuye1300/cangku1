<script setup>
import { ref } from 'vue'
import { useRouter, RouterLink } from 'vue-router'
import { useAuthStore } from '../stores/auth'
import AuthLayout from '../components/AuthLayout.vue'

const router = useRouter()
const auth = useAuthStore()

const form = ref({
  username: 'alice',
  password: '123456'
})
const loading = ref(false)
const error = ref('')

async function oauthLogin() {
  loading.value = true
  error.value = ''
  try {
    await auth.loginOAuthMock('demo_user')
    router.push('/')
  } catch (e) {
    error.value = e.message
  } finally {
    loading.value = false
  }
}

async function handleSubmit() {
  loading.value = true
  error.value = ''
  try {
    await auth.login(form.value)
    router.push('/')
  } catch (e) {
    error.value = e.message
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <AuthLayout title="登录" subtitle="使用账号密码或 Mock OAuth 进入平台">
    <form class="auth-form" @submit.prevent="handleSubmit">
      <label class="field">
        用户名
        <input v-model="form.username" type="text" placeholder="alice" required autocomplete="username" />
      </label>
      <label class="field">
        密码
        <input
          v-model="form.password"
          type="password"
          placeholder="123456"
          required
          autocomplete="current-password"
        />
      </label>

      <p v-if="error" class="text-error">{{ error }}</p>

      <button type="submit" class="btn btn-primary btn-block" :disabled="loading">
        {{ loading ? '登录中...' : '登录' }}
      </button>
      <button type="button" class="btn btn-ghost btn-block" :disabled="loading" @click="oauthLogin">
        GitHub OAuth（Mock）
      </button>
      <p class="foot text-muted">
        还没有账号？
        <RouterLink to="/register">去注册</RouterLink>
        · 测试账号 alice / 123456
      </p>
    </form>
  </AuthLayout>
</template>

<style scoped>
.auth-form {
  margin: 0;
}

.foot {
  margin: 16px 0 0;
  text-align: center;
  font-size: 14px;
}

.foot a {
  color: var(--color-primary);
  font-weight: 500;
}

.btn-block + .btn-block {
  margin-top: 10px;
}
</style>
