import { useEffect, useState } from 'react'
import { rentalApi } from '../lib/api'
import { useApi } from '../lib/useApi'
import { saveBlobUrl } from '../lib/http'
import {
  dateRange,
  fileSize,
  formatDateTime,
  formatRupiah,
  rentalStatus,
  timeAgo,
} from '../lib/format'
import { AdminShell } from '../components/AdminShell'
import {
  Alert,
  Empty,
  ErrorBox,
  Icon,
  Loading,
  Modal,
  StatCard,
} from '../components/ui'

const LIMIT = 10

const STATUS_FILTERS = [
  { value: '', label: 'Semua status' },
  { value: 'pending', label: 'Menunggu' },
  { value: 'approved', label: 'Disetujui' },
  { value: 'completed', label: 'Selesai' },
  { value: 'rejected', label: 'Ditolak' },
  { value: 'cancelled', label: 'Dibatalkan' },
]

const DOCUMENT_LABELS = {
  agreement: 'Surat Perjanjian',
  handover_proof: 'Bukti Serah Terima',
  return_proof: 'Bukti Pengembalian',
  cod_receipt: 'Kuitansi COD',
  other: 'Dokumen Tambahan',
}

function documentFilename(type) {
  switch (type) {
    case 'agreement': return 'surat-perjanjian-sewa.pdf'
    case 'cod_receipt': return 'kuitansi-cod.pdf'
    case 'handover_proof': return 'bukti-serah-terima.jpg'
    case 'return_proof': return 'bukti-pengembalian.jpg'
    default: return 'dokumen-rental.pdf'
  }
}

