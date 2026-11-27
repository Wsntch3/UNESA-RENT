import { useCallback, useEffect, useRef, useState } from 'react'

/**
 * Small data-fetching helper: runs `loader` whenever `deps` change.
 * `reload()` re-runs it manually.
 */
export function useApi(loader, deps = []) {
  const [data, setData] = useState(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(null)
  const [nonce, setNonce] = useState(0)

  const loaderRef = useRef(loader)
  loaderRef.current = loader

  useEffect(() => {
    let alive = true
    setLoading(true)
    setError(null)

    Promise.resolve()
      .then(() => loaderRef.current())
      .then((result) => {
        if (!alive) return
        setData(result)
        setLoading(false)
      })
      .catch((err) => {
        if (!alive) return
        setError(err)
        setLoading(false)
      })

    return () => {
      alive = false
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [...deps, nonce])

  const reload = useCallback(() => setNonce((n) => n + 1), [])

  return { data, loading, error, reload, setData }
}
