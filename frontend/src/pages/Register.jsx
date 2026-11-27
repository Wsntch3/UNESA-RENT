import { useEffect, useState } from 'react'
import { Link, useLocation, useRouter } from '../router'
import { useAuth } from '../auth/useAuth'
import { Alert, Icon, PageLoader } from '../components/ui'

const PROGRAMS = [{ code: '120', label: 'S1 Teknik Informatika (kode NIM 120)' }]

const STUDENT_EMAIL = /^[^\s@]+@mhs\.unesa\.ac\.id$/i

export default function Register() {
  const { navigate } = useRouter()
  const { query } = useLocation()
  const {
    register,
    requestOtp,
    verifyOtp,
    status,
    pending,
    error,
    clearError,
  } = useAuth()

  const [step, setStep] = useState('form')
  const [form, setForm] = useState({
    name: '',
    email: '',
    password: '',
    confirmPassword: '',
    nim: '',
    program: '120',
    agree: false,
  })
  const [otp, setOtp] = useState('')
  const [showPassword, setShowPassword] = useState(false)
  const [localError, setLocalError] = useState(null)
  const [otpError, setOtpError] = useState(null)
  const [otpBusy, setOtpBusy] = useState(false)
  const [cooldown, setCooldown] = useState(0)

  const target = query.next || '/profil'

  useEffect(() => {
    if (status === 'authenticated') navigate(target, { replace: true })
  }, [status, navigate, target])

  useEffect(() => {
    if (cooldown <= 0) return undefined
    const timer = setInterval(() => setCooldown((value) => Math.max(0, value - 1)), 1000)
    return () => clearInterval(timer)
  }, [cooldown])

  if (status === 'loading' || status === 'authenticated') return <PageLoader />

  const set = (key) => (event) => {
    const value = event.target.type === 'checkbox' ? event.target.checked : event.target.value
    setForm((prev) => ({ ...prev, [key]: value }))
  }

  const validateForm = () => {
    const fields = {}
    if (!form.name.trim()) fields.name = 'Nama lengkap wajib diisi.'
    if (!STUDENT_EMAIL.test(form.email.trim())) {
      fields.email = 'Gunakan email resmi mahasiswa @mhs.unesa.ac.id.'
    }
    if (form.password.length < 8) fields.password = 'Password minimal 8 karakter.'
    if (form.password !== form.confirmPassword) fields.confirmPassword = 'Konfirmasi kata sandi tidak sama.'
    if (!/^\d{11}$/.test(form.nim.trim())) fields.nim = 'NIM harus 11 digit angka.'
    else if (form.nim.trim().slice(2, 4) !== '05' || form.nim.trim().slice(4, 7) !== '120') {
      fields.nim = 'NIM harus memuat kode FT 05 dan prodi 120 pada digit 3–7.'
    }
    if (!form.agree) fields.agree = 'Kamu harus menyetujui ketentuan layanan.'
    return fields
  }

  const submitForm = async (event) => {
    event.preventDefault()
    setLocalError(null)
    clearError?.()

    const fields = validateForm()
    if (Object.keys(fields).length) {
      setLocalError({ message: 'Periksa kembali data pendaftaran.', fields })
      return
    }

    setOtpBusy(true)
    try {
      await requestOtp({ email: form.email.trim(), purpose: 'register' })
      setStep('otp')
      setOtp('')
      setOtpError(null)
      setCooldown(60)
    } catch (err) {
      setLocalError(err)
    } finally {
      setOtpBusy(false)
    }
  }

  const resend = async () => {
    if (cooldown > 0 || otpBusy) return
    setOtpError(null)
    setOtpBusy(true)
    try {
      await requestOtp({ email: form.email.trim(), purpose: 'register' })
      setCooldown(60)
    } catch (err) {
      setOtpError(err)
    } finally {
      setOtpBusy(false)
    }
  }

  const submitOtp = async (event) => {
    event.preventDefault()
    setOtpError(null)
    clearError?.()

    if (!/^\d{6}$/.test(otp.trim())) {
      setOtpError({ message: 'Masukkan 6 digit kode verifikasi.' })
      return
    }

    setOtpBusy(true)
    try {
      const verified = await verifyOtp({
        email: form.email.trim(),
        purpose: 'register',
        code: otp.trim(),
      })
      await register({
        name: form.name.trim(),
        email: form.email.trim(),
        password: form.password,
        nim: form.nim.trim(),
        verification_token: verified.verification_token,
      })
      navigate(target, { replace: true })
    } catch (err) {
      setOtpError(err)
      if (err?.code === 'invalid_ticket' || err?.fields?.verification_token) {
        setStep('otp')
      }
    } finally {
      setOtpBusy(false)
    }
  }

  const activeError = localError || error

  return (
    <div className="container auth-wrap">
      <section className="auth-card">
        <div>
          <span className="badge badge-primary">
            <Icon name="school" filled size={13} /> Pendaftaran Resmi
          </span>
          <h1 className="auth-title mt-3">
            {step === 'form' ? 'Mulai sewanya dari sini' : 'Verifikasi Email Mahasiswa'}
          </h1>
          <p className="subtitle mt-2">
            {step === 'form'
              ? 'Buat akun untuk menyewa dan menyewakan barang bersama sesama mahasiswa Fakultas Teknik UNESA.'
              : `Kami mengirim 6 digit kode ke ${form.email.trim()}.`}
          </p>
        </div>

        <div className="row-wrap mt-4">
          <span className={`verification-step ${step === 'form' ? 'active' : 'done'}`.trim()}>
            <span className="num">1</span> Buat Akun
          </span>
          <Icon name="chevron_right" size={15} className="muted" />
          <span className={`verification-step ${step === 'otp' ? 'active' : ''}`.trim()}>
            <span className="num">2</span> Verifikasi Email
          </span>
          <Icon name="chevron_right" size={15} className="muted" />
          <span className="verification-step">
            <span className="num">3</span> Verifikasi KTM
          </span>
        </div>

        {step === 'form' && activeError ? (
          <div className="mt-4">
            <Alert tone="error" icon="error" title="Pendaftaran gagal">
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

        {step === 'form' ? (
          <form className="auth-form" onSubmit={submitForm} noValidate>
            <div className="field">
              <label className="label" htmlFor="name">Nama Lengkap <span className="req">*</span></label>
              <div className="input-wrap">
                <Icon name="person" />
                <input
                  id="name"
                  className="input"
                  placeholder="Sesuai identitas mahasiswa / Siakadu"
                  value={form.name}
                  onChange={set('name')}
                  autoComplete="name"
                  required
                />
              </div>
            </div>

            <div className="field">
              <label className="label" htmlFor="reg-email">Email Mahasiswa UNESA <span className="req">*</span></label>
              <div className="input-wrap">
                <Icon name="mail" />
                <input
                  id="reg-email"
                  className="input"
                  type="email"
                  placeholder="nama@mhs.unesa.ac.id"
                  value={form.email}
                  onChange={set('email')}
                  autoComplete="email"
                  required
                />
              </div>
              <span className="hint">Wajib berakhiran @mhs.unesa.ac.id. Kode verifikasi dikirim ke email ini.</span>
            </div>

            <div className="form-grid">
              <div className="field">
                <label className="label" htmlFor="program">Program Studi <span className="req">*</span></label>
                <select id="program" className="select" value={form.program} onChange={set('program')}>
                  {PROGRAMS.map((program) => (
                    <option key={program.code} value={program.code}>
                      {program.label}
                    </option>
                  ))}
                </select>
                <span className="hint">Prodi ditentukan otomatis dari prefix NIM.</span>
              </div>

              <div className="field">
                <label className="label" htmlFor="nim">NIM <span className="req">*</span></label>
                <div className="input-wrap">
                  <Icon name="tag" />
                  <input
                    id="nim"
                    className="input"
                    inputMode="numeric"
                    maxLength={11}
                    placeholder="24051204999"
                    value={form.nim}
                    onChange={(event) =>
                      setForm((prev) => ({ ...prev, nim: event.target.value.replace(/\D/g, '') }))
                    }
                    required
                  />
                </div>
                <span className="hint">11 digit: angkatan (2) + 05 (FT) + 120 (prodi) + jenjang + urutan.</span>
              </div>
            </div>

            <div className="form-grid">
              <div className="field">
                <label className="label" htmlFor="reg-password">Kata Sandi <span className="req">*</span></label>
                <div className="input-wrap">
                  <Icon name="lock" />
                  <input
                    id="reg-password"
                    className="input has-action"
                    type={showPassword ? 'text' : 'password'}
                    placeholder="Min. 8 karakter"
                    value={form.password}
                    onChange={set('password')}
                    autoComplete="new-password"
                    required
                  />
                  <button
                    type="button"
                    className="input-action"
                    onClick={() => setShowPassword((value) => !value)}
                    aria-label="Tampilkan kata sandi"
                  >
                    <Icon name={showPassword ? 'visibility_off' : 'visibility'} />
                  </button>
                </div>
              </div>

              <div className="field">
                <label className="label" htmlFor="confirm">Konfirmasi Kata Sandi <span className="req">*</span></label>
                <div className="input-wrap">
                  <Icon name="lock_reset" />
                  <input
                    id="confirm"
                    className="input"
                    type={showPassword ? 'text' : 'password'}
                    placeholder="Ulangi kata sandi"
                    value={form.confirmPassword}
                    onChange={set('confirmPassword')}
                    autoComplete="new-password"
                    required
                  />
                </div>
              </div>
            </div>

            <label className="checkbox">
              <input type="checkbox" checked={form.agree} onChange={set('agree')} />
              <span>
                Saya menyetujui ketentuan layanan dan kebijakan privasi komunitas UNESA Rent, serta
                bersedia memverifikasi KTM pada langkah berikutnya.
              </span>
            </label>

            <button type="submit" className="btn btn-primary btn-block btn-lg" disabled={pending || otpBusy}>
              {pending || otpBusy ? <span className="spinner" /> : <Icon name="mail" />}
              {pending || otpBusy ? 'Mengirim kode…' : 'Kirim Kode Verifikasi'}
            </button>
          </form>
        ) : (
          <form className="auth-form" onSubmit={submitOtp} noValidate>
            {otpError ? (
              <Alert tone="error" icon="error" title="Verifikasi gagal">
                {otpError.message}
                {otpError.fields ? (
                  <ul className="mt-1" style={{ listStyle: 'disc', paddingLeft: 18 }}>
                    {Object.entries(otpError.fields).map(([field, message]) => (
                      <li key={field}>{message}</li>
                    ))}
                  </ul>
                ) : null}
              </Alert>
            ) : null}

            <div className="field">
              <label className="label" htmlFor="otp">Kode Verifikasi <span className="req">*</span></label>
              <div className="input-wrap">
                <Icon name="pin" />
                <input
                  id="otp"
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

            <button type="submit" className="btn btn-primary btn-block btn-lg" disabled={otpBusy || pending}>
              {otpBusy || pending ? <span className="spinner" /> : <Icon name="verified" />}
              {otpBusy || pending ? 'Memverifikasi…' : 'Verifikasi & Buat Akun'}
            </button>

            <div className="row-wrap" style={{ justifyContent: 'space-between' }}>
              <button
                type="button"
                className="btn btn-ghost btn-sm"
                onClick={() => {
                  setStep('form')
                  setOtpError(null)
                }}
              >
                <Icon name="arrow_back" /> Ubah data
              </button>
              <button
                type="button"
                className="btn btn-outline btn-sm"
                onClick={resend}
                disabled={cooldown > 0 || otpBusy}
              >
                <Icon name="refresh" /> {cooldown > 0 ? `Kirim ulang (${cooldown}s)` : 'Kirim ulang kode'}
              </button>
            </div>

            <Alert tone="info" icon="info">
              Email terverifikasi. Lanjutkan verifikasi KTM di halaman profil.
            </Alert>
          </form>
        )}

        <div className="auth-switch">
          Sudah memiliki akun? <Link to="/masuk">Masuk di sini</Link>
        </div>
      </section>

      <aside className="auth-visual">
        <span className="eyebrow">
          <Icon name="handshake" size={15} /> Komunitas Kampus
        </span>
        <h2 className="auth-title" style={{ fontSize: 26 }}>Kenapa bergabung di UNESA Rent?</h2>

        <div className="info-list">
          <li>
            <Icon name="check_circle" />
            <span><strong>Hemat biaya.</strong> Sewa alat praktikum hanya saat dibutuhkan, tanpa beli mahal.</span>
          </li>
          <li>
            <Icon name="check_circle" />
            <span><strong>Lingkaran terpercaya.</strong> Semua transaksi hanya antar mahasiswa FT UNESA.</span>
          </li>
          <li>
            <Icon name="check_circle" />
            <span><strong>COD aman.</strong> Cek fisik barang dan KTM langsung di area kampus.</span>
          </li>
          <li>
            <Icon name="check_circle" />
            <span><strong>Dokumen jelas.</strong> Riwayat status dan surat perjanjian tersimpan digital.</span>
          </li>
        </div>

        <div className="card card-pad card-flat" style={{ background: 'var(--subtle)', borderColor: 'transparent' }}>
          <div className="row">
            <span className="stat-icon">
              <Icon name="badge" />
            </span>
            <div>
              <div className="card-title">Verifikasi KTM 1× saja</div>
              <div className="tiny muted mt-1">Setelah disetujui admin, kamu bisa menyewa dan menyewakan.</div>
            </div>
          </div>
        </div>

        <div className="card card-pad card-flat" style={{ background: 'var(--yellow-panel)', borderColor: 'transparent' }}>
          <div className="card-title">
            <Icon name="lightbulb" filled /> Catatan NIM
          </div>
          <p className="small mt-1">
            NIM wajib 11 digit dengan format <strong>AA FF PPP J UUU</strong>. Pastikan NIM dan
            program studi sesuai KTM.
          </p>
        </div>

        <p className="tiny muted">
          Dengan mendaftar, kamu menyetujui bahwa data KTM hanya dipakai untuk verifikasi internal dan
          tidak ditampilkan ke publik.
        </p>
      </aside>
    </div>
  )
}
