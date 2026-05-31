<script setup>
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { uploadVideo } from '../api/video'

const router = useRouter()
const title = ref('')
const description = ref('')
const file = ref(null)
const loading = ref(false)
const error = ref('')
const message = ref('')

function onFileChange(e) {
  file.value = e.target.files?.[0] || null
}

async function handleSubmit() {
  if (!file.value) {
    error.value = '请选择视频文件'
    return
  }
  loading.value = true
  error.value = ''
  message.value = ''
  try {
    const form = new FormData()
    form.append('title', title.value)
    form.append('description', description.value)
    form.append('file', file.value)
    const res = await uploadVideo(form)
    message.value = `上传成功，视频 ID：${res.data.video.id}，状态：${res.data.video.status}`
    setTimeout(() => router.push({ path: '/', query: { v: res.data.video.id } }), 800)
  } catch (e) {
    error.value = e.message
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="page-center">
    <form class="card card-body-lg upload-form" @submit.prevent="handleSubmit">
      <h1 class="page-title">上传视频</h1>
      <p class="page-desc">上传后将进入转码队列，自动生成 HLS 并打标签</p>

      <label class="field">
        标题
        <input v-model="title" required maxlength="80" />
      </label>
      <label class="field">
        简介
        <textarea v-model="description" rows="3" maxlength="300" />
      </label>
      <label class="field">
        视频文件
        <input type="file" accept="video/*" required @change="onFileChange" />
      </label>

      <p v-if="error" class="text-error">{{ error }}</p>
      <p v-if="message" class="text-success">{{ message }}</p>

      <button type="submit" class="btn btn-primary btn-block" :disabled="loading">
        {{ loading ? '上传中...' : '开始上传' }}
      </button>
    </form>
  </div>
</template>

<style scoped>
.upload-form {
  width: min(560px, 100%);
}
</style>
