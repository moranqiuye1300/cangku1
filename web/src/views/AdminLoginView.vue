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
  <div class="page-center console-auth">
    <form class="card card-body-lg form" @submit.prevent="handleSubmit">
      <h1 class="page-title">内部管理入口</h1>
      <p class="page-desc">仅限管理员与审核员访问</p>

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
    </form>
  </div>
</template>

<style scoped>
.console-auth {
  background: var(--color-bg);
}

.form {
  width: min(400px, 100%);
}
</style>
