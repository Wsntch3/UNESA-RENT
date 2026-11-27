import { useEffect, useMemo, useState } from 'react'
import { Link, useLocation } from '../router'
import { adminKtmApi, ktmApi } from '../lib/api'
import { useApi } from '../lib/useApi'
import { fileSize, formatDate, formatDateTime, ktmStatus, timeAgo } from '../lib/format'
import { AdminShell } from '../components/AdminShell'
import {
  Alert,
  Avatar,
  Empty,
  ErrorBox,
  Icon,
  Loading,
  Modal,
} from '../components/ui'

const LIMIT = 6

const TABS = [
  { value: 'pending', label: 'Menunggu' },
  { value: 'verified', label: 'Terverifikasi' },
  { value: 'rejected', label: 'Ditolak' },
  { value: 'revoked', label: 'Dicabut' },
]

function KtmPreview({ submission }) {
  const [url, setUrl] = useState(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState(null)

  const purged = Boolean(
    submission?.file_purged_at ||
      error?.code === 'ktm_file_purged' ||
      error?.status === 410,
  )

  useEffect(() => {
    if (!submission || submission.file_purged_at) {
      setUrl(null)
      setError(null)
      return undefined
    }

    let alive = true
    let objectUrl = null
    setUrl(null)
    setError(null)
    setLoading(true)

    ktmApi
      .fileBlobUrl(submission.id)
      .then((result) => {
        if (!alive) {
          URL.revokeObjectURL(result)
          return
        }
        objectUrl = result
        setUrl(result)
      })
      .catch((err) => {
        if (alive) setError(err)
      })
      .finally(() => {
        if (alive) setLoading(false)
      })

    return () => {
      alive = false
      if (objectUrl) URL.revokeObjectURL(objectUrl)
    }
  }, [submission?.id, submission?.file_purged_at])

  if (!submission) {
    return (
      <div className="doc-preview">
        <div className="center muted">
          <Icon name="badge" size={38} />
          <p className="small mt-2">Pilih berkas untuk melihat dokumen KTM.</p>
        </div>
      </div>
    )
  }

  if (purged) {
    return (
      <div className="doc-preview">
        <div className="center muted">
          <Icon name="hide_image" size={38} />
          <p className="small mt-2">File sudah dihapus setelah verifikasi selesai.</p>
        </div>
      </div>
    )
  }

  return (
    <div className="doc-preview" style={{ minHeight: 260 }}>
      {loading ? (
        <div className="center muted">
          <span className="spinner lg" />
          <p className="small mt-2">Mengambil dokumen privat…</p>
        </div>
      ) : error ? (
        <div className="center text-danger">
          <Icon name="error" size={32} />
          <p className="small mt-2">{error.message}</p>
        </div>
      ) : url ? (
        <img src={url} alt={`KTM ${submission.user_name}`} />
      ) : null}
    </div>
  )
}

export default function AdminKTM() {
  const { query } = useLocation()
  const initialStatus = TABS.some((tab) => tab.value === query.status) ? query.status : 'pending'
  const [status, setStatus] = useState(initialStatus)
  const [offset, setOffset] = useState(0)
  const [selectedId, setSelectedId] = useState(null)
  const [confirmNim, setConfirmNim] = useState(false)
  const [busy, setBusy] = useState(null)
  const [feedback, setFeedback] = useState(null)
  const [actionError, setActionError] = useState(null)
  const [reasonMode, setReasonMode] = useState(null)
  const [reason, setReason] = useState('')
  const [modalBusy, setModalBusy] = useState(false)
  const [modalError, setModalError] = useState(null)

  const { data, loading, error, reload } = useApi(
    () => adminKtmApi.list({ status, limit: LIMIT, offset }),
    [status, offset],
  )

  const submissions = useMemo(() => data?.submissions || [], [data])
  const selected = submissions.find((item) => item.id === selectedId) || null

  useEffect(() => {
    if (!submissions.length) {
      setSelectedId(null)
      return
    }
    if (!submissions.some((item) => item.id === selectedId)) {
      setSelectedId(submissions[0].id)
    }
  }, [submissions, selectedId])

  useEffect(() => {
    setConfirmNim(false)
  }, [selectedId])

  useEffect(() => {
    if (TABS.some((tab) => tab.value === query.status) && query.status !== status) {
      setStatus(query.status)
      setOffset(0)
      setSelectedId(null)
      setFeedback(null)
      setActionError(null)
    }
  }, [query.status]) 

  const changeStatus = (value) => {
    setStatus(value)
    setOffset(0)
    setSelectedId(null)
    setFeedback(null)
    setActionError(null)
  }

  const verify = async () => {
    if (!selected) return
    setBusy('verify')
    setActionError(null)
    setFeedback(null)
    try {
      await adminKtmApi.verify(selected.id)
      setFeedback({ tone: 'success', message: `KTM ${selected.user_name} berhasil diverifikasi.` })
      setConfirmNim(false)
      await reload()
    } catch (err) {
      setActionError(err)
    } finally {
      setBusy(null)
    }
  }

  const submitReason = async (event) => {
    event.preventDefault()
    if (!selected || !reasonMode) return
    if (!reason.trim()) {
      setModalError({ message: 'Alasan wajib diisi.' })
      return
    }

    setModalBusy(true)
    setModalError(null)
    try {
      if (reasonMode === 'reject') {
        await adminKtmApi.reject(selected.id, reason.trim())
        setFeedback({ tone: 'success', message: `Pengajuan ${selected.user_name} ditolak.` })
      } else {
        await adminKtmApi.revoke(selected.id, reason.trim())
        setFeedback({ tone: 'success', message: `Verifikasi ${selected.user_name} dicabut.` })
      }
      setReasonMode(null)
      setReason('')
      await reload()
    } catch (err) {
      setModalError(err)
    } finally {
      setModalBusy(false)
    }
  }

  const meta = selected ? ktmStatus(selected.status) : null

  return (
    <AdminShell
      eyebrow="Verifikasi Identitas Mahasiswa"
      title="Antrean verifikasi KTM"
      subtitle="Validasi data mahasiswa aktif Fakultas Teknik UNESA untuk akses sewa-menyewa alat praktikum dan laboratorium."
      actions={
        <span className="badge badge-primary">
          <Icon name="lock" size={13} /> Dokumen privat
        </span>
      }
    >
      <div className="tabs mb-4">
        {TABS.map((tab) => (
          <button
            key={tab.value}
            type="button"
            className={`tab ${status === tab.value ? 'active' : ''}`.trim()}
            onClick={() => changeStatus(tab.value)}
          >
            {tab.label}
            {status === tab.value && submissions.length ? (
              <span className="count">{submissions.length}</span>
            ) : null}
          </button>
        ))}
      </div>

      {feedback ? (
        <div className="mb-4">
          <Alert tone={feedback.tone} icon="check_circle">{feedback.message}</Alert>
        </div>
      ) : null}

      {actionError ? (
        <div className="mb-4">
          <ErrorBox error={actionError} title="Aksi gagal" />
        </div>
      ) : null}

      {error ? <ErrorBox error={error} onRetry={reload} /> : null}

      {loading ? (
        <Loading label="Memuat antrean KTM…" />
      ) : submissions.length ? (
        <div className="grid cols-2" style={{ alignItems: 'start' }}>
          <section className="stack-sm">
            {submissions.map((submission) => {
              const badge = ktmStatus(submission.status)
              const active = submission.id === selectedId
              return (
                <button
                  key={submission.id}
                  type="button"
                  className="card card-pad"
                  onClick={() => setSelectedId(submission.id)}
                  style={{
                    textAlign: 'left',
                    cursor: 'pointer',
                    borderColor: active ? 'var(--primary)' : undefined,
                    boxShadow: active ? '0 0 0 3px var(--primary-soft)' : undefined,
                    background: 'var(--white)',
                  }}
                >
                  <div className="row" style={{ alignItems: 'flex-start', gap: 12 }}>
                    <Avatar name={submission.user_name} />
                    <div style={{ flex: 1, minWidth: 0 }}>
                      <div className="between" style={{ gap: 6 }}>
                        <strong className="truncate">{submission.user_name}</strong>
                        <span className={`badge badge-${badge.tone}`}>{badge.label}</span>
                      </div>
                      <div className="tiny muted truncate">{submission.user_email}</div>
                      <div className="tiny muted mt-1">
                        NIM {submission.user_nim || '—'} · {fileSize(submission.file_size_bytes)} ·{' '}
                        {timeAgo(submission.submitted_at)}
                      </div>
                    </div>
                  </div>
                </button>
              )
            })}

            <div className="between" style={{ marginTop: 6 }}>
              <span className="tiny muted">
                Menampilkan {submissions.length} berkas {data?.status ? `(${data.status})` : ''}
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
                  disabled={submissions.length < LIMIT}
                  onClick={() => setOffset((value) => value + LIMIT)}
                >
                  Berikutnya <Icon name="chevron_right" />
                </button>
              </div>
            </div>
          </section>

          <section className="card card-pad-lg sticky-side">
            {selected ? (
              <>
                <div className="row" style={{ alignItems: 'flex-start', gap: 12 }}>
                  <Avatar name={selected.user_name} size="lg" />
                  <div style={{ flex: 1, minWidth: 0 }}>
                    <h2 className="section-title" style={{ fontSize: 19 }}>{selected.user_name}</h2>
                    <div className="tiny muted">{selected.user_email}</div>
                    <div className="row-wrap mt-2">
                      <span className={`badge badge-${meta.tone}`}>
                        <Icon name={meta.icon} size={12} /> {meta.label}
                      </span>
                      {selected.reviewer_confirmed_nim ? (
                        <span className="badge badge-success">
                          <Icon name="check" size={12} /> NIM dikonfirmasi
                        </span>
                      ) : null}
                    </div>
                  </div>
                </div>

                <div className="spec-grid mt-4">
                  <div className="spec">
                    <span>NIM akun</span>
                    <strong>{selected.user_nim || '—'}</strong>
                  </div>
                  <div className="spec">
                    <span>Dikirim</span>
                    <strong>{formatDate(selected.submitted_at, 'short')}</strong>
                  </div>
                  <div className="spec">
                    <span>Ukuran file</span>
                    <strong>{fileSize(selected.file_size_bytes)}</strong>
                  </div>
                  <div className="spec">
                    <span>Format</span>
                    <strong>{selected.mime_type === 'image/png' ? 'PNG' : 'JPEG'}</strong>
                  </div>
                </div>

                <h3 className="card-title mt-4 mb-2">Lampiran KTM</h3>
                <KtmPreview submission={selected} />

                {selected.reviewed_at ? (
                  <p className="hint mt-2">Ditinjau {formatDateTime(selected.reviewed_at)}.</p>
                ) : null}
                {selected.rejection_reason ? (
                  <Alert tone="error" icon="cancel" title="Alasan penolakan">
                    {selected.rejection_reason}
                  </Alert>
                ) : null}
                {selected.revoke_reason ? (
                  <Alert tone="error" icon="gpp_bad" title="Alasan pencabutan">
                    {selected.revoke_reason}
                  </Alert>
                ) : null}

                {selected.status === 'pending' ? (
                  <div className="stack mt-4">
                    <label className="checkbox">
                      <input
                        type="checkbox"
                        checked={confirmNim}
                        onChange={(event) => setConfirmNim(event.target.checked)}
                      />
                      <span>
                        Saya sudah mencocokkan nama dan NIM pada kartu dengan data akun
                        <strong> {selected.user_nim}</strong>.
                      </span>
                    </label>
                    <div className="row">
                      <button
                        type="button"
                        className="btn btn-primary"
                        disabled={!confirmNim || busy === 'verify'}
                        onClick={verify}
                      >
                        {busy === 'verify' ? <span className="spinner" /> : <Icon name="verified_user" />}
                        Verifikasi KTM
                      </button>
                      <button
                        type="button"
                        className="btn btn-danger-soft"
                        disabled={Boolean(busy)}
                        onClick={() => {
                          setReason('')
                          setModalError(null)
                          setReasonMode('reject')
                        }}
                      >
                        <Icon name="cancel" /> Tolak
                      </button>
                    </div>
                  </div>
                ) : null}

                {selected.status === 'verified' ? (
                  <button
                    type="button"
                    className="btn btn-danger-soft btn-block mt-4"
                    disabled={Boolean(busy)}
                    onClick={() => {
                      setReason('')
                      setModalError(null)
                      setReasonMode('revoke')
                    }}
                  >
                    <Icon name="gpp_bad" /> Cabut Verifikasi
                  </button>
                ) : null}

                <Alert tone="info" icon="shield" title="Kerahasiaan terjaga">
                  Dokumen identitas hanya boleh diakses untuk keperluan verifikasi. Setiap pemeriksaan
                  tercatat pada audit log.
                </Alert>
              </>
            ) : (
              <Empty icon="badge" title="Pilih berkas">
                Pilih salah satu pengajuan di sebelah kiri untuk mulai meninjau.
              </Empty>
            )}
          </section>
        </div>
      ) : (
        <Empty icon="task_alt" title="Antrean kosong">
          Tidak ada berkas KTM dengan status {TABS.find((tab) => tab.value === status)?.label.toLowerCase()}.
        </Empty>
      )}

      <p className="hint mt-4">
        <Link to="/admin">Kembali ke dashboard admin</Link>
      </p>

      {reasonMode ? (
        <Modal
          title={reasonMode === 'reject' ? 'Tolak pengajuan KTM' : 'Cabut verifikasi KTM'}
          subtitle={selected ? `${selected.user_name} · NIM ${selected.user_nim}` : undefined}
          onClose={() => setReasonMode(null)}
          footer={
            <>
              <button type="button" className="btn btn-ghost" onClick={() => setReasonMode(null)}>Batal</button>
              <button type="submit" form="ktm-reason-form" className="btn btn-danger" disabled={modalBusy}>
                {modalBusy ? <span className="spinner" /> : <Icon name={reasonMode === 'reject' ? 'cancel' : 'gpp_bad'} />}
                {reasonMode === 'reject' ? 'Tolak Berkas' : 'Cabut Verifikasi'}
              </button>
            </>
          }
        >
          <form id="ktm-reason-form" className="stack" onSubmit={submitReason}>
            <Alert tone="warning" icon="warning" title="Keputusan tercatat permanen">
              Alasan akan dikirim ke mahasiswa dan disimpan pada audit log sistem.
            </Alert>
            <div className="field">
              <label className="label" htmlFor="ktm-reason">Alasan <span className="req">*</span></label>
              <textarea
                id="ktm-reason"
                className="textarea"
                value={reason}
                onChange={(event) => setReason(event.target.value)}
                placeholder={
                  reasonMode === 'reject'
                    ? 'Contoh: Foto KTM buram dan NIM tidak terbaca jelas.'
                    : 'Contoh: Identitas tidak lagi valid berdasarkan laporan resmi.'
                }
              />
            </div>
            {modalError ? <Alert tone="error" icon="error">{modalError.message}</Alert> : null}
          </form>
        </Modal>
      ) : null}
    </AdminShell>
  )
}
