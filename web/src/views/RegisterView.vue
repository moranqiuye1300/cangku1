<script setup>
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '../stores/auth'

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
  <div class="page-center">
    <form class="card card-body-lg form" @submit.prevent="handleSubmit">
      <h1 class="page-title">注册</h1>
      <p class="page-desc">创建账号后即可登录平台</p>

      <label class="field">
        用户名
        <input v-model="form.username" type="text" minlength="3" required />
      </label>
      <label class="field">
        昵称
        <input v-model="form.nickname" type="text" />
      </label>
      <label class="field">
        密码
        <input v-model="form.password" type="password" minlength="6" required />
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
</style>
