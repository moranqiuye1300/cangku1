import Hls from 'hls.js'

const QUALITY_ORDER = ['1080p', '720p', '480p', '360p']

export function sortedQualities(playUrls) {
  if (!playUrls) return []
  return Object.keys(playUrls).sort((a, b) => {
    const ai = QUALITY_ORDER.indexOf(a)
    const bi = QUALITY_ORDER.indexOf(b)
    return (ai === -1 ? 999 : ai) - (bi === -1 ? 999 : bi)
  })
}

export function defaultQuality(playUrls) {
  if (!playUrls) return ''
  if (playUrls['720p']) return '720p'
  if (playUrls['1080p']) return '1080p'
  const sorted = sortedQualities(playUrls)
  return sorted[0] || ''
}

export function pickPlayUrl(playUrls, quality = '') {
  if (!playUrls) return ''
  const q = quality && playUrls[quality] ? quality : defaultQuality(playUrls)
  return q ? playUrls[q] : ''
}

export function hasPlayUrls(playUrls) {
  return playUrls && Object.keys(playUrls).length > 0
}

export function mountHls(videoEl, url) {
  if (!videoEl || !url) return null
  if (Hls.isSupported()) {
    const hls = new Hls({
      enableWorker: true,
      maxBufferLength: 20,
      maxMaxBufferLength: 40,
      startLevel: -1,
      capLevelToPlayerSize: true
    })
    hls.loadSource(url)
    hls.attachMedia(videoEl)
    return hls
  }
  if (videoEl.canPlayType('application/vnd.apple.mpegurl')) {
    videoEl.src = url
  }
  return null
}

export function remountAtTime(videoEl, url, currentTime, prevHls) {
  destroyHls(prevHls)
  if (!videoEl || !url) return null
  const hls = mountHls(videoEl, url)
  const seek = () => {
    if (currentTime > 0 && Number.isFinite(currentTime)) {
      videoEl.currentTime = Math.min(currentTime, videoEl.duration || currentTime)
    }
  }
  if (videoEl.readyState >= 1) {
    seek()
  } else {
    videoEl.addEventListener('loadedmetadata', seek, { once: true })
  }
  return hls
}

export function destroyHls(hls) {
  if (hls) {
    hls.destroy()
  }
}

export function seekWhenReady(videoEl, time, onSeeked) {
  if (!videoEl || !Number.isFinite(time) || time <= 0) return
  const seek = () => {
    const max = videoEl.duration || time
    videoEl.currentTime = Math.min(time, max)
    onSeeked?.(videoEl.currentTime)
  }
  if (videoEl.readyState >= 1) {
    seek()
  } else {
    videoEl.addEventListener('loadedmetadata', seek, { once: true })
  }
}
