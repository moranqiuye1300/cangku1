<script setup>
import { ref } from 'vue'
import { useRouter, RouterLink } from 'vue-router'
import { useAuthStore } from '../stores/auth'
import AuthLayout from '../components/AuthLayout.vue'

const router = useRouter()
const auth = useAuthStore()

const form = ref({
  username: '',
  password: '',
  nickname: ''
})
const loading = ref(false)
const error = ref('')
const success = ref('')

async function handleSubmit() {
  loading.value = true
  error.value = ''
  success.value = ''
  try {
    await auth.register(form.value)
    success.value = '注册成功，请登录'
    setTimeout(() => router.push('/login'), 800)
  } catch (e) {
    error.value = e.message
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <AuthLayout title="注册" subtitle="创建账号后即可上传与互动">
    <form class="auth-form" @submit.prevent="handleSubmit">
      <label class="field">
        用户名
        <input v-model="form.username" type="text" minlength="3" required autocomplete="username" />
      </label>
      <label class="field">
        昵称
        <input v-model="form.nickname" type="text" autocomplete="nickname" />
      </label>
      <label class="field">
        密码
        <input
          v-model="form.password"
          type="password"
          minlength="6"
          required
          autocomplete="new-password"
        />
      </label>

      <p v-if="error" class="text-error">{{ error }}</p>
      <p v-if="success" class="text-success">{{ success }}</p>

      <button type="submit" class="btn btn-primary btn-block" :disabled="loading">
        {{ loading ? '提交中...' : '注册' }}
      </button>
      <p class="foot text-muted">
        已有账号？
        <RouterLink to="/login">去登录</RouterLink>
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
</style>
