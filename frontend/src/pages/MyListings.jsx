import { useState } from 'react'
import { Link } from '../router'
import { useAuth } from '../auth/useAuth'
import { catalogApi } from '../lib/api'
import { useApi } from '../lib/useApi'
import {
  categoryIcon,
  conditionLabel,
  dateRange,
  formatDate,
  formatRupiah,
  listingPhotoUrl,
  todayISO,
} from '../lib/format'
import {
  Alert,
  Empty,
  ErrorBox,
  Icon,
  Loading,
  Modal,
  PageHeader,
} from '../components/ui'

function BlocksModal({ listing, onClose, onChanged }) {
  const { data, loading, error, reload } = useApi(() => catalogApi.blocks(listing.id), [listing.id])
  const [start, setStart] = useState(todayISO())
  const [end, setEnd] = useState(todayISO())
  const [reason, setReason] = useState('')
  const [busy, setBusy] = useState(false)
  const [formError, setFormError] = useState(null)

  const blocks = data?.blocks || []

  const addBlock = async (event) => {
    event.preventDefault()
    setFormError(null)
    setBusy(true)
    try {
      await catalogApi.addBlock(listing.id, {
        start_date: start,
        end_date: end,
        reason: reason.trim(),
      })
      setReason('')
      reload()
      onChanged?.()
    } catch (err) {
      setFormError(err)
    } finally {
      setBusy(false)
    }
  }

  const removeBlock = async (blockId) => {
    setBusy(true)
    setFormError(null)
    try {
      await catalogApi.deleteBlock(listing.id, blockId)
      reload()
      onChanged?.()
    } catch (err) {
      setFormError(err)
    } finally {
      setBusy(false)
    }
  }

  return (
    <Modal
      title="Atur ketersediaan"
      subtitle={listing.title}
      onClose={onClose}
      footer={
        <button type="button" className="btn btn-ghost" onClick={onClose}>
          Tutup
        </button>
      }
    >
      <form className="stack" onSubmit={addBlock}>
        <div className="form-grid">
          <div className="field">
            <label className="label" htmlFor="block-start">Mulai</label>
            <input
              id="block-start"
              type="date"
              className="input"
              min={todayISO()}
              value={start}
              onChange={(event) => {
                setStart(event.target.value)
                if (event.target.value > end) setEnd(event.target.value)
              }}
              required
            />
          </div>
          <div className="field">
            <label className="label" htmlFor="block-end">Selesai</label>
            <input
              id="block-end"
              type="date"
              className="input"
              min={start}
              value={end}
              onChange={(event) => setEnd(event.target.value)}
              required
            />
          </div>
        </div>
        <div className="field">
          <label className="label" htmlFor="block-reason">Alasan (opsional)</label>
          <input
            id="block-reason"
            className="input"
            placeholder="Contoh: Dipakai sendiri untuk praktikum"
            value={reason}
            onChange={(event) => setReason(event.target.value)}
          />
        </div>
        <button type="submit" className="btn btn-primary" disabled={busy}>
          {busy ? <span className="spinner" /> : <Icon name="event_busy" />}
          Blokir tanggal
        </button>
        {formError ? <Alert tone="error" icon="error">{formError.message}</Alert> : null}
      </form>

      <hr className="divider" style={{ margin: '18px 0' }} />

      <h3 className="card-title mb-2">Periode diblokir</h3>
      {loading ? (
        <Loading label="Memuat periode…" />
      ) : error ? (
        <ErrorBox error={error} onRetry={reload} />
      ) : blocks.length ? (
        <div className="stack-sm">
          {blocks.map((block) => (
            <div className="block-row" key={block.id}>
              <div>
                <div className="strong">{dateRange(block.start_date, block.end_date)}</div>
                {block.reason ? <div className="tiny muted">{block.reason}</div> : null}
              </div>
              <button
                type="button"
                className="btn btn-danger-soft btn-sm"
                disabled={busy}
                onClick={() => removeBlock(block.id)}
              >
                <Icon name="delete" /> Hapus
              </button>
            </div>
          ))}
        </div>
      ) : (
        <p className="hint">Belum ada tanggal yang diblokir.</p>
      )}
    </Modal>
  )
}

