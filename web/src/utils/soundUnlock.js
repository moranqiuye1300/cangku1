const KEY = 'svp_sound_unlocked'

export function isSoundUnlocked() {
  return sessionStorage.getItem(KEY) === '1'
}

export function markSoundUnlocked() {
  sessionStorage.setItem(KEY, '1')
}
