import { useState } from 'react'
import { Link, useRouter } from '../router'
import { useAuth } from '../auth/useAuth'
import { catalogApi } from '../lib/api'
import { useApi } from '../lib/useApi'
import { ListingCard, ListingCardSkeleton } from '../components/ListingCard'
import { Empty, Icon, SectionHeading } from '../components/ui'

const FALLBACK_CATEGORIES = [
  { id: 'elektronik', slug: 'elektronik', name: 'Elektronik' },
  { id: 'fotografi', slug: 'fotografi', name: 'Fotografi' },
  { id: 'perkakas', slug: 'perkakas', name: 'Perkakas' },
  { id: 'olahraga', slug: 'olahraga', name: 'Olahraga' },
  { id: 'alat-tulis', slug: 'alat-tulis', name: 'Alat Tulis' },
  { id: 'lainnya', slug: 'lainnya', name: 'Lainnya' },
]

const CATEGORY_ICONS = {
  elektronik: 'devices',
  fotografi: 'photo_camera',
  perkakas: 'handyman',
  olahraga: 'sports_soccer',
  'alat-tulis': 'edit',
  lainnya: 'category',
}

export default function Home() {
  const { navigate } = useRouter()
  const { isAuthenticated, ktmVerified } = useAuth()
  const [search, setSearch] = useState('')

  const { data: categoriesData } = useApi(() => catalogApi.categories(), [])
  const { data: featured, loading } = useApi(
    () => catalogApi.search({ limit: 8, sort: 'newest' }),
    [],
  )

  const categories = categoriesData?.categories?.length
    ? categoriesData.categories
    : FALLBACK_CATEGORIES
  const items = featured?.items || []

  const submit = (event) => {
    event.preventDefault()
    const q = search.trim()
    navigate(q ? `/jelajahi?q=${encodeURIComponent(q)}` : '/jelajahi')
  }

  return (
    <>
      <section className="hero">
        <div className="container hero-inner">
          <span className="eyebrow">
            <Icon name="school" size={15} /> Komunitas Mandiri Mahasiswa FT UNESA
          </span>
          <h1 className="hero-title">
            Butuh barang? <span className="highlight">Sewa aja</span> di sini.
          </h1>
          <p className="hero-sub">
            Temukan kebutuhan praktikum, tugas, dan kegiatan kampus dari sesama mahasiswa Fakultas
            Teknik UNESA. COD aman di area kampus, penyedia terverifikasi KTM.
          </p>

          <form className="hero-search" onSubmit={submit} role="search">
            <Icon name="search" />
            <input
              value={search}
              onChange={(event) => setSearch(event.target.value)}
              placeholder="Cari kamera, multimeter, tenda, proyektor…"
              aria-label="Cari barang"
            />
            <button type="submit" className="btn btn-primary">
              <Icon name="search" />
              <span>Cari Barang</span>
            </button>
          </form>

          <div className="hero-meta">
            <span className="badge badge-primary">
              <Icon name="verified" filled size={13} /> KTM Terverifikasi
            </span>
            <span className="badge badge-yellow">
              <Icon name="place" size={13} /> COD Kampus Ketintang
            </span>
            <span className="badge badge-mint">
              <Icon name="payments" size={13} /> Tanpa biaya admin
            </span>
          </div>
        </div>
      </section>

      <div className="container" style={{ paddingBottom: 48 }}>
        <section aria-labelledby="kategori-title" style={{ marginBottom: 34 }}>
          <SectionHeading
            id="kategori-title"
            title="Jelajahi kategori"
            subtitle="Pilih sesuai kebutuhan lab, tugas, dan kegiatanmu."
            action={
              <Link to="/jelajahi" className="btn btn-ghost btn-sm">
                Lihat semua <Icon name="arrow_forward" />
              </Link>
            }
          />
          <div className="chip-row">
            {categories.map((category) => (
              <Link
                key={category.id || category.slug}
                to={`/jelajahi?category=${encodeURIComponent(category.slug)}`}
                className="chip"
              >
                <Icon name={CATEGORY_ICONS[category.slug] || 'category'} />
                {category.name}
              </Link>
            ))}
          </div>
        </section>

        <section className="hero-panel" style={{ marginBottom: 40 }}>
          <div className="feature-tile blue">
            <div className="big-icon">
              <Icon name="verified_user" filled />
            </div>
            <h3>Penyedia terverifikasi KTM</h3>
            <p>
              Setiap penyedia melewati verifikasi identitas mahasiswa FT. Transaksi lebih tenang,
              barang lebih terpercaya.
            </p>
          </div>
          <div className="feature-tile yellow">
            <div className="big-icon">
              <Icon name="handshake" />
            </div>
            <h3>COD di kampus</h3>
            <p>Sepakati titik temu di Gedung FT, Lab Terpadu, atau area parkir. Cek fisik sebelum bayar.</p>
          </div>
          <div className="feature-tile mint">
            <div className="big-icon">
              <Icon name="savings" />
            </div>
            <h3>Tanpa biaya admin</h3>
            <p>Harga sewa langsung dari pemilik. Tidak ada potongan platform untuk sesama mahasiswa.</p>
          </div>
          <div className="feature-tile lavender">
            <div className="big-icon">
              <Icon name="chat" />
            </div>
            <h3>Chat & perjanjian</h3>
            <p>Ajukan tanggal, diskusikan detail lewat chat, lalu unduh surat perjanjian sewa digital.</p>
          </div>
        </section>

        <section aria-labelledby="tersedia-title">
          <SectionHeading
            id="tersedia-title"
            title="Barang tersedia"
            subtitle={
              featured?.total
                ? `${featured.total} barang aktif dari mahasiswa FT UNESA`
                : 'Temukan barang yang sesuai dengan kebutuhanmu.'
            }
            action={
              <Link to="/jelajahi" className="btn btn-outline btn-sm">
                Jelajahi katalog <Icon name="arrow_outward" />
              </Link>
            }
          />
          {loading ? (
            <div className="listing-grid">
              {Array.from({ length: 4 }).map((_, index) => (
                <ListingCardSkeleton key={index} />
              ))}
            </div>
          ) : items.length ? (
            <div className="listing-grid">
              {items.map((listing) => (
                <ListingCard key={listing.id} listing={listing} />
              ))}
            </div>
          ) : (
            <Empty icon="inventory_2" title="Belum ada barang yang disewakan">
              Jadilah yang pertama menyewakan barangmu ke sesama mahasiswa FT.
            </Empty>
          )}
        </section>

        <section className="grid cols-3 mt-6" style={{ marginTop: 36 }}>
          <div className="card card-pad">
            <span className="stat-icon">
              <Icon name="search" />
            </span>
            <h3 className="card-title mt-3">1. Cari & pilih barang</h3>
            <p className="subtitle mt-1">
              Filter kategori, lokasi COD, dan rentang harga. Cek kondisi serta kelengkapan barang.
            </p>
          </div>
          <div className="card card-pad">
            <span className="stat-icon mint">
              <Icon name="event_available" />
            </span>
            <h3 className="card-title mt-3">2. Ajukan tanggal sewa</h3>
            <p className="subtitle mt-1">
              Tentukan periode sewa. Penyedia akan menyetujui atau menolak pengajuanmu.
            </p>
          </div>
          <div className="card card-pad">
            <span className="stat-icon yellow">
              <Icon name="handshake" />
            </span>
            <h3 className="card-title mt-3">3. COD & selesai</h3>
            <p className="subtitle mt-1">
              Bertemu di kampus, cek barang bersama, lalu tandai selesai setelah dikembalikan.
            </p>
          </div>
        </section>

        <section
          className="card card-pad-lg mt-6"
          style={{
            marginTop: 40,
            background: 'var(--primary)',
            borderColor: 'transparent',
            color: '#fff',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            gap: 20,
            flexWrap: 'wrap',
          }}
        >
          <div style={{ maxWidth: 620 }}>
            <h2 style={{ fontSize: 26, letterSpacing: '-0.9px' }}>
              Punya barang yang menganggur? <span className="highlight" style={{ color: 'var(--ink)' }}>Sewakan</span> di sini.
            </h2>
            <p style={{ marginTop: 8, fontSize: 13.5, opacity: 0.85 }}>
              Bantu teman satu fakultas, dapat penghasilan tambahan, dan rawat ekosistem pinjam-meminjam
              alat praktikum FT UNESA.
            </p>
          </div>
          <Link
            to={isAuthenticated ? '/barang-saya/baru' : '/daftar'}
            className="btn btn-white btn-lg"
          >
            <Icon name="add_circle" />
            {isAuthenticated ? 'Tambah Barang' : 'Daftar & Sewakan'}
          </Link>
        </section>

        {isAuthenticated && !ktmVerified ? (
          <div className="notice-strip" style={{ marginTop: 16 }}>
            <Icon name="info" />
            <span>
              Verifikasi KTM dulu agar bisa menyewa dan menyewakan barang.{' '}
              <Link to="/profil" style={{ color: 'var(--ink)', textDecoration: 'underline' }}>
                Verifikasi sekarang
              </Link>
            </span>
          </div>
        ) : null}
      </div>
    </>
  )
}
