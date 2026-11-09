import { computed, ref } from 'vue'

export interface Video {
  name: string
  relPath: string
  dir: string
  size: number
  durationMs: number | null
}

interface VideosResponse {
  videos: Video[]
}

const videos = ref<Video[]>([])
const query = ref('')
const selectedPath = ref<string | null>(null)
const loading = ref(false)
const error = ref<string | null>(null)

export function useLibrary() {
  const filtered = computed(() => {
    const q = query.value.trim().toLowerCase()
    if (!q) return videos.value
    return videos.value.filter((v) => {
      return (
        v.name.toLowerCase().includes(q) ||
        v.relPath.toLowerCase().includes(q) ||
        v.dir.toLowerCase().includes(q)
      )
    })
  })

  const groups = computed(() => {
    const map = new Map<string, Video[]>()
    for (const v of filtered.value) {
      const key = v.dir || '(root)'
      const list = map.get(key)
      if (list) list.push(v)
      else map.set(key, [v])
    }
    return [...map.entries()].map(([dir, items]) => ({ dir, items }))
  })

  const selected = computed(() => {
    if (!selectedPath.value) return null
    return videos.value.find((v) => v.relPath === selectedPath.value) ?? null
  })

  const selectedIndex = computed(() => {
    if (!selectedPath.value) return -1
    return filtered.value.findIndex((v) => v.relPath === selectedPath.value)
  })

  async function load() {
    loading.value = true
    error.value = null
    try {
      const res = await fetch('/api/videos')
      if (!res.ok) throw new Error(`HTTP ${res.status}`)
      const data = (await res.json()) as VideosResponse
      videos.value = data.videos ?? []
      if (!selectedPath.value && videos.value.length > 0) {
        selectedPath.value = videos.value[0].relPath
      }
    } catch (e) {
      error.value = e instanceof Error ? e.message : String(e)
    } finally {
      loading.value = false
    }
  }

  function select(relPath: string) {
    selectedPath.value = relPath
  }

  function selectRelative(delta: number) {
    const list = filtered.value
    if (list.length === 0) return
    const cur = selectedIndex.value
    const next = cur < 0 ? 0 : (cur + delta + list.length) % list.length
    selectedPath.value = list[next].relPath
  }

  return {
    videos,
    query,
    selectedPath,
    selected,
    filtered,
    groups,
    loading,
    error,
    load,
    select,
    selectRelative,
  }
}

export function formatSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`
  const units = ['KB', 'MB', 'GB', 'TB']
  let n = bytes
  let i = -1
  do {
    n /= 1024
    i++
  } while (n >= 1024 && i < units.length - 1)
  return `${n.toFixed(n >= 10 ? 0 : 1)} ${units[i]}`
}

export function formatDuration(ms: number | null | undefined): string {
  if (ms == null || ms < 0) return '—'
  const total = Math.round(ms / 1000)
  const h = Math.floor(total / 3600)
  const m = Math.floor((total % 3600) / 60)
  const s = total % 60
  if (h > 0) return `${h}:${String(m).padStart(2, '0')}:${String(s).padStart(2, '0')}`
  return `${m}:${String(s).padStart(2, '0')}`
}

export function mediaURL(relPath: string): string {
  return `/media/${relPath.split('/').map(encodeURIComponent).join('/')}`
}
