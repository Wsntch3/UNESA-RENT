import { useState } from 'react'
import { Link, useLocation } from '../router'
import { useAuth } from '../auth/useAuth'
import { reportApi } from '../lib/api'
import { useApi } from '../lib/useApi'
import { formatDateTime, timeAgo } from '../lib/format'
import {
  Alert,
  Empty,
  ErrorBox,
  Field,
  Icon,
  Loading,
  PageHeader,
} from '../components/ui'

const TYPES = [
  { value: 'technical', label: 'Masalah teknis' },
  { value: 'safety', label: 'Keamanan barang' },
  { value: 'fraud', label: 'Penipuan' },
  { value: 'harassment', label: 'Pelecehan' },
  { value: 'other', label: 'Lainnya' },
]

const STATUS_LABELS = {
  open: { label: 'Terbuka', tone: 'warning' },
  in_review: { label: 'Ditinjau', tone: 'primary' },
  resolved: { label: 'Diselesaikan', tone: 'success' },
  rejected: { label: 'Ditolak', tone: 'danger' },
}

export default function Report() {
  const { query } = useLocation()
  const { user } = useAuth()
  const [type, setType] = useState('technical')
  const [subject, setSubject] = useState('')
  const [body, setBody] = useState('')
  const [errors, setErrors] = useState(null)
  const [busy, setBusy] = useState(false)
  const [feedback, setFeedback] = useState(null)

  const rentalID = query.rental || ''
  const listingID = query.listing || ''
  const mineQuery = useApi(() => reportApi.mine({ limit: 20 }), [])

  const validate = () => {
    const fields = {}
    if (!TYPES.some((item) => item.value === type)) fields.type = 'Jenis laporan tidak valid.'
    if (subject.trim().length < 3 || subject.trim().length > 150) {
      fields.subject = 'Judul harus 3-150 karakter.'
    }
    if (body.trim().length < 10 || body.trim().length > 5000) {
      fields.body = 'Isi laporan harus 10-5000 karakter.'
    }
    setErrors(Object.keys(fields).length ? fields : null)
    return Object.keys(fields).length === 0
  }

  const submit = async (event) => {
    event.preventDefault()
    if (!validate()) return

    setBusy(true)
    setFeedback(null)
    try {
      await reportApi.create({
        type,
        subject: subject.trim(),
        body: body.trim(),
        rental_id: rentalID || undefined,
        listing_id: listingID || undefined,
      })
      setFeedback({
        tone: 'success',
        message: 'Laporan terkirim. Tim admin BEM FT akan meninjau dan memberi kabar lewat notifikasi.',
      })
      setSubject('')
      setBody('')
      setType('technical')
      setErrors(null)
      await mineQuery.reload()
    } catch (err) {
      setErrors(err.fields || { submit: err.message })
      setFeedback({ tone: 'error', message: err.message })
    } finally {
      setBusy(false)
    }
  }

  const reports = mineQuery.data?.items || []

  return (
    <div className="container page">
      <PageHeader
        eyebrow="Kanal Pelaporan Komunitas"
        title="Laporkan masalah"
        subtitle="Bantu menjaga keamanan ekosistem sewa FT UNESA. Laporanmu ditinjau oleh admin dan bersifat rahasia."
        actions={
          <Link to="/penyewaan" className="btn btn-outline">
            <Icon name="arrow_back" /> Kembali ke Penyewaan
          </Link>
        }
      />

      <div className="grid cols-2" style={{ alignItems: 'start', gap: 20 }}>
        <section className="card card-pad-lg">
          <h2 className="section-title" style={{ fontSize: 18 }}>Formulir laporan</h2>
          <p className="hint mt-1">
            Isi dengan jelas agar admin dapat menindaklanjuti. Jangan sertakan data pribadi sensitif.
          </p>

          {feedback ? (
            <div className="mt-4">
              <Alert tone={feedback.tone} icon={feedback.tone === 'success' ? 'check_circle' : 'error'}>
                {feedback.message}
              </Alert>
            </div>
          ) : null}

          {errors?.submit ? (
            <div className="mt-4"><ErrorBox error={{ message: errors.submit }} title="Laporan gagal dikirim" /></div>
          ) : null}

          <form className="stack mt-4" onSubmit={submit}>
            {rentalID || listingID ? (
              <Alert tone="info" icon="link" title="Laporan terkait transaksi">
                {rentalID ? `Sewa #${rentalID.slice(0, 8).toUpperCase()}` : `Listing #${listingID.slice(0, 8).toUpperCase()}`}
                {' '}terlampir otomatis pada laporan ini.
              </Alert>
            ) : null}

            <Field label="Jenis masalah" htmlFor="report-type" required error={errors?.type}>
              <select
                id="report-type"
                className="select"
                value={type}
                onChange={(event) => setType(event.target.value)}
              >
                {TYPES.map((item) => (
                  <option key={item.value} value={item.value}>{item.label}</option>
                ))}
              </select>
            </Field>

            <Field label="Judul laporan" htmlFor="report-subject" required error={errors?.subject}>
              <input
                id="report-subject"
                className="input"
                value={subject}
                onChange={(event) => setSubject(event.target.value)}
                placeholder="Contoh: Barang tidak sesuai deskripsi"
                maxLength={150}
              />
            </Field>

            <Field label="Isi laporan" htmlFor="report-body" required error={errors?.body}>
              <textarea
                id="report-body"
                className="textarea"
                rows={7}
                value={body}
                onChange={(event) => setBody(event.target.value)}
                placeholder="Ceritakan kronologi, waktu kejadian, dan pihak yang terlibat."
                maxLength={5000}
              />
              <span className="hint">Minimal 10 karakter. Maksimal 5000 karakter.</span>
            </Field>

            <button type="submit" className="btn btn-primary btn-block" disabled={busy}>
              {busy ? <span className="spinner" /> : <Icon name="send" />}
              Kirim Laporan
            </button>
          </form>
        </section>

        <section className="card card-pad-lg">
          <div className="between" style={{ marginBottom: 12 }}>
            <div>
              <h2 className="section-title" style={{ fontSize: 18 }}>Laporan saya</h2>
              <p className="hint mt-1">Pantau status laporan yang sudah kamu kirim.</p>
            </div>
          </div>

          {mineQuery.error ? <ErrorBox error={mineQuery.error} onRetry={mineQuery.reload} /> : null}
          {mineQuery.loading ? <Loading label="Memuat laporan…" /> : null}

          {!mineQuery.loading && !mineQuery.error && !reports.length ? (
            <Empty icon="flag" title="Belum ada laporan">
              Laporan yang kamu kirim akan muncul di sini.
            </Empty>
          ) : null}

          {reports.length ? (
            <div className="stack-sm">
              {reports.map((item) => {
                const meta = STATUS_LABELS[item.status] || { label: item.status, tone: 'muted' }
                return (
                  <div className="party" key={item.id}>
                    <div className="party-body">
                      <div className="party-name">{item.subject}</div>
                      <div className="party-meta">
                        {TYPES.find((entry) => entry.value === item.type)?.label || item.type} ·{' '}
                        {timeAgo(item.created_at)}
                      </div>
                      {item.resolution_note ? (
                        <div className="tiny muted mt-1">{item.resolution_note}</div>
                      ) : null}
                      {item.resolved_at ? (
                        <div className="tiny muted">Ditutup {formatDateTime(item.resolved_at)}</div>
                      ) : null}
                    </div>
                    <span className={`badge badge-${meta.tone}`}>{meta.label}</span>
                  </div>
                )
              })}
            </div>
          ) : null}
        </section>
      </div>

      {user ? (
        <Alert tone="info" icon="lock" title="Kerja sama dengan admin">
          Laporan hanya dapat dilihat oleh pelapor dan tim admin. Informasi kontak pribadi tidak
          diperlukan untuk menindaklanjuti laporan.
        </Alert>
      ) : null}
    </div>
  )
}
