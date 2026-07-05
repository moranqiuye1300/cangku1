export function throttle(fn, wait = 200) {
  let last = 0
  let timer = null
  return (...args) => {
    const now = Date.now()
    if (now - last >= wait) {
      last = now
      fn(...args)
      return
    }
    clearTimeout(timer)
    timer = setTimeout(() => {
      last = Date.now()
      fn(...args)
    }, wait - (now - last))
  }
}
