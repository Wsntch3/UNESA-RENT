import { createContext, useCallback, useEffect, useMemo, useState } from 'react'
import { authApi } from '../lib/authApi'
import { ktmApi } from '../lib/api'
import { clearToken, getToken, saveToken } from './tokenStorage'

export const AuthContext = createContext(null)

export function AuthProvider({ children }) {
  const [user, setUser] = useState(null)
  const [token, setToken] = useState(() => getToken())
  const [status, setStatus] = useState(() => (getToken() ? 'loading' : 'anonymous'))
  const [error, setError] = useState(null)
  const [pending, setPending] = useState(false)

  const [ktm, setKtm] = useState(null)
  const [ktmLoading, setKtmLoading] = useState(false)
  const [ktmNonce, setKtmNonce] = useState(0)

  useEffect(() => {
    if (!getToken()) return undefined

    const controller = new AbortController()

    ;(async () => {
      try {
        const data = await authApi.me(controller.signal)
        setUser(data.user)
        setStatus('authenticated')
      } catch (err) {
        if (controller.signal.aborted) return

        if (err.status === 401) {
          clearToken()
          setToken(null)
        } else {
          setError(err)
        }
        setUser(null)
        setStatus('anonymous')
      }
    })()

    return () => controller.abort()
  }, [])

  
  useEffect(() => {
    if (status !== 'authenticated' || user?.role !== 'mahasiswa') {
      setKtm(null)
      setKtmLoading(false)
      return undefined
    }

    let alive = true
    setKtmLoading(true)
    ktmApi
      .mine()
      .then((data) => {
        if (alive) setKtm(data)
      })
      .catch(() => {
        if (alive) setKtm(null)
      })
      .finally(() => {
        if (alive) setKtmLoading(false)
      })

    return () => {
      alive = false
    }
  }, [status, user?.id, user?.role, ktmNonce])

  const refreshKtm = useCallback(() => setKtmNonce((n) => n + 1), [])

  const applySession = useCallback((data) => {
    saveToken(data.token)
    setToken(data.token)
    setUser(data.user)
    setStatus('authenticated')
    setError(null)
  }, [])

  const login = useCallback(
    async (credentials) => {
      setPending(true)
      setError(null)
      try {
        const data = await authApi.login(credentials)
        
        if (data.otp_required) return data
        applySession(data)
        return data.user
      } catch (err) {
        setError(err)
        throw err
      } finally {
        setPending(false)
      }
    },
    [applySession],
  )

  const loginOtp = useCallback(
    async (payload) => {
      setPending(true)
      setError(null)
      try {
        const data = await authApi.loginOtp(payload)
        applySession(data)
        return data.user
      } catch (err) {
        setError(err)
        throw err
      } finally {
        setPending(false)
      }
    },
    [applySession],
  )

  const register = useCallback(
    async (payload) => {
      setPending(true)
      setError(null)
      try {
        const data = await authApi.register(payload)
        applySession(data)
        return data.user
      } catch (err) {
        setError(err)
        throw err
      } finally {
        setPending(false)
      }
    },
    [applySession],
  )

  const requestOtp = useCallback(async (payload) => {
    setError(null)
    return authApi.requestOtp(payload)
  }, [])

  const verifyOtp = useCallback(async (payload) => {
    setError(null)
    return authApi.verifyOtp(payload)
  }, [])

  const forgotPassword = useCallback(async (payload) => {
    setError(null)
    return authApi.forgotPassword(payload)
  }, [])

  const resetPassword = useCallback(async (payload) => {
    setError(null)
    return authApi.resetPassword(payload)
  }, [])

  const logout = useCallback(() => {
    clearToken()
    setToken(null)
    setUser(null)
    setError(null)
    setKtm(null)
    setStatus('anonymous')
  }, [])

  const value = useMemo(
    () => ({
      user,
      token,
      status,
      error,
      pending,
      isAuthenticated: status === 'authenticated',
      login,
      loginOtp,
      register,
      requestOtp,
      verifyOtp,
      forgotPassword,
      resetPassword,
      logout,
      clearError: () => setError(null),
      ktm,
      ktmLoading,
      ktmVerified: Boolean(ktm?.verified),
      ktmSubmission: ktm?.submissions?.[0] || null,
      refreshKtm,
    }),
    [
      user,
      token,
      status,
      error,
      pending,
      login,
      loginOtp,
      register,
      requestOtp,
      verifyOtp,
      forgotPassword,
      resetPassword,
      logout,
      ktm,
      ktmLoading,
      refreshKtm,
    ],
  )

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}