export default function AdminRentals() {
  const [status, setStatus] = useState('')
  const [from, setFrom] = useState('')
  const [to, setTo] = useState('')
  const [overdue, setOverdue] = useState(false)
  const [searchInput, setSearchInput] = useState('')
  const [query, setQuery] = useState('')
  const [offset, setOffset] = useState(0)

  const [detailID, setDetailID] = useState(null)
  const [tab, setTab] = useState('summary')
  const [busy, setBusy] = useState(false)
  const [modalError, setModalError] = useState(null)
  const [feedback, setFeedback] = useState(null)

  const [reason, setReason] = useState('')
  const [password, setPassword] = useState('')
  const [docTarget, setDocTarget] = useState(null)
  const [revealTarget, setRevealTarget] = useState(null)
  const [transcriptOpen, setTranscriptOpen] = useState(false)
  const [revealed, setRevealed] = useState({})

  useEffect(() => {
    const timer = setTimeout(() => {
      setQuery(searchInput.trim())
      setOffset(0)
    }, 350)
    return () => clearTimeout(timer)
  }, [searchInput])

  const statsQuery = useApi(() => rentalApi.adminStats(), [])
  const { data, loading, error, reload } = useApi(
    () => rentalApi.adminList({
      status,
      from,
      to,
      q: query,
      overdue: overdue ? 'true' : undefined,
      limit: LIMIT,
      offset,
    }),
    [status, from, to, query, overdue, offset],
  )

  const detailQuery = useApi(
    () => (detailID ? rentalApi.adminDetail(detailID) : Promise.resolve(null)),
    [detailID],
  )
  const documentsQuery = useApi(
    () => (detailID && tab === 'documents' ? rentalApi.adminDocuments(detailID) : Promise.resolve(null)),
    [detailID, tab],
  )
  const messagesQuery = useApi(
    () => (detailID && tab === 'chat' ? rentalApi.adminMessages(detailID, { limit: 50 }) : Promise.resolve(null)),
    [detailID, tab],
  )

  const items = data?.items || []
  const total = data?.total || 0
  const stats = statsQuery.data?.stats || {}
  const rental = detailQuery.data?.rental || null
  const documents = documentsQuery.data?.documents || []
  const messages = messagesQuery.data?.messages || []

  const changeFilter = (setter, value) => {
    setter(value)
    setOffset(0)
  }

  const openDetail = (id) => {
    setDetailID(id)
    setTab('summary')
    setRevealed({})
    setReason('')
    setPassword('')
    setModalError(null)
    setFeedback(null)
  }

  const closeDetail = () => {
    setDetailID(null)
    setDocTarget(null)
    setRevealTarget(null)
    setTranscriptOpen(false)
    setReason('')
    setPassword('')
    setModalError(null)
  }

  const validateReason = () => {
    if (reason.trim().length < 10) {
      setModalError({ message: 'Alasan wajib minimal 10 karakter.' })
      return false
    }
    return true
  }

  const validatePassword = () => {
    if (!password) {
      setModalError({ message: 'Kata sandi admin wajib diisi.' })
      return false
    }
    return true
  }

  const exportCSV = async () => {
    setBusy(true)
    setFeedback(null)
    try {
      const url = await rentalApi.adminExportBlobUrl({
        status,
        from,
        to,
        q: query,
        overdue: overdue ? 'true' : undefined,
      })
      saveBlobUrl(url, 'penyewaan.csv')
      setFeedback({ tone: 'success', message: 'Data penyewaan diekspor. Aktivitas ini tercatat di audit log.' })
    } catch (err) {
      setFeedback({ tone: 'error', message: err.message })
    } finally {
      setBusy(false)
    }
  }

  const downloadDocument = async (event) => {
    event.preventDefault()
    if (!docTarget || !validateReason()) return
    setBusy(true)
    setModalError(null)
    try {
      const url = await rentalApi.adminDocumentBlobUrl(detailID, docTarget.id, reason.trim())
      saveBlobUrl(url, documentFilename(docTarget.type))
      setDocTarget(null)
      setReason('')
      setFeedback({
        tone: 'success',
        message: `Dokumen ${DOCUMENT_LABELS[docTarget.type] || docTarget.type} diunduh. Aksi tercatat di audit log.`,
      })
    } catch (err) {
      setModalError(err)
    } finally {
      setBusy(false)
    }
  }

  const revealMessage = async (event) => {
    event.preventDefault()
    if (!revealTarget || !validateReason() || !validatePassword()) return
    setBusy(true)
    setModalError(null)
    try {
      const result = await rentalApi.revealMessage(detailID, revealTarget.id, {
        reason: reason.trim(),
        password,
      })
      setRevealed((prev) => ({ ...prev, [revealTarget.id]: result.body || '' }))
      setFeedback({
        tone: 'warning',
        message: 'Isi pesan ditampilkan. Akses ini tercatat di audit log.',
      })
      setRevealTarget(null)
      setReason('')
      setPassword('')
    } catch (err) {
      setModalError(err)
    } finally {
      setBusy(false)
    }
  }

  const exportTranscript = async (event) => {
    event.preventDefault()
    if (!validateReason() || !validatePassword()) return
    setBusy(true)
    setModalError(null)
    try {
      const url = await rentalApi.transcriptBlobUrl(detailID, {
        reason: reason.trim(),
        password,
      })
      saveBlobUrl(url, 'transkrip-chat.txt')
      setTranscriptOpen(false)
      setReason('')
      setPassword('')
      setFeedback({
        tone: 'warning',
        message: 'Transkrip diunduh. Akses ini tercatat di audit log.',
      })
    } catch (err) {
      setModalError(err)
    } finally {
      setBusy(false)
    }
  }

  return (
    <AdminShell
      eyebrow="Monitoring & Audit Transaksi"
      title="Monitoring penyewaan"
      subtitle="Pantau seluruh transaksi, dokumen COD, dan percakapan untuk keperluan moderasi. Semua akses sensitif tercatat."
      actions={
        <>
          <button type="button" className="btn btn-outline" disabled={busy} onClick={exportCSV}>
            <Icon name="download" /> Ekspor CSV
          </button>
          <span className="badge badge-muted">
            <Icon name="receipt_long" size={13} /> {total} transaksi
          </span>
        </>
      }
    >
      {feedback ? (
        <div className="mb-4">
          <Alert
            tone={feedback.tone}
            icon={feedback.tone === 'success' ? 'check_circle' : 'gavel'}
          >
            {feedback.message}
          </Alert>
        </div>
      ) : null}

      {statsQuery.error ? <ErrorBox error={statsQuery.error} onRetry={statsQuery.reload} title="Statistik gagal dimuat" /> : null}

      <div className="grid cols-4" style={{ marginBottom: 20 }}>
        <StatCard icon="receipt_long" value={stats.total || 0} label="Total transaksi" />
        <StatCard
          icon="hourglass_top"
          tone="yellow"
          value={stats.by_status?.pending || 0}
          label="Menunggu persetujuan"
          note={`${stats.by_status?.approved || 0} disetujui`}
        />
        <StatCard
          icon="event_busy"
          tone="danger"
          value={stats.overdue || 0}
          label="Terlambat"
          note="Perlu tindak lanjut"
        />
        <StatCard
          icon="description"
          tone="mint"
          value={stats.documents || 0}
          label="Dokumen"
          note={fileSize(stats.storage_bytes || 0)}
        />
      </div>

      <div className="between mb-4" style={{ flexWrap: 'wrap', gap: 10 }}>
        <div className="toolbar">
          <select
            className="select"
            value={status}
            onChange={(event) => changeFilter(setStatus, event.target.value)}
            aria-label="Filter status rental"
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
          <div className="input-wrap" style={{ minWidth: 220 }}>
            <Icon name="search" />
            <input
              className="input"
              value={searchInput}
              onChange={(event) => setSearchInput(event.target.value)}
              placeholder="Cari barang…"
              aria-label="Cari barang"
            />
          </div>
          <label className="row" style={{ gap: 6 }}>
            <input
              type="checkbox"
              checked={overdue}
              onChange={(event) => changeFilter(setOverdue, event.target.checked)}
            />
            <span className="tiny strong">Hanya yang terlambat</span>
          </label>
        </div>
      </div>

      {error ? <ErrorBox error={error} onRetry={reload} /> : null}

      {loading ? (
        <Loading label="Memuat penyewaan…" />
      ) : items.length ? (
        <>
          <div className="table-wrap">
            <table className="table">
              <thead>
                <tr>
                  <th>Barang</th>
                  <th>Penyewa</th>
                  <th>Penyedia</th>
                  <th>Periode</th>
                  <th>Total</th>
                  <th>Status</th>
                  <th>Dibuat</th>
                  <th>Aksi</th>
                </tr>
              </thead>
              <tbody>
                {items.map((item) => {
                  const meta = rentalStatus(item.status)
                  return (
                    <tr key={item.id}>
                      <td className="strong">{item.listing?.title}</td>
                      <td>{item.renter?.name}</td>
                      <td>{item.owner?.name}</td>
                      <td className="tiny muted">{dateRange(item.start_date, item.end_date)}</td>
                      <td className="strong">{formatRupiah(item.total_price)}</td>
                      <td>
                        <span className={`badge badge-${meta.tone}`}>{meta.label}</span>
                      </td>
                      <td className="tiny muted">{timeAgo(item.created_at)}</td>
                      <td>
                        <button type="button" className="btn btn-outline btn-sm" onClick={() => openDetail(item.id)}>
                          <Icon name="visibility" /> Detail
                        </button>
                      </td>
                    </tr>
                  )
                })}
              </tbody>
            </table>
          </div>

          <div className="between" style={{ marginTop: 14 }}>
            <span className="small muted">
              Menampilkan {items.length} dari {total} transaksi
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
        <Empty icon="receipt_long" title="Tidak ada transaksi">
          {query || status || from || to || overdue
            ? 'Tidak ada transaksi yang cocok dengan filter ini.'
            : 'Belum ada penyewaan yang tercatat.'}
        </Empty>
      )}

      <Alert tone="info" icon="lock" title="Akses data privat">
        Isi percakapan dan dokumen hanya ditampilkan setelah alasan serta kata sandi admin diverifikasi.
        Setiap akses tercatat di log audit.
      </Alert>

      {detailID ? (
        <Modal
          title="Detail transaksi"
          subtitle={`#${detailID.slice(0, 8).toUpperCase()}`}
          onClose={closeDetail}
          size="modal-lg"
        >
          <div className="tabs" style={{ marginBottom: 14 }}>
            {[
              { value: 'summary', label: 'Ringkasan' },
              { value: 'documents', label: 'Dokumen' },
              { value: 'chat', label: 'Chat' },
            ].map((item) => (
              <button
                key={item.value}
                type="button"
                className={`tab ${tab === item.value ? 'active' : ''}`.trim()}
                onClick={() => {
                  setTab(item.value)
                  setModalError(null)
                }}
              >
                {item.label}
              </button>
            ))}
          </div>

          {detailQuery.loading ? <Loading label="Memuat detail…" /> : null}
          {detailQuery.error ? <ErrorBox error={detailQuery.error} onRetry={detailQuery.reload} /> : null}

          {tab === 'summary' && rental ? (
            <div className="stack">
              <div className="card card-pad">
                <h3 className="card-title">{rental.listing?.title}</h3>
                <div className="grid cols-2 mt-3">
                  <div className="spec">
                    <span>Penyewa</span>
                    <strong>{rental.renter?.name}</strong>
                  </div>
                  <div className="spec">
                    <span>Penyedia</span>
                    <strong>{rental.owner?.name}</strong>
                  </div>
                  <div className="spec">
                    <span>Periode</span>
                    <strong>{dateRange(rental.start_date, rental.end_date)}</strong>
                  </div>
                  <div className="spec">
                    <span>Total</span>
                    <strong>{formatRupiah(rental.total_price)}</strong>
                  </div>
                </div>
                <div className="row-wrap mt-3">
                  <span className={`badge badge-${rentalStatus(rental.status).tone}`}>
                    {rentalStatus(rental.status).label}
                  </span>
                  <span className="tiny muted">Dibuat {formatDateTime(rental.created_at)}</span>
                </div>
              </div>

              <div>
                <h3 className="card-title">Riwayat status</h3>
                {rental.events?.length ? (
                  <div className="timeline mt-3">
                    {rental.events.map((event, index) => (
                      <div
                        key={event.id ?? index}
                        className={`timeline-item ${index === rental.events.length - 1 ? 'current' : 'done'}`}
                      >
                        <span className="timeline-dot" />
                        <div className="timeline-title">{event.note || rentalStatus(event.to_status).label}</div>
                        <div className="timeline-meta">{formatDateTime(event.created_at)}</div>
                      </div>
                    ))}
                  </div>
                ) : (
                  <p className="hint mt-2">Belum ada riwayat perubahan status.</p>
                )}
              </div>
            </div>
          ) : null}

          {tab === 'documents' ? (
            <div className="stack">
              <Alert tone="info" icon="description" title="Dokumen privat">
                Unduh dokumen memerlukan alasan minimal 10 karakter dan tercatat di audit log.
              </Alert>
              {documentsQuery.loading ? <Loading label="Memuat dokumen…" /> : null}
              {documentsQuery.error ? (
                <ErrorBox error={documentsQuery.error} onRetry={documentsQuery.reload} />
              ) : null}
              {!documentsQuery.loading && !documentsQuery.error && !documents.length ? (
                <Empty icon="description" title="Belum ada dokumen" />
              ) : null}
              {documents.map((doc) => (
                <div className="party" key={doc.id}>
                  <span className="stat-icon"><Icon name={doc.type === 'cod_receipt' ? 'receipt_long' : 'description'} /></span>
                  <div className="party-body">
                    <div className="party-name">{DOCUMENT_LABELS[doc.type] || doc.type}</div>
                    <div className="party-meta">
                      {formatDateTime(doc.created_at)} · {fileSize(doc.size_bytes)}
                    </div>
                  </div>
                  <button
                    type="button"
                    className="btn btn-outline btn-sm"
                    onClick={() => {
                      setDocTarget(doc)
                      setReason('')
                      setModalError(null)
                    }}
                  >
                    <Icon name="download" /> Unduh
                  </button>
                </div>
              ))}
            </div>
          ) : null}

          {tab === 'chat' ? (
            <div className="stack">
              <div className="between" style={{ flexWrap: 'wrap', gap: 8 }}>
                <Alert tone="info" icon="lock" title="Metadata saja">
                  Isi pesan sengaja disembunyikan. Buka satu pesan untuk melihat isinya dengan verifikasi.
                </Alert>
                <button
                  type="button"
                  className="btn btn-outline btn-sm"
                  onClick={() => {
                    setTranscriptOpen(true)
                    setReason('')
                    setPassword('')
                    setModalError(null)
                  }}
                >
                  <Icon name="download" /> Ekspor transkrip
                </button>
              </div>

              {messagesQuery.loading ? <Loading label="Memuat metadata chat…" /> : null}
              {messagesQuery.error ? (
                <ErrorBox error={messagesQuery.error} onRetry={messagesQuery.reload} />
              ) : null}

              {!messagesQuery.loading && !messagesQuery.error && !messages.length ? (
                <Empty icon="chat" title="Belum ada pesan" />
              ) : null}

              {messages.length ? (
                <div className="table-wrap">
                  <table className="table">
                    <thead>
                      <tr>
                        <th>Pengirim</th>
                        <th>Waktu</th>
                        <th>Panjang</th>
                        <th>Hash pendek</th>
                        <th>Aksi</th>
                      </tr>
                    </thead>
                    <tbody>
                      {messages.map((message) => (
                        <tr key={message.id}>
                          <td className="strong">{message.sender?.name}</td>
                          <td className="tiny muted">{formatDateTime(message.created_at)}</td>
                          <td>{message.body_length} karakter</td>
                          <td className="mono tiny">{message.body_sha256?.slice(0, 12) || '—'}</td>
                          <td>
                            {revealed[message.id] ? (
                              <div style={{ maxWidth: 260 }}>
                                <div className="body-text">“{revealed[message.id]}”</div>
                                <div className="tiny text-danger mt-1">Akses ini tercatat di audit log.</div>
                              </div>
                            ) : (
                              <button
                                type="button"
                                className="btn btn-outline btn-sm"
                                onClick={() => {
                                  setRevealTarget(message)
                                  setReason('')
                                  setPassword('')
                                  setModalError(null)
                                }}
                              >
                                <Icon name="visibility" /> Buka isi pesan
                              </button>
                            )}
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              ) : null}
            </div>
          ) : null}
        </Modal>
      ) : null}

      {docTarget ? (
        <Modal
          title="Unduh dokumen"
          subtitle={DOCUMENT_LABELS[docTarget.type] || docTarget.type}
          onClose={() => setDocTarget(null)}
          footer={
            <>
              <button type="button" className="btn btn-ghost" onClick={() => setDocTarget(null)}>Batal</button>
              <button type="submit" form="sensitive-form" className="btn btn-primary" disabled={busy}>
                {busy ? <span className="spinner" /> : <Icon name="download" />} Unduh
              </button>
            </>
          }
        >
          <form id="sensitive-form" className="stack" onSubmit={downloadDocument}>
            <Alert tone="warning" icon="gavel" title="Akses tercatat">
              Alasan dan identitas admin akan tersimpan pada log audit.
            </Alert>
            <div className="field">
              <label className="label" htmlFor="download-reason">Alasan unduh <span className="req">*</span></label>
              <textarea
                id="download-reason"
                className="textarea"
                rows={3}
                value={reason}
                onChange={(event) => setReason(event.target.value)}
                placeholder="Contoh: Meninjau bukti serah terima untuk laporan masuk."
              />
              <span className="hint">Minimal 10 karakter.</span>
            </div>
            {modalError ? <Alert tone="error" icon="error">{modalError.message}</Alert> : null}
          </form>
        </Modal>
      ) : null}

      {revealTarget ? (
        <Modal
          title="Buka isi pesan"
          subtitle={`Dikirim ${formatDateTime(revealTarget.created_at)} · ${revealTarget.sender?.name}`}
          onClose={() => setRevealTarget(null)}
          footer={
            <>
              <button type="button" className="btn btn-ghost" onClick={() => setRevealTarget(null)}>Batal</button>
              <button type="submit" form="sensitive-form" className="btn btn-primary" disabled={busy}>
                {busy ? <span className="spinner" /> : <Icon name="visibility" />} Buka Pesan
              </button>
            </>
          }
        >
          <form id="sensitive-form" className="stack" onSubmit={revealMessage}>
            <Alert tone="warning" icon="gavel" title="Akses sensitif">
              Password admin diverifikasi ulang. Setiap pembukaan tercatat di audit log.
            </Alert>
            <div className="field">
              <label className="label" htmlFor="reveal-reason">Alasan <span className="req">*</span></label>
              <textarea
                id="reveal-reason"
                className="textarea"
                rows={3}
                value={reason}
                onChange={(event) => setReason(event.target.value)}
                placeholder="Contoh: Memeriksa isi pesan terkait laporan penipuan."
              />
              <span className="hint">Minimal 10 karakter.</span>
            </div>
            <div className="field">
              <label className="label" htmlFor="reveal-password">Kata sandi admin <span className="req">*</span></label>
              <input
                id="reveal-password"
                className="input"
                type="password"
                value={password}
                onChange={(event) => setPassword(event.target.value)}
                autoComplete="current-password"
              />
            </div>
            {modalError ? <Alert tone="error" icon="error">{modalError.message}</Alert> : null}
          </form>
        </Modal>
      ) : null}

      {transcriptOpen ? (
        <Modal
          title="Ekspor transkrip chat"
          subtitle="Berkas transkrip dihasilkan dari seluruh pesan transaksi."
          onClose={() => setTranscriptOpen(false)}
          footer={
            <>
              <button type="button" className="btn btn-ghost" onClick={() => setTranscriptOpen(false)}>Batal</button>
              <button type="submit" form="sensitive-form" className="btn btn-primary" disabled={busy}>
                {busy ? <span className="spinner" /> : <Icon name="download" />} Ekspor
              </button>
            </>
          }
        >
          <form id="sensitive-form" className="stack" onSubmit={exportTranscript}>
            <Alert tone="warning" icon="gavel" title="Akses tercatat">
              Ekspor transkrip memerlukan kata sandi admin dan akan diaudit.
            </Alert>
            <div className="field">
              <label className="label" htmlFor="transcript-reason">Alasan <span className="req">*</span></label>
              <textarea
                id="transcript-reason"
                className="textarea"
                rows={3}
                value={reason}
                onChange={(event) => setReason(event.target.value)}
                placeholder="Contoh: Menyediakan bukti untuk peninjauan laporan."
              />
              <span className="hint">Minimal 10 karakter.</span>
            </div>
            <div className="field">
              <label className="label" htmlFor="transcript-password">Kata sandi admin <span className="req">*</span></label>
              <input
                id="transcript-password"
                className="input"
                type="password"
                value={password}
                onChange={(event) => setPassword(event.target.value)}
                autoComplete="current-password"
              />
            </div>
            {modalError ? <Alert tone="error" icon="error">{modalError.message}</Alert> : null}
          </form>
        </Modal>
      ) : null}
    </AdminShell>
  )
}
