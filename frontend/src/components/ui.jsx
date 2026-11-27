import { useEffect } from 'react'
import { initials } from '../lib/format'

export function Icon({ name, filled = false, size, className = '', style }) {
  return (
    <span
      className={`material-symbols-outlined ${filled ? 'icon-filled' : ''} ${className}`.trim()}
      aria-hidden="true"
      style={{ ...(size ? { fontSize: `${size}px` } : null), ...style }}
    >
      {name}
    </span>
  )
}

const AVATAR_TONES = ['', 'mint', 'yellow', 'lavender']

export function Avatar({ name, size = '', tone, src, className = '' }) {
  const autoTone =
    tone ?? AVATAR_TONES[(String(name || '').charCodeAt(0) || 0) % AVATAR_TONES.length]
  return (
    <span className={`avatar ${size} ${autoTone} ${className}`.trim()} title={name || undefined}>
      {src ? <img src={src} alt={name || 'Avatar'} /> : initials(name)}
    </span>
  )
}

export function Spinner({ large = false }) {
  return <span className={`spinner ${large ? 'lg' : ''}`.trim()} aria-label="Memuat" />
}

export function Loading({ label = 'Memuat data…' }) {
  return (
    <div className="loading-block" role="status">
      <Spinner />
      <span>{label}</span>
    </div>
  )
}

export function Empty({ icon = 'inbox', title = 'Belum ada data', children, action }) {
  return (
    <div className="empty">
      <Icon name={icon} size={38} />
      <strong>{title}</strong>
      {children ? <p>{children}</p> : null}
      {action}
    </div>
  )
}

export function ErrorBox({ error, onRetry, title = 'Terjadi kesalahan' }) {
  if (!error) return null
  return (
    <div className="alert alert-error" role="alert">
      <Icon name="error" />
      <div className="alert-body">
        <strong>{title}</strong>
        {error.message}
        {error.fields ? (
          <ul className="mt-2" style={{ listStyle: 'disc', paddingLeft: 18 }}>
            {Object.entries(error.fields).map(([field, message]) => (
              <li key={field}>
                {field}: {message}
              </li>
            ))}
          </ul>
        ) : null}
      </div>
      {onRetry ? (
        <button type="button" className="btn btn-sm btn-ghost" onClick={onRetry}>
          Coba lagi
        </button>
      ) : null}
    </div>
  )
}

export function Alert({ tone = 'info', icon = 'info', title, children }) {
  return (
    <div className={`alert alert-${tone}`} role={tone === 'error' ? 'alert' : undefined}>
      <Icon name={icon} />
      <div className="alert-body">
        {title ? <strong>{title}</strong> : null}
        {children}
      </div>
    </div>
  )
}

export function Field({ label, htmlFor, required, hint, error, children, className = '' }) {
  return (
    <div className={`field ${className}`.trim()}>
      {label ? (
        <label className="label" htmlFor={htmlFor}>
          {label}
          {required ? <span className="req">*</span> : null}
        </label>
      ) : null}
      {children}
      {error ? <span className="field-error">{error}</span> : null}
      {hint && !error ? <span className="hint">{hint}</span> : null}
    </div>
  )
}

export function PageHeader({ eyebrow, title, subtitle, actions, children }) {
  return (
    <div className="page-head">
      <div>
        {eyebrow ? <div className="eyebrow">{eyebrow}</div> : null}
        <h1 className="page-title">{title}</h1>
        {subtitle ? <p className="subtitle mt-2" style={{ maxWidth: 680 }}>{subtitle}</p> : null}
        {children}
      </div>
      {actions ? <div className="row-wrap">{actions}</div> : null}
    </div>
  )
}

export function StatCard({ icon, tone = '', value, label, note }) {
  return (
    <div className="stat-card">
      <span className={`stat-icon ${tone}`.trim()}>
        <Icon name={icon} />
      </span>
      <div>
        <div className="stat-value">{value}</div>
        <div className="stat-label">{label}</div>
        {note ? <div className="stat-note">{note}</div> : null}
      </div>
    </div>
  )
}

export function Modal({ title, subtitle, onClose, children, footer, size = '' }) {
  useEffect(() => {
    const onKey = (event) => {
      if (event.key === 'Escape') onClose?.()
    }
    document.addEventListener('keydown', onKey)
    const previous = document.body.style.overflow
    document.body.style.overflow = 'hidden'
    return () => {
      document.removeEventListener('keydown', onKey)
      document.body.style.overflow = previous
    }
  }, [onClose])

  return (
    <div className="modal-backdrop" onMouseDown={(event) => event.target === event.currentTarget && onClose?.()}>
      <div className={`modal ${size}`.trim()} role="dialog" aria-modal="true" aria-label={title}>
        <div className="modal-head">
          <div>
            <h2>{title}</h2>
            {subtitle ? <p className="hint mt-1">{subtitle}</p> : null}
          </div>
          <button type="button" className="btn-icon" onClick={onClose} aria-label="Tutup">
            <Icon name="close" size={18} />
          </button>
        </div>
        {children}
        {footer ? <div className="modal-foot">{footer}</div> : null}
      </div>
    </div>
  )
}

export function SectionHeading({ title, subtitle, action, id }) {
  return (
    <div className="between" style={{ alignItems: 'flex-end', marginBottom: 14 }}>
      <div>
        <h2 className="section-title" id={id}>{title}</h2>
        {subtitle ? <p className="subtitle mt-1">{subtitle}</p> : null}
      </div>
      {action}
    </div>
  )
}

export function PageLoader() {
  return (
    <div className="container page">
      <div className="loading-block">
        <Spinner large />
        <span>Memeriksa sesi…</span>
      </div>
    </div>
  )
}
