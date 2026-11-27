import { useEffect } from 'react'
import { Link, RouterProvider, matchPath, useLocation, useRouter } from './router'
import { useAuth } from './auth/useAuth'
import { Layout } from './components/Layout'
import { Icon, PageLoader } from './components/ui'

import Home from './pages/Home'
import Catalog from './pages/Catalog'
import ListingDetail from './pages/ListingDetail'
import Login from './pages/Login'
import Register from './pages/Register'
import ForgotPassword from './pages/ForgotPassword'
import Dashboard from './pages/Dashboard'
import MyListings from './pages/MyListings'
import ListingForm from './pages/ListingForm'
import Rentals from './pages/Rentals'
import RentalDetail from './pages/RentalDetail'
import Messages from './pages/Messages'
import Profile from './pages/Profile'
import Report from './pages/Report'
import AdminDashboard from './pages/Admin'
import AdminKTM from './pages/AdminKTM'
import AdminListings from './pages/AdminListings'
import AdminRentals from './pages/AdminRentals'
import AdminUsers from './pages/AdminUsers'
import AdminReports from './pages/AdminReports'
import AdminAudit from './pages/AdminAudit'

function RequireUser({ admin = false, children }) {
  const { status, user } = useAuth()
  const { navigate } = useRouter()
  const { path } = useLocation()

  useEffect(() => {
    if (status === 'anonymous') {
      navigate(`/masuk?next=${encodeURIComponent(path)}`, { replace: true })
    }
  }, [status, path, navigate])

  if (status !== 'authenticated') {
    return <PageLoader />
  }

  if (admin && user?.role !== 'admin_dev') {
    return (
      <div className="container page">
        <div className="empty">
          <Icon name="lock" size={38} />
          <strong>Akses khusus administrator</strong>
          <p>Halaman ini hanya dapat dibuka oleh akun admin Fakultas Teknik.</p>
          <Link to="/dashboard" className="btn btn-outline btn-sm">
            Kembali ke dashboard
          </Link>
        </div>
      </div>
    )
  }

  return children
}

function NotFound() {
  return (
    <div className="container page">
      <div className="empty" style={{ paddingBlock: 80 }}>
        <Icon name="explore_off" size={44} />
        <strong style={{ fontSize: 18 }}>Halaman tidak ditemukan</strong>
        <p>Tautan mungkin salah atau halaman sudah dipindahkan.</p>
        <div className="row-wrap" style={{ justifyContent: 'center' }}>
          <Link to="/" className="btn btn-primary btn-sm">
            <Icon name="home" /> Beranda
          </Link>
          <Link to="/jelajahi" className="btn btn-outline btn-sm">
            <Icon name="storefront" /> Jelajahi Barang
          </Link>
        </div>
      </div>
    </div>
  )
}

function renderPrivate(pathname) {
  if (pathname === '/dashboard') return <Dashboard />
  if (pathname === '/barang-saya') return <MyListings />
  if (pathname === '/barang-saya/baru') return <ListingForm />

  const editMatch = matchPath('/barang-saya/:id/edit', pathname)
  if (editMatch) return <ListingForm id={editMatch.id} />

  if (pathname === '/penyewaan') return <Rentals />

  const rentalMatch = matchPath('/penyewaan/:id', pathname)
  if (rentalMatch) return <RentalDetail id={rentalMatch.id} />

  if (pathname === '/pesan') return <Messages />
  if (pathname === '/profil') return <Profile />
  if (pathname === '/lapor') return <Report />

  return null
}

function renderAdmin(pathname) {
  if (pathname === '/admin') return <AdminDashboard />
  if (pathname === '/admin/ktm') return <AdminKTM />
  if (pathname === '/admin/listings') return <AdminListings />
  if (pathname === '/admin/rentals') return <AdminRentals />
  if (pathname === '/admin/users') return <AdminUsers />
  if (pathname === '/admin/reports') return <AdminReports />
  if (pathname === '/admin/audit') return <AdminAudit />
  return null
}

function Routes() {
  const { pathname } = useLocation()

  if (pathname === '/') return <Home />
  if (pathname === '/jelajahi') return <Catalog />
  if (pathname === '/masuk') return <Login />
  if (pathname === '/daftar') return <Register />
  if (pathname === '/lupa-sandi') return <ForgotPassword />

  const listingMatch = matchPath('/barang/:id', pathname)
  if (listingMatch) return <ListingDetail id={listingMatch.id} />

  if (pathname === '/admin' || pathname.startsWith('/admin/')) {
    const adminPage = renderAdmin(pathname)
    if (adminPage) return <RequireUser admin>{adminPage}</RequireUser>
    return <NotFound />
  }

  const privatePage = renderPrivate(pathname)
  if (privatePage) return <RequireUser>{privatePage}</RequireUser>

  return <NotFound />
}

export default function App() {
  return (
    <RouterProvider>
      <Layout>
        <Routes />
      </Layout>
    </RouterProvider>
  )
}
