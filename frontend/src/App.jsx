import { useAuth } from './auth/useAuth'

export default function App() {
  const { status, user, error, logout } = useAuth()

  return (
    <main className="app">
      <h1>UNESARENT</h1>
      <p className="hint">
        Logika autentikasi sudah siap. Halaman ini sengaja minimal — ganti dengan tampilan buatanmu.
      </p>

      {status === 'loading' && <p>Memeriksa sesi...</p>}

      {status === 'anonymous' && (
        <p>
          Belum login. Buat form lalu panggil <code>login({'{ email, password }'})</code> atau{' '}
          <code>register({'{ name, email, password, nim }'})</code> dari <code>useAuth()</code>.
        </p>
      )}

      {status === 'authenticated' && (
        <section>
          <p>
            Login sebagai <strong>{user.name}</strong> ({user.email})
          </p>
          <button type="button" onClick={logout}>
            Logout
          </button>
        </section>
      )}

      {error && (
        <p role="alert" className="error">
          {error.message}
          {error.status === 0 ? '' : ` (${error.code})`}
        </p>
      )}
    </main>
  )
}
