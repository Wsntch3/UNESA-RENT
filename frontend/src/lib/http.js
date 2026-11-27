import { getToken } from '../auth/tokenStorage'

const BASE_URL = (import.meta.env.VITE_API_URL || '').replace(/\/+$/, '')

export class ApiError extends Error {
  constructor(message, { status = 0, code = 'unknown_error', fields } = {}) {
    super(message)
    this.name = 'ApiError'
    this.status = status
    this.code = code
    this.fields = fields
  }
}

function networkError() {
  return new ApiError(
    'Tidak dapat terhubung ke server. Cek koneksi atau apakah backend sudah berjalan.',
    { code: 'network_error' },
  )
}

async function readJson(response) {
  const contentType = response.headers.get('content-type') || ''
  if (!contentType.includes('application/json')) return null
  return response.json().catch(() => null)
}

async function throwIfNotOk(response) {
  if (response.ok) return
  const data = await readJson(response)
  const apiError = data?.error
  throw new ApiError(apiError?.message || `Request gagal (HTTP ${response.status})`, {
    status: response.status,
    code: apiError?.code || 'http_error',
    fields: apiError?.fields,
  })
}

export async function request(path, { method = 'GET', body, auth = true, signal } = {}) {
  const headers = { Accept: 'application/json' }

  if (body !== undefined) {
    headers['Content-Type'] = 'application/json'
  }

  if (auth) {
    const token = getToken()
    if (token) headers.Authorization = `Bearer ${token}`
  }

  let response
  try {
    response = await fetch(`${BASE_URL}${path}`, {
      method,
      headers,
      body: body === undefined ? undefined : JSON.stringify(body),
      signal,
    })
  } catch (error) {
    if (error.name === 'AbortError') throw error
    throw networkError()
  }

  const data = await readJson(response)
  if (!response.ok) {
    const apiError = data?.error
    throw new ApiError(apiError?.message || `Request gagal (HTTP ${response.status})`, {
      status: response.status,
      code: apiError?.code || 'http_error',
      fields: apiError?.fields,
    })
  }

  return data
}

export async function upload(path, file, field = 'file', extra = {}) {
  const form = new FormData()
  form.append(field, file)
  Object.entries(extra).forEach(([key, value]) => {
    if (value !== undefined && value !== null) form.append(key, String(value))
  })

  const token = getToken()
  let response
  try {
    response = await fetch(`${BASE_URL}${path}`, {
      method: 'POST',
      headers: token ? { Authorization: `Bearer ${token}` } : {},
      body: form,
    })
  } catch {
    throw networkError()
  }

  await throwIfNotOk(response)
  return readJson(response)
}

export async function fetchBlobUrl(path) {
  const token = getToken()
  let response
  try {
    response = await fetch(`${BASE_URL}${path}`, {
      headers: token ? { Authorization: `Bearer ${token}` } : {},
    })
  } catch {
    throw networkError()
  }

  await throwIfNotOk(response)
  const blob = await response.blob()
  return URL.createObjectURL(blob)
}

/** POST JSON yang mengembalikan berkas; caller harus mencabut URL blob setelah dipakai. */
export async function postBlobUrl(path, body) {
  const token = getToken()
  let response
  try {
    response = await fetch(`${BASE_URL}${path}`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        ...(token ? { Authorization: `Bearer ${token}` } : {}),
      },
      body: JSON.stringify(body),
    })
  } catch {
    throw networkError()
  }

  await throwIfNotOk(response)
  const blob = await response.blob()
  return URL.createObjectURL(blob)
}

/** Unduh URL blob lalu cabut URL tersebut setelah sempat dipakai browser. */
export function saveBlobUrl(url, filename) {
  const link = document.createElement('a')
  link.href = url
  link.download = filename
  document.body.appendChild(link)
  link.click()
  link.remove()
  setTimeout(() => URL.revokeObjectURL(url), 15000)
}

/** Build a WebSocket URL for a backend path, honouring VITE_API_URL. */
export function wsUrl(path) {
  if (/^https?:\/\//i.test(BASE_URL)) {
    const base = new URL(BASE_URL)
    const protocol = base.protocol === 'https:' ? 'wss:' : 'ws:'
    return `${protocol}//${base.host}${path}`
  }
  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
  return `${protocol}//${window.location.host}${path}`
}

export { BASE_URL }
