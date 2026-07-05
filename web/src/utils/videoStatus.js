export const VIDEO_STATUS_LABELS = {
  pending_source_review: '待源审',
  transcoding: '转码中',
  pending_final_review: '待发布',
  ready: '已发布',
  failed: '转码失败',
  pending: '处理中'
}

export function videoStatusLabel(status) {
  return VIDEO_STATUS_LABELS[status] || status || '-'
}

export function isPublishedStatus(status) {
  return status === 'ready'
}
