<script setup lang="ts">
import { nextTick, onMounted, onUnmounted, ref, watch } from 'vue'
import type { Video } from '../composables/useLibrary'
import { formatDuration, mediaURL } from '../composables/useLibrary'

const props = defineProps<{
  video: Video | null
}>()

const emit = defineEmits<{
  next: []
  prev: []
}>()

const videoEl = ref<HTMLVideoElement | null>(null)

watch(
  () => props.video?.relPath,
  async () => {
    await nextTick()
    const el = videoEl.value
    if (!el) return
    el.load()
    void el.play().catch(() => {
      /* autoplay may be blocked until user gesture */
    })
  },
)

function togglePlay() {
  const el = videoEl.value
  if (!el) return
  if (el.paused) void el.play()
  else el.pause()
}

function seek(delta: number) {
  const el = videoEl.value
  if (!el) return
  el.currentTime = Math.max(0, Math.min(el.duration || 0, el.currentTime + delta))
}

function onKey(e: KeyboardEvent) {
  const tag = (e.target as HTMLElement | null)?.tagName
  if (tag === 'INPUT' || tag === 'TEXTAREA' || tag === 'SELECT') return

  switch (e.key) {
    case ' ':
      e.preventDefault()
      togglePlay()
      break
    case 'ArrowLeft':
      e.preventDefault()
      seek(-5)
      break
    case 'ArrowRight':
      e.preventDefault()
      seek(5)
      break
    case 'n':
    case 'N':
      e.preventDefault()
      emit('next')
      break
    case 'p':
    case 'P':
      e.preventDefault()
      emit('prev')
      break
  }
}

onMounted(() => {
  window.addEventListener('keydown', onKey)
})

onUnmounted(() => {
  window.removeEventListener('keydown', onKey)
})
</script>

<template>
  <section class="player">
    <header v-if="video" class="head">
      <h2>{{ video.name }}</h2>
      <p>{{ video.relPath }} · {{ formatDuration(video.durationMs) }}</p>
    </header>
    <header v-else class="head">
      <h2>No video selected</h2>
      <p>Pick a file from the list.</p>
    </header>

    <div class="stage">
      <video
        v-if="video"
        ref="videoEl"
        :key="video.relPath"
        controls
        playsinline
        preload="metadata"
      >
        <source :src="mediaURL(video.relPath)" type="video/mp4" />
      </video>
      <div v-else class="placeholder">Vaultplay</div>
    </div>

    <div class="controls">
      <button type="button" @click="$emit('prev')">Previous</button>
      <button type="button" class="primary" @click="togglePlay">Play / Pause</button>
      <button type="button" @click="$emit('next')">Next</button>
    </div>
    <p class="hints">Space play/pause · ← → seek · N next · P previous</p>
  </section>
</template>

<style scoped>
.player {
  display: grid;
  grid-template-rows: auto 1fr auto auto;
  gap: 0.85rem;
  min-height: 0;
}

.head h2 {
  margin: 0;
  font-size: 1.15rem;
  font-weight: 600;
}

.head p {
  margin: 0.25rem 0 0;
  color: var(--muted);
  font-family: var(--font-mono);
  font-size: 0.8rem;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.stage {
  min-height: 0;
  background: #050607;
  border: 1px solid var(--line);
  display: grid;
  place-items: center;
  overflow: hidden;
}

video {
  width: 100%;
  height: 100%;
  max-height: min(62vh, 720px);
  background: #000;
}

.placeholder {
  color: var(--muted);
  font-family: var(--font-mono);
  letter-spacing: 0.2em;
  text-transform: uppercase;
  padding: 4rem 1rem;
}

.controls {
  display: flex;
  gap: 0.55rem;
  flex-wrap: wrap;
}

.controls button {
  border: 1px solid var(--line);
  background: var(--bg-elevated);
  padding: 0.55rem 0.9rem;
  border-radius: 4px;
}

.controls button:hover {
  border-color: var(--accent-dim);
}

.controls .primary {
  border-color: var(--accent-dim);
  color: var(--accent);
}

.hints {
  margin: 0;
  color: var(--muted);
  font-size: 0.78rem;
  font-family: var(--font-mono);
}
</style>