export default function MyListings() {
  const { ktmVerified, ktmLoading } = useAuth()
  const { data, loading, error, reload } = useApi(() => catalogApi.mine({ limit: 60 }), [])

  const [filter, setFilter] = useState('all')
  const [feedback, setFeedback] = useState(null)
  const [deleteTarget, setDeleteTarget] = useState(null)
  const [blocksTarget, setBlocksTarget] = useState(null)
  const [deleting, setDeleting] = useState(false)
  const [deleteError, setDeleteError] = useState(null)

  const listings = data?.items || []
  const active = listings.filter((item) => item.is_active)
  const inactive = listings.filter((item) => !item.is_active)
  const shown =
    filter === 'active' ? active : filter === 'inactive' ? inactive : listings

  const confirmDelete = async () => {
    if (!deleteTarget) return
    setDeleting(true)
    setDeleteError(null)
    try {
      await catalogApi.remove(deleteTarget.id)
      setDeleteTarget(null)
      setFeedback({ tone: 'success', message: 'Listing berhasil dihapus permanen.' })
      reload()
    } catch (err) {
      setDeleteError(err)
    } finally {
      setDeleting(false)
    }
  }

  return (
    <div className="container page">
      <PageHeader
        eyebrow="Inventaris Penyedia"
        title="Barang saya"
        subtitle="Kelola status ketersediaan, edit detail, dan pantau barang yang kamu sewakan kepada sesama mahasiswa Ketintang."
        actions={
          <Link to="/barang-saya/baru" className="btn btn-primary">
            <Icon name="add_circle" /> Tambah Barang
          </Link>
        }
      />

      {!ktmLoading && !ktmVerified ? (
        <div className="notice-strip mb-4">
          <Icon name="verified_user" />
          <span>
            Kamu perlu verifikasi KTM sebelum bisa menambah listing.{' '}
            <Link to="/profil" style={{ color: 'var(--ink)', textDecoration: 'underline' }}>
              Verifikasi sekarang
            </Link>
          </span>
        </div>
      ) : null}

      {feedback ? (
        <div className="mb-4">
          <Alert tone={feedback.tone} icon={feedback.tone === 'success' ? 'check_circle' : 'error'}>
            {feedback.message}
          </Alert>
        </div>
      ) : null}

      <div className="between mb-4" style={{ flexWrap: 'wrap', gap: 10 }}>
        <div className="tabs">
          {[
            { value: 'all', label: `Semua (${listings.length})` },
            { value: 'active', label: `Aktif (${active.length})` },
            { value: 'inactive', label: `Tidak aktif (${inactive.length})` },
          ].map((tab) => (
            <button
              key={tab.value}
              type="button"
              className={`tab ${filter === tab.value ? 'active' : ''}`.trim()}
              onClick={() => setFilter(tab.value)}
            >
              {tab.label}
            </button>
          ))}
        </div>
        <span className="badge badge-muted">
          <Icon name="info" size={13} /> Maksimal 3 foto per barang
        </span>
      </div>

      {error ? <ErrorBox error={error} onRetry={reload} /> : null}

      {loading ? (
        <Loading label="Memuat barangmu…" />
      ) : shown.length ? (
        <div className="stack">
          {shown.map((listing) => {
            const photo = listingPhotoUrl(listing)
            return (
              <article className="card card-pad" key={listing.id}>
                <div className="row" style={{ alignItems: 'flex-start', gap: 14, flexWrap: 'wrap' }}>
                  <div
                    style={{
                      width: 104,
                      height: 84,
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
                      <img src={photo} alt={listing.title} style={{ width: '100%', height: '100%', objectFit: 'cover' }} />
                    ) : (
                      <Icon name={categoryIcon(listing.category?.slug)} size={32} />
                    )}
                  </div>

                  <div style={{ flex: 1, minWidth: 220 }}>
                    <div className="row-wrap">
                      <span className={`badge ${listing.is_active ? 'badge-success' : 'badge-danger'}`}>
                        {listing.is_active ? 'Aktif' : 'Tidak aktif'}
                      </span>
                      <span className="badge badge-muted">{listing.category?.name}</span>
                      <span className="badge badge-muted">Kondisi {conditionLabel(listing.condition_percent)}</span>
                    </div>
                    <h3 className="card-title mt-2">{listing.title}</h3>
                    <div className="tiny muted mt-1">
                      <Icon name="place" size={13} /> {listing.campus_location?.name} ·{' '}
                      <Icon name="calendar_today" size={13} /> Ditambahkan {formatDate(listing.created_at, 'short')}
                    </div>
                    <div className="price mt-2">
                      {formatRupiah(listing.price_per_day)} <small>/ hari</small>
                    </div>
                  </div>

                  <div className="row-wrap" style={{ alignSelf: 'center' }}>
                    {listing.is_active ? (
                      <Link to={`/barang/${listing.id}`} className="btn btn-ghost btn-sm">
                        <Icon name="visibility" /> Lihat
                      </Link>
                    ) : null}
                    <Link to={`/barang-saya/${listing.id}/edit`} className="btn btn-outline btn-sm">
                      <Icon name="edit" /> Edit
                    </Link>
                    <button
                      type="button"
                      className="btn btn-soft btn-sm"
                      onClick={() => setBlocksTarget(listing)}
                    >
                      <Icon name="event_busy" /> Ketersediaan
                    </button>
                    <button
                      type="button"
                      className="btn btn-danger-soft btn-sm"
                      onClick={() => {
                        setDeleteTarget(listing)
                        setDeleteError(null)
                      }}
                    >
                      <Icon name="delete" /> Hapus
                    </button>
                  </div>
                </div>

                {!listing.is_active && listing.takedown_reason ? (
                  <div className="alert alert-warning mt-3">
                    <Icon name="warning" />
                    <div className="alert-body">
                      <strong>Dinonaktifkan admin</strong>
                      {listing.takedown_reason}
                    </div>
                  </div>
                ) : null}
              </article>
            )
          })}
        </div>
      ) : (
        <Empty
          icon="inventory_2"
          title={filter === 'inactive' ? 'Tidak ada listing non-aktif' : 'Belum ada barang'}
          action={
            <Link to="/barang-saya/baru" className="btn btn-primary btn-sm">
              <Icon name="add_circle" /> Tambah Barang
            </Link>
          }
        >
          {filter === 'inactive'
            ? 'Semua listingmu aktif dan tampil di katalog.'
            : 'Sewakan barang yang jarang kamu pakai dan bantu teman satu fakultas.'}
        </Empty>
      )}

      {blocksTarget ? (
        <BlocksModal
          listing={blocksTarget}
          onClose={() => setBlocksTarget(null)}
          onChanged={reload}
        />
      ) : null}

      {deleteTarget ? (
        <Modal
          title="Hapus listing?"
          subtitle={deleteTarget.title}
          onClose={() => setDeleteTarget(null)}
          footer={
            <>
              <button type="button" className="btn btn-ghost" onClick={() => setDeleteTarget(null)}>
                Batal
              </button>
              <button type="button" className="btn btn-danger" disabled={deleting} onClick={confirmDelete}>
                {deleting ? <span className="spinner" /> : <Icon name="delete_forever" />}
                Hapus permanen
              </button>
            </>
          }
        >
          <p className="body-text">
            Listing akan dihapus permanen. Jika barang sudah pernah dipakai transaksi, penghapusan
            akan ditolak dan kamu bisa menonaktifkannya lewat admin.
          </p>
          {deleteError ? (
            <div className="mt-3">
              <Alert tone="error" icon="error" title="Gagal menghapus">
                {deleteError.message}
              </Alert>
            </div>
          ) : null}
        </Modal>
      ) : null}
    </div>
  )
}
