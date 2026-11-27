import { useEffect, useMemo, useState } from 'react'
import { useLocation } from '../router'
import { catalogApi } from '../lib/api'
import { useApi } from '../lib/useApi'
import { ListingCard, ListingCardSkeleton } from '../components/ListingCard'
import { Empty, ErrorBox, Icon, PageHeader } from '../components/ui'

const LIMIT = 12

const PRICE_OPTIONS = [
  { value: '', label: 'Semua harga' },
  { value: 'low', label: '< Rp 20.000 / hari' },
  { value: 'mid', label: 'Rp 20.000 – Rp 50.000' },
  { value: 'high', label: '> Rp 50.000 / hari' },
]

const SORT_OPTIONS = [
  { value: 'newest', label: 'Terbaru' },
  { value: 'price_asc', label: 'Termurah' },
  { value: 'price_desc', label: 'Termahal' },
]

export default function Catalog() {
  const { query } = useLocation()

  const [searchInput, setSearchInput] = useState(query.q || '')
  const [appliedQ, setAppliedQ] = useState(query.q || '')
  const [category, setCategory] = useState(query.category || '')
  const [location, setLocation] = useState('')
  const [price, setPrice] = useState('')
  const [sort, setSort] = useState('newest')
  const [offset, setOffset] = useState(0)

  useEffect(() => {
    const timer = setTimeout(() => {
      setAppliedQ(searchInput.trim())
      setOffset(0)
    }, 350)
    return () => clearTimeout(timer)
  }, [searchInput])

  
  useEffect(() => {
    setSearchInput(query.q || '')
    setAppliedQ(query.q || '')
    setCategory(query.category || '')
    setOffset(0)
  }, [query.q, query.category])

  const { data: categoriesData } = useApi(() => catalogApi.categories(), [])
  const { data: locationsData } = useApi(() => catalogApi.locations(), [])

  const categories = categoriesData?.categories || []
  const locations = locationsData?.campus_locations || []

  const params = useMemo(() => {
    const p = { q: appliedQ, category, location, sort, limit: LIMIT, offset }
    if (price === 'low') p.max_price = 20000
    if (price === 'mid') {
      p.min_price = 20000
      p.max_price = 50000
    }
    if (price === 'high') p.min_price = 50000
    return p
  }, [appliedQ, category, location, price, sort, offset])

  const { data, loading, error, reload } = useApi(() => catalogApi.search(params), [params])

  const items = data?.items || []
  const total = data?.total || 0
  const hasFilter = Boolean(appliedQ || category || location || price || sort !== 'newest')

  const reset = () => {
    setSearchInput('')
    setAppliedQ('')
    setCategory('')
    setLocation('')
    setPrice('')
    setSort('newest')
    setOffset(0)
  }

  const change = (setter) => (event) => {
    setter(event.target.value)
    setOffset(0)
  }

  return (
    <div className="container page">
      <PageHeader
        eyebrow="Katalog FT UNESA"
        title="Jelajahi barang"
        subtitle="Temukan perlengkapan praktikum, tugas, dan kegiatan kampus dari sesama mahasiswa Fakultas Teknik."
        actions={
          <span className="badge badge-primary">
            <Icon name="verified" filled size={13} /> Penyedia terverifikasi KTM
          </span>
        }
      />

      <div className="card card-pad" style={{ marginBottom: 20 }}>
        <div className="toolbar">
          <div className="input-wrap" style={{ flex: 1 }}>
            <Icon name="search" />
            <input
              className="input"
              value={searchInput}
              onChange={(event) => setSearchInput(event.target.value)}
              placeholder="Cari nama barang, merek, atau tipe…"
              aria-label="Cari barang"
            />
          </div>
          <select className="select" value={location} onChange={change(setLocation)} aria-label="Lokasi COD">
            <option value="">Semua lokasi COD</option>
            {locations.map((item) => (
              <option key={item.id} value={item.id}>
                {item.name}
              </option>
            ))}
          </select>
          <select className="select" value={price} onChange={change(setPrice)} aria-label="Rentang harga">
            {PRICE_OPTIONS.map((option) => (
              <option key={option.value} value={option.value}>
                {option.label}
              </option>
            ))}
          </select>
          <select className="select" value={sort} onChange={change(setSort)} aria-label="Urutkan">
            {SORT_OPTIONS.map((option) => (
              <option key={option.value} value={option.value}>
                Urutkan: {option.label}
              </option>
            ))}
          </select>
          {hasFilter ? (
            <button type="button" className="btn btn-ghost btn-sm" onClick={reset}>
              <Icon name="restart_alt" /> Reset
            </button>
          ) : null}
        </div>

        <div className="chip-row mt-4">
          <button
            type="button"
            className={`chip ${category === '' ? 'active' : ''}`.trim()}
            onClick={() => {
              setCategory('')
              setOffset(0)
            }}
          >
            <Icon name="apps" /> Semua Kategori
          </button>
          {categories.map((item) => (
            <button
              key={item.id}
              type="button"
              className={`chip ${category === item.slug ? 'active' : ''}`.trim()}
              onClick={() => {
                setCategory(item.slug)
                setOffset(0)
              }}
            >
              {item.name}
            </button>
          ))}
        </div>
      </div>

      <div className="between" style={{ marginBottom: 12 }}>
        <span className="subtitle">
          {loading ? 'Memuat barang…' : `Menampilkan ${items.length} dari ${total} barang aktif`}
        </span>
        {appliedQ ? <span className="badge badge-muted">Pencarian: “{appliedQ}”</span> : null}
      </div>

      {error ? <ErrorBox error={error} onRetry={reload} /> : null}

      {loading ? (
        <div className="listing-grid">
          {Array.from({ length: 8 }).map((_, index) => (
            <ListingCardSkeleton key={index} />
          ))}
        </div>
      ) : items.length ? (
        <>
          <div className="listing-grid">
            {items.map((listing) => (
              <ListingCard key={listing.id} listing={listing} />
            ))}
          </div>

          {total > LIMIT ? (
            <div className="between" style={{ marginTop: 20 }}>
              <span className="small muted">
                Halaman {Math.floor(offset / LIMIT) + 1} dari {Math.max(1, Math.ceil(total / LIMIT))}
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
          ) : null}
        </>
      ) : (
        <Empty
          icon="search_off"
          title="Barang tidak ditemukan"
          action={
            hasFilter ? (
              <button type="button" className="btn btn-outline btn-sm" onClick={reset}>
                Hapus filter
              </button>
            ) : null
          }
        >
          Coba kata kunci lain, ubah kategori, atau longgarkan filter harga.
        </Empty>
      )}
    </div>
  )
}
