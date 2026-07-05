<script setup>
defineProps({
  title: { type: String, required: true },
  count: { type: [Number, String], default: '' },
  suffix: { type: String, default: '个' }
})
</script>

<template>
  <div class="page-section-head">
    <h2>{{ title }}</h2>
    <span v-if="count !== '' && count !== null">{{ count }} {{ suffix }}</span>
    <slot v-else name="extra" />
  </div>
</template>
