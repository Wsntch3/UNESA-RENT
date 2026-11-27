import { useEffect, useState } from 'react'
import { Link } from '../router'
import { useAuth } from '../auth/useAuth'
import { adminApi } from '../lib/api'
import { useApi } from '../lib/useApi'
import { saveBlobUrl } from '../lib/http'
import { formatDate, roleLabel, timeAgo } from '../lib/format'
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

const ROLE_FILTERS = [
  { value: '', label: 'Semua peran' },
  { value: 'mahasiswa', label: 'Mahasiswa' },
  { value: 'admin_dev', label: 'Admin' },
]

const STATUS_FILTERS = [
  { value: '', label: 'Semua status' },
  { value: 'active', label: 'Aktif' },
  { value: 'suspended', label: 'Ditangguhkan' },
]

const VERIFIED_FILTERS = [
  { value: '', label: 'Semua verifikasi' },
  { value: 'true', label: 'Terverifikasi' },
  { value: 'false', label: 'Belum verifikasi' },
]

export default function AdminUsers() {
  const { user } = useAuth()
  const [queryInput, setQueryInput] = useState('')
  const [query, setQuery] = useState('')
  const [role, setRole] = useState('')
  const [status, setStatus] = useState('')
  const [verified, setVerified] = useState('')
  const [offset, setOffset] = useState(0)

  const [target, setTarget] = useState(null)
  const [action, setAction] = useState(null)
  const [reason, setReason] = useState('')
  const [busy, setBusy] = useState(false)
  const [modalError, setModalError] = useState(null)
  const [feedback, setFeedback] = useState(null)

  useEffect(() => {
    const timer = setTimeout(() => {
      setQuery(queryInput.trim())
      setOffset(0)
    }, 350)
    return () => clearTimeout(timer)
  }, [queryInput])

  const statsQuery = useApi(() => adminApi.stats(), [])
  const { data, loading, error, reload } = useApi(
    () => adminApi.users({ q: query, role, status, verified, limit: LIMIT, offset }),
    [query, role, status, verified, offset],
  )

  const users = data?.items || []
  const total = data?.total || 0
  const stats = statsQuery.data?.stats || {}

  const changeFilter = (setter, value) => {
    setter(value)
    setOffset(0)
  }

  const openAction = (item, nextAction) => {
    setTarget(item)
    setAction(nextAction)
    setReason('')
    setModalError(null)
  }

  const closeAction = () => {
    setTarget(null)
    setAction(null)
    setReason('')
    setModalError(null)
  }

  const submit = async (event) => {
    event.preventDefault()
    if (!target || !action) return
    if (reason.trim().length < 10) {
      setModalError({ message: 'Alasan wajib minimal 10 karakter.' })
      return
    }

    setBusy(true)
    setModalError(null)
    try {
      const text = reason.trim()
      if (action === 'suspend') await adminApi.suspend(target.id, text)
      if (action === 'reactivate') await adminApi.reactivate(target.id, text)
      if (action === 'delete') await adminApi.removeUser(target.id, text)
      const messages = {
        suspend: `Akun ${target.name} ditangguhkan dan sesi aktifnya dicabut.`,
        reactivate: `Akun ${target.name} diaktifkan kembali.`,
        delete: `Akun ${target.name} dihapus. NIM dan email kini dapat digunakan kembali.`,
      }
      closeAction()
      setFeedback({ tone: 'success', message: messages[action] })
      await reload()
    } catch (err) {
      setModalError(err)
    } finally {
      setBusy(false)
    }
  }

  const exportCSV = async () => {
    setBusy(true)
    setFeedback(null)
    try {
      const url = await adminApi.exportUsersBlobUrl({ q: query, role, status, verified })
      saveBlobUrl(url, 'pengguna.csv')
      setFeedback({ tone: 'success', message: 'Data pengguna diekspor. Aktivitas ini tercatat di audit log.' })
    } catch (err) {
      setFeedback({ tone: 'error', message: err.message })
    } finally {
      setBusy(false)
    }
  }

  return (
    <AdminShell
      eyebrow="Governance Pengguna"
      title="Kelola pengguna"
      subtitle="Verifikasi status akun, tangguhkan pelanggar, dan hapus akun belum terverifikasi yang tidak memiliki aktivitas."
      actions={
        <>
          <button type="button" className="btn btn-outline" disabled={busy} onClick={exportCSV}>
            <Icon name="download" /> Ekspor CSV
          </button>
          <Link to="/admin/reports" className="btn btn-outline">
            <Icon name="flag" /> Laporan
          </Link>
          <span className="badge badge-muted">
            <Icon name="groups" size={13} /> {total} pengguna
          </span>
        </>
      }
    >
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

      {statsQuery.error ? <ErrorBox error={statsQuery.error} onRetry={statsQuery.reload} title="Statistik gagal dimuat" /> : null}

      <div className="grid cols-3" style={{ marginBottom: 20 }}>
        <StatCard icon="groups" value={stats.total_users || 0} label="Total pengguna" />
        <StatCard
          icon="verified_user"
          tone="mint"
          value={stats.verified_users || 0}
          label="Terverifikasi"
          note="KTM sudah disetujui"
        />
        <StatCard
          icon="flag"
          tone="yellow"
          value={stats.open_reports || 0}
          label="Laporan terbuka"
          note="Perlu peninjauan"
        />
      </div>

      <div className="between mb-4" style={{ flexWrap: 'wrap', gap: 10 }}>
        <div className="toolbar">
          <div className="input-wrap" style={{ minWidth: 220 }}>
            <Icon name="search" />
            <input
              className="input"
              value={queryInput}
              onChange={(event) => setQueryInput(event.target.value)}
              placeholder="Cari nama, email, atau NIM…"
              aria-label="Cari pengguna"
            />
          </div>
          <select
            className="select"
            value={role}
            onChange={(event) => changeFilter(setRole, event.target.value)}
            aria-label="Filter peran"
          >
            {ROLE_FILTERS.map((option) => (
              <option key={option.value} value={option.value}>{option.label}</option>
            ))}
          </select>
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
          <select
            className="select"
            value={verified}
            onChange={(event) => changeFilter(setVerified, event.target.value)}
            aria-label="Filter verifikasi"
          >
            {VERIFIED_FILTERS.map((option) => (
              <option key={option.value} value={option.value}>{option.label}</option>
            ))}
          </select>
        </div>
      </div>

      {error ? <ErrorBox error={error} onRetry={reload} /> : null}

      {loading ? (
        <Loading label="Memuat pengguna…" />
      ) : users.length ? (
        <>
          <div className="table-wrap">
            <table className="table">
              <thead>
                <tr>
                  <th>Nama</th>
                  <th>Email</th>
                  <th>Peran</th>
                  <th>Status</th>
                  <th>Verifikasi</th>
                  <th>Dibuat</th>
                  <th>Aksi</th>
                </tr>
              </thead>
              <tbody>
                {users.map((item) => {
                  const suspended = item.status === 'suspended'
                  const isSelf = user?.id === item.id
                  return (
                    <tr key={item.id}>
                      <td>
                        <div className="row" style={{ gap: 8 }}>
                          <Avatar name={item.name} size="sm" />
                          <div>
                            <div className="strong">{item.name}</div>
                            {item.nim ? <div className="tiny muted">NIM {item.nim}</div> : null}
                          </div>
                        </div>
                      </td>
                      <td className="tiny">{item.email}</td>
                      <td>{roleLabel(item.role)}</td>
                      <td>
                        <span className={`badge ${suspended ? 'badge-danger' : 'badge-success'}`}>
                          {suspended ? 'Ditangguhkan' : 'Aktif'}
                        </span>
                      </td>
                      <td>
                        <span className={`badge ${item.is_verified ? 'badge-mint' : 'badge-muted'}`}>
                          {item.is_verified ? 'Terverifikasi' : 'Belum'}
                        </span>
                      </td>
                      <td className="tiny muted">{formatDate(item.created_at, 'short')}</td>
                      <td>
                        <div className="row-wrap">
                          {suspended ? (
                            <button
                              type="button"
                              className="btn btn-success-soft btn-sm"
                              onClick={() => openAction(item, 'reactivate')}
                            >
                              <Icon name="restore" /> Aktifkan
                            </button>
                          ) : (
                            <button
                              type="button"
                              className="btn btn-danger-soft btn-sm"
                              disabled={isSelf}
                              title={isSelf ? 'Tidak dapat menangguhkan akun sendiri' : undefined}
                              onClick={() => openAction(item, 'suspend')}
                            >
                              <Icon name="block" /> Tangguhkan
                            </button>
                          )}
                          {!item.is_verified ? (
                            <button
                              type="button"
                              className="btn btn-ghost btn-sm"
                              onClick={() => openAction(item, 'delete')}
                            >
                              <Icon name="delete" /> Hapus
                            </button>
                          ) : null}
                        </div>
                      </td>
                    </tr>
                  )
                })}
              </tbody>
            </table>
          </div>

          <div className="between" style={{ marginTop: 14 }}>
            <span className="small muted">
              Menampilkan {users.length} dari {total} pengguna
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
        <Empty icon="groups" title="Tidak ada pengguna">
          {query || role || status || verified
            ? 'Tidak ada pengguna yang cocok dengan filter ini.'
            : 'Belum ada pengguna yang terdaftar.'}
        </Empty>
      )}

      <Alert tone="info" icon="gavel" title="Kebijakan moderasi">
        Penangguhan mencabut sesi login dan memerlukan alasan jelas. Penghapusan hanya tersedia untuk
        akun belum terverifikasi tanpa listing, sewa, pesan, atau laporan.
      </Alert>

      {target && action ? (
        <Modal
          title={{
            suspend: 'Tangguhkan pengguna',
            reactivate: 'Aktifkan kembali pengguna',
            delete: 'Hapus akun belum terverifikasi',
          }[action]}
          subtitle={`${target.name} · ${target.email}`}
          onClose={closeAction}
          footer={
            <>
              <button type="button" className="btn btn-ghost" onClick={closeAction}>Batal</button>
              <button
                type="submit"
                form="user-action-form"
                className={`btn ${action === 'reactivate' ? 'btn-primary' : 'btn-danger'}`}
                disabled={busy}
              >
                {busy ? <span className="spinner" /> : <Icon name={action === 'reactivate' ? 'restore' : action === 'delete' ? 'delete' : 'block'} />}
                {{
                  suspend: 'Tangguhkan',
                  reactivate: 'Aktifkan',
                  delete: 'Hapus Akun',
                }[action]}
              </button>
            </>
          }
        >
          <form id="user-action-form" className="stack" onSubmit={submit}>
            {action === 'suspend' ? (
              <Alert tone="warning" icon="block" title="Sesi langsung dicabut">
                Pengguna harus login ulang setelah akunnya diaktifkan kembali.
              </Alert>
            ) : null}
            {action === 'delete' ? (
              <Alert tone="error" icon="warning" title="Tindakan tidak dapat dikembalikan">
                NIM dan email akan dibebaskan. Hapus hanya akun yang belum terverifikasi dan tidak
                memiliki aktivitas.
              </Alert>
            ) : null}
            {action === 'reactivate' ? (
              <Alert tone="success" icon="check_circle" title="Akses dipulihkan">
                Pengguna dapat login kembali setelah tindakan ini.
              </Alert>
            ) : null}
            <div className="field">
              <label className="label" htmlFor="user-action-reason">Alasan <span className="req">*</span></label>
              <textarea
                id="user-action-reason"
                className="textarea"
                rows={3}
                value={reason}
                onChange={(event) => setReason(event.target.value)}
                placeholder="Contoh: Melanggar ketentuan komunitas setelah peninjauan laporan."
              />
              <span className="hint">Minimal 10 karakter. Tercatat di log audit.</span>
            </div>
            {modalError ? <ErrorBox error={modalError} title="Tindakan gagal" /> : null}
            <p className="tiny muted">Terakhir login: {timeAgo(target.last_login_at)}</p>
          </form>
        </Modal>
      ) : null}
    </AdminShell>
  )
}
