import { useState } from 'react'
import { Link } from '../router'
import { adminApi } from '../lib/api'
import { useApi } from '../lib/useApi'
import { formatDateTime } from '../lib/format'
import { AdminShell } from '../components/AdminShell'
import {
  Alert,
  Empty,
  ErrorBox,
  Icon,
  Loading,
  StatCard,
} from '../components/ui'

const LIMIT = 10

const ACTIONS = [
  { value: '', label: 'Semua tindakan' },
  { value: 'reveal', label: 'Buka isi chat' },
  { value: 'download_document', label: 'Unduh dokumen' },
  { value: 'export_data', label: 'Ekspor data' },
  { value: 'resolve_report', label: 'Selesaikan laporan' },
  { value: 'reject_report', label: 'Tolak laporan' },
  { value: 'suspend_user', label: 'Tangguhkan pengguna' },
  { value: 'reactivate_user', label: 'Aktifkan pengguna' },
  { value: 'delete_unverified_user', label: 'Hapus akun' },
  { value: 'takedown_listing', label: 'Turunkan listing' },
  { value: 'restore_listing', label: 'Pulihkan listing' },
  { value: 'verify_ktm', label: 'Verifikasi KTM' },
  { value: 'revoke_ktm', label: 'Cabut KTM' },
]

const TARGET_LABELS = {
  listing: 'Listing',
  user: 'Pengguna',
  report: 'Laporan',
  rental: 'Sewa',
  rental_document: 'Dokumen',
  ktm: 'KTM',
  chat: 'Chat',
}

export default function AdminAudit() {
  const [action, setAction] = useState('')
  const [actorID, setActorID] = useState('')
  const [actorInput, setActorInput] = useState('')
  const [from, setFrom] = useState('')
  const [to, setTo] = useState('')
  const [offset, setOffset] = useState(0)

  const { data, loading, error, reload } = useApi(
    () => adminApi.audit({
      action,
      actor_id: actorID,
      from,
      to,
      limit: LIMIT,
      offset,
    }),
    [action, actorID, from, to, offset],
  )

  const entries = data?.items || []
  const total = data?.total || 0

  const changeFilter = (setter, value) => {
    setter(value)
    setOffset(0)
  }

  return (
    <AdminShell
      eyebrow="Log Aktivitas Administrator"
      title="Log audit"
      subtitle="Catatan read-only atas seluruh tindakan sensitif: verifikasi, moderasi, unduh dokumen, dan ekspor data."
      actions={
        <>
          <Link to="/admin/reports" className="btn btn-outline">
            <Icon name="flag" /> Laporan
          </Link>
          <span className="badge badge-muted">
            <Icon name="history" size={13} /> {total} entri
          </span>
        </>
      }
    >
      <div className="grid cols-3" style={{ marginBottom: 20 }}>
        <StatCard icon="gavel" value={total} label="Total aktivitas" note="Semua entri audit" />
        <StatCard icon="lock" tone="mint" value="Read-only" label="Akses log" note="Tidak dapat diubah" />
        <StatCard icon="schedule" tone="yellow" value="Tersimpan" label="Retensi" note="Ikuti kebijakan kampus" />
      </div>

      <div className="between mb-4" style={{ flexWrap: 'wrap', gap: 10 }}>
        <div className="toolbar">
          <select
            className="select"
            value={action}
            onChange={(event) => changeFilter(setAction, event.target.value)}
            aria-label="Filter tindakan"
          >
            {ACTIONS.map((item) => (
              <option key={item.value} value={item.value}>{item.label}</option>
            ))}
          </select>
          <div className="input-wrap" style={{ minWidth: 220 }}>
            <Icon name="person_search" />
            <input
              className="input"
              value={actorInput}
              onChange={(event) => setActorInput(event.target.value)}
              onBlur={() => changeFilter(setActorID, actorInput.trim())}
              placeholder="ID admin (opsional)"
              aria-label="Filter ID admin"
            />
          </div>
          <input
            type="date"
            className="input"
            value={from}
            onChange={(event) => changeFilter(setFrom, event.target.value)}
            aria-label="Dari tanggal"
          />
          <input
            type="date"
            className="input"
            value={to}
            onChange={(event) => changeFilter(setTo, event.target.value)}
            aria-label="Sampai tanggal"
          />
        </div>
      </div>

      {error ? <ErrorBox error={error} onRetry={reload} /> : null}

      {loading ? (
        <Loading label="Memuat log audit…" />
      ) : entries.length ? (
        <>
          <div className="table-wrap">
            <table className="table">
              <thead>
                <tr>
                  <th>Waktu</th>
                  <th>Tindakan</th>
                  <th>Aktor</th>
                  <th>Target</th>
                  <th>Alasan</th>
                  <th>IP</th>
                </tr>
              </thead>
              <tbody>
                {entries.map((entry) => (
                  <tr key={entry.id}>
                    <td className="tiny muted">{formatDateTime(entry.created_at)}</td>
                    <td>
                      <span className="badge badge-muted">
                        {ACTIONS.find((item) => item.value === entry.action)?.label || entry.action}
                      </span>
                    </td>
                    <td>
                      <div className="strong">{entry.actor_name || 'Sistem'}</div>
                      <div className="tiny muted mono">{entry.actor_id?.slice(0, 8)}</div>
                    </td>
                    <td>
                      <div className="tiny">
                        {TARGET_LABELS[entry.target_type] || entry.target_type || '—'}
                      </div>
                      <div className="tiny muted mono">{entry.target_id?.slice(0, 8)}</div>
                    </td>
                    <td className="tiny" style={{ maxWidth: 320 }}>
                      {entry.reason || '—'}
                    </td>
                    <td className="tiny muted mono">{entry.ip_address || '—'}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>

          <div className="between" style={{ marginTop: 14 }}>
            <span className="small muted">
              Menampilkan {entries.length} dari {total} entri
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
        <Empty icon="history" title="Belum ada aktivitas">
          {action || actorID || from || to
            ? 'Tidak ada entri yang cocok dengan filter ini.'
            : 'Tindakan admin akan tercatat di sini setelah ada aktivitas.'}
        </Empty>
      )}

      <Alert tone="info" icon="lock" title="Log bersifat append-only">
        Entri audit tidak dapat diedit atau dihapus dari antarmuka. Gunakan filter untuk menelusuri
        aktivitas tertentu.
      </Alert>
    </AdminShell>
  )
}
