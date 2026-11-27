import { useEffect, useState } from 'react'
import { Link, useRouter } from '../router'
import { useAuth } from '../auth/useAuth'
import { Alert, Icon } from '../components/ui'

const STEPS = [
  { key: 'email', label: 'Email' },
  { key: 'otp', label: 'Verifikasi' },
  { key: 'password', label: 'Sandi Baru' },
]

export default function ForgotPassword() {
  const { navigate } = useRouter()
  const { forgotPassword, verifyOtp, resetPassword, error, clearError } = useAuth()

  const [step, setStep] = useState('email')
  const [email, setEmail] = useState('')
  const [otp, setOtp] = useState('')
  const [ticket, setTicket] = useState('')
  const [password, setPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [showPassword, setShowPassword] = useState(false)
  const [localError, setLocalError] = useState(null)
  const [busy, setBusy] = useState(false)
  const [cooldown, setCooldown] = useState(0)

  useEffect(() => {
    if (cooldown <= 0) return undefined
    const timer = setInterval(() => setCooldown((value) => Math.max(0, value - 1)), 1000)
    return () => clearInterval(timer)
  }, [cooldown])

  const activeError = localError || error

  const requestCode = async (event) => {
    event?.preventDefault()
    setLocalError(null)
    clearError?.()

    if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email.trim())) {
      setLocalError({ message: 'Masukkan alamat email yang valid.' })
      return
    }

    setBusy(true)
    try {
      await forgotPassword({ email: email.trim() })
      setStep('otp')
      setOtp('')
      setCooldown(60)
    } catch (err) {
      setLocalError(err)
    } finally {
      setBusy(false)
    }
  }

  const resend = async () => {
    if (cooldown > 0 || busy) return
    setLocalError(null)
    setBusy(true)
    try {
      await forgotPassword({ email: email.trim() })
      setCooldown(60)
    } catch (err) {
      setLocalError(err)
    } finally {
      setBusy(false)
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

    setBusy(true)
    try {
      const verified = await verifyOtp({
        email: email.trim(),
        purpose: 'reset_password',
        code: otp.trim(),
      })
      setTicket(verified.verification_token)
      setStep('password')
    } catch (err) {
      setLocalError(err)
    } finally {
      setBusy(false)
    }
  }

  const submitPassword = async (event) => {
    event.preventDefault()
    setLocalError(null)
    clearError?.()

    if (password.length < 8) {
      setLocalError({ message: 'Kata sandi baru minimal 8 karakter.' })
      return
    }
    if (password !== confirmPassword) {
      setLocalError({ message: 'Konfirmasi kata sandi tidak sama.' })
      return
    }

    setBusy(true)
    try {
      await resetPassword({
        email: email.trim(),
        ticket,
        new_password: password,
      })
      setStep('done')
    } catch (err) {
      setLocalError(err)
    } finally {
      setBusy(false)
    }
  }

  return (
    <div className="container auth-wrap">
      <section className="auth-card">
        <div>
          <span className="badge badge-primary">
            <Icon name="lock_reset" filled size={13} /> Pemulihan Akun
          </span>
          <h1 className="auth-title mt-3">
            {step === 'done' ? 'Kata Sandi Diperbarui' : 'Lupa Kata Sandi?'}
          </h1>
          <p className="subtitle mt-2">
            {step === 'email'
              ? 'Masukkan email akunmu. Kami akan mengirim kode verifikasi untuk membuat kata sandi baru.'
              : step === 'otp'
                ? 'Jika email terdaftar, kode verifikasi telah dikirim.'
                : step === 'password'
                  ? 'Buat kata sandi baru yang kuat dan mudah kamu ingat.'
                  : 'Kata sandi baru sudah tersimpan. Silakan masuk kembali dengan kata sandi tersebut.'}
          </p>
        </div>

        {step !== 'done' ? (
          <div className="row-wrap mt-4">
            {STEPS.map((item, index) => (
              <span key={item.key} className="row">
                <span className={`verification-step ${step === item.key ? 'active' : ''}`.trim()}>
                  <span className="num">{index + 1}</span> {item.label}
                </span>
                {index < STEPS.length - 1 ? (
                  <Icon name="chevron_right" size={15} className="muted" />
                ) : null}
              </span>
            ))}
          </div>
        ) : null}

        {activeError ? (
          <div className="mt-4">
            <Alert tone="error" icon="error" title="Gagal memproses">
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

        {step === 'email' ? (
          <form className="auth-form" onSubmit={requestCode} noValidate>
            <div className="field">
              <label className="label" htmlFor="reset-email">Email Akun</label>
              <div className="input-wrap">
                <Icon name="mail" />
                <input
                  id="reset-email"
                  className="input"
                  type="email"
                  autoComplete="email"
                  placeholder="nama@mhs.unesa.ac.id"
                  value={email}
                  onChange={(event) => setEmail(event.target.value)}
                  required
                />
              </div>
              <span className="hint">Gunakan email yang terdaftar di akun UNESA Rent.</span>
            </div>

            <button type="submit" className="btn btn-primary btn-block btn-lg" disabled={busy}>
              {busy ? <span className="spinner" /> : <Icon name="mail" />}
              {busy ? 'Mengirim kode…' : 'Kirim Kode Verifikasi'}
            </button>
          </form>
        ) : null}

        {step === 'otp' ? (
          <form className="auth-form" onSubmit={submitOtp} noValidate>
            <div className="field">
              <label className="label" htmlFor="reset-otp">Kode Verifikasi</label>
              <div className="input-wrap">
                <Icon name="pin" />
                <input
                  id="reset-otp"
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
              <span className="hint">Kode berlaku 10 menit dan hanya bisa dipakai sekali.</span>
            </div>

            <button type="submit" className="btn btn-primary btn-block btn-lg" disabled={busy}>
              {busy ? <span className="spinner" /> : <Icon name="verified" />}
              {busy ? 'Memverifikasi…' : 'Verifikasi Kode'}
            </button>

            <div className="row-wrap" style={{ justifyContent: 'space-between' }}>
              <button
                type="button"
                className="btn btn-ghost btn-sm"
                onClick={() => {
                  setStep('email')
                  setOtp('')
                  setLocalError(null)
                  clearError?.()
                }}
              >
                <Icon name="arrow_back" /> Ubah email
              </button>
              <button
                type="button"
                className="btn btn-outline btn-sm"
                onClick={resend}
                disabled={cooldown > 0 || busy}
              >
                <Icon name="refresh" /> {cooldown > 0 ? `Kirim ulang (${cooldown}s)` : 'Kirim ulang kode'}
              </button>
            </div>
          </form>
        ) : null}

        {step === 'password' ? (
          <form className="auth-form" onSubmit={submitPassword} noValidate>
            <div className="field">
              <label className="label" htmlFor="new-password">Kata Sandi Baru</label>
              <div className="input-wrap">
                <Icon name="lock" />
                <input
                  id="new-password"
                  className="input has-action"
                  type={showPassword ? 'text' : 'password'}
                  autoComplete="new-password"
                  placeholder="Min. 8 karakter"
                  value={password}
                  onChange={(event) => setPassword(event.target.value)}
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

            <div className="field">
              <label className="label" htmlFor="confirm-new-password">Konfirmasi Kata Sandi Baru</label>
              <div className="input-wrap">
                <Icon name="lock_reset" />
                <input
                  id="confirm-new-password"
                  className="input"
                  type={showPassword ? 'text' : 'password'}
                  autoComplete="new-password"
                  placeholder="Ulangi kata sandi baru"
                  value={confirmPassword}
                  onChange={(event) => setConfirmPassword(event.target.value)}
                  required
                />
              </div>
            </div>

            <button type="submit" className="btn btn-primary btn-block btn-lg" disabled={busy}>
              {busy ? <span className="spinner" /> : <Icon name="save" />}
              {busy ? 'Menyimpan…' : 'Simpan Kata Sandi Baru'}
            </button>
          </form>
        ) : null}

        {step === 'done' ? (
          <div className="auth-form">
            <Alert tone="success" icon="check_circle" title="Berhasil">
              Kata sandi akun {email.trim()} sudah diperbarui. Sesi lama otomatis tidak berlaku.
            </Alert>
            <button
              type="button"
              className="btn btn-primary btn-block btn-lg"
              onClick={() => navigate('/masuk', { replace: true })}
            >
              <Icon name="login" /> Masuk dengan Kata Sandi Baru
            </button>
          </div>
        ) : null}

        <div className="auth-switch">
          Ingat kata sandi? <Link to="/masuk">Kembali ke halaman masuk</Link>
        </div>
      </section>

      <aside className="auth-visual">
        <span className="eyebrow">
          <Icon name="shield" size={15} /> Keamanan Akun
        </span>
        <h2 className="auth-title" style={{ fontSize: 26 }}>Pemulihan yang aman</h2>

        <div className="info-list">
          <li>
            <Icon name="check_circle" />
            <span><strong>Verifikasi email.</strong> Kode hanya dikirim ke email yang terdaftar.</span>
          </li>
          <li>
            <Icon name="check_circle" />
            <span><strong>Sekali pakai.</strong> Setiap kode berlaku 10 menit dan tidak bisa dipakai ulang.</span>
          </li>
          <li>
            <Icon name="check_circle" />
            <span><strong>Sesi lama berakhir.</strong> Setelah sandi diganti, token lama otomatis tidak berlaku.</span>
          </li>
        </div>

        <Alert tone="info" icon="info">
          Admin UNESA Rent tidak pernah meminta kode verifikasi atau kata sandimu. Jangan bagikan ke siapa pun.
        </Alert>
      </aside>
    </div>
  )
}
