<script setup lang="ts">
import { onMounted } from 'vue'
import FileList from './components/FileList.vue'
import Player from './components/Player.vue'
import SearchBox from './components/SearchBox.vue'
import { useLibrary } from './composables/useLibrary'

const {
  query,
  selectedPath,
  selected,
  groups,
  loading,
  error,
  load,
  select,
  selectRelative,
} = useLibrary()

onMounted(() => {
  void load()
})
</script>

<template>
  <div class="shell">
    <header class="brand">
      <div>
        <p class="eyebrow">Local · Rootless · Offline</p>
        <h1>Vaultplay</h1>
      </div>
      <p class="status" :data-state="error ? 'err' : 'ok'">
        <template v-if="loading">Indexing…</template>
        <template v-else-if="error">{{ error }}</template>
        <template v-else>Ready</template>
      </p>
    </header>

    <main class="layout">
      <aside class="library">
        <SearchBox v-model="query" />
        <FileList
          :groups="groups"
          :selected-path="selectedPath"
          @select="select"
        />
      </aside>
      <Player
        :video="selected"
        @next="selectRelative(1)"
        @prev="selectRelative(-1)"
      />
    </main>
  </div>
</template>

<style scoped>
.shell {
  height: 100%;
  display: grid;
  grid-template-rows: auto 1fr;
  gap: 1rem;
  padding: 1.25rem clamp(1rem, 2vw, 1.75rem) 1.5rem;
}

.brand {
  display: flex;
  justify-content: space-between;
  align-items: end;
  gap: 1rem;
  border-bottom: 1px solid var(--line);
  padding-bottom: 0.85rem;
}

.eyebrow {
  margin: 0 0 0.2rem;
  font-family: var(--font-mono);
  font-size: 0.72rem;
  letter-spacing: 0.14em;
  text-transform: uppercase;
  color: var(--muted);
}

h1 {
  margin: 0;
  font-size: clamp(1.8rem, 3vw, 2.4rem);
  font-weight: 600;
  letter-spacing: -0.02em;
}

.status {
  margin: 0;
  font-family: var(--font-mono);
  font-size: 0.8rem;
  color: var(--muted);
}

.status[data-state='err'] {
  color: var(--danger);
}

.layout {
  min-height: 0;
  display: grid;
  grid-template-columns: minmax(280px, 0.95fr) minmax(320px, 1.15fr);
  gap: 1.25rem;
}

.library {
  min-height: 0;
  display: grid;
  grid-template-rows: auto 1fr;
  gap: 0.85rem;
  background: color-mix(in srgb, var(--bg-elevated) 88%, transparent);
  border: 1px solid var(--line);
  padding: 0.9rem;
}

@media (max-width: 960px) {
  .layout {
    grid-template-columns: 1fr;
    grid-template-rows: minmax(240px, 40vh) 1fr;
  }
}
</style>
