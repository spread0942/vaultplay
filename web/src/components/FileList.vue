<script setup lang="ts">
import type { Video } from '../composables/useLibrary'
import { formatDuration, formatSize } from '../composables/useLibrary'

defineProps<{
  groups: { dir: string; items: Video[] }[]
  selectedPath: string | null
}>()

defineEmits<{
  select: [relPath: string]
}>()
</script>

<template>
  <div class="list" role="list">
    <section v-for="group in groups" :key="group.dir" class="group">
      <h2>{{ group.dir }}</h2>
      <button
        v-for="item in group.items"
        :key="item.relPath"
        type="button"
        class="row"
        role="listitem"
        :class="{ active: item.relPath === selectedPath }"
        :aria-current="item.relPath === selectedPath ? 'true' : undefined"
        @click="$emit('select', item.relPath)"
      >
        <span class="name">{{ item.name }}</span>
        <span class="path" :title="item.relPath">{{ item.relPath }}</span>
        <span class="meta">{{ formatSize(item.size) }}</span>
        <span class="meta">{{ formatDuration(item.durationMs) }}</span>
      </button>
    </section>
    <p v-if="groups.length === 0" class="empty">No matching videos.</p>
  </div>
</template>

<style scoped>
.list {
  display: flex;
  flex-direction: column;
  gap: 1.25rem;
  overflow: auto;
  min-height: 0;
  padding-right: 0.25rem;
}

.group h2 {
  margin: 0 0 0.45rem;
  font-size: 0.75rem;
  font-weight: 600;
  letter-spacing: 0.06em;
  text-transform: uppercase;
  color: var(--accent);
  font-family: var(--font-mono);
}

.row {
  width: 100%;
  display: grid;
  grid-template-columns: minmax(0, 1.4fr) minmax(0, 1.6fr) 5.5rem 4.5rem;
  gap: 0.75rem;
  align-items: center;
  text-align: left;
  border: 0;
  border-bottom: 1px solid var(--line);
  background: transparent;
  padding: 0.7rem 0.55rem;
  transition: background 120ms ease;
}

.row:hover {
  background: var(--bg-hover);
}

.row.active {
  background: linear-gradient(90deg, rgba(224, 162, 74, 0.16), transparent 70%);
  box-shadow: inset 3px 0 0 var(--accent);
}

.name {
  font-weight: 500;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.path,
.meta {
  font-family: var(--font-mono);
  font-size: 0.8rem;
  color: var(--muted);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.meta {
  text-align: right;
}

.empty {
  color: var(--muted);
  margin: 1rem 0;
}

@media (max-width: 900px) {
  .row {
    grid-template-columns: 1fr auto;
    grid-template-areas:
      'name meta'
      'path dur';
  }

  .name {
    grid-area: name;
  }

  .path {
    grid-area: path;
  }

  .meta:nth-of-type(3) {
    grid-area: meta;
  }

  .meta:nth-of-type(4) {
    grid-area: dur;
  }
}
</style>
