const KEY = 'svp_progress'
const TTL = 7 * 24 * 60 * 60 * 1000

function readMap() {
  try {
    return JSON.parse(localStorage.getItem(KEY) || '{}')
  } catch {
    return {}
  }
}

export function saveProgress(videoId, time) {
  if (!videoId || !Number.isFinite(time) || time < 1) return
  const map = readMap()
  map[videoId] = { t: time, at: Date.now() }
  localStorage.setItem(KEY, JSON.stringify(map))
}

export function loadProgress(videoId) {
  if (!videoId) return 0
  const item = readMap()[videoId]
  if (!item || Date.now() - item.at > TTL) return 0
  return Number.isFinite(item.t) ? item.t : 0
}

export function formatPlaybackTime(seconds) {
  const s = Math.max(0, Math.floor(seconds))
  const m = Math.floor(s / 60)
  const sec = s % 60
  return `${m}:${String(sec).padStart(2, '0')}`
}
