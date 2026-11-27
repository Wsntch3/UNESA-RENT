import { createContext, useCallback, useContext, useEffect, useMemo, useState } from 'react'

const RouterContext = createContext(null)

function readPath() {
  const raw = window.location.hash.replace(/^#/, '')
  if (!raw) return '/'
  return raw.startsWith('/') ? raw : `/${raw}`
}

export function RouterProvider({ children }) {
  const [path, setPath] = useState(readPath)

  useEffect(() => {
    const onChange = () => setPath(readPath())
    window.addEventListener('hashchange', onChange)
    return () => window.removeEventListener('hashchange', onChange)
  }, [])

  const navigate = useCallback((to, { replace = false } = {}) => {
    const target = to.startsWith('/') ? to : `/${to}`
    if (replace) {
      const url = `${window.location.pathname}${window.location.search}#${target}`
      window.history.replaceState(null, '', url)
      setPath(target)
    } else if (readPath() === target) {
      setPath(target)
    } else {
      window.location.hash = target
    }
    window.scrollTo({ top: 0, behavior: 'auto' })
  }, [])

  const value = useMemo(() => ({ path, navigate }), [path, navigate])
  return <RouterContext.Provider value={value}>{children}</RouterContext.Provider>
}

export function useRouter() {
  const ctx = useContext(RouterContext)
  if (!ctx) throw new Error('useRouter harus dipakai di dalam <RouterProvider>')
  return ctx
}

export function useLocation() {
  const { path } = useRouter()
  const [pathname, searchString = ''] = path.split('?')
  const query = useMemo(
    () => Object.fromEntries(new URLSearchParams(searchString)),
    [searchString],
  )
  return { pathname: pathname || '/', query, path }
}

export function Link({ to, children, className, onClick, ...rest }) {
  const { navigate } = useRouter()
  return (
    <a
      href={`#${to}`}
      className={className}
      onClick={(event) => {
        if (event.metaKey || event.ctrlKey || event.shiftKey || event.button !== 0) return
        event.preventDefault()
        onClick?.(event)
        navigate(to)
      }}
      {...rest}
    >
      {children}
    </a>
  )
}


export function matchPath(pattern, pathname) {
  const p = pattern.split('/').filter(Boolean)
  const s = pathname.split('/').filter(Boolean)
  if (p.length !== s.length) return null
  const params = {}
  for (let i = 0; i < p.length; i += 1) {
    if (p[i].startsWith(':')) {
      params[p[i].slice(1)] = decodeURIComponent(s[i])
    } else if (p[i] !== s[i]) {
      return null
    }
  }
  return params
}
