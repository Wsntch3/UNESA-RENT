import { useEffect, useState } from 'react'
import { Link } from '../router'
import { useAuth } from '../auth/useAuth'
import { rentalApi } from '../lib/api'
import { useApi } from '../lib/useApi'
import { saveBlobUrl } from '../lib/http'
import {
  dateRange,
  formatRupiah,
  rentalStatus,
  timeAgo,
} from '../lib/format'
import {
  Avatar,
  Empty,
  ErrorBox,
  Icon,
  Loading,
  PageHeader,
} from '../components/ui'

const LIMIT = 12

const STATUS_FILTERS = [
  { value: '', label: 'Semua status' },
  { value: 'pending', label: 'Menunggu' },
  { value: 'approved', label: 'Disetujui' },
  { value: 'completed', label: 'Selesai' },
  { value: 'rejected', label: 'Ditolak' },
  { value: 'cancelled', label: 'Dibatalkan' },
]

export default function Rentals() {
  const { user } = useAuth()

  const [roleTab, setRoleTab] = useState('all')
  const [status, setStatus] = useState('')
  const [from, setFrom] = useState('')
  const [to, setTo] = useState('')
  const [searchInput, setSearchInput] = useState('')
  const [search, setSearch] = useState('')
  const [offset, setOffset] = useState(0)
  const [exportError, setExportError] = useState(null)

  useEffect(() => {
    const timer = setTimeout(() => {
      setSearch(searchInput.trim())
      setOffset(0)
    }, 350)
    return () => clearTimeout(timer)
  }, [searchInput])

  const summaryQuery = useApi(() => rentalApi.mine({ limit: 60 }), [])
  const { data, loading, error, reload } = useApi(
    () => rentalApi.mine({
      role: roleTab === 'all' ? undefined : roleTab,
      status,
      from,
      to,
      q: search,
      limit: LIMIT,
      offset,
    }),
    [roleTab, status, from, to, search, offset],
  )

  const rentals = data?.items || []
  const total = data?.total || 0
  const allRentals = summaryQuery.data?.items || []
  const asRenter = allRentals.filter((item) => item.my_role === 'renter')
  const asOwner = allRentals.filter((item) => item.my_role === 'owner')
  const activeRentals = allRentals.filter((item) => item.status === 'approved')

  const changeFilter = (setter, value) => {
    setter(value)
    setOffset(0)
  }

  const exportCSV = () => {
    setExportError(null)
    try {
      const rows = [
        ['ID', 'Barang', 'Peran', 'Pihak lain', 'Mulai', 'Selesai', 'Hari', 'Total', 'Status', 'Dibuat'],
        ...rentals.map((item) => [
          item.id,
          item.listing?.title || '',
          item.my_role === 'owner' ? 'Penyedia' : 'Penyewa',
          item.my_role === 'owner' ? item.renter?.name || '' : item.owner?.name || '',
          item.start_date,
          item.end_date,
          item.total_days,
          item.total_price,
          rentalStatus(item.status).label,
          item.created_at,
        ]),
      ]
      const csv = rows
        .map((row) => row.map((cell) => `"${String(cell ?? '').replace(/"/g, '""')}"`).join(','))
        .join('\r\n')
      const blob = new Blob([`\uFEFF${csv}`], { type: 'text/csv;charset=utf-8' })
      saveBlobUrl(URL.createObjectURL(blob), 'penyewaan-saya.csv')
    } catch (err) {
      setExportError(err)
    }
  }

  return (
    <div className="container page">
      <PageHeader
        eyebrow="Aktivitas Transaksi Kampus Ketintang"
        title="Penyewaan saya"
        subtitle="Pantau pengajuan sewa, penyewaan aktif, dan riwayat transaksi bersama sesama mahasiswa FT UNESA."
        actions={
          <>
            <button type="button" className="btn btn-outline" onClick={exportCSV} disabled={!rentals.length}>
              <Icon name="download" /> Ekspor CSV
            </button>
            <Link to="/lapor" className="btn btn-ghost">
              <Icon name="flag" /> Laporkan Masalah
            </Link>
            <Link to="/jelajahi" className="btn btn-outline">
              <Icon name="search" /> Cari Barang
            </Link>
          </>
        }
      />

      <div className="grid cols-3" style={{ marginBottom: 20 }}>
        <div className="stat-card">
          <span className="stat-icon"><Icon name="shopping_bag" /></span>
          <div>
            <div className="stat-value">{asRenter.length}</div>
            <div className="stat-label">Sebagai penyewa</div>
          </div>
        </div>
        <div className="stat-card">
          <span className="stat-icon mint"><Icon name="storefront" /></span>
          <div>
            <div className="stat-value">{asOwner.length}</div>
            <div className="stat-label">Sebagai pemilik</div>
          </div>
        </div>
        <div className="stat-card">
          <span className="stat-icon yellow"><Icon name="sync_alt" /></span>
          <div>
            <div className="stat-value">{activeRentals.length}</div>
            <div className="stat-label">Sedang berlangsung</div>
          </div>
        </div>
      </div>

      <div className="between mb-4" style={{ flexWrap: 'wrap', gap: 10 }}>
        <div className="tabs">
          {[
            { value: 'all', label: `Semua (${allRentals.length})` },
            { value: 'renter', label: `Penyewa (${asRenter.length})` },
            { value: 'owner', label: `Pemilik (${asOwner.length})` },
          ].map((tab) => (
            <button
              key={tab.value}
              type="button"
              className={`tab ${roleTab === tab.value ? 'active' : ''}`.trim()}
              onClick={() => changeFilter(setRoleTab, tab.value)}
            >
              {tab.label}
            </button>
          ))}
        </div>

        <div className="toolbar">
          <div className="input-wrap" style={{ minWidth: 200 }}>
            <Icon name="search" />
            <input
              className="input"
              value={searchInput}
              onChange={(event) => setSearchInput(event.target.value)}
              placeholder="Cari barang atau nama…"
              aria-label="Cari transaksi"
            />
          </div>
          <select
            className="select"
            value={status}
            onChange={(event) => changeFilter(setStatus, event.target.value)}
            aria-label="Filter status"
          >
            {STATUS_FILTERS.map((option) => (
              <option key={option.value} value={option.value}>{option.label}</option>
            ))}
          </select>
          <input
            type="date"
            className="input"
            value={from}
            onChange={(event) => changeFilter(setFrom, event.target.value)}
            aria-label="Dibuat dari tanggal"
          />
          <input
            type="date"
            className="input"
            value={to}
            onChange={(event) => changeFilter(setTo, event.target.value)}
            aria-label="Dibuat sampai tanggal"
          />
        </div>
      </div>

      {error ? <ErrorBox error={error} onRetry={reload} /> : null}
      {summaryQuery.error ? <ErrorBox error={summaryQuery.error} onRetry={summaryQuery.reload} title="Ringkasan gagal dimuat" /> : null}
      {exportError ? <ErrorBox error={exportError} title="Ekspor gagal" /> : null}

      {loading ? (
        <Loading label="Memuat penyewaan…" />
      ) : rentals.length ? (
        <>
        <div className="stack">
          {rentals.map((item) => {
            const meta = rentalStatus(item.status)
            const isOwner = item.my_role === 'owner'
            const other = isOwner ? item.renter : item.owner
            const photo = item.listing?.cover_photo_url

            return (
              <article className="card card-pad" key={item.id}>
                <div className="row" style={{ alignItems: 'flex-start', gap: 14, flexWrap: 'wrap' }}>
                  <div
                    style={{
                      width: 96,
                      height: 78,
                      borderRadius: 10,
                      overflow: 'hidden',
                      background: 'linear-gradient(135deg,#eef1ff,#fff3d1)',
                      display: 'grid',
                      placeItems: 'center',
                      color: 'var(--primary)',
                      flexShrink: 0,
                    }}
                  >
                    {photo ? (
                      <img src={photo} alt={item.listing?.title} style={{ width: '100%', height: '100%', objectFit: 'cover' }} />
                    ) : (
                      <Icon name="inventory_2" size={30} />
                    )}
                  </div>

                  <div style={{ flex: 1, minWidth: 240 }}>
                    <div className="row-wrap">
                      <span className={`badge ${isOwner ? 'badge-mint' : 'badge-primary'}`}>
                        <Icon name={isOwner ? 'storefront' : 'shopping_bag'} size={12} />
                        {isOwner ? 'Pemilik' : 'Penyewa'}
                      </span>
                      <span className={`badge badge-${meta.tone}`}>
                        <Icon name={meta.icon} size={12} /> {meta.label}
                      </span>
                      <span className="tiny muted">#{item.id.slice(0, 8)}</span>
                    </div>

                    <h3 className="card-title mt-2">{item.listing?.title}</h3>
                    <div className="tiny muted mt-1">
                      <Icon name="event" size={13} /> {dateRange(item.start_date, item.end_date)} ·{' '}
                      {item.total_days} hari · <Icon name="payments" size={13} />{' '}
                      {formatRupiah(item.total_price)}
                    </div>

                    <div className="row mt-2">
                      <Avatar name={other?.name} size="sm" />
                      <span className="tiny muted">
                        {isOwner ? 'Penyewa' : 'Pemilik'}: <strong style={{ color: 'var(--ink)' }}>{other?.name}</strong>
                        {' · '}diajukan {timeAgo(item.created_at)}
                      </span>
                    </div>
                  </div>

                  <div className="row-wrap" style={{ alignSelf: 'center' }}>
                    <Link to={`/penyewaan/${item.id}`} className="btn btn-primary btn-sm">
                      <Icon name="visibility" /> Detail
                    </Link>
                    <Link to={`/penyewaan/${item.id}?chat=1`} className="btn btn-outline btn-sm">
                      <Icon name="chat" /> Chat
                    </Link>
                  </div>
                </div>
              </article>
            )
          })}
        </div>

        <div className="between" style={{ marginTop: 14 }}>
          <span className="small muted">
            Menampilkan {rentals.length} dari {total} transaksi
          </span>
          <div className="row">
            <button
              type="button"
              className="btn btn-outline btn-sm"
              disabled={offset === 0}
              onClick={() => setOffset((value) => Math.max(0, value - LIMIT))}
            >
              <Icon name="chevron_left" /> Sebelumnya
            </button>
            <button
              type="button"
              className="btn btn-outline btn-sm"
              disabled={offset + LIMIT >= total}
              onClick={() => setOffset((value) => value + LIMIT)}
            >
              Berikutnya <Icon name="chevron_right" />
            </button>
          </div>
        </div>
        </>
      ) : (
        <Empty
          icon="receipt_long"
          title="Belum ada transaksi"
          action={
            <Link to="/jelajahi" className="btn btn-primary btn-sm">
              <Icon name="search" /> Jelajahi Barang
            </Link>
          }
        >
          {search || status || from || to || roleTab !== 'all'
            ? 'Tidak ada transaksi yang cocok dengan filtermu.'
            : 'Ajukan sewa barang atau tunggu pengajuan masuk ke barangmu.'}
        </Empty>
      )}

      {user ? (
        <p className="hint mt-4">
          Tips: buka detail transaksi untuk menyetujui pengajuan, membatalkan sewa, mengunduh surat
          perjanjian, atau mengobrol dengan pihak lain.
        </p>
      ) : null}
    </div>
  )
}
