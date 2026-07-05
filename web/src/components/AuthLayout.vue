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
      <div class="auth-form-card">
        <h2 class="auth-form-title">{{ title }}</h2>
        <p v-if="subtitle" class="auth-form-desc">{{ subtitle }}</p>
        <slot />
      </div>
    </div>
  </div>
</template>

<style scoped>
.auth-shell {
  min-height: calc(100vh - var(--header-height));
  display: grid;
  place-items: center;
  padding: var(--space-5);
  background: linear-gradient(145deg, #0f0f0f 0%, #1a0a0f 50%, #0a0f1a 100%);
}

.auth-card {
  width: min(420px, 100%);
}

.auth-brand {
  text-align: center;
  margin-bottom: var(--space-5);
}

.auth-brand-mark {
  width: 56px;
  height: 56px;
  margin: 0 auto 16px;
  display: grid;
  place-items: center;
  border-radius: var(--radius-lg);
  background: var(--color-primary-gradient);
  color: #fff;
  font-weight: 700;
  font-size: 20px;
  box-shadow: 0 4px 24px rgba(254, 44, 85, 0.35);
}

.auth-brand h1 {
  margin: 0 0 6px;
  font-size: 24px;
  font-weight: 700;
  color: #fff;
  letter-spacing: -0.02em;
}

.auth-brand p {
  margin: 0;
  color: var(--color-text-muted);
  font-size: 14px;
}

.auth-form-card {
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  padding: 28px 24px;
  box-shadow: var(--shadow-card);
}

.auth-form-title {
  margin: 0 0 4px;
  font-size: 20px;
  font-weight: 600;
  color: var(--color-text);
}

.auth-form-desc {
  margin: 0 0 20px;
  color: var(--color-text-secondary);
  font-size: 14px;
}

@media (max-width: 768px) {
  .auth-shell {
    padding: var(--space-4);
  }
}
</style>
