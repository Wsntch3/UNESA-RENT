import { request } from './http'

export const authApi = {
  register({ name, email, password, nim, verification_token }) {
    return request('/api/auth/register', {
      method: 'POST',
      auth: false,
      body: { name, email, password, nim, verification_token },
    })
  },

  login({ email, password }) {
    return request('/api/auth/login', {
      method: 'POST',
      auth: false,
      body: { email, password },
    })
  },

  loginOtp({ challenge, code }) {
    return request('/api/auth/login/otp', {
      method: 'POST',
      auth: false,
      body: { challenge, code },
    })
  },

  requestOtp({ email, purpose }) {
    return request('/api/auth/otp/request', {
      method: 'POST',
      auth: false,
      body: { email, purpose },
    })
  },

  verifyOtp({ email, purpose, code }) {
    return request('/api/auth/otp/verify', {
      method: 'POST',
      auth: false,
      body: { email, purpose, code },
    })
  },

  forgotPassword({ email }) {
    return request('/api/auth/password/forgot', {
      method: 'POST',
      auth: false,
      body: { email },
    })
  },

  resetPassword({ email, ticket, new_password }) {
    return request('/api/auth/password/reset', {
      method: 'POST',
      auth: false,
      body: { email, ticket, new_password },
    })
  },

  me(signal) {
    return request('/api/auth/me', { signal })
  },
}
