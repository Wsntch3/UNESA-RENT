import { useEffect, useRef, useState } from 'react'
import { Link, useRouter } from '../router'
import { useAuth } from '../auth/useAuth'
import { catalogApi } from '../lib/api'
import { useApi } from '../lib/useApi'
import { conditionLabel, dateRange, formatDate, todayISO } from '../lib/format'
import {
  Alert,
  ErrorBox,
  Field,
  Icon,
  Loading,
  PageHeader,
} from '../components/ui'

const EMPTY_FORM = {
  title: '',
  description: '',
  category_id: '',
  campus_location_id: '',
  price_per_day: '',
  condition_percent: 90,
}

export default function ListingForm({ id }) {
  const editing = Boolean(id)
  const { navigate } = useRouter()
  const { ktmVerified, ktmLoading } = useAuth()

  const { data: categoriesData } = useApi(() => catalogApi.categories(), [])
  const { data: locationsData } = useApi(() => catalogApi.locations(), [])

  const {
    data: mineData,
    loading,
    error,
    reload,
  } = useApi(
    () => (editing ? catalogApi.mine({ limit: 60 }) : Promise.resolve(null)),
    [id],
  )

  const listing = editing ? (mineData?.items || []).find((item) => item.id === id) : null

  const {
    data: blocksData,
    reload: reloadBlocks,
  } = useApi(
    () => (editing ? catalogApi.blocks(id) : Promise.resolve(null)),
    [id],
  )

  const [form, setForm] = useState(EMPTY_FORM)
  const [saving, setSaving] = useState(false)
  const [saveError, setSaveError] = useState(null)
  const [feedback, setFeedback] = useState(null)

  const [uploading, setUploading] = useState(false)
  const [photoError, setPhotoError] = useState(null)
  const [photoBusy, setPhotoBusy] = useState(null)
  const fileInput = useRef(null)

  const [blockStart, setBlockStart] = useState(todayISO())
  const [blockEnd, setBlockEnd] = useState(todayISO())
  const [blockReason, setBlockReason] = useState('')
  const [blockBusy, setBlockBusy] = useState(false)
  const [blockError, setBlockError] = useState(null)

  const categories = categoriesData?.categories || []
  const locations = locationsData?.campus_locations || []
  const blocks = blocksData?.blocks || []
  const photos = listing?.photos?.length
    ? listing.photos
    : listing?.cover_photo
      ? [listing.cover_photo]
      : []

  useEffect(() => {
    if (!listing) return
    setForm({
      title: listing.title || '',
      description: listing.description || '',
      category_id: listing.category?.id || '',
      campus_location_id: listing.campus_location?.id || '',
      price_per_day: String(listing.price_per_day || ''),
      condition_percent: listing.condition_percent || 90,
    })
  }, [listing?.id, listing?.updated_at])

  useEffect(() => {
    if (!categories.length) return
    setForm((prev) => {
      if (prev.category_id) return prev
      return { ...prev, category_id: categories[0].id }
    })
  }, [categories])

  useEffect(() => {
    if (!locations.length) return
    setForm((prev) => {
      if (prev.campus_location_id) return prev
      return { ...prev, campus_location_id: locations[0].id }
    })
  }, [locations])

  const set = (key) => (event) => setForm((prev) => ({ ...prev, [key]: event.target.value }))

  const submit = async (event) => {
    event.preventDefault()
    setSaveError(null)
    setFeedback(null)
    setSaving(true)

    const payload = {
      title: form.title.trim(),
      description: form.description.trim(),
      category_id: form.category_id,
      campus_location_id: form.campus_location_id,
      price_per_day: Number(form.price_per_day),
      condition_percent: Number(form.condition_percent),
    }

    try {
      if (editing) {
        await catalogApi.update(id, payload)
        await reload()
        setFeedback({ tone: 'success', message: 'Perubahan listing berhasil disimpan.' })
      } else {
        const result = await catalogApi.create(payload)
        navigate(`/barang-saya/${result.listing.id}/edit`)
      }
    } catch (err) {
      setSaveError(err)
    } finally {
      setSaving(false)
    }
  }

  const uploadPhoto = async (event) => {
    const file = event.target.files?.[0]
    event.target.value = ''
    if (!file) return

    setPhotoError(null)
    if (!['image/jpeg', 'image/png'].includes(file.type)) {
      setPhotoError({ message: 'Format file harus JPEG atau PNG.' })
      return
    }
    if (file.size > 5 * 1024 * 1024) {
      setPhotoError({ message: 'Ukuran file maksimal 5 MB.' })
      return
    }

    setUploading(true)
    try {
      await catalogApi.addPhoto(id, file)
      await reload()
    } catch (err) {
      setPhotoError(err)
    } finally {
      setUploading(false)
    }
  }

  const deletePhoto = async (photoId) => {
    setPhotoBusy(photoId)
    setPhotoError(null)
    try {
      await catalogApi.deletePhoto(id, photoId)
      await reload()
    } catch (err) {
      setPhotoError(err)
    } finally {
      setPhotoBusy(null)
    }
  }

  const addBlock = async (event) => {
    event.preventDefault()
    setBlockError(null)
    setBlockBusy(true)
    try {
      await catalogApi.addBlock(id, {
        start_date: blockStart,
        end_date: blockEnd,
        reason: blockReason.trim(),
      })
      setBlockReason('')
      reloadBlocks()
    } catch (err) {
      setBlockError(err)
    } finally {
      setBlockBusy(false)
    }
  }

  const deleteBlock = async (blockId) => {
    setBlockBusy(true)
    setBlockError(null)
    try {
      await catalogApi.deleteBlock(id, blockId)
      reloadBlocks()
    } catch (err) {
      setBlockError(err)
    } finally {
      setBlockBusy(false)
    }
  }

  if (editing && loading) {
    return (
      <div className="container page">
        <Loading label="Memuat listing…" />
      </div>
    )
  }

  if (editing && (error || !listing)) {
    return (
      <div className="container page">
        <ErrorBox error={error} onRetry={reload} title="Listing tidak bisa dimuat" />
        <div className="empty">
          <Icon name="inventory_2" size={38} />
          <strong>Listing tidak ditemukan</strong>
          <p>Listing mungkin sudah dihapus atau bukan milik akunmu.</p>
          <Link to="/barang-saya" className="btn btn-outline btn-sm">Kembali ke Barang Saya</Link>
        </div>
      </div>
    )
  }

  return (
    <div className="container page">
      <div className="breadcrumbs">
        <Link to="/barang-saya">Barang Saya</Link>
        <Icon name="chevron_right" />
        <span>{editing ? 'Edit Barang' : 'Tambah Barang'}</span>
      </div>

      <PageHeader
        eyebrow="P2P Rental Kampus FT UNESA"
        title={editing ? 'Edit barang' : 'Sewakan barangmu'}
        subtitle="Punya barang yang sedang tidak dipakai? Sewakan ke sesama mahasiswa teknik dengan aman dan praktis via COD kampus."
        actions={
          editing && listing?.is_active ? (
            <Link to={`/barang/${id}`} className="btn btn-outline">
              <Icon name="visibility" /> Lihat di katalog
            </Link>
          ) : null
        }
      />

      {!ktmLoading && !ktmVerified ? (
        <div className="notice-strip mb-4">
          <Icon name="verified_user" />
          <span>
            Akunmu belum terverifikasi KTM. Verifikasi dulu agar listing bisa dipublikasikan.{' '}
            <Link to="/profil" style={{ color: 'var(--ink)', textDecoration: 'underline' }}>
              Verifikasi sekarang
            </Link>
          </span>
        </div>
      ) : null}

      {feedback ? (
        <div className="mb-4">
          <Alert tone={feedback.tone} icon="check_circle">{feedback.message}</Alert>
        </div>
      ) : null}

      <div className="detail-grid">
        <div className="stack">
          <form className="card card-pad-lg stack" onSubmit={submit} noValidate>
            <h2 className="section-title" style={{ fontSize: 18 }}>Informasi barang</h2>

            <Field
              label="Nama barang & tipe model"
              htmlFor="title"
              required
              hint="Tulis nama spesifik beserta merek/tipe agar mudah ditemukan."
              error={saveError?.fields?.title}
            >
              <input
                id="title"
                className="input"
                value={form.title}
                onChange={set('title')}
                placeholder="Contoh: Multimeter Digital Sanwa CD800a"
                maxLength={120}
                required
              />
            </Field>

            <div className="form-grid">
              <Field label="Kategori" htmlFor="category" required error={saveError?.fields?.category_id}>
                <select id="category" className="select" value={form.category_id} onChange={set('category_id')} required>
                  <option value="" disabled>Pilih kategori</option>
                  {categories.map((category) => (
                    <option key={category.id} value={category.id}>{category.name}</option>
                  ))}
                </select>
              </Field>

              <Field label="Lokasi COD" htmlFor="location" required error={saveError?.fields?.campus_location_id}>
                <select id="location" className="select" value={form.campus_location_id} onChange={set('campus_location_id')} required>
                  <option value="" disabled>Pilih lokasi</option>
                  {locations.map((location) => (
                    <option key={location.id} value={location.id}>{location.name}</option>
                  ))}
                </select>
              </Field>

              <Field
                label="Harga sewa per hari (Rp)"
                htmlFor="price"
                required
                hint="1 – 10.000.000"
                error={saveError?.fields?.price_per_day}
              >
                <input
                  id="price"
                  className="input"
                  type="number"
                  inputMode="numeric"
                  min="1"
                  max="10000000"
                  value={form.price_per_day}
                  onChange={set('price_per_day')}
                  placeholder="25000"
                  required
                />
              </Field>

              <Field
                label={`Kondisi barang: ${conditionLabel(form.condition_percent)}`}
                htmlFor="condition"
                hint="Geser untuk menunjukkan kondisi fisik barang."
                error={saveError?.fields?.condition_percent}
              >
                <div className="row">
                  <input
                    id="condition"
                    className="range"
                    type="range"
                    min="1"
                    max="100"
                    value={form.condition_percent}
                    onChange={set('condition_percent')}
                  />
                  <span className="badge badge-muted nowrap">{form.condition_percent}%</span>
                </div>
              </Field>
            </div>

            <Field
              label="Deskripsi barang"
              htmlFor="description"
              hint={`${form.description.length}/2000 karakter. Jelaskan kelengkapan, kondisi, dan aturan pemakaian.`}
              error={saveError?.fields?.description}
            >
              <textarea
                id="description"
                className="textarea"
                value={form.description}
                onChange={set('description')}
                placeholder="Lengkap dengan kabel, charger, tas, dan catatan kondisi. Sebutkan juga kebutuhan praktikum yang cocok."
                maxLength={2000}
              />
            </Field>

            {saveError && !saveError.fields ? (
              <Alert tone="error" icon="error" title="Gagal menyimpan">
                {saveError.message}
              </Alert>
            ) : null}

            <div className="row" style={{ justifyContent: 'flex-end' }}>
              <Link to="/barang-saya" className="btn btn-ghost">Batal</Link>
              <button type="submit" className="btn btn-primary" disabled={saving}>
                {saving ? <span className="spinner" /> : <Icon name="save" />}
                {saving ? 'Menyimpan…' : editing ? 'Simpan Perubahan' : 'Buat Listing'}
              </button>
            </div>
          </form>

          {editing ? (
            <section className="card card-pad-lg">
              <div className="between">
                <div>
                  <h2 className="section-title" style={{ fontSize: 18 }}>Foto barang</h2>
                  <p className="hint mt-1">{photos.length} dari 3 foto terunggah. JPEG/PNG, maks 5 MB.</p>
                </div>
                <span className="badge badge-muted">Maks. 3 foto</span>
              </div>

              {photoError ? (
                <div className="mt-3">
                  <Alert tone="error" icon="error">{photoError.message}</Alert>
                </div>
              ) : null}

              <div className="photo-grid mt-4">
                {photos.map((photo, index) => (
                  <div className="photo-tile" key={photo.id || index}>
                    <img src={photo.url} alt={`Foto ${index + 1}`} />
                    {index === 0 ? (
                      <span className="badge badge-primary primary-tag">
                        <Icon name="star" filled size={12} /> Utama
                      </span>
                    ) : null}
                    <button
                      type="button"
                      className="remove"
                      aria-label="Hapus foto"
                      disabled={photoBusy === photo.id}
                      onClick={() => deletePhoto(photo.id)}
                    >
                      <Icon name={photoBusy === photo.id ? 'hourglass_top' : 'close'} />
                    </button>
                  </div>
                ))}

                {photos.length < 3 ? (
                  <label className="upload-tile" style={{ cursor: uploading ? 'wait' : 'pointer' }}>
                    {uploading ? <span className="spinner" /> : <Icon name="add_photo_alternate" size={26} />}
                    <span>{uploading ? 'Mengunggah…' : 'Tambah foto'}</span>
                    <input
                      ref={fileInput}
                      type="file"
                      accept="image/jpeg,image/png"
                      hidden
                      onChange={uploadPhoto}
                      disabled={uploading}
                    />
                  </label>
                ) : null}
              </div>

              <div className="alert alert-info mt-4">
                <Icon name="lightbulb" filled />
                <div className="alert-body">
                  Ambil foto dengan pencahayaan terang yang menunjukkan kelengkapan (kabel/aksesori)
                  dan kondisi fisik alat secara nyata.
                </div>
              </div>
            </section>
          ) : (
            <div className="alert alert-info">
              <Icon name="photo_camera" />
              <div className="alert-body">
                Setelah listing dibuat, kamu bisa mengunggah maksimal 3 foto dan mengatur tanggal
                yang diblokir di halaman edit.
              </div>
            </div>
          )}

          {editing ? (
            <section className="card card-pad-lg">
              <h2 className="section-title" style={{ fontSize: 18 }}>Ketersediaan</h2>
              <p className="hint mt-1">
                Blokir tanggal saat barang sedang dipakai sendiri atau tidak bisa disewakan.
              </p>

              <form className="stack mt-4" onSubmit={addBlock}>
                <div className="form-grid">
                  <Field label="Mulai" htmlFor="block-start">
                    <input
                      id="block-start"
                      type="date"
                      className="input"
                      min={todayISO()}
                      value={blockStart}
                      onChange={(event) => {
                        setBlockStart(event.target.value)
                        if (event.target.value > blockEnd) setBlockEnd(event.target.value)
                      }}
                    />
                  </Field>
                  <Field label="Selesai" htmlFor="block-end">
                    <input
                      id="block-end"
                      type="date"
                      className="input"
                      min={blockStart}
                      value={blockEnd}
                      onChange={(event) => setBlockEnd(event.target.value)}
                    />
                  </Field>
                </div>
                <Field label="Alasan (opsional)" htmlFor="block-reason">
                  <input
                    id="block-reason"
                    className="input"
                    value={blockReason}
                    onChange={(event) => setBlockReason(event.target.value)}
                    placeholder="Contoh: Dipakai sendiri untuk praktikum"
                  />
                </Field>
                <div className="row" style={{ justifyContent: 'flex-end' }}>
                  <button type="submit" className="btn btn-soft btn-sm" disabled={blockBusy}>
                    {blockBusy ? <span className="spinner" /> : <Icon name="event_busy" />}
                    Blokir tanggal
                  </button>
                </div>
              </form>

              {blockError ? (
                <div className="mt-3">
                  <Alert tone="error" icon="error">{blockError.message}</Alert>
                </div>
              ) : null}

              <div className="stack-sm mt-4">
                {blocks.length ? (
                  blocks.map((block) => (
                    <div className="block-row" key={block.id}>
                      <div>
                        <div className="strong">{dateRange(block.start_date, block.end_date)}</div>
                        {block.reason ? <div className="tiny muted">{block.reason}</div> : null}
                      </div>
                      <button
                        type="button"
                        className="btn btn-danger-soft btn-sm"
                        disabled={blockBusy}
                        onClick={() => deleteBlock(block.id)}
                      >
                        <Icon name="delete" /> Hapus
                      </button>
                    </div>
                  ))
                ) : (
                  <p className="hint">Belum ada tanggal yang diblokir.</p>
                )}
              </div>
            </section>
          ) : null}
        </div>

        <aside className="sticky-side stack">
          <div className="card card-pad">
            <h3 className="card-title">Status listing</h3>
            <div className="stack-sm mt-3">
              <div className="kv">
                <dt>Status</dt>
                <dd>
                  <span className={`badge ${listing?.is_active ? 'badge-success' : 'badge-warning'}`}>
                    {editing ? (listing?.is_active ? 'Aktif' : 'Tidak aktif') : 'Draft baru'}
                  </span>
                </dd>
              </div>
              {editing ? (
                <>
                  <div className="kv">
                    <dt>Dibuat</dt>
                    <dd>{formatDate(listing?.created_at, 'short')}</dd>
                  </div>
                  <div className="kv">
                    <dt>Diperbarui</dt>
                    <dd>{formatDate(listing?.updated_at, 'short')}</dd>
                  </div>
                  <div className="kv">
                    <dt>Foto</dt>
                    <dd>{photos.length} / 3</dd>
                  </div>
                </>
              ) : null}
            </div>
          </div>

          <div className="card card-pad">
            <h3 className="card-title">
              <Icon name="tips_and_updates" /> Tips cepat
            </h3>
            <ul className="info-list mt-3">
              <li><Icon name="check_circle" /><span>Gunakan judul spesifik: merek + tipe.</span></li>
              <li><Icon name="check_circle" /><span>Sebutkan kelengkapan di deskripsi.</span></li>
              <li><Icon name="check_circle" /><span>Pasang harga wajar dibanding barang sejenis.</span></li>
              <li><Icon name="check_circle" /><span>Blokir tanggal saat barang tidak tersedia.</span></li>
            </ul>
          </div>

          {editing ? (
            <div className="card card-pad">
              <h3 className="card-title">Tautan publik</h3>
              <p className="hint mt-1">Bagikan listing ke teman satu angkatan.</p>
              <Link to={`/barang/${id}`} className="btn btn-outline btn-sm btn-block mt-3">
                <Icon name="open_in_new" /> Buka halaman barang
              </Link>
            </div>
          ) : null}
        </aside>
      </div>
    </div>
  )
}
