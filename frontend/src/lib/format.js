const rupiah = new Intl.NumberFormat('id-ID', {
  style: 'currency',
  currency: 'IDR',
  maximumFractionDigits: 0,
})

const dateLong = new Intl.DateTimeFormat('id-ID', {
  day: 'numeric',
  month: 'long',
  year: 'numeric',
})

const dateShort = new Intl.DateTimeFormat('id-ID', {
  day: 'numeric',
  month: 'short',
  year: 'numeric',
})

const dateTime = new Intl.DateTimeFormat('id-ID', {
  day: 'numeric',
  month: 'short',
  year: 'numeric',
  hour: '2-digit',
  minute: '2-digit',
})

export function formatRupiah(value) {
  const n = Number(value)
  if (!Number.isFinite(n)) return 'Rp 0'
  return rupiah.format(n).replace(/\u00a0/g, ' ')
}


export function parseDateOnly(value) {
  if (!value) return null
  const match = /^(\d{4})-(\d{2})-(\d{2})/.exec(String(value))
  if (!match) {
    const d = new Date(value)
    return Number.isNaN(d.getTime()) ? null : d
  }
  return new Date(Number(match[1]), Number(match[2]) - 1, Number(match[3]))
}

export function formatDate(value, style = 'long') {
  if (!value) return '—'
  const d = String(value).length === 10 ? parseDateOnly(value) : new Date(value)
  if (!d || Number.isNaN(d.getTime())) return '—'
  if (style === 'short') return dateShort.format(d)
  if (style === 'datetime') return dateTime.format(d)
  return dateLong.format(d)
}

export function formatDateTime(value) {
  return formatDate(value, 'datetime')
}

export function dateRange(start, end) {
  if (!start && !end) return '—'
  if (start === end || !end) return formatDate(start, 'short')
  return `${formatDate(start, 'short')} – ${formatDate(end, 'short')}`
}

export function daysInclusive(start, end) {
  const a = parseDateOnly(start)
  const b = parseDateOnly(end)
  if (!a || !b) return 0
  const diff = Math.round((b.getTime() - a.getTime()) / 86400000)
  return diff >= 0 ? diff + 1 : 0
}

export function todayISO(offsetDays = 0) {
  const d = new Date()
  d.setDate(d.getDate() + offsetDays)
  const month = String(d.getMonth() + 1).padStart(2, '0')
  const day = String(d.getDate()).padStart(2, '0')
  return `${d.getFullYear()}-${month}-${day}`
}

export function initials(name = '') {
  return name
    .trim()
    .split(/\s+/)
    .slice(0, 2)
    .map((part) => part[0] || '')
    .join('')
    .toUpperCase() || '?'
}

export function conditionLabel(percent) {
  const n = Number(percent) || 0
  return `${(n / 10).toFixed(1)}/10`
}

const RENTAL_STATUS = {
  pending: { label: 'Menunggu', tone: 'warning', icon: 'hourglass_top' },
  approved: { label: 'Disetujui', tone: 'primary', icon: 'check_circle' },
  rejected: { label: 'Ditolak', tone: 'danger', icon: 'cancel' },
  cancelled: { label: 'Dibatalkan', tone: 'muted', icon: 'block' },
  completed: { label: 'Selesai', tone: 'success', icon: 'task_alt' },
}

export function rentalStatus(status) {
  return RENTAL_STATUS[status] || { label: status || '—', tone: 'muted', icon: 'info' }
}

const KTM_STATUS = {
  pending: { label: 'Menunggu Verifikasi', tone: 'warning', icon: 'hourglass_top' },
  verified: { label: 'Terverifikasi', tone: 'success', icon: 'verified' },
  rejected: { label: 'Ditolak', tone: 'danger', icon: 'cancel' },
  revoked: { label: 'Dicabut', tone: 'danger', icon: 'gpp_bad' },
}

export function ktmStatus(status) {
  return KTM_STATUS[status] || { label: 'Belum Verifikasi', tone: 'muted', icon: 'badge' }
}

export function roleLabel(role) {
  return role === 'admin_dev' ? 'Admin' : 'Mahasiswa'
}

const PRODI = {
  '120': 'S1 Teknik Informatika',
}

export function prodiName(code) {
  if (!code) return 'Mahasiswa FT UNESA'
  return PRODI[code] || `Prodi ${code}`
}

export function timeAgo(value) {
  if (!value) return '—'
  const d = new Date(value)
  if (Number.isNaN(d.getTime())) return '—'
  const seconds = Math.floor((Date.now() - d.getTime()) / 1000)
  if (seconds < 60) return 'baru saja'
  const minutes = Math.floor(seconds / 60)
  if (minutes < 60) return `${minutes} menit lalu`
  const hours = Math.floor(minutes / 60)
  if (hours < 24) return `${hours} jam lalu`
  const days = Math.floor(hours / 24)
  if (days < 30) return `${days} hari lalu`
  return formatDate(value, 'short')
}

export function fileSize(bytes) {
  const n = Number(bytes) || 0
  if (n < 1024) return `${n} B`
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(0)} KB`
  return `${(n / (1024 * 1024)).toFixed(1)} MB`
}

export function listingPhotoUrl(listing) {
  if (!listing) return null
  if (listing.cover_photo?.url) return listing.cover_photo.url
  const photo = listing.photos?.[0]
  return photo?.url || null
}

export function categoryIcon(slug) {
  switch (slug) {
    case 'elektronik':
      return 'devices'
    case 'fotografi':
      return 'photo_camera'
    case 'olahraga':
      return 'sports_soccer'
    case 'alat-tulis':
      return 'edit'
    case 'perkakas':
      return 'handyman'
    default:
      return 'inventory_2'
  }
}
