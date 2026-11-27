import { useState } from 'react'
import { Link } from '../router'
import { reportApi } from '../lib/api'
import { useApi } from '../lib/useApi'
import { formatDateTime, timeAgo } from '../lib/format'
import { AdminShell } from '../components/AdminShell'
import {
  Alert,
  Avatar,
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
  { value: 'open', label: 'Terbuka' },
  { value: 'in_review', label: 'Ditinjau' },
  { value: 'resolved', label: 'Diselesaikan' },
  { value: 'rejected', label: 'Ditolak' },
]

const TYPE_FILTERS = [
  { value: '', label: 'Semua jenis' },
  { value: 'technical', label: 'Teknis' },
  { value: 'safety', label: 'Keamanan' },
  { value: 'fraud', label: 'Penipuan' },
  { value: 'harassment', label: 'Pelecehan' },
  { value: 'other', label: 'Lainnya' },
]

const TYPE_LABELS = Object.fromEntries(TYPE_FILTERS.map((item) => [item.value, item.label]))

const STATUS_META = {
  open: { label: 'Terbuka', tone: 'warning' },
  in_review: { label: 'Ditinjau', tone: 'primary' },
  resolved: { label: 'Diselesaikan', tone: 'success' },
  rejected: { label: 'Ditolak', tone: 'danger' },
}

function statusMeta(status) {
  return STATUS_META[status] || { label: status || '—', tone: 'muted' }
}

