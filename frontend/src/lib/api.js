import { request, upload, fetchBlobUrl, postBlobUrl } from './http'

function qs(params = {}) {
  const search = new URLSearchParams()
  Object.entries(params).forEach(([key, value]) => {
    if (value === undefined || value === null || value === '') return
    search.set(key, String(value))
  })
  const raw = search.toString()
  return raw ? `?${raw}` : ''
}

export const catalogApi = {
  categories: () => request('/api/categories', { auth: false }),
  locations: () => request('/api/campus-locations', { auth: false }),
  search: (params) => request(`/api/listings${qs(params)}`, { auth: false }),
  detail: (id) => request(`/api/listings/${id}`, { auth: false }),
  availability: (id, from, to) =>
    request(`/api/listings/${id}/availability${qs({ from, to })}`, { auth: false }),
  mine: (params) => request(`/api/listings/mine${qs(params)}`),
  create: (body) => request('/api/listings', { method: 'POST', body }),
  update: (id, body) => request(`/api/listings/${id}`, { method: 'PUT', body }),
  remove: (id) => request(`/api/listings/${id}`, { method: 'DELETE' }),
  addPhoto: (id, file) => upload(`/api/listings/${id}/photos`, file),
  deletePhoto: (id, photoId) =>
    request(`/api/listings/${id}/photos/${photoId}`, { method: 'DELETE' }),
  blocks: (id) => request(`/api/listings/${id}/blocks`),
  addBlock: (id, body) => request(`/api/listings/${id}/blocks`, { method: 'POST', body }),
  deleteBlock: (id, blockId) =>
    request(`/api/listings/${id}/blocks/${blockId}`, { method: 'DELETE' }),
  adminList: (params) => request(`/api/admin/listings${qs(params)}`),
  takedown: (id, reason) =>
    request(`/api/admin/listings/${id}/takedown`, { method: 'POST', body: { reason } }),
  restore: (id) => request(`/api/admin/listings/${id}/restore`, { method: 'POST' }),
}

export const rentalApi = {
  submit: (listingId, body) =>
    request(`/api/listings/${listingId}/rentals`, { method: 'POST', body }),
  mine: (params) => request(`/api/rentals/mine${qs(params)}`),
  detail: (id) => request(`/api/rentals/${id}`),
  approve: (id) => request(`/api/rentals/${id}/approve`, { method: 'POST' }),
  reject: (id, reason) =>
    request(`/api/rentals/${id}/reject`, { method: 'POST', body: { reason } }),
  cancel: (id, reason) =>
    request(`/api/rentals/${id}/cancel`, { method: 'POST', body: { reason } }),
  complete: (id) => request(`/api/rentals/${id}/complete`, { method: 'POST' }),
  messages: (id, params) => request(`/api/rentals/${id}/messages${qs(params)}`),
  sendMessage: (id, body) =>
    request(`/api/rentals/${id}/messages`, { method: 'POST', body: { body } }),
  wsTicket: (id) => request(`/api/rentals/${id}/ws-ticket`, { method: 'POST' }),
  agreementBlobUrl: (id) => fetchBlobUrl(`/api/rentals/${id}/agreement/file`),
  history: (params) => request(`/api/rentals/history${qs(params)}`),

  documents: (id) => request(`/api/rentals/${id}/documents`),
  documentBlobUrl: (rentalID, docID) =>
    fetchBlobUrl(`/api/rentals/${rentalID}/documents/${docID}/file`),
  cod: (id) => request(`/api/rentals/${id}/cod`),
  handover: (id, file, note) => upload(`/api/rentals/${id}/cod/handover`, file, 'file', { note }),
  confirmHandover: (id) => request(`/api/rentals/${id}/cod/handover/confirm`, { method: 'POST' }),
  returnProof: (id, file, note) => upload(`/api/rentals/${id}/cod/return`, file, 'file', { note }),
  confirmReturn: (id) => request(`/api/rentals/${id}/cod/return/confirm`, { method: 'POST' }),
  settle: (id, body) => request(`/api/rentals/${id}/cod/settle`, { method: 'POST', body }),

  adminList: (params) => request(`/api/admin/rentals${qs(params)}`),
  adminDetail: (id) => request(`/api/admin/rentals/${id}`),
  adminStats: () => request('/api/admin/rentals/stats'),
  adminExportBlobUrl: (params) => fetchBlobUrl(`/api/admin/rentals/export.csv${qs(params)}`),
  adminDocuments: (id) => request(`/api/admin/rentals/${id}/documents`),
  adminDocumentBlobUrl: (rentalID, docID, reason) =>
    postBlobUrl(`/api/admin/rentals/${rentalID}/documents/${docID}/download`, { reason }),
  adminMessages: (id, params) => request(`/api/admin/rentals/${id}/messages${qs(params)}`),
  revealMessage: (rentalID, msgID, body) =>
    request(`/api/admin/rentals/${rentalID}/messages/${msgID}/reveal`, { method: 'POST', body }),
  transcriptBlobUrl: (id, body) =>
    postBlobUrl(`/api/admin/rentals/${id}/transcript/export`, body),
}

export const ktmApi = {
  mine: () => request('/api/ktm/me'),
  submit: (file) => upload('/api/ktm', file),
  fileBlobUrl: (id) => fetchBlobUrl(`/api/ktm/${id}/file`),
}

export const adminKtmApi = {
  list: (params) => request(`/api/admin/ktm${qs(params)}`),
  verify: (id) =>
    request(`/api/admin/ktm/${id}/verify`, { method: 'POST', body: { confirm_nim: true } }),
  reject: (id, reason) =>
    request(`/api/admin/ktm/${id}/reject`, { method: 'POST', body: { reason } }),
  revoke: (id, reason) =>
    request(`/api/admin/ktm/${id}/revoke`, { method: 'POST', body: { reason } }),
}

export const notificationApi = {
  list: (params) => request(`/api/notifications${qs(params)}`),
  read: (id) => request(`/api/notifications/${id}/read`, { method: 'POST' }),
}

export const reportApi = {
  create: (body) => request('/api/reports', { method: 'POST', body }),
  mine: (params) => request(`/api/reports/mine${qs(params)}`),
  adminList: (params) => request(`/api/admin/reports${qs(params)}`),
  detail: (id) => request(`/api/admin/reports/${id}`),
  resolve: (id, note) =>
    request(`/api/admin/reports/${id}/resolve`, { method: 'POST', body: { resolution_note: note } }),
  reject: (id, note) =>
    request(`/api/admin/reports/${id}/reject`, { method: 'POST', body: { resolution_note: note } }),
}

export const adminApi = {
  users: (params) => request(`/api/admin/users${qs(params)}`),
  user: (id) => request(`/api/admin/users/${id}`),
  suspend: (id, reason) =>
    request(`/api/admin/users/${id}/suspend`, { method: 'POST', body: { reason } }),
  reactivate: (id, reason) =>
    request(`/api/admin/users/${id}/reactivate`, { method: 'POST', body: { reason } }),
  removeUser: (id, reason) => request(`/api/admin/users/${id}`, { method: 'DELETE', body: { reason } }),
  exportUsersBlobUrl: (params) => fetchBlobUrl(`/api/admin/users/export.csv${qs(params)}`),
  audit: (params) => request(`/api/admin/audit${qs(params)}`),
  stats: () => request('/api/admin/stats'),
}
