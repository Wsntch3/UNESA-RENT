import { useState } from 'react'
import { Link } from '../router'
import { useAuth } from '../auth/useAuth'
import { catalogApi, rentalApi } from '../lib/api'
import { useApi } from '../lib/useApi'
import { dateRange, formatRupiah, rentalStatus, todayISO } from '../lib/format'
import { ListingCard } from '../components/ListingCard'
import {
  Alert,
  Avatar,
  Empty,
  ErrorBox,
  Icon,
  Loading,
  Modal,
  PageHeader,
  StatCard,
} from '../components/ui'

export default function Dashboard() {
  const { user, ktmVerified, ktmLoading } = useAuth()

  const {
    data: rentalData,
    loading: rentalsLoading,
    error: rentalsError,
    reload: reloadRentals,
  } = useApi(() => rentalApi.mine({ limit: 60 }), [])

  const {
    data: listingData,
    loading: listingsLoading,
    error: listingsError,
    reload: reloadListings,
  } = useApi(() => catalogApi.mine({ limit: 60 }), [])

  const [feedback, setFeedback] = useState(null)
  const [busyId, setBusyId] = useState(null)
  const [rejectTarget, setRejectTarget] = useState(null)
  const [reason, setReason] = useState('')
  const [rejectError, setRejectError] = useState(null)

  const rentals = rentalData?.items || []
  const listings = listingData?.items || []

  const incoming = rentals.filter((item) => item.my_role === 'owner' && item.status === 'pending')
  const activeOwner = rentals.filter((item) => item.my_role === 'owner' && item.status === 'approved')
  const activeRenter = rentals.filter(
    (item) => item.my_role === 'renter' && ['pending', 'approved'].includes(item.status),
  )
  const activeListings = listings.filter((item) => item.is_active)
  const completedRentals = rentals.filter((item) => item.status === 'completed')
  const overdueRentals = rentals.filter(
    (item) => item.status === 'approved' && item.end_date < todayISO(),
  )

  const refresh = async () => {
    await Promise.all([reloadRentals(), reloadListings()])
  }

  const approve = async (id) => {
    setBusyId(id)
    setFeedback(null)
    try {
      await rentalApi.approve(id)
      await refresh()
      setFeedback({ tone: 'success', message: 'Pengajuan sewa disetujui. Penyewa akan menerima notifikasi.' })
    } catch (err) {
      setFeedback({ tone: 'error', message: err.message })
    } finally {
      setBusyId(null)
    }
  }

  const submitReject = async (event) => {
    event.preventDefault()
    if (!rejectTarget) return
    if (!reason.trim()) {
      setRejectError({ message: 'Alasan penolakan wajib diisi.' })
      return
    }
    setBusyId(rejectTarget.id)
    setRejectError(null)
    try {
      await rentalApi.reject(rejectTarget.id, reason.trim())
      setRejectTarget(null)
      setReason('')
      await refresh()
      setFeedback({ tone: 'success', message: 'Pengajuan ditolak dan penyewa diberi tahu.' })
    } catch (err) {
      setRejectError(err)
    } finally {
      setBusyId(null)
    }
  }

  const loading = rentalsLoading || listingsLoading

  return (
    <div className="container page">
      <PageHeader
        eyebrow="Dashboard Penyewaan"
        title={`Halo, ${user?.name?.split(' ')[0] || 'Mahasiswa'}!`}
        subtitle="Ringkasan aktivitas penyewaan barang dan inventaris milikmu di ekosistem FT UNESA."
        actions={
          <>
            <Link to="/barang-saya/baru" className="btn btn-primary">
              <Icon name="add_circle" /> Tambah Barang
            </Link>
            <Link to="/penyewaan" className="btn btn-outline">
              <Icon name="receipt_long" /> Penyewaan Saya
            </Link>
          </>
        }
      />

      {!ktmLoading && !ktmVerified ? (
        <div className="notice-strip mb-4">
          <Icon name="verified_user" />
          <span>
            Akunmu belum terverifikasi KTM. Kamu belum bisa membuat listing atau mengajukan sewa.{' '}
            <Link to="/profil" style={{ color: 'var(--ink)', textDecoration: 'underline' }}>
              Verifikasi sekarang
            </Link>
          </span>
        </div>
      ) : null}

      {feedback ? (
        <div className="mb-4">
          <Alert
            tone={feedback.tone}
            icon={feedback.tone === 'success' ? 'check_circle' : 'error'}
          >
            {feedback.message}
          </Alert>
        </div>
      ) : null}

      <div className="grid cols-4" style={{ marginBottom: 22 }}>
        <StatCard
          icon="inventory_2"
          value={activeListings.length}
          label="Listing aktif"
          note={`${listings.length - activeListings.length} non-aktif`}
        />
        <StatCard
          icon="inbox"
          tone="yellow"
          value={incoming.length}
          label="Pengajuan baru"
          note="Butuh responmu"
        />
        <StatCard
          icon="sync_alt"
          tone="mint"
          value={activeOwner.length}
          label="Penyewaan berlangsung"
          note="Sebagai penyedia"
        />
        <StatCard
          icon="shopping_bag"
          value={activeRenter.length}
          label="Sewa aktifku"
          note="Sebagai penyewa"
        />
      </div>

      <div className="grid cols-3" style={{ marginBottom: 22 }}>
        <StatCard
          icon="sync_alt"
          tone="mint"
          value={activeOwner.length + activeRenter.length}
          label="Rental berlangsung"
          note="Sewa aktif dan disetujui"
        />
        <StatCard
          icon="task_alt"
          value={completedRentals.length}
          label="Rental selesai"
          note="Transaksi sudah ditutup"
        />
        <StatCard
          icon="event_busy"
          tone="danger"
          value={overdueRentals.length}
          label="Terlambat dikembalikan"
          note="Perlu ditindaklanjuti"
        />
      </div>

      {rentalsError ? <ErrorBox error={rentalsError} onRetry={reloadRentals} /> : null}
      {listingsError ? <ErrorBox error={listingsError} onRetry={reloadListings} /> : null}

      {loading ? (
        <Loading />
      ) : (
        <div className="grid cols-2" style={{ alignItems: 'start', gap: 20 }}>
          <section>
            <div className="between" style={{ marginBottom: 12 }}>
              <h2 className="section-title">
                Pengajuan masuk{' '}
                {incoming.length ? <span className="badge badge-yellow">{incoming.length} baru</span> : null}
              </h2>
              <Link to="/penyewaan" className="btn btn-ghost btn-sm">
                Semua <Icon name="arrow_forward" />
              </Link>
            </div>

            {incoming.length ? (
              <div className="stack">
                {incoming.map((item) => {
                  const status = rentalStatus(item.status)
                  return (
                    <article className="card card-pad" key={item.id}>
                      <div className="between">
                        <div className="row">
                          <Avatar name={item.renter?.name} />
                          <div>
                            <div className="card-title">{item.renter?.name}</div>
                            <div className="tiny muted">
                              {dateRange(item.start_date, item.end_date)} · {item.total_days} hari
                            </div>
                          </div>
                        </div>
                        <span className={`badge badge-${status.tone}`}>{status.label}</span>
                      </div>

                      <div className="party mt-3">
                        <span className="stat-icon">
                          <Icon name="inventory_2" />
                        </span>
                        <div className="party-body">
                          <div className="party-name">{item.listing?.title}</div>
                          <div className="party-meta">
                            Total sewa <strong>{formatRupiah(item.total_price)}</strong> ·{' '}
                            {formatRupiah(item.price_per_day_snapshot)}/hari
                          </div>
                        </div>
                      </div>

                      <div className="row-wrap mt-3">
                        <button
                          type="button"
                          className="btn btn-primary btn-sm"
                          disabled={busyId === item.id}
                          onClick={() => approve(item.id)}
                        >
                          {busyId === item.id ? <span className="spinner" /> : <Icon name="check_circle" />}
                          Setujui
                        </button>
                        <button
                          type="button"
                          className="btn btn-danger-soft btn-sm"
                          disabled={busyId === item.id}
                          onClick={() => {
                            setRejectTarget(item)
                            setReason('')
                            setRejectError(null)
                          }}
                        >
                          <Icon name="cancel" /> Tolak
                        </button>
                        <Link to={`/penyewaan/${item.id}`} className="btn btn-outline btn-sm">
                          <Icon name="visibility" /> Detail
                        </Link>
                        <Link to={`/penyewaan/${item.id}?chat=1`} className="btn btn-ghost btn-sm">
                          <Icon name="chat" /> Chat
                        </Link>
                      </div>
                    </article>
                  )
                })}
              </div>
            ) : (
              <Empty icon="inbox" title="Tidak ada pengajuan baru">
                Pengajuan sewa dari mahasiswa lain akan muncul di sini.
              </Empty>
            )}
          </section>

          <section className="stack">
            <div>
              <div className="between" style={{ marginBottom: 12 }}>
                <h2 className="section-title">Listing saya</h2>
                <Link to="/barang-saya" className="btn btn-ghost btn-sm">
                  Kelola <Icon name="arrow_forward" />
                </Link>
              </div>

              {listings.length ? (
                <div className="stack-sm">
                  {listings.slice(0, 4).map((listing) => (
                    <Link
                      key={listing.id}
                      to={`/barang-saya/${listing.id}/edit`}
                      className="card card-pad row"
                      style={{ textDecoration: 'none', color: 'inherit', gap: 12 }}
                    >
                      <span className="stat-icon">
                        <Icon name="inventory_2" />
                      </span>
                      <div style={{ flex: 1, minWidth: 0 }}>
                        <div className="card-title truncate">{listing.title}</div>
                        <div className="tiny muted">
                          {formatRupiah(listing.price_per_day)}/hari ·{' '}
                          {listing.is_active ? 'Aktif' : 'Tidak aktif'}
                        </div>
                      </div>
                      <Icon name="chevron_right" className="muted" />
                    </Link>
                  ))}
                </div>
              ) : (
                <Empty
                  icon="add_box"
                  title="Belum punya listing"
                  action={
                    <Link to="/barang-saya/baru" className="btn btn-primary btn-sm">
                      Tambah Barang
                    </Link>
                  }
                >
                  Mulai sewakan barang yang jarang kamu pakai.
                </Empty>
              )}
            </div>

            <div>
              <div className="between" style={{ marginBottom: 12 }}>
                <h2 className="section-title">Sewa aktifku</h2>
                <Link to="/penyewaan" className="btn btn-ghost btn-sm">
                  Semua <Icon name="arrow_forward" />
                </Link>
              </div>
              {activeRenter.length ? (
                <div className="stack-sm">
                  {activeRenter.slice(0, 3).map((item) => {
                    const status = rentalStatus(item.status)
                    return (
                      <Link
                        key={item.id}
                        to={`/penyewaan/${item.id}`}
                        className="card card-pad"
                        style={{ textDecoration: 'none', color: 'inherit' }}
                      >
                        <div className="between">
                          <div className="card-title truncate">{item.listing?.title}</div>
                          <span className={`badge badge-${status.tone}`}>{status.label}</span>
                        </div>
                        <div className="tiny muted mt-1">
                          {dateRange(item.start_date, item.end_date)} · {formatRupiah(item.total_price)}
                        </div>
                      </Link>
                    )
                  })}
                </div>
              ) : (
                <Empty icon="shopping_bag" title="Belum ada sewa aktif">
                  <Link to="/jelajahi">Jelajahi barang</Link> untuk mulai menyewa.
                </Empty>
              )}
            </div>
          </section>
        </div>
      )}

      {rejectTarget ? (
        <Modal
          title="Tolak pengajuan sewa"
          subtitle={`${rejectTarget.renter?.name} · ${rejectTarget.listing?.title}`}
          onClose={() => setRejectTarget(null)}
          footer={
            <>
              <button type="button" className="btn btn-ghost" onClick={() => setRejectTarget(null)}>
                Batal
              </button>
              <button
                type="submit"
                form="reject-form"
                className="btn btn-danger"
                disabled={busyId === rejectTarget.id}
              >
                {busyId === rejectTarget.id ? <span className="spinner" /> : <Icon name="cancel" />}
                Tolak Pengajuan
              </button>
            </>
          }
        >
          <form id="reject-form" className="stack" onSubmit={submitReject}>
            <div className="field">
              <label className="label" htmlFor="reject-reason">Alasan penolakan <span className="req">*</span></label>
              <textarea
                id="reject-reason"
                className="textarea"
                placeholder="Contoh: Barang sedang dipakai untuk praktikum minggu depan."
                value={reason}
                onChange={(event) => setReason(event.target.value)}
              />
            </div>
            {rejectError ? <Alert tone="error" icon="error">{rejectError.message}</Alert> : null}
          </form>
        </Modal>
      ) : null}

      {listings.length && activeListings.length === 0 ? (
        <p className="hint mt-4">
          Semua listingmu sedang non-aktif. Aktifkan kembali lewat halaman <Link to="/barang-saya">Barang Saya</Link>.
        </p>
      ) : null}
    </div>
  )
}
