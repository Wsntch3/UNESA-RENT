import { defineConfig, loadEnv } from 'vite'
import react from '@vitejs/plugin-react'

function cspOrigins(apiUrl) {
  const httpOrigins = []
  const wsOrigins = []

  if (apiUrl) {
    try {
      const url = new URL(apiUrl, 'http://localhost')
      const isAbsolute = /^(https?:)?\/\//i.test(apiUrl)
      if (isAbsolute && url.origin !== 'http://localhost') {
        httpOrigins.push(url.origin)
        const wsProtocol = url.protocol === 'https:' ? 'wss:' : 'ws:'
        wsOrigins.push(`${wsProtocol}//${url.host}`)
      }
    } catch {
      // Abaikan nilai VITE_API_URL yang bukan URL valid; default tetap same-origin.
    }
  }

  return { httpOrigins, wsOrigins }
}

function contentSecurityPolicy(apiUrl, isDev) {
  const { httpOrigins, wsOrigins } = cspOrigins(apiUrl)
  const connectSources = ["'self'", ...httpOrigins, ...wsOrigins].join(' ')
  const imageSources = ["'self'", 'data:', 'blob:', ...httpOrigins].join(' ')
  const scriptSources = isDev ? "'self' 'unsafe-inline'" : "'self'"

  return [
    "default-src 'self'",
    `img-src ${imageSources}`,
    `connect-src ${connectSources}`,
    "style-src 'self' 'unsafe-inline' https://fonts.googleapis.com",
    "font-src 'self' https://fonts.gstatic.com data:",
    `script-src ${scriptSources}`,
    "object-src 'none'",
    "base-uri 'self'",
    "form-action 'self'",
    "frame-ancestors 'none'",
  ].join('; ')
}

export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), '')
  const isDev = mode !== 'production'
  const csp = contentSecurityPolicy(env.VITE_API_URL, isDev)

  return {
    plugins: [
      react(),
      {
        name: 'security-csp',
        transformIndexHtml(html) {
          return html.replace(
            '<head>',
            `<head>\n    <meta http-equiv="Content-Security-Policy" content="${csp}" />`,
          )
        },
      },
    ],
    server: {
      port: 5173,
      proxy: {
        '/api': {
          target: process.env.VITE_PROXY_TARGET || 'http://localhost:8080',
          changeOrigin: true,
          ws: true,
        },
      },
    },
  }
})
