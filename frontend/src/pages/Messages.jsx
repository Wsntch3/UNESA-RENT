import { useMemo, useState } from 'react'
import { Link } from '../router'
import { useAuth } from '../auth/useAuth'
import { rentalApi } from '../lib/api'
import { useApi } from '../lib/useApi'
import { dateRange, formatRupiah, rentalStatus, timeAgo } from '../lib/format'
import {
  Avatar,
  Empty,
  ErrorBox,
  Icon,
  Loading,
  PageHeader,
} from '../components/ui'

function ConversationRow({ rental, userId }) {
  const { data, loading } = useApi(
    () => rentalApi.messages(rental.id, { limit: 100 }),
    [rental.id],
  )

  const messages = data?.messages || []
  const last = messages[messages.length - 1]
  const isOwner = rental.my_role === 'owner'
  const other = isOwner ? rental.renter : rental.owner
  const meta = rentalStatus(rental.status)
  const mine = last?.sender?.id === userId

  return (
    <Link
      to={`/penyewaan/${rental.id}?chat=1`}
      className="card card-pad card-hover"
      style={{ color: 'inherit', textDecoration: 'none', display: 'block' }}
    >
      <div className="row" style={{ alignItems: 'flex-start', gap: 12 }}>
        <Avatar name={other?.name} tone={isOwner ? 'yellow' : 'mint'} />
        <div style={{ flex: 1, minWidth: 0 }}>
          <div className="between" style={{ gap: 8, alignItems: 'flex-start' }}>
            <div style={{ minWidth: 0 }}>
              <div className="card-title truncate">{other?.name}</div>
              <div className="tiny muted truncate">
                {isOwner ? 'Penyewa' : 'Pemilik'} · {rental.listing?.title}
              </div>
            </div>
            <span className={`badge badge-${meta.tone}`}>
              <Icon name={meta.icon} size={12} /> {meta.label}
            </span>
          </div>

          <div
            className="small body-text mt-2 truncate"
            style={{ fontStyle: loading || !last ? 'italic' : 'normal' }}
          >
            {loading
              ? 'Memuat pesan…'
              : last
                ? `${mine ? 'Kamu: ' : ''}${last.body}`
                : 'Belum ada pesan. Mulai percakapan untuk menyepakati COD.'}
          </div>

          <div className="row-wrap mt-2" style={{ justifyContent: 'space-between' }}>
            <span className="tiny muted">
              <Icon name="event" size={12} /> {dateRange(rental.start_date, rental.end_date)} ·{' '}
              {formatRupiah(rental.total_price)}
            </span>
            <span className="tiny muted">
              {last ? timeAgo(last.created_at) : `diajukan ${timeAgo(rental.created_at)}`}
            </span>
          </div>
        </div>
      </div>
    </Link>
  )
}

export default function Messages() {
  const { user } = useAuth()
  const { data, loading, error, reload } = useApi(() => rentalApi.mine({ limit: 60 }), [])
  const [tab, setTab] = useState('active')
  const [search, setSearch] = useState('')

  const rentals = data?.items || []
  const active = rentals.filter((item) => ['pending', 'approved'].includes(item.status))

  const source = tab === 'active' ? active : rentals
  const shown = useMemo(() => {
    const q = search.trim().toLowerCase()
    return source
      .filter((item) => {
        if (!q) return true
        const other = (item.my_role === 'owner' ? item.renter : item.owner)?.name?.toLowerCase() || ''
        const title = item.listing?.title?.toLowerCase() || ''
        return other.includes(q) || title.includes(q)
      })
      .slice(0, 24)
  }, [source, search])

  return (
    <div className="container page">
      <PageHeader
        eyebrow="Pesan & Negosiasi"
        title="Percakapan"
        subtitle="Diskusikan waktu COD, kondisi barang, dan detail pengembalian dengan penyewa atau pemilik."
        actions={
          <Link to="/jelajahi" className="btn btn-outline">
            <Icon name="search" /> Cari Barang
          </Link>
        }
      />

      <div className="grid cols-3 mb-4">
        <div className="stat-card">
          <span className="stat-icon"><Icon name="forum" /></span>
          <div>
            <div className="stat-value">{active.length}</div>
            <div className="stat-label">Percakapan aktif</div>
          </div>
        </div>
        <div className="stat-card">
          <span className="stat-icon mint"><Icon name="receipt_long" /></span>
          <div>
            <div className="stat-value">{rentals.length}</div>
            <div className="stat-label">Total transaksi</div>
          </div>
        </div>
        <div className="stat-card">
          <span className="stat-icon yellow"><Icon name="handshake" /></span>
          <div>
            <div className="stat-value">
              {rentals.filter((item) => item.status === 'completed').length}
            </div>
            <div className="stat-label">Selesai</div>
          </div>
        </div>
      </div>

      <div className="between mb-4" style={{ flexWrap: 'wrap', gap: 10 }}>
        <div className="tabs">
          <button
            type="button"
            className={`tab ${tab === 'active' ? 'active' : ''}`.trim()}
            onClick={() => setTab('active')}
          >
            <Icon name="chat_bubble" size={15} /> Aktif ({active.length})
          </button>
          <button
            type="button"
            className={`tab ${tab === 'all' ? 'active' : ''}`.trim()}
            onClick={() => setTab('all')}
          >
            <Icon name="history" size={15} /> Semua ({rentals.length})
          </button>
        </div>

        <div className="input-wrap" style={{ minWidth: 220 }}>
          <Icon name="search" />
          <input
            className="input"
            value={search}
            onChange={(event) => setSearch(event.target.value)}
            placeholder="Cari nama atau barang…"
            aria-label="Cari percakapan"
          />
        </div>
      </div>

      {error ? <ErrorBox error={error} onRetry={reload} /> : null}

      {loading ? (
        <Loading label="Memuat percakapan…" />
      ) : shown.length ? (
        <div className="stack-sm">
          {shown.map((rental) => (
            <ConversationRow key={rental.id} rental={rental} userId={user?.id} />
          ))}
        </div>
      ) : (
        <Empty
          icon="chat_bubble_outline"
          title={tab === 'active' ? 'Belum ada percakapan aktif' : 'Belum ada percakapan'}
          action={
            <Link to="/penyewaan" className="btn btn-outline btn-sm">
              <Icon name="receipt_long" /> Lihat Penyewaan
            </Link>
          }
        >
          Percakapan muncul setelah kamu mengajukan sewa atau menerima pengajuan.
        </Empty>
      )}

      <div className="alert alert-info mt-4">
        <Icon name="shield" />
        <div className="alert-body">
          <strong>Jaga transaksi tetap di dalam platform</strong>
          Gunakan chat UNESA Rent agar riwayat kesepakatan tersimpan. Jangan membagikan data rekening
          atau identitas sebelum COD.
        </div>
      </div>
    </div>
  )
}
