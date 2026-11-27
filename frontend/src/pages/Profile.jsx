import { useEffect, useRef, useState } from 'react'
import { Link } from '../router'
import { useAuth } from '../auth/useAuth'
import { ktmApi } from '../lib/api'
import {
  fileSize,
  formatDate,
  formatDateTime,
  ktmStatus,
  prodiName,
  roleLabel,
} from '../lib/format'
import {
  Alert,
  Avatar,
  Empty,
  Icon,
  Loading,
  Modal,
  PageHeader,
} from '../components/ui'

export default function Profile() {
  const { user, ktm, ktmLoading, ktmVerified, refreshKtm, logout } = useAuth()

  const [file, setFile] = useState(null)
  const [preview, setPreview] = useState(null)
  const [uploading, setUploading] = useState(false)
  const [uploadError, setUploadError] = useState(null)
  const [feedback, setFeedback] = useState(null)
  const inputRef = useRef(null)

  const [previewUrl, setPreviewUrl] = useState(null)
  const [previewBusy, setPreviewBusy] = useState(false)
  const [previewError, setPreviewError] = useState(null)
  const [previewTitle, setPreviewTitle] = useState('')

  const submissions = ktm?.submissions || []
  const latest = submissions[0] || null
  const pending = submissions.find((item) => item.status === 'pending')
  const isAdmin = user?.role === 'admin_dev'

  const effectiveStatus = ktmVerified ? 'verified' : latest?.status || null
  const meta = ktmStatus(effectiveStatus)

  useEffect(() => {
    return () => {
      if (previewUrl) URL.revokeObjectURL(previewUrl)
    }
  }, [previewUrl])

  useEffect(() => {
    return () => {
      if (preview) URL.revokeObjectURL(preview)
    }
  }, [preview])

  const chooseFile = (event) => {
    const picked = event.target.files?.[0]
    event.target.value = ''
    setUploadError(null)
    setFeedback(null)
    if (!picked) return

    if (!['image/jpeg', 'image/png'].includes(picked.type)) {
      setUploadError({ message: 'Format file harus JPEG atau PNG.' })
      return
    }
    if (picked.size > 5 * 1024 * 1024) {
      setUploadError({ message: 'Ukuran file maksimal 5 MB.' })
      return
    }

    setFile(picked)
    setPreview((old) => {
      if (old) URL.revokeObjectURL(old)
      return URL.createObjectURL(picked)
    })
  }

  const submit = async (event) => {
    event.preventDefault()
    if (!file) return

    setUploading(true)
    setUploadError(null)
    setFeedback(null)
    try {
      await ktmApi.submit(file)
      setFile(null)
      setPreview((old) => {
        if (old) URL.revokeObjectURL(old)
        return null
      })
      refreshKtm()
      setFeedback({
        tone: 'success',
        message: 'Dokumen KTM terkirim. Admin BEM FT akan meninjau dalam 1×24 jam kerja.',
      })
    } catch (err) {
      setUploadError(err)
    } finally {
      setUploading(false)
    }
  }

  const openPreview = async (submission) => {
    if (submission.file_purged_at) {
      setPreviewTitle(submission.user_name || user?.name || 'KTM')
      setPreviewError({
        message: 'File sudah dihapus setelah proses verifikasi selesai.',
        code: 'ktm_file_purged',
        status: 410,
      })
      return
    }
    setPreviewBusy(true)
    setPreviewError(null)
    setPreviewTitle(submission.user_name || user?.name || 'KTM')
    try {
      const url = await ktmApi.fileBlobUrl(submission.id)
      setPreviewUrl((old) => {
        if (old) URL.revokeObjectURL(old)
        return url
      })
    } catch (err) {
      setPreviewError(err)
    } finally {
      setPreviewBusy(false)
    }
  }

  const closePreview = () => {
    setPreviewUrl((old) => {
      if (old) URL.revokeObjectURL(old)
      return null
    })
    setPreviewError(null)
  }

  const canSubmit = !isAdmin && !ktmVerified && !pending

  if (isAdmin) {
    return (
      <div className="container page">
        <PageHeader
          eyebrow="Akun Administrator"
          title="Profil admin"
          subtitle="Akun admin_dev tidak memerlukan verifikasi KTM mahasiswa."
        />
        <div className="grid cols-2">
          <section className="card card-pad-lg">
            <div className="row">
              <Avatar name={user?.name} size="xl" />
              <div>
                <h2 className="section-title" style={{ fontSize: 21 }}>{user?.name}</h2>
                <div className="subtitle">{user?.email}</div>
                <div className="row-wrap mt-2">
                  <span className="badge badge-primary">
                    <Icon name="admin_panel_settings" size={13} /> Admin FT
                  </span>
                  <span className="badge badge-muted">{roleLabel(user?.role)}</span>
                </div>
              </div>
            </div>
          </section>
          <section className="card card-pad-lg">
            <h3 className="card-title">Akses portal</h3>
            <p className="hint mt-1">Gunakan portal admin untuk verifikasi KTM dan moderasi listing.</p>
            <div className="stack-sm mt-3">
              <Link to="/admin" className="btn btn-primary btn-block">
                <Icon name="space_dashboard" /> Dashboard Admin
              </Link>
              <Link to="/admin/ktm" className="btn btn-outline btn-block">
                <Icon name="verified_user" /> Verifikasi KTM
              </Link>
              <Link to="/admin/listings" className="btn btn-outline btn-block">
                <Icon name="inventory_2" /> Moderasi Listing
              </Link>
            </div>
          </section>
        </div>
      </div>
    )
  }

  return (
    <div className="container page">
      <PageHeader
        eyebrow="Akun & Verifikasi"
        title="Profil saya"
        subtitle="Kelola identitas mahasiswa dan pantau status verifikasi KTM untuk membuka seluruh fitur sewa-menyewa."
        actions={
          <span className={`badge badge-${meta.tone}`}>
            <Icon name={meta.icon} filled size={13} /> {meta.label}
          </span>
        }
      />

      {feedback ? (
        <div className="mb-4">
          <Alert tone={feedback.tone} icon="check_circle">{feedback.message}</Alert>
        </div>
      ) : null}

      <div className="detail-grid">
        <div className="stack">
          <section className="card card-pad-lg">
            <div className="row" style={{ alignItems: 'center', gap: 16, flexWrap: 'wrap' }}>
              <Avatar name={user?.name} size="xl" />
              <div style={{ flex: 1, minWidth: 200 }}>
                <h2 className="section-title" style={{ fontSize: 22 }}>{user?.name}</h2>
                <div className="subtitle">{user?.email}</div>
                <div className="row-wrap mt-2">
                  <span className="badge badge-muted">
                    <Icon name="school" size={13} /> {prodiName(user?.study_program_code)}
                  </span>
                  <span className="badge badge-muted">
                    <Icon name="tag" size={13} /> NIM {user?.nim || '—'}
                  </span>
                  <span className="badge badge-primary">
                    <Icon name="badge" size={13} /> {roleLabel(user?.role)}
                  </span>
                </div>
              </div>
            </div>

            <hr className="divider" style={{ margin: '18px 0' }} />

            <div className="grid cols-3">
              <div className="spec">
                <span>Terdaftar sejak</span>
                <strong>{formatDate(user?.created_at, 'short')}</strong>
              </div>
              <div className="spec">
                <span>Kampus</span>
                <strong>Ketintang, FT UNESA</strong>
              </div>
              <div className="spec">
                <span>Status akun</span>
                <strong>{user?.status === 'suspended' ? 'Ditangguhkan' : 'Aktif'}</strong>
              </div>
            </div>
          </section>

          <section className="card card-pad-lg">
            <div className="between" style={{ alignItems: 'flex-start', flexWrap: 'wrap', gap: 10 }}>
              <div>
                <h2 className="section-title" style={{ fontSize: 18 }}>Status verifikasi KTM</h2>
                <p className="hint mt-1">
                  Kartu Tanda Mahasiswa hanya dipakai untuk validasi internal dan tidak ditampilkan
                  ke publik.
                </p>
              </div>
              <span className={`badge badge-${meta.tone}`}>
                <Icon name={meta.icon} size={13} /> {meta.label}
              </span>
            </div>

            {ktmLoading ? (
              <Loading label="Memeriksa status KTM…" />
            ) : ktmVerified ? (
              <div className="mt-4">
                <Alert tone="success" icon="verified_user" title="Terverifikasi penuh">
                  Identitas mahasiswa FT UNESA sudah valid. Kamu bisa menyewa dan menyewakan barang.
                </Alert>
                <ul className="info-list mt-3">
                  <li><Icon name="check_circle" /><span>Membuat listing dan mengunggah foto barang.</span></li>
                  <li><Icon name="check_circle" /><span>Mengajukan sewa dan berdiskusi dengan penyedia.</span></li>
                  <li><Icon name="check_circle" /><span>Menandatangani perjanjian sewa digital.</span></li>
                </ul>
                {latest ? (
                  <div className="row-wrap mt-4">
                    <button type="button" className="btn btn-outline btn-sm" onClick={() => openPreview(latest)}>
                      <Icon name="visibility" /> Lihat dokumen terverifikasi
                    </button>
                    <span className="tiny muted">Diverifikasi {formatDate(latest.reviewed_at || latest.updated_at, 'short')}</span>
                  </div>
                ) : null}
              </div>
            ) : pending ? (
              <div className="mt-4">
                <Alert tone="warning" icon="hourglass_top" title="Dokumen sedang ditinjau">
                  Estimasi 1×24 jam kerja. Kamu tetap bisa menjelajahi katalog selama menunggu.
                </Alert>
                <div className="row-wrap mt-3">
                  <button
                    type="button"
                    className="btn btn-outline btn-sm"
                    onClick={() => refreshKtm()}
                  >
                    <Icon name="refresh" /> Cek status ulang
                  </button>
                  <button type="button" className="btn btn-ghost btn-sm" onClick={() => openPreview(pending)}>
                    <Icon name="visibility" /> Lihat dokumen terkirim
                  </button>
                  <span className="tiny muted">Diunggah {formatDateTime(pending.submitted_at)}</span>
                </div>
              </div>
            ) : (
              <div className="mt-4">
                <Alert
                  tone={latest ? 'error' : 'info'}
                  icon={latest ? 'warning' : 'info'}
                  title={latest ? 'Perlu perbaikan dokumen' : 'Belum verifikasi'}
                >
                  {latest?.rejection_reason
                    ? latest.rejection_reason
                    : latest?.revoke_reason
                      ? latest.revoke_reason
                      : 'Unggah foto KTM yang jelas untuk membuka fitur sewa-menyewa.'}
                </Alert>
              </div>
            )}

            {canSubmit ? (
              <form className="stack mt-4" onSubmit={submit}>
                <div className="field">
                  <label className="label" htmlFor="ktm-file">
                    Foto KTM <span className="req">*</span>
                  </label>
                  <input
                    ref={inputRef}
                    id="ktm-file"
                    type="file"
                    className="input"
                    accept="image/jpeg,image/png"
                    onChange={chooseFile}
                  />
                  <span className="hint">JPEG atau PNG, maksimal 5 MB. Pastikan NIM dan nama terbaca.</span>
                </div>

                {preview ? (
                  <div className="doc-preview" style={{ minHeight: 160 }}>
                    <img src={preview} alt="Pratinjau KTM" style={{ maxHeight: 240 }} />
                  </div>
                ) : null}

                {uploadError ? (
                  <Alert tone="error" icon="error" title="Upload gagal">{uploadError.message}</Alert>
                ) : null}

                <div className="row" style={{ justifyContent: 'flex-end' }}>
                  {file ? (
                    <button
                      type="button"
                      className="btn btn-ghost"
                      onClick={() => {
                        setFile(null)
                        setPreview((old) => {
                          if (old) URL.revokeObjectURL(old)
                          return null
                        })
                      }}
                    >
                      Hapus pilihan
                    </button>
                  ) : null}
                  <button type="submit" className="btn btn-primary" disabled={!file || uploading}>
                    {uploading ? <span className="spinner" /> : <Icon name="upload_file" />}
                    {uploading ? 'Mengunggah…' : 'Kirim Verifikasi'}
                  </button>
                </div>
              </form>
            ) : null}
          </section>

          <section className="card card-pad-lg">
            <h2 className="section-title" style={{ fontSize: 18 }}>Riwayat pengajuan</h2>
            {submissions.length ? (
              <div className="stack-sm mt-3">
                {submissions.map((submission) => {
                  const status = ktmStatus(submission.status)
                  return (
                    <div className="block-row" key={submission.id} style={{ alignItems: 'flex-start' }}>
                      <div>
                        <div className="row-wrap">
                          <span className={`badge badge-${status.tone}`}>
                            <Icon name={status.icon} size={12} /> {status.label}
                          </span>
                          <span className="tiny muted">Dikirim {formatDateTime(submission.submitted_at)}</span>
                        </div>
                        <div className="tiny muted mt-1">
                          {submission.mime_type} · {fileSize(submission.file_size_bytes)}
                          {submission.reviewed_at ? ` · ditinjau ${formatDate(submission.reviewed_at, 'short')}` : ''}
                        </div>
                        {submission.rejection_reason ? (
                          <div className="tiny text-danger mt-1">Alasan: {submission.rejection_reason}</div>
                        ) : null}
                        {submission.revoke_reason ? (
                          <div className="tiny text-danger mt-1">Dicabut: {submission.revoke_reason}</div>
                        ) : null}
                      </div>
                      <button
                        type="button"
                        className="btn btn-outline btn-sm"
                        disabled={Boolean(submission.file_purged_at)}
                        onClick={() => openPreview(submission)}
                      >
                        <Icon name={submission.file_purged_at ? 'hide_image' : 'visibility'} />
                        {submission.file_purged_at ? 'File dihapus' : 'Lihat'}
                      </button>
                    </div>
                  )
                })}
              </div>
            ) : (
              <Empty icon="badge" title="Belum ada pengajuan KTM">
                Unggah dokumen pertamamu untuk mulai verifikasi.
              </Empty>
            )}
          </section>
        </div>

        <aside className="sticky-side stack">
          <section className="card card-pad-lg">
            <span className="eyebrow">Verifikasi kampus digital</span>
            <h3 className="card-title mt-2">Kenapa perlu KTM?</h3>
            <ul className="info-list mt-3">
              <li><Icon name="shield" /><span>Mencegah akun palsu dan penipuan antar mahasiswa.</span></li>
              <li><Icon name="handshake" /><span>Membuat penyedia lebih percaya sebelum COD.</span></li>
              <li><Icon name="description" /><span>Perjanjian sewa hanya berlaku untuk akun terverifikasi.</span></li>
            </ul>
          </section>

          <section className="card card-pad">
            <div className="verification-step done">
              <span className="num"><Icon name="check" size={12} /></span> Akun dibuat
            </div>
            <div className={`verification-step ${ktmVerified ? 'done' : pending ? 'active' : ''}`.trim()} style={{ marginTop: 10 }}>
              <span className="num">{ktmVerified ? <Icon name="check" size={12} /> : '2'}</span>
              {ktmVerified ? 'KTM terverifikasi' : pending ? 'Sedang ditinjau admin' : 'Unggah KTM'}
            </div>
            <div className={`verification-step ${ktmVerified ? 'active' : ''}`.trim()} style={{ marginTop: 10 }}>
              <span className="num">3</span> Mulai sewa-menyewa
            </div>
          </section>

          <section className="card card-pad">
            <h3 className="card-title">Keamanan akun</h3>
            <p className="hint mt-1">Keluar dari perangkat ini jika memakai komputer bersama.</p>
            <button type="button" className="btn btn-danger-soft btn-block mt-3" onClick={logout}>
              <Icon name="logout" /> Keluar dari akun
            </button>
          </section>
        </aside>
      </div>

      {previewUrl || previewError ? (
        <Modal
          title={previewTitle || 'Dokumen KTM'}
          subtitle="Dokumen privat — jangan dibagikan."
          size="modal-lg"
          onClose={closePreview}
          footer={<button type="button" className="btn btn-ghost" onClick={closePreview}>Tutup</button>}
        >
          {previewError ? (
            previewError.code === 'ktm_file_purged' || previewError.status === 410 ? (
              <div className="doc-preview">
                <div className="center muted">
                  <Icon name="hide_image" size={38} />
                  <p className="small mt-2">File sudah dihapus setelah proses verifikasi selesai.</p>
                </div>
              </div>
            ) : (
              <Alert tone="error" icon="error">{previewError.message}</Alert>
            )
          ) : (
            <div className="doc-preview">
              {previewUrl ? <img src={previewUrl} alt="Dokumen KTM" /> : <span className="spinner lg" />}
            </div>
          )}
        </Modal>
      ) : previewBusy ? (
        <Modal title="Memuat dokumen…" onClose={closePreview}>
          <Loading label="Mengambil file privat…" />
        </Modal>
      ) : null}
    </div>
  )
}
