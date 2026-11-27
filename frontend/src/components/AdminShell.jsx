import { Link, useLocation } from '../router'
import { useAuth } from '../auth/useAuth'
import { Avatar, Icon } from './ui'

const ADMIN_LINKS = [
  { to: '/admin', label: 'Dashboard', icon: 'dashboard', exact: true },
  { to: '/admin/ktm', label: 'Verifikasi KTM', icon: 'verified_user' },
  { to: '/admin/listings', label: 'Listing Barang', icon: 'inventory_2' },
  { to: '/admin/rentals', label: 'Penyewaan', icon: 'receipt_long' },
  { to: '/admin/users', label: 'Pengguna', icon: 'groups' },
  { to: '/admin/reports', label: 'Laporan', icon: 'flag' },
  { to: '/admin/audit', label: 'Log Audit', icon: 'history' },
]

export function AdminShell({ eyebrow, title, subtitle, actions, children }) {
  const { pathname } = useLocation()
  const { user } = useAuth()

  return (
    <div className="container page">
      <div className="admin-shell">
        <aside className="admin-side">
          <div className="row" style={{ padding: '6px 8px 12px', gap: 10 }}>
            <Avatar name={user?.name} size="sm" />
            <div style={{ minWidth: 0 }}>
              <div className="tiny strong truncate">Admin FT UNESA</div>
              <div className="tiny muted">Moderator / Verifikator</div>
            </div>
          </div>
          <hr className="divider" style={{ margin: '0 4px 8px' }} />
          {ADMIN_LINKS.map((link) => {
            const active = link.exact ? pathname === link.to : pathname.startsWith(link.to)
            return (
              <Link key={link.to} to={link.to} className={`nav-link ${active ? 'active' : ''}`.trim()}>
                <Icon name={link.icon} /> {link.label}
              </Link>
            )
          })}
          <hr className="divider" style={{ margin: '8px 4px' }} />
          <Link to="/" className="nav-link">
            <Icon name="arrow_back" /> Kembali ke Marketplace
          </Link>
        </aside>

        <div>
          <div className="page-head">
            <div>
              {eyebrow ? <div className="eyebrow">{eyebrow}</div> : null}
              <h1 className="page-title">{title}</h1>
              {subtitle ? <p className="subtitle mt-2" style={{ maxWidth: 680 }}>{subtitle}</p> : null}
            </div>
            {actions ? <div className="row-wrap">{actions}</div> : null}
          </div>
          {children}
        </div>
      </div>
    </div>
  )
}
