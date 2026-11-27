import { useEffect, useState } from 'react'
import { Link, useLocation, useRouter } from '../router'
import { useAuth } from '../auth/useAuth'
import { Alert, Icon, PageLoader } from '../components/ui'

export default function Login({ next }) {
  const { navigate } = useRouter()
  const { query } = useLocation()
  const { login, loginOtp, status, pending, error, clearError } = useAuth()

  const [form, setForm] = useState({ email: '', password: '' })
  const [showPassword, setShowPassword] = useState(false)
  const [localError, setLocalError] = useState(null)
  const [step, setStep] = useState('credentials')
  const [challenge, setChallenge] = useState('')
  const [otp, setOtp] = useState('')

  const target = next || query.next || '/'

  useEffect(() => {
    if (status === 'authenticated') {
      navigate(target, { replace: true })
    }
  }, [status, navigate, target])

  if (status === 'loading') return <PageLoader />
  if (status === 'authenticated') return <PageLoader />

  const submit = async (event) => {
    event.preventDefault()
    setLocalError(null)
    clearError?.()
    if (!form.email.trim() || !form.password) {
      setLocalError({ message: 'Email dan kata sandi wajib diisi.' })
      return
    }
    try {
      const result = await login({ email: form.email.trim(), password: form.password })
      if (result?.otp_required) {
        setChallenge(result.challenge)
        setOtp('')
        setStep('otp')
        return
      }
      navigate(target, { replace: true })
    } catch {
      
    }
  }

  const submitOtp = async (event) => {
    event.preventDefault()
    setLocalError(null)
    clearError?.()
    if (!/^\d{6}$/.test(otp.trim())) {
      setLocalError({ message: 'Masukkan 6 digit kode verifikasi.' })
      return
    }
    try {
      await loginOtp({ challenge, code: otp.trim() })
      navigate(target, { replace: true })
    } catch {
      
    }
  }

  const activeError = localError || error

  return (
    <div className="container auth-wrap">
      <section className="auth-card">
        <div>
          <span className="badge badge-primary">
            <Icon name="lock" filled size={13} /> Portal Masuk Mahasiswa FT
          </span>
          <h1 className="auth-title mt-3">
            {step === 'otp' ? 'Verifikasi Admin' : 'Selamat Datang Kembali'}
          </h1>
          <p className="subtitle mt-2">
            {step === 'otp'
              ? 'Masukkan kode yang dikirim ke email admin.'
              : 'Masuk dan lanjutkan aktivitas sewamu di platform UNESA Rent.'}
          </p>
        </div>

        {activeError ? (
          <div className="mt-4">
            <Alert tone="error" icon="error" title="Gagal Masuk">
              {activeError.message}
              {activeError.fields ? (
                <ul className="mt-1" style={{ listStyle: 'disc', paddingLeft: 18 }}>
                  {Object.entries(activeError.fields).map(([field, message]) => (
                    <li key={field}>{message}</li>
                  ))}
                </ul>
              ) : null}
            </Alert>
          </div>
        ) : null}

        {step === 'credentials' ? (
        <form className="auth-form" onSubmit={submit} noValidate>
          <div className="field">
            <label className="label" htmlFor="email">Email Mahasiswa UNESA</label>
            <div className="input-wrap">
              <Icon name="mail" />
              <input
                id="email"
                className="input"
                type="email"
                autoComplete="email"
                placeholder="nama@mhs.unesa.ac.id"
                value={form.email}
                onChange={(event) => setForm({ ...form, email: event.target.value })}
                required
              />
            </div>
          </div>

          <div className="field">
            <div className="between">
              <label className="label" htmlFor="password">Kata Sandi</label>
              <Link to="/lupa-sandi" className="tiny">Lupa kata sandi?</Link>
            </div>
            <div className="input-wrap">
              <Icon name="key" />
              <input
                id="password"
                className={`input has-action ${showPassword ? '' : ''}`.trim()}
                type={showPassword ? 'text' : 'password'}
                autoComplete="current-password"
                placeholder="••••••••"
                value={form.password}
                onChange={(event) => setForm({ ...form, password: event.target.value })}
                required
              />
              <button
                type="button"
                className="input-action"
                aria-label={showPassword ? 'Sembunyikan kata sandi' : 'Tampilkan kata sandi'}
                onClick={() => setShowPassword((value) => !value)}
              >
                <Icon name={showPassword ? 'visibility_off' : 'visibility'} />
              </button>
            </div>
          </div>

          <label className="checkbox">
            <input type="checkbox" defaultChecked />
            <span>Ingat sesi saya di perangkat ini</span>
          </label>

          <button type="submit" className="btn btn-primary btn-block btn-lg" disabled={pending}>
            {pending ? <span className="spinner" /> : <Icon name="login" />}
            {pending ? 'Memproses…' : 'Masuk ke Akun'}
          </button>
        </form>
        ) : (
          <form className="auth-form" onSubmit={submitOtp} noValidate>
            <div className="field">
              <label className="label" htmlFor="admin-otp">Kode Verifikasi Admin</label>
              <div className="input-wrap">
                <Icon name="pin" />
                <input
                  id="admin-otp"
                  className="input"
                  inputMode="numeric"
                  maxLength={6}
                  placeholder="6 digit kode"
                  value={otp}
                  onChange={(event) => setOtp(event.target.value.replace(/\D/g, '').slice(0, 6))}
                  autoComplete="one-time-code"
                  required
                />
              </div>
              <span className="hint">Kode berlaku 10 menit. Jangan bagikan kode ke siapa pun.</span>
            </div>

            <button type="submit" className="btn btn-primary btn-block btn-lg" disabled={pending}>
              {pending ? <span className="spinner" /> : <Icon name="verified" />}
              {pending ? 'Memverifikasi…' : 'Masuk sebagai Admin'}
            </button>

            <button
              type="button"
              className="btn btn-ghost btn-sm"
              onClick={() => {
                setStep('credentials')
                setChallenge('')
                setOtp('')
                setLocalError(null)
                clearError?.()
              }}
            >
              <Icon name="arrow_back" /> Kembali ke halaman masuk
            </button>
          </form>
        )}

        <div className="alert alert-info mt-4">
          <Icon name="info" filled />
          <div className="alert-body">
            Satu akun untuk <strong>menyewa perlengkapan</strong> sekaligus{' '}
            <strong>menyewakan barangmu</strong>. Verifikasi KTM dilakukan setelah pendaftaran.
          </div>
        </div>

        <div className="auth-switch">
          Belum punya akun? <Link to="/daftar">Daftar Mahasiswa Baru</Link>
        </div>
      </section>

      <aside className="auth-visual">
        <div className="row-wrap">
          <span className="badge badge-primary">
            <Icon name="verified" filled size={13} /> KTM Terverifikasi Mahasiswa FT
          </span>
          <span className="badge badge-yellow">
            <Icon name="place" size={13} /> COD Lobi E1 & Gazebo A9
          </span>
        </div>

        <div className="metric-row">
          <div className="metric">
            <strong>KTM</strong>
            <span>Verifikasi mahasiswa</span>
          </div>
          <div className="metric">
            <strong>FT</strong>
            <span>Khusus Fakultas Teknik</span>
          </div>
          <div className="metric">
            <strong>0%</strong>
            <span>Biaya admin</span>
          </div>
        </div>

        <div className="party">
          <span className="stat-icon mint">
            <Icon name="shield" />
          </span>
          <div className="party-body">
            <div className="party-name">Verifikasi KTM langsung</div>
            <div className="party-meta">Transaksi aman COD di area resmi Fakultas Teknik Ketintang.</div>
          </div>
          <span className="badge badge-success">Aman</span>
        </div>
      </aside>
    </div>
  )
}
