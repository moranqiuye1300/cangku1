<script setup>
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { completeChunkUpload, initChunkUpload, uploadChunk } from '../api/video'
import { useAuthStore } from '../stores/auth'

const router = useRouter()
const auth = useAuthStore()
const title = ref('')
const description = ref('')
const file = ref(null)
const fileName = ref('')
const dragOver = ref(false)
const loading = ref(false)
const error = ref('')
const message = ref('')
const fileInput = ref(null)
const progress = ref(0)
const progressText = ref('')

function onFileChange(e) {
  const f = e.target.files?.[0] || null
  setFile(f)
}

function setFile(f) {
  file.value = f
  fileName.value = f?.name || ''
}

function onDrop(e) {
  dragOver.value = false
  const f = e.dataTransfer?.files?.[0]
  if (f && f.type.startsWith('video/')) {
    setFile(f)
  } else if (f) {
    error.value = '请选择视频文件'
  }
}

function pickFile() {
  fileInput.value?.click()
}

async function handleSubmit() {
  if (!file.value) {
    error.value = '请选择视频文件'
    return
  }
  loading.value = true
  error.value = ''
  message.value = ''
  progress.value = 0
  progressText.value = ''
  try {
    const chunkSize = 5 * 1024 * 1024
    const totalChunks = Math.max(1, Math.ceil(file.value.size / chunkSize))
    const initRes = await initChunkUpload({
      title: title.value,
      description: description.value,
      filename: file.value.name,
      content_type: file.value.type || 'application/octet-stream',
      size: file.value.size,
      chunk_size: chunkSize
    })
    const sessionId = initRes.data.session_id
    for (let i = 0; i < totalChunks; i++) {
      const start = i * chunkSize
      const end = Math.min(start + chunkSize, file.value.size)
      const chunk = file.value.slice(start, end)
      const form = new FormData()
      form.append('session_id', sessionId)
      form.append('chunk_index', String(i))
      form.append('file', chunk, `${file.value.name}.part${i}`)
      await uploadChunk(form)
      progress.value = Math.round(((i + 1) / totalChunks) * 100)
      progressText.value = `已上传 ${i + 1}/${totalChunks} 分片`
    }
    const res = await completeChunkUpload({
      session_id: sessionId,
      title: title.value,
      description: description.value
    })
    message.value = '上传成功，等待源片审核，通过后将自动转码'
    const userId = auth.user?.id || res.data.video?.user_id
    setTimeout(() => {
      if (userId) {
        router.push(`/users/${userId}`)
      }
    }, 1200)
  } catch (e) {
    error.value = e?.response?.data?.message || e.message || '上传失败'
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="page-center">
    <form class="card card-body-lg upload-form" @submit.prevent="handleSubmit">
      <h1 class="page-title">上传视频</h1>
      <p class="page-desc">上传后需经源片快审与精审发布，通过后才会出现在推荐与发现页</p>

      <div class="upload-steps" role="list">
        <div class="upload-step" role="listitem">
          <strong>1</strong>
          源片快审
        </div>
        <div class="upload-step" role="listitem">
          <strong>2</strong>
          自动转码
        </div>
        <div class="upload-step" role="listitem">
          <strong>3</strong>
          精审发布
        </div>
      </div>

      <label class="field">
        标题
        <input v-model="title" required maxlength="80" placeholder="给视频起个标题" />
      </label>
      <label class="field">
        简介
        <textarea v-model="description" rows="3" maxlength="300" placeholder="可选，介绍视频内容" />
      </label>

      <div
        class="drop-zone"
        :class="{ 'drop-zone--active': dragOver }"
        @dragover.prevent="dragOver = true"
        @dragleave.prevent="dragOver = false"
        @drop.prevent="onDrop"
        @click="pickFile"
      >
        <input ref="fileInput" type="file" accept="video/*" @change="onFileChange" />
        <span v-if="fileName">{{ fileName }}</span>
        <template v-else>
          <span>点击或拖拽视频到此处</span>
          <span class="text-muted" style="font-size: 12px">支持常见视频格式，最大 200MB</span>
        </template>
      </div>

      <div v-if="loading" class="upload-progress" aria-label="上传进度">
        <div class="upload-progress__bar" :style="{ width: progress + '%' }"></div>
      </div>
      <p v-if="progressText" class="text-muted">{{ progressText }}</p>
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

.upload-progress {
  width: 100%;
  height: 8px;
  background: #eef2ff;
  border-radius: 999px;
  overflow: hidden;
  margin-bottom: 8px;
}

.upload-progress__bar {
  height: 100%;
  background: linear-gradient(90deg, #4f46e5, #06b6d4);
  transition: width 0.2s ease;
}
</style>
