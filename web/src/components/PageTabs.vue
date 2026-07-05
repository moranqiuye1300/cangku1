<script setup>
const model = defineModel({ type: String, required: true })

defineProps({
  tabs: {
    type: Array,
    required: true
  }
})

function select(key) {
  model.value = key
}
</script>

<template>
  <nav class="tab-group">
    <button
      v-for="tab in tabs"
      :key="tab.key"
      type="button"
      class="tab-item"
      :class="{ active: model === tab.key }"
      @click="select(tab.key)"
    >
      {{ tab.label }}
    </button>
  </nav>
</template>
