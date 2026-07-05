<script setup>
defineProps({
  title: { type: String, required: true },
  subtitle: { type: String, default: '' }
})
</script>

<template>
  <div class="auth-shell">
    <div class="auth-card">
      <div class="auth-brand">
        <div class="auth-brand-mark">SV</div>
        <h1>短视频平台</h1>
        <p>发现 · 创作 · 分享</p>
      </div>
      <div class="card card-body-lg">
        <h2 class="page-title">{{ title }}</h2>
        <p v-if="subtitle" class="page-desc">{{ subtitle }}</p>
        <slot />
      </div>
    </div>
  </div>
</template>