export default function AdminReports() {
  const [status, setStatus] = useState('')
  const [type, setType] = useState('')
  const [offset, setOffset] = useState(0)

  const [selected, setSelected] = useState(null)
  const [decision, setDecision] = useState(null)
  const [note, setNote] = useState('')
  const [busy, setBusy] = useState(false)
  const [modalError, setModalError] = useState(null)
  const [feedback, setFeedback] = useState(null)

  const statsQuery = useApi(async () => {
    const [open, inReview, resolved] = await Promise.all([
      reportApi.adminList({ status: 'open', limit: 1 }),
      reportApi.adminList({ status: 'in_review', limit: 1 }),
      reportApi.adminList({ status: 'resolved', limit: 1 }),
    ])
    return {
      open: open.total || 0,
      inReview: inReview.total || 0,
      resolved: resolved.total || 0,
    }
  }, [])

  const { data, loading, error, reload } = useApi(
    () => reportApi.adminList({ status, type, limit: LIMIT, offset }),
    [status, type, offset],
  )

  const detailQuery = useApi(
    () => (selected ? reportApi.detail(selected.id) : Promise.resolve(null)),
    [selected?.id],
  )

  const reports = data?.items || []
  const total = data?.total || 0
  const stats = statsQuery.data || {}
  const report = detailQuery.data?.report || null

  const changeFilter = (setter, value) => {
    setter(value)
    setOffset(0)
  }

  const openDetail = (item) => {
    setSelected(item)
    setDecision(null)
    setNote('')
    setModalError(null)
    setFeedback(null)
  }

  const closeDetail = () => {
    setSelected(null)
    setDecision(null)
    setNote('')
    setModalError(null)
  }

  const submitDecision = async (event) => {
    event.preventDefault()
    if (!selected) return
    if (note.trim().length < 10) {
      setModalError({ message: 'Catatan penyelesaian wajib minimal 10 karakter.' })
      return
    }

    setBusy(true)
    setModalError(null)
    try {
      await reportApi[decision](selected.id, note.trim())
      closeDetail()
      setFeedback({
        tone: 'success',
        message: decision === 'resolve'
          ? 'Laporan diselesaikan. Pelapor menerima notifikasi.'
          : 'Laporan ditolak. Pelapor menerima notifikasi.',
      })
      await Promise.all([reload(), statsQuery.reload()])
    } catch (err) {
      setModalError(err)
    } finally {
      setBusy(false)
    }
  }

  return (
    <AdminShell
      eyebrow="Peninjauan Laporan Mahasiswa"
      title="Antrean laporan"
      subtitle="Tinjau laporan teknis, keamanan, dan pelanggaran komunitas. Berikan keputusan beserta catatan yang jelas."
      actions={
        <>
          <Link to="/admin/users" className="btn btn-outline">
            <Icon name="groups" /> Kelola Pengguna
          </Link>
          <span className="badge badge-muted">
            <Icon name="flag" size={13} /> {total} laporan
          </span>
        </>
      }
    >
      {feedback ? (
        <div className="mb-4">
          <Alert tone="success" icon="check_circle">{feedback.message}</Alert>
        </div>
      ) : null}

      {statsQuery.error ? <ErrorBox error={statsQuery.error} onRetry={statsQuery.reload} title="Statistik gagal dimuat" /> : null}

      <div className="grid cols-3" style={{ marginBottom: 20 }}>
        <StatCard icon="flag" tone="yellow" value={stats.open || 0} label="Laporan terbuka" />
        <StatCard icon="hourglass_top" value={stats.inReview || 0} label="Sedang ditinjau" />
        <StatCard icon="task_alt" tone="mint" value={stats.resolved || 0} label="Diselesaikan" />
      </div>

      <div className="between mb-4" style={{ flexWrap: 'wrap', gap: 10 }}>
        <div className="toolbar">
          <select
            className="select"
            value={status}
            onChange={(event) => changeFilter(setStatus, event.target.value)}
            aria-label="Filter status laporan"
          >
            {STATUS_FILTERS.map((option) => (
              <option key={option.value} value={option.value}>{option.label}</option>
            ))}
          </select>
          <select
            className="select"
            value={type}
            onChange={(event) => changeFilter(setType, event.target.value)}
            aria-label="Filter jenis laporan"
          >
            {TYPE_FILTERS.map((option) => (
              <option key={option.value} value={option.value}>{option.label}</option>
            ))}
          </select>
        </div>
      </div>

      {error ? <ErrorBox error={error} onRetry={reload} /> : null}

      {loading ? (
        <Loading label="Memuat laporan…" />
      ) : reports.length ? (
        <>
          <div className="table-wrap">
            <table className="table">
              <thead>
                <tr>
                  <th>Judul</th>
                  <th>Jenis</th>
                  <th>Pelapor</th>
                  <th>Status</th>
                  <th>Dibuat</th>
                  <th>Aksi</th>
                </tr>
              </thead>
              <tbody>
                {reports.map((item) => {
                  const meta = statusMeta(item.status)
                  return (
                    <tr key={item.id}>
                      <td>
                        <div className="strong">{item.subject}</div>
                        <div className="tiny muted truncate" style={{ maxWidth: 320 }}>{item.body}</div>
                      </td>
                      <td><span className="badge badge-muted">{TYPE_LABELS[item.type] || item.type}</span></td>
                      <td>{item.reporter_name}</td>
                      <td><span className={`badge badge-${meta.tone}`}>{meta.label}</span></td>
                      <td className="tiny muted">{timeAgo(item.created_at)}</td>
                      <td>
                        <button type="button" className="btn btn-outline btn-sm" onClick={() => openDetail(item)}>
                          <Icon name="visibility" /> Tinjau
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
              Menampilkan {reports.length} dari {total} laporan
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
        <Empty icon="flag" title="Tidak ada laporan">
          {status || type
            ? 'Tidak ada laporan yang cocok dengan filter ini.'
            : 'Antrean laporan kosong. Semua laporan mahasiswa akan muncul di sini.'}
        </Empty>
      )}

      {selected ? (
        <Modal
          title="Tinjau laporan"
          subtitle={`#${selected.id.slice(0, 8).toUpperCase()}`}
          onClose={closeDetail}
          size="modal-lg"
        >
          {detailQuery.loading ? <Loading label="Memuat laporan…" /> : null}
          {detailQuery.error ? <ErrorBox error={detailQuery.error} onRetry={detailQuery.reload} /> : null}

          {report ? (
            <div className="stack">
              <div className="between" style={{ flexWrap: 'wrap', gap: 8 }}>
                <div>
                  <h3 className="card-title">{report.subject}</h3>
                  <div className="row mt-2">
                    <Avatar name={report.reporter_name} size="sm" />
                    <span className="tiny muted">
                      {report.reporter_name} · {formatDateTime(report.created_at)}
                    </span>
                  </div>
                </div>
                <div className="row-wrap">
                  <span className="badge badge-muted">{TYPE_LABELS[report.type] || report.type}</span>
                  <span className={`badge badge-${statusMeta(report.status).tone}`}>
                    {statusMeta(report.status).label}
                  </span>
                </div>
              </div>

              <div className="card card-pad">
                <span className="eyebrow">Isi laporan</span>
                <p className="body-text mt-2" style={{ whiteSpace: 'pre-wrap' }}>{report.body}</p>
              </div>

              <div className="grid cols-2">
                <div className="spec">
                  <span>Barang terkait</span>
                  <strong>
                    {report.listing_id ? (
                      <Link to={`/barang/${report.listing_id}`}>Lihat listing</Link>
                    ) : (
                      'Tidak ada'
                    )}
                  </strong>
                </div>
                <div className="spec">
                  <span>Sewa terkait</span>
                  <strong className="mono tiny">{report.rental_id?.slice(0, 8) || 'Tidak ada'}</strong>
                </div>
              </div>

              {report.resolution_note ? (
                <Alert tone="info" icon="check_circle" title="Catatan keputusan">
                  {report.resolution_note}
                  {report.resolved_at ? (
                    <div className="tiny muted mt-1">Ditutup {formatDateTime(report.resolved_at)}</div>
                  ) : null}
                </Alert>
              ) : null}

              {['open', 'in_review'].includes(report.status) ? (
                <div className="row-wrap">
                  <button
                    type="button"
                    className="btn btn-primary btn-sm"
                    onClick={() => {
                      setDecision('resolve')
                      setNote('')
                      setModalError(null)
                    }}
                  >
                    <Icon name="task_alt" /> Selesaikan
                  </button>
                  <button
                    type="button"
                    className="btn btn-danger-soft btn-sm"
                    onClick={() => {
                      setDecision('reject')
                      setNote('')
                      setModalError(null)
                    }}
                  >
                    <Icon name="cancel" /> Tolak Laporan
                  </button>
                </div>
              ) : (
                <Alert tone="success" icon="lock" title="Laporan sudah ditutup">
                  Keputusan tidak dapat diubah kembali.
                </Alert>
              )}
            </div>
          ) : null}

          {decision ? (
            <Modal
              title={decision === 'resolve' ? 'Selesaikan laporan' : 'Tolak laporan'}
              subtitle={selected.subject}
              onClose={() => setDecision(null)}
              footer={
                <>
                  <button type="button" className="btn btn-ghost" onClick={() => setDecision(null)}>Batal</button>
                  <button
                    type="submit"
                    form="report-decision-form"
                    className={`btn ${decision === 'resolve' ? 'btn-primary' : 'btn-danger'}`}
                    disabled={busy}
                  >
                    {busy ? <span className="spinner" /> : <Icon name="gavel" />}
                    {decision === 'resolve' ? 'Selesaikan' : 'Tolak'}
                  </button>
                </>
              }
            >
              <form id="report-decision-form" className="stack" onSubmit={submitDecision}>
                <Alert tone="warning" icon="notifications" title="Pelapor akan diberitahu">
                  Catatan keputusan dikirim sebagai notifikasi dan tercatat di log audit.
                </Alert>
                <div className="field">
                  <label className="label" htmlFor="report-note">Catatan keputusan <span className="req">*</span></label>
                  <textarea
                    id="report-note"
                    className="textarea"
                    rows={4}
                    value={note}
                    onChange={(event) => setNote(event.target.value)}
                    placeholder="Contoh: Temuan sesuai laporan. Listing telah diturunkan dan pemilik diberi peringatan."
                  />
                  <span className="hint">Minimal 10 karakter.</span>
                </div>
                {modalError ? <ErrorBox error={modalError} title="Keputusan gagal" /> : null}
              </form>
            </Modal>
          ) : null}
        </Modal>
      ) : null}
    </AdminShell>
  )
}
