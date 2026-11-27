import { useEffect, useMemo, useState } from 'react'
import { catalogApi } from '../lib/api'
import { useApi } from '../lib/useApi'
import {
  categoryIcon,
  conditionLabel,
  formatDate,
  formatRupiah,
  listingPhotoUrl,
} from '../lib/format'
import { AdminShell } from '../components/AdminShell'
import {
  Alert,
  Empty,
  ErrorBox,
  Icon,
  Loading,
  Modal,
} from '../components/ui'

const LIMIT = 10

const TABS = [
  { value: '', label: 'Semua' },
  { value: 'active', label: 'Aktif' },
  { value: 'taken_down', label: 'Diturunkan' },
]

export default function AdminListings() {
  const [status, setStatus] = useState('')
  const [searchInput, setSearchInput] = useState('')
  const [query, setQuery] = useState('')
  const [offset, setOffset] = useState(0)

  const [takedownTarget, setTakedownTarget] = useState(null)
  const [restoreTarget, setRestoreTarget] = useState(null)
  const [reason, setReason] = useState('')
  const [busy, setBusy] = useState(false)
  const [modalError, setModalError] = useState(null)
  const [feedback, setFeedback] = useState(null)
  const [actionError, setActionError] = useState(null)

  useEffect(() => {
    const timer = setTimeout(() => {
      setQuery(searchInput.trim())
      setOffset(0)
    }, 350)
    return () => clearTimeout(timer)
  }, [searchInput])

  const { data, loading, error, reload } = useApi(
    () => catalogApi.adminList({ status, q: query, limit: LIMIT, offset }),
    [status, query, offset],
  )

  const items = useMemo(() => data?.items || [], [data])
  const total = data?.total || 0

  const submitTakedown = async (event) => {
    event.preventDefault()
    if (!takedownTarget) return
    if (!reason.trim()) {
      setModalError({ message: 'Alasan penurunan wajib diisi.' })
      return
    }

    setBusy(true)
    setModalError(null)
    setActionError(null)
    try {
      await catalogApi.takedown(takedownTarget.id, reason.trim())
      setFeedback({
        tone: 'success',
        message: `Listing “${takedownTarget.title}” diturunkan dari katalog.`,
      })
      setTakedownTarget(null)
      setReason('')
      await reload()
    } catch (err) {
      setModalError(err)
    } finally {
      setBusy(false)
    }
  }

  const submitRestore = async () => {
    if (!restoreTarget) return
    setBusy(true)
    setActionError(null)
    try {
      await catalogApi.restore(restoreTarget.id)
      setFeedback({
        tone: 'success',
        message: `Listing “${restoreTarget.title}” dipulihkan dan tampil kembali di katalog.`,
      })
      setRestoreTarget(null)
      await reload()
    } catch (err) {
      setActionError(err)
    } finally {
      setBusy(false)
    }
  }

  return (
    <AdminShell
      eyebrow="Moderasi Komunitas & Listing"
      title="Moderasi listing barang"
      subtitle="Pantau kualitas dan kepatuhan barang yang disewakan. Turunkan listing yang melanggar ketentuan komunitas."
      actions={
        <span className="badge badge-muted">
          <Icon name="inventory_2" size={13} /> {total} listing
        </span>
      }
    >
      <div className="between mb-4" style={{ flexWrap: 'wrap', gap: 10 }}>
        <div className="tabs">
          {TABS.map((tab) => (
            <button
              key={tab.value}
              type="button"
              className={`tab ${status === tab.value ? 'active' : ''}`.trim()}
              onClick={() => {
                setStatus(tab.value)
                setOffset(0)
              }}
            >
              {tab.label}
            </button>
          ))}
        </div>

        <div className="input-wrap" style={{ minWidth: 240 }}>
          <Icon name="search" />
          <input
            className="input"
            value={searchInput}
            onChange={(event) => setSearchInput(event.target.value)}
            placeholder="Cari judul barang atau penyedia…"
            aria-label="Cari listing"
          />
        </div>
      </div>

      {feedback ? (
        <div className="mb-4">
          <Alert tone={feedback.tone} icon="check_circle">{feedback.message}</Alert>
        </div>
      ) : null}

      {actionError ? (
        <div className="mb-4">
          <ErrorBox error={actionError} title="Aksi gagal" onRetry={reload} />
        </div>
      ) : null}

      {error ? <ErrorBox error={error} onRetry={reload} /> : null}

      {loading ? (
        <Loading label="Memuat listing…" />
      ) : items.length ? (
        <>
          <div className="table-wrap">
            <table className="table">
              <thead>
                <tr>
                  <th>Barang</th>
                  <th>Penyedia</th>
                  <th>Kategori</th>
                  <th>Harga / hari</th>
                  <th>Status</th>
                  <th>Ditambahkan</th>
                  <th>Aksi</th>
                </tr>
              </thead>
              <tbody>
                {items.map((listing) => {
                  const photo = listingPhotoUrl(listing)
                  return (
                    <tr key={listing.id}>
                      <td>
                        <div className="row" style={{ gap: 10, alignItems: 'flex-start' }}>
                          <div
                            style={{
                              width: 52,
                              height: 44,
                              borderRadius: 8,
                              overflow: 'hidden',
                              background: 'linear-gradient(135deg,#eef1ff,#fff3d1)',
                              display: 'grid',
                              placeItems: 'center',
                              color: 'var(--primary)',
                              flexShrink: 0,
                            }}
                          >
                            {photo ? (
                              <img
                                src={photo}
                                alt={listing.title}
                                style={{ width: '100%', height: '100%', objectFit: 'cover' }}
                              />
                            ) : (
                              <Icon name={categoryIcon(listing.category?.slug)} size={20} />
                            )}
                          </div>
                          <div style={{ minWidth: 160 }}>
                            <div className="strong" style={{ maxWidth: 240 }}>{listing.title}</div>
                            <div className="tiny muted">
                              Kondisi {conditionLabel(listing.condition_percent)} · ID{' '}
                              {listing.id.slice(0, 8)}
                            </div>
                            {!listing.is_active && listing.takedown_reason ? (
                              <div className="tiny text-danger mt-1" style={{ maxWidth: 240 }}>
                                {listing.takedown_reason}
                              </div>
                            ) : null}
                          </div>
                        </div>
                      </td>
                      <td>
                        <div className="strong">{listing.owner?.name}</div>
                        <div className="tiny muted">{listing.campus_location?.name}</div>
                      </td>
                      <td>{listing.category?.name}</td>
                      <td className="strong">{formatRupiah(listing.price_per_day)}</td>
                      <td>
                        <span className={`badge ${listing.is_active ? 'badge-success' : 'badge-danger'}`}>
                          {listing.is_active ? 'Aktif' : 'Diturunkan'}
                        </span>
                      </td>
                      <td className="tiny muted">{formatDate(listing.created_at, 'short')}</td>
                      <td>
                        {listing.is_active ? (
                          <button
                            type="button"
                            className="btn btn-danger-soft btn-sm"
                            onClick={() => {
                              setTakedownTarget(listing)
                              setReason('')
                              setModalError(null)
                            }}
                          >
                            <Icon name="visibility_off" /> Turunkan
                          </button>
                        ) : (
                          <button
                            type="button"
                            className="btn btn-success-soft btn-sm"
                            onClick={() => setRestoreTarget(listing)}
                          >
                            <Icon name="restore" /> Pulihkan
                          </button>
                        )}
                      </td>
                    </tr>
                  )
                })}
              </tbody>
            </table>
          </div>

          <div className="between" style={{ marginTop: 14 }}>
            <span className="small muted">
              Menampilkan {items.length} dari {total} listing
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
        <Empty icon="inventory_2" title="Tidak ada listing">
          {query
            ? `Tidak ada listing yang cocok dengan “${query}”.`
            : 'Belum ada listing pada filter ini.'}
        </Empty>
      )}

      <Alert tone="info" icon="gavel" title="Kebijakan moderasi">
        Turunkan listing yang melanggar ketentuan (barang terlarang, foto tidak pantas, atau deskripsi
        menyesatkan). Sertakan alasan yang jelas agar penyedia dapat memperbaiki.
      </Alert>

      {takedownTarget ? (
        <Modal
          title="Turunkan listing"
          subtitle={takedownTarget.title}
          onClose={() => setTakedownTarget(null)}
          footer={
            <>
              <button type="button" className="btn btn-ghost" onClick={() => setTakedownTarget(null)}>Batal</button>
              <button type="submit" form="takedown-form" className="btn btn-danger" disabled={busy}>
                {busy ? <span className="spinner" /> : <Icon name="visibility_off" />} Turunkan
              </button>
            </>
          }
        >
          <form id="takedown-form" className="stack" onSubmit={submitTakedown}>
            <Alert tone="warning" icon="warning" title="Listing disembunyikan dari katalog">
              Penyedia tetap dapat melihat listing dan alasan penurunan di halaman Barang Saya.
            </Alert>
            <div className="field">
              <label className="label" htmlFor="takedown-reason">Alasan penurunan <span className="req">*</span></label>
              <textarea
                id="takedown-reason"
                className="textarea"
                value={reason}
                onChange={(event) => setReason(event.target.value)}
                placeholder="Contoh: Foto barang tidak sesuai atau deskripsi menyesatkan."
              />
            </div>
            {modalError ? <Alert tone="error" icon="error">{modalError.message}</Alert> : null}
          </form>
        </Modal>
      ) : null}

      {restoreTarget ? (
        <Modal
          title="Pulihkan listing"
          subtitle={restoreTarget.title}
          onClose={() => setRestoreTarget(null)}
          footer={
            <>
              <button type="button" className="btn btn-ghost" onClick={() => setRestoreTarget(null)}>Batal</button>
              <button type="button" className="btn btn-primary" disabled={busy} onClick={submitRestore}>
                {busy ? <span className="spinner" /> : <Icon name="restore" />} Pulihkan
              </button>
            </>
          }
        >
          <p className="body-text">
            Listing akan tampil kembali di katalog publik jika penyedia masih berstatus mahasiswa aktif
            dengan KTM terverifikasi.
          </p>
        </Modal>
      ) : null}
    </AdminShell>
  )
}
