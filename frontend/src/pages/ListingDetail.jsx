import { useEffect, useMemo, useState } from 'react'
import { Link, useRouter } from '../router'
import { useAuth } from '../auth/useAuth'
import { catalogApi, rentalApi } from '../lib/api'
import { useApi } from '../lib/useApi'
import {
  categoryIcon,
  conditionLabel,
  dateRange,
  daysInclusive,
  formatDate,
  formatRupiah,
  listingPhotoUrl,
  todayISO,
} from '../lib/format'
import { ListingCard } from '../components/ListingCard'
import { Alert, Avatar, Empty, ErrorBox, Icon, Loading } from '../components/ui'

export default function ListingDetail({ id }) {
  const { navigate } = useRouter()
  const { isAuthenticated, ktmVerified, user } = useAuth()

  const { data, loading, error, reload } = useApi(() => catalogApi.detail(id), [id])
  const listing = data?.listing || null

  const [photoIndex, setPhotoIndex] = useState(0)
  const [startDate, setStartDate] = useState('')
  const [endDate, setEndDate] = useState('')
  const [submitting, setSubmitting] = useState(false)
  const [submitError, setSubmitError] = useState(null)
  const [createdRental, setCreatedRental] = useState(null)
  const [unavailable, setUnavailable] = useState(null)

  const categorySlug = listing?.category?.slug || ''
  const { data: relatedData } = useApi(
    () => (categorySlug ? catalogApi.search({ category: categorySlug, limit: 5 }) : Promise.resolve({ items: [] })),
    [categorySlug],
  )

  useEffect(() => {
    setPhotoIndex(0)
  }, [id])

  useEffect(() => {
    if (!startDate || !endDate || endDate < startDate) {
      setUnavailable(null)
      return undefined
    }
    let alive = true
    catalogApi
      .availability(id, startDate, endDate)
      .then((result) => {
        if (alive) setUnavailable(result.unavailable || [])
      })
      .catch(() => {
        if (alive) setUnavailable(null)
      })
    return () => {
      alive = false
    }
  }, [id, startDate, endDate])

  const photos = useMemo(() => {
    if (!listing) return []
    if (listing.photos?.length) return listing.photos
    if (listing.cover_photo) return [listing.cover_photo]
    return []
  }, [listing])

  const days = daysInclusive(startDate, endDate)
  const total = listing ? days * listing.price_per_day : 0
  const isOwner = listing && user?.id === listing.owner?.id
  const conflict = Array.isArray(unavailable) && unavailable.length > 0

  const related = (relatedData?.items || []).filter((item) => item.id !== id).slice(0, 4)

  if (loading) {
    return (
      <div className="container page">
        <Loading label="Memuat detail barang…" />
      </div>
    )
  }

  if (error || !listing) {
    return (
      <div className="container page">
        <ErrorBox error={error} onRetry={reload} title="Barang tidak bisa dimuat" />
        {!error ? (
          <Empty
            icon="inventory_2"
            title="Barang tidak ditemukan"
            action={
              <Link to="/jelajahi" className="btn btn-outline btn-sm">
                Kembali ke katalog
              </Link>
            }
          >
            Barang mungkin sudah dihapus atau dinonaktifkan admin.
          </Empty>
        ) : null}
      </div>
    )
  }

  const handleSubmit = async (event) => {
    event.preventDefault()
    setSubmitError(null)
    setCreatedRental(null)

    if (!isAuthenticated) {
      navigate(`/masuk?next=${encodeURIComponent(`/barang/${id}`)}`)
      return
    }
    if (!ktmVerified) {
      setSubmitError({
        message: 'Akun kamu belum terverifikasi KTM. Verifikasi dulu sebelum mengajukan sewa.',
        code: 'ktm_not_verified',
      })
      return
    }
    if (!startDate || !endDate) {
      setSubmitError({ message: 'Tentukan tanggal mulai dan tanggal selesai sewa.', code: 'validation_error' })
      return
    }
    if (days < 1) {
      setSubmitError({ message: 'Tanggal selesai harus sama atau setelah tanggal mulai.', code: 'validation_error' })
      return
    }

    setSubmitting(true)
    try {
      const result = await rentalApi.submit(id, { start_date: startDate, end_date: endDate })
      setCreatedRental(result.rental)
      setStartDate('')
      setEndDate('')
      setUnavailable(null)
    } catch (err) {
      setSubmitError(err)
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <div className="container page">
      <div className="breadcrumbs">
        <Link to="/">Beranda</Link>
        <Icon name="chevron_right" />
        <Link to="/jelajahi">Jelajahi Barang</Link>
        <Icon name="chevron_right" />
        <span>{listing.category?.name}</span>
        <Icon name="chevron_right" />
        <span className="truncate" style={{ maxWidth: 260 }}>{listing.title}</span>
      </div>

      <div className="detail-grid">
        <div className="stack">
          <div className="gallery">
            <div className="main">
              {photos[photoIndex] ? (
                <img src={photos[photoIndex].url} alt={listing.title} />
              ) : (
                <div className="placeholder">
                  <Icon name={categoryIcon(listing.category?.slug)} size={48} />
                </div>
              )}
              <div className="thumb-top" style={{ position: 'absolute' }}>
                <span className="badge badge-primary">
                  <Icon name="verified" filled size={13} /> KTM Terverifikasi
                </span>
                <span className="badge badge-dark">
                  <Icon name="place" size={13} /> {listing.campus_location?.name}
                </span>
              </div>
            </div>
            {photos.slice(0, 4).map((photo, index) => (
              <button
                key={photo.id || index}
                type="button"
                className="thumb"
                onClick={() => setPhotoIndex(index)}
                aria-label={`Lihat foto ${index + 1}`}
                style={{
                  border: index === photoIndex ? '2px solid var(--primary)' : '1px solid var(--border-soft)',
                  padding: 0,
                  cursor: 'pointer',
                  background: 'var(--subtle)',
                }}
              >
                <img src={photo.url} alt={`${listing.title} foto ${index + 1}`} />
              </button>
            ))}
          </div>

          <div className="card card-pad-lg">
            <div className="between" style={{ alignItems: 'flex-start' }}>
              <div>
                <span className="eyebrow">{listing.category?.name} &middot; {listing.campus_location?.name}</span>
                <h1 className="page-title mt-2" style={{ fontSize: 28 }}>{listing.title}</h1>
              </div>
              <span className={`badge ${listing.is_active ? 'badge-success' : 'badge-danger'}`}>
                {listing.is_active ? 'Tersedia' : 'Tidak aktif'}
              </span>
            </div>

            <div className="row-wrap mt-3">
              <span className="badge badge-muted">
                <Icon name="thumb_up" size={13} /> Kondisi {conditionLabel(listing.condition_percent)}
              </span>
              <span className="badge badge-muted">
                <Icon name="calendar_today" size={13} /> Ditambahkan {formatDate(listing.created_at, 'short')}
              </span>
              <span className="badge badge-muted">
                <Icon name="visibility" size={13} /> Aktif di katalog
              </span>
              <Link to={`/lapor?listing=${listing.id}`} className="btn btn-ghost btn-sm">
                <Icon name="flag" /> Laporkan
              </Link>
            </div>

            <hr className="divider" style={{ margin: '18px 0' }} />

            <h2 className="section-title" style={{ fontSize: 18 }}>Deskripsi barang</h2>
            <p className="body-text mt-2" style={{ whiteSpace: 'pre-line', lineHeight: 1.65 }}>
              {listing.description || 'Penyedia belum menambahkan deskripsi.'}
            </p>

            <h2 className="section-title mt-6" style={{ fontSize: 18 }}>Spesifikasi & info</h2>
            <div className="spec-grid mt-3">
              <div className="spec">
                <span>Kondisi barang</span>
                <strong>{conditionLabel(listing.condition_percent)} ({listing.condition_percent}%)</strong>
              </div>
              <div className="spec">
                <span>Kategori</span>
                <strong>{listing.category?.name}</strong>
              </div>
              <div className="spec">
                <span>Lokasi serah-terima</span>
                <strong>{listing.campus_location?.name}</strong>
              </div>
              <div className="spec">
                <span>Tarif sewa</span>
                <strong>{formatRupiah(listing.price_per_day)} / hari</strong>
              </div>
            </div>

            {listing.blocked_periods?.length ? (
              <>
                <h2 className="section-title mt-6" style={{ fontSize: 18 }}>Tanggal tidak tersedia</h2>
                <div className="stack-sm mt-3">
                  {listing.blocked_periods.map((period, index) => (
                    <div className="block-row" key={`${period.start_date}-${index}`}>
                      <span>
                        <Icon name="event_busy" size={15} /> {dateRange(period.start_date, period.end_date)}
                      </span>
                      <span className="badge badge-muted">Diblokir penyedia</span>
                    </div>
                  ))}
                </div>
              </>
            ) : null}
          </div>

          <div className="card card-pad-lg">
            <h2 className="section-title" style={{ fontSize: 18 }}>Penyedia barang</h2>
            <div className="party mt-3">
              <Avatar name={listing.owner?.name} size="lg" />
              <div className="party-body">
                <div className="party-name">{listing.owner?.name}</div>
                <div className="party-meta">Mahasiswa FT UNESA &middot; Penyedia terverifikasi</div>
              </div>
              <span className="badge badge-success">
                <Icon name="verified" filled size={13} /> KTM
              </span>
            </div>
            <p className="hint mt-3">
              Data pribadi penyedia seperti email, NIM, dan nomor WhatsApp tidak ditampilkan. Gunakan
              fitur chat setelah pengajuan sewa dibuat untuk berkoordinasi.
            </p>
          </div>
        </div>

        <aside className="sticky-side">
          <div className="card card-pad-lg">
            <div className="price price-lg">
              {formatRupiah(listing.price_per_day)} <small>/ hari</small>
            </div>
            <p className="hint mt-1">Harga ditetapkan penyedia. Tanpa biaya admin.</p>

            <hr className="divider" style={{ margin: '16px 0' }} />

            {isOwner ? (
              <div className="stack">
                <Alert tone="info" icon="info" title="Ini barang milikmu">
                  Kamu tidak bisa menyewa barang sendiri. Kelola ketersediaan dan pengajuan masuk dari
                  dashboard.
                </Alert>
                <Link to={`/barang-saya/${listing.id}/edit`} className="btn btn-primary btn-block">
                  <Icon name="tune" /> Kelola Barang
                </Link>
                <Link to="/dashboard" className="btn btn-outline btn-block">
                  <Icon name="space_dashboard" /> Dashboard Penyewaan
                </Link>
              </div>
            ) : (
              <form className="stack" onSubmit={handleSubmit}>
                <div className="field">
                  <label className="label" htmlFor="start-date">Tanggal mulai</label>
                  <input
                    id="start-date"
                    type="date"
                    className="input"
                    min={todayISO()}
                    value={startDate}
                    onChange={(event) => {
                      setStartDate(event.target.value)
                      if (endDate && event.target.value > endDate) setEndDate(event.target.value)
                    }}
                  />
                </div>
                <div className="field">
                  <label className="label" htmlFor="end-date">Tanggal selesai</label>
                  <input
                    id="end-date"
                    type="date"
                    className="input"
                    min={startDate || todayISO()}
                    value={endDate}
                    onChange={(event) => setEndDate(event.target.value)}
                  />
                  <span className="hint">Tanggal selesai termasuk hari terakhir sewa.</span>
                </div>

                {days > 0 ? (
                  <div>
                    <dl className="kv">
                      <dt>Durasi</dt>
                      <dd>{days} hari</dd>
                    </dl>
                    <dl className="kv">
                      <dt>{formatRupiah(listing.price_per_day)} × {days} hari</dt>
                      <dd>{formatRupiah(total)}</dd>
                    </dl>
                    <dl className="kv total">
                      <dt>Total sewa</dt>
                      <dd>{formatRupiah(total)}</dd>
                    </dl>
                  </div>
                ) : null}

                {conflict ? (
                  <Alert tone="warning" icon="event_busy" title="Tanggal tidak tersedia">
                    Sebagian tanggal sudah diblokir atau disewa orang lain. Pilih periode lain.
                  </Alert>
                ) : null}

                {createdRental ? (
                  <Alert tone="success" icon="check_circle" title="Pengajuan terkirim">
                    Menunggu persetujuan penyedia.{' '}
                    <Link to={`/penyewaan/${createdRental.id}`}>Lihat pengajuan</Link>
                  </Alert>
                ) : null}

                {submitError ? (
                  <Alert tone="error" icon="error" title="Pengajuan gagal">
                    {submitError.message}
                  </Alert>
                ) : null}

                {!isAuthenticated ? (
                  <Link
                    to={`/masuk?next=${encodeURIComponent(`/barang/${id}`)}`}
                    className="btn btn-primary btn-block btn-lg"
                  >
                    <Icon name="login" /> Masuk untuk menyewa
                  </Link>
                ) : !ktmVerified ? (
                  <Link to="/profil" className="btn btn-primary btn-block">
                    <Icon name="verified_user" /> Verifikasi KTM dulu
                  </Link>
                ) : (
                  <button
                    type="submit"
                    className="btn btn-primary btn-block btn-lg"
                    disabled={submitting || !startDate || !endDate || days < 1 || conflict}
                  >
                    {submitting ? <span className="spinner" /> : <Icon name="send" />}
                    {submitting ? 'Mengirim…' : 'Ajukan Sewa'}
                  </button>
                )}

                <p className="hint">
                  Pengajuan belum transaksi final. Penyedia dapat menyetujui atau menolak. Pembayaran
                  dilakukan tunai saat COD.
                </p>
              </form>
            )}
          </div>

          <div className="card card-pad mt-4" style={{ marginTop: 16 }}>
            <div className="info-list">
              <li>
                <Icon name="verified_user" />
                <span>Penyedia sudah diverifikasi KTM mahasiswa FT UNESA.</span>
              </li>
              <li>
                <Icon name="handshake" />
                <span>Serah-terima COD di {listing.campus_location?.name}.</span>
              </li>
              <li>
                <Icon name="description" />
                <span>Surat perjanjian digital tersedia setelah rental disetujui.</span>
              </li>
            </div>
          </div>
        </aside>
      </div>

      {related.length ? (
        <section style={{ marginTop: 40 }}>
          <div className="between" style={{ marginBottom: 14 }}>
            <h2 className="section-title">Barang lain di {listing.category?.name}</h2>
            <Link to={`/jelajahi?category=${listing.category?.slug}`} className="btn btn-ghost btn-sm">
              Lihat semua <Icon name="arrow_forward" />
            </Link>
          </div>
          <div className="listing-grid">
            {related.map((item) => (
              <ListingCard key={item.id} listing={item} />
            ))}
          </div>
        </section>
      ) : null}
    </div>
  )
}
