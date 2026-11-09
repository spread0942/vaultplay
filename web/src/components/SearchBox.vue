<script setup lang="ts">
defineProps<{
  modelValue: string
}>()

defineEmits<{
  'update:modelValue': [value: string]
}>()
</script>

<template>
  <label class="search">
    <span class="label">Search</span>
    <input
      type="search"
      :value="modelValue"
      placeholder="Filter by name or path…"
      autocomplete="off"
      spellcheck="false"
      @input="$emit('update:modelValue', ($event.target as HTMLInputElement).value)"
    />
  </label>
</template>

<style scoped>
.search {
  display: grid;
  gap: 0.35rem;
}

.label {
  font-size: 0.75rem;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  color: var(--muted);
}

input {
  width: 100%;
  border: 1px solid var(--line);
  background: var(--bg);
  border-radius: 4px;
  padding: 0.65rem 0.75rem;
}

input::placeholder {
  color: var(--muted);
}
</style>
