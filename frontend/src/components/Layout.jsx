import { useCallback, useEffect, useRef, useState } from 'react'
import { Link, useLocation, useRouter } from '../router'
import { useAuth } from '../auth/useAuth'
import { notificationApi } from '../lib/api'
import { timeAgo } from '../lib/format'
import { Avatar, Icon } from './ui'

const PUBLIC_LINKS = [{ to: '/jelajahi', label: 'Jelajahi Barang', icon: 'storefront' }]

const PRIVATE_LINKS = [
  { to: '/barang-saya/baru', label: 'Sewakan Barang', icon: 'add_circle' },
  { to: '/pesan', label: 'Pesan', icon: 'chat_bubble' },
  { to: '/penyewaan', label: 'Penyewaan Saya', icon: 'receipt_long' },
]

function isActive(pathname, to) {
  if (to === '/') return pathname === '/'
  return pathname === to || pathname.startsWith(`${to}/`)
}

export function Layout({ children }) {
  const { pathname } = useLocation()
  const { navigate } = useRouter()
  const { status, user, logout, ktmVerified } = useAuth()

  const authed = status === 'authenticated'
  const isAdmin = user?.role === 'admin_dev'

  const [menuOpen, setMenuOpen] = useState(false)
  const [bellOpen, setBellOpen] = useState(false)
  const [userOpen, setUserOpen] = useState(false)
  const [search, setSearch] = useState('')
  const [notifications, setNotifications] = useState({ items: [], unread_count: 0 })

  const navRef = useRef(null)

  useEffect(() => {
    setMenuOpen(false)
    setBellOpen(false)
    setUserOpen(false)
  }, [pathname])

  useEffect(() => {
    const onDocClick = (event) => {
      if (navRef.current && !navRef.current.contains(event.target)) {
        setBellOpen(false)
        setUserOpen(false)
        setMenuOpen(false)
      }
    }
    document.addEventListener('mousedown', onDocClick)
    return () => document.removeEventListener('mousedown', onDocClick)
  }, [])

  const loadNotifications = useCallback(() => {
    if (!authed) return
    notificationApi
      .list({ unread: 1, limit: 8 })
      .then((data) => setNotifications(data))
      .catch(() => {})
  }, [authed])

  useEffect(() => {
    if (!authed) {
      setNotifications({ items: [], unread_count: 0 })
      return undefined
    }
    loadNotifications()
    const timer = setInterval(loadNotifications, 60000)
    return () => clearInterval(timer)
  }, [authed, pathname, loadNotifications])

  const submitSearch = (event) => {
    event.preventDefault()
    const q = search.trim()
    navigate(q ? `/jelajahi?q=${encodeURIComponent(q)}` : '/jelajahi')
    setMenuOpen(false)
  }

  const openNotification = async (item) => {
    try {
      if (!item.read_at) await notificationApi.read(item.id)
    } catch {
      // ignore — navigation is more important
    }
    setBellOpen(false)
    loadNotifications()
    const rentalId = item.data?.rental_id
    if (rentalId) navigate(`/penyewaan/${rentalId}`)
  }

  const links = authed ? [...PUBLIC_LINKS, ...PRIVATE_LINKS] : PUBLIC_LINKS
  const unread = notifications.unread_count || 0

  return (
    <div className="app-shell">
      <header className="site-nav" ref={navRef}>
        <div className="container nav-inner">
          <Link to="/" className="brand">
            <span className="brand-mark">
              <Icon name="handyman" filled />
            </span>
            <span>
              UNESA<span className="brand-accent">Rent</span>
            </span>
            <span className="brand-pill">FT</span>
          </Link>

          <nav className={`nav-links ${menuOpen ? 'open' : ''}`} aria-label="Navigasi utama">
            {links.map((link) => (
              <Link
                key={link.to}
                to={link.to}
                className={`nav-link ${isActive(pathname, link.to) ? 'active' : ''}`}
              >
                <Icon name={link.icon} />
                {link.label}
              </Link>
            ))}
            {isAdmin ? (
              <Link to="/admin" className={`nav-link ${isActive(pathname, '/admin') ? 'active' : ''}`}>
                <Icon name="admin_panel_settings" />
                Admin
              </Link>
            ) : null}
          </nav>

          <div className="nav-actions">
            <form className="nav-search" onSubmit={submitSearch} role="search">
              <Icon name="search" />
              <input
                value={search}
                onChange={(event) => setSearch(event.target.value)}
                placeholder="Cari alat lab & teknik…"
                aria-label="Cari barang"
              />
            </form>

            {authed ? (
              <>
                <div className="dropdown">
                  <button
                    type="button"
                    className="btn-icon nav-bell"
                    aria-label={`Notifikasi${unread ? `, ${unread} belum dibaca` : ''}`}
                    onClick={() => {
                      setBellOpen((open) => !open)
                      setUserOpen(false)
                      if (!bellOpen) loadNotifications()
                    }}
                  >
                    <Icon name="notifications" />
                    {unread > 0 ? <span className="dot">{unread > 9 ? '9+' : unread}</span> : null}
                  </button>
                  {bellOpen ? (
                    <div className="dropdown-panel" role="menu">
                      <div className="dropdown-head">
                        <span>Notifikasi</span>
                        <span className="badge badge-primary">{unread} baru</span>
                      </div>
                      {notifications.items?.length ? (
                        notifications.items.map((item) => (
                          <button
                            key={item.id}
                            type="button"
                            className={`dropdown-item ${item.read_at ? '' : 'unread'}`.trim()}
                            onClick={() => openNotification(item)}
                          >
                            <Icon name="notifications_active" />
                            <span>
                              <strong>{item.title}</strong>
                              {item.body ? <span className="small muted" style={{ display: 'block' }}>{item.body}</span> : null}
                              <span className="tiny muted">{timeAgo(item.created_at)}</span>
                            </span>
                          </button>
                        ))
                      ) : (
                        <div className="dropdown-empty">Belum ada notifikasi baru.</div>
                      )}
                    </div>
                  ) : null}
                </div>

                <div className="dropdown">
                  <button
                    type="button"
                    className="nav-avatar"
                    onClick={() => {
                      setUserOpen((open) => !open)
                      setBellOpen(false)
                    }}
                    aria-label="Menu akun"
                  >
                    <Avatar name={user?.name} size="sm" />
                    <span className="nav-name">{user?.name?.split(' ')[0]}</span>
                    <Icon name="expand_more" size={16} />
                  </button>
                  {userOpen ? (
                    <div className="dropdown-panel menu" role="menu">
                      <div style={{ padding: '8px 10px 10px', borderBottom: '1px solid var(--border-soft)' }}>
                        <div className="strong">{user?.name}</div>
                        <div className="tiny muted truncate">{user?.email}</div>
                        <div className="row-wrap mt-2">
                          {isAdmin ? (
                            <span className="badge badge-primary">Admin FT</span>
                          ) : ktmVerified ? (
                            <span className="badge badge-success">
                              <Icon name="verified" filled size={12} /> KTM Terverifikasi
                            </span>
                          ) : (
                            <span className="badge badge-warning">KTM Belum Verifikasi</span>
                          )}
                        </div>
                      </div>
                      <Link to="/dashboard" className="dropdown-item">
                        <Icon name="space_dashboard" /> Dashboard
                      </Link>
                      <Link to="/profil" className="dropdown-item">
                        <Icon name="person" /> Profil & Verifikasi
                      </Link>
                      <Link to="/barang-saya" className="dropdown-item">
                        <Icon name="inventory_2" /> Barang Saya
                      </Link>
                      <Link to="/penyewaan" className="dropdown-item">
                        <Icon name="receipt_long" /> Penyewaan Saya
                      </Link>
                      {isAdmin ? (
                        <Link to="/admin" className="dropdown-item">
                          <Icon name="admin_panel_settings" /> Portal Admin
                        </Link>
                      ) : null}
                      <hr className="divider" style={{ margin: '6px 4px' }} />
                      <button
                        type="button"
                        className="dropdown-item"
                        onClick={() => {
                          logout()
                          navigate('/')
                        }}
                      >
                        <Icon name="logout" /> Keluar
                      </button>
                    </div>
                  ) : null}
                </div>
              </>
            ) : (
              <div className="row" style={{ gap: 6 }}>
                <Link to="/masuk" className="btn btn-ghost btn-sm">
                  Masuk
                </Link>
                <Link to="/daftar" className="btn btn-primary btn-sm">
                  Daftar
                </Link>
              </div>
            )}

            <button
              type="button"
              className="btn-icon nav-toggle"
              aria-label="Buka menu"
              aria-expanded={menuOpen}
              onClick={() => setMenuOpen((open) => !open)}
            >
              <Icon name={menuOpen ? 'close' : 'menu'} />
            </button>
          </div>
        </div>
      </header>

      <main className="site-main">{children}</main>

      <footer className="site-footer">
        <div className="container footer-inner">
          <span>
            © 2026 UNESA Rent — Fakultas Teknik Universitas Negeri Surabaya. Inisiatif kolaboratif
            mahasiswa FT.
          </span>
          <span className="footer-links">
            <Link to="/jelajahi">Jelajahi</Link>
            <Link to="/profil">Verifikasi KTM</Link>
            <Link to="/penyewaan">Penyewaan</Link>
          </span>
        </div>
      </footer>
    </div>
  )
}
