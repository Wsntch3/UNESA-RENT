import { Link } from '../router'
import { useAuth } from '../auth/useAuth'
import { adminApi, adminKtmApi, catalogApi, rentalApi } from '../lib/api'
import { useApi } from '../lib/useApi'
import { dateRange, formatRupiah, ktmStatus, rentalStatus, timeAgo } from '../lib/format'
import { AdminShell } from '../components/AdminShell'
import {
  Alert,
  Avatar,
  Empty,
  ErrorBox,
  Icon,
  Loading,
  StatCard,
} from '../components/ui'

export default function AdminDashboard() {
  const { user } = useAuth()

  const { data, loading, error, reload } = useApi(
    async () => {
      const [pending, verified, rejected, active, takenDown, adminStats, rentalStats, recent] =
        await Promise.all([
          adminKtmApi.list({ status: 'pending', limit: 60 }),
          adminKtmApi.list({ status: 'verified', limit: 60 }),
          adminKtmApi.list({ status: 'rejected', limit: 60 }),
          catalogApi.adminList({ status: 'active', limit: 1 }),
          catalogApi.adminList({ status: 'taken_down', limit: 6 }),
          adminApi.stats(),
          rentalApi.adminStats(),
          rentalApi.adminList({ limit: 5 }),
        ])
      return {
        pending: pending.submissions || [],
        verified: verified.submissions || [],
        rejected: rejected.submissions || [],
        activeTotal: active.total || 0,
        takenDown: takenDown.items || [],
        takenDownTotal: takenDown.total || 0,
        adminStats: adminStats.stats || {},
        rentalStats: rentalStats.stats || {},
        recent: recent.items || [],
      }
    },
    [],
  )

  const pending = data?.pending || []
  const verified = data?.verified || []
  const rejected = data?.rejected || []
  const takenDown = data?.takenDown || []
  const activeTotal = data?.activeTotal || 0
  const takenDownTotal = data?.takenDownTotal || 0
  const adminStats = data?.adminStats || {}
  const rentalStats = data?.rentalStats || {}
  const recent = data?.recent || []

  return (
    <AdminShell
      eyebrow="Portal Administrasi Kampus FT"
      title={`Ruang kerja, ${user?.name?.split(' ')[0] || 'Admin'}`}
      subtitle="Pantau antrean verifikasi identitas mahasiswa dan kelancaran sirkulasi inventaris komunitas UNESA Rent."
      actions={
        <>
          <Link to="/admin/ktm" className="btn btn-primary">
            <Icon name="verified_user" /> Tinjau KTM
          </Link>
          <Link to="/admin/listings" className="btn btn-outline">
            <Icon name="inventory_2" /> Moderasi Listing
          </Link>
          <Link to="/admin/rentals" className="btn btn-outline">
            <Icon name="receipt_long" /> Monitoring Sewa
          </Link>
        </>
      }
    >
      {pending.length ? (
        <div className="notice-strip mb-4">
          <Icon name="notifications_active" />
          <span>
            {pending.length} berkas KTM menunggu verifikasi. SLA peninjauan 1×24 jam kerja.
          </span>
        </div>
      ) : null}

      {error ? <ErrorBox error={error} onRetry={reload} title="Gagal memuat data admin" /> : null}

      {loading ? (
        <Loading label="Memuat ringkasan admin…" />
      ) : (
        <>
          <div className="grid cols-4" style={{ marginBottom: 22 }}>
            <StatCard
              icon="pending_actions"
              tone="yellow"
              value={pending.length}
              label="KTM menunggu"
              note="Prioritas peninjauan"
            />
            <StatCard
              icon="how_to_reg"
              tone="mint"
              value={verified.length}
              label="KTM terverifikasi"
              note={`${rejected.length} pernah ditolak`}
            />
            <StatCard
              icon="inventory_2"
              value={activeTotal}
              label="Listing aktif"
              note="Tampil di katalog"
            />
            <StatCard
              icon="report_problem"
              tone="danger"
              value={takenDownTotal}
              label="Listing diturunkan"
              note="Perlu pemantauan"
            />
          </div>

          <div className="grid cols-4" style={{ marginBottom: 22 }}>
            <StatCard
              icon="groups"
              value={adminStats.total_users || 0}
              label="Total pengguna"
              note={`${adminStats.verified_users || 0} terverifikasi`}
            />
            <StatCard
              icon="receipt_long"
              tone="mint"
              value={rentalStats.total || 0}
              label="Total penyewaan"
              note={`${rentalStats.overdue || 0} terlambat`}
            />
            <StatCard
              icon="flag"
              tone="yellow"
              value={adminStats.open_reports || 0}
              label="Laporan terbuka"
              note="Menunggu peninjauan"
            />
            <StatCard
              icon="description"
              value={adminStats.total_documents || 0}
              label="Dokumen tersimpan"
              note="Arsip privat transaksi"
            />
          </div>

          <section className="card card-pad-lg" style={{ marginBottom: 20 }}>
            <div className="between" style={{ marginBottom: 14 }}>
              <div>
                <h2 className="section-title" style={{ fontSize: 18 }}>Penyewaan terbaru</h2>
                <p className="hint mt-1">Sewa yang baru diajukan oleh mahasiswa.</p>
              </div>
              <Link to="/admin/rentals" className="btn btn-ghost btn-sm">
                Semua <Icon name="arrow_forward" />
              </Link>
            </div>
            {recent.length ? (
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
                      <th>Aksi</th>
                    </tr>
                  </thead>
                  <tbody>
                    {recent.map((item) => {
                      const status = rentalStatus(item.status)
                      return (
                        <tr key={item.id}>
                          <td className="strong">{item.listing?.title}</td>
                          <td>{item.renter?.name}</td>
                          <td>{item.owner?.name}</td>
                          <td className="tiny muted">{dateRange(item.start_date, item.end_date)}</td>
                          <td className="strong">{formatRupiah(item.total_price)}</td>
                          <td>
                            <span className={`badge badge-${status.tone}`}>{status.label}</span>
                          </td>
                          <td>
                            <Link to="/admin/rentals" className="btn btn-outline btn-sm">
                              Kelola
                            </Link>
                          </td>
                        </tr>
                      )
                    })}
                  </tbody>
                </table>
              </div>
            ) : (
              <Empty icon="receipt_long" title="Belum ada penyewaan">
                Transaksi baru akan muncul di sini setelah mahasiswa mengajukan sewa.
              </Empty>
            )}
          </section>

          <div className="grid cols-2" style={{ alignItems: 'start', gap: 20 }}>
            <section className="card card-pad-lg">
              <div className="between" style={{ marginBottom: 14 }}>
                <div>
                  <h2 className="section-title" style={{ fontSize: 18 }}>Antrean verifikasi KTM</h2>
                  <p className="hint mt-1">Berkas terbaru yang menunggu keputusan.</p>
                </div>
                <Link to="/admin/ktm" className="btn btn-ghost btn-sm">
                  Semua <Icon name="arrow_forward" />
                </Link>
              </div>

              {pending.length ? (
                <div className="stack-sm">
                  {pending.slice(0, 5).map((submission) => {
                    const meta = ktmStatus(submission.status)
                    return (
                      <div className="party" key={submission.id}>
                        <Avatar name={submission.user_name} />
                        <div className="party-body">
                          <div className="party-name">{submission.user_name}</div>
                          <div className="party-meta">
                            NIM {submission.user_nim} · {timeAgo(submission.submitted_at)}
                          </div>
                        </div>
                        <span className={`badge badge-${meta.tone}`}>{meta.label}</span>
                        <Link to="/admin/ktm" className="btn btn-outline btn-sm">
                          Tinjau
                        </Link>
                      </div>
                    )
                  })}
                </div>
              ) : (
                <Empty icon="task_alt" title="Antrean bersih">
                  Tidak ada berkas KTM yang menunggu.
                </Empty>
              )}
            </section>

            <section className="card card-pad-lg">
              <div className="between" style={{ marginBottom: 14 }}>
                <div>
                  <h2 className="section-title" style={{ fontSize: 18 }}>Listing diturunkan</h2>
                  <p className="hint mt-1">Barang yang disembunyikan dari katalog publik.</p>
                </div>
                <Link to="/admin/listings" className="btn btn-ghost btn-sm">
                  Semua <Icon name="arrow_forward" />
                </Link>
              </div>

              {takenDown.length ? (
                <div className="stack-sm">
                  {takenDown.slice(0, 5).map((listing) => (
                    <div className="party" key={listing.id}>
                      <Avatar name={listing.owner?.name} tone="yellow" />
                      <div className="party-body">
                        <div className="party-name truncate">{listing.title}</div>
                        <div className="party-meta truncate">
                          {listing.owner?.name} · {formatRupiah(listing.price_per_day)}/hari
                        </div>
                        {listing.takedown_reason ? (
                          <div className="tiny text-danger mt-1 truncate">{listing.takedown_reason}</div>
                        ) : null}
                      </div>
                      <Link to="/admin/listings" className="btn btn-soft btn-sm">
                        Kelola
                      </Link>
                    </div>
                  ))}
                </div>
              ) : (
                <Empty icon="inventory_2" title="Tidak ada listing diturunkan">
                  Semua listing aktif mematuhi ketentuan komunitas.
                </Empty>
              )}
            </section>
          </div>

          <section className="card card-pad-lg" style={{ marginTop: 20 }}>
            <div className="between" style={{ marginBottom: 14 }}>
              <div>
                <h2 className="section-title" style={{ fontSize: 18 }}>Panduan singkat moderator</h2>
                <p className="hint mt-1">Alur standar operasional verifikasi dan moderasi.</p>
              </div>
              <span className="badge badge-primary">SOP FT-2026</span>
            </div>
            <div className="grid cols-3">
              <div className="spec">
                <span>1. Verifikasi KTM</span>
                <strong>Cocokkan nama & NIM pada kartu dengan data akun.</strong>
              </div>
              <div className="spec">
                <span>2. Moderasi listing</span>
                <strong>Turunkan barang yang melanggar dengan alasan jelas.</strong>
              </div>
              <div className="spec">
                <span>3. Audit</span>
                <strong>Setiap keputusan tercatat pada log sistem.</strong>
              </div>
            </div>
            <Alert tone="info" icon="lock" title="Dokumen privat">
              Berkas KTM hanya boleh diakses untuk keperluan verifikasi. Jangan mengunduh atau
              membagikannya ke pihak lain.
            </Alert>
          </section>
        </>
      )}
    </AdminShell>
  )
}
