<script setup>
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '../stores/auth'

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
  <div class="page-center">
    <form class="card card-body-lg form" @submit.prevent="handleSubmit">
      <h1 class="page-title">登录</h1>
      <p class="page-desc">测试账号：alice / 123456</p>

      <label class="field">
        用户名
        <input v-model="form.username" type="text" required />
      </label>
      <label class="field">
        密码
        <input v-model="form.password" type="password" required />
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
      </p>
    </form>
  </div>
</template>

<style scoped>
.form {
  width: min(420px, 100%);
}

.foot {
  margin: 16px 0 0;
  text-align: center;
  font-size: 14px;
}

.btn-block + .btn-block {
  margin-top: 10px;
}
</style>
