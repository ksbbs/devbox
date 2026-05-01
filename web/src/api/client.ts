import axios from 'axios'
import router from '../router'

const api = axios.create({
  baseURL: '/api',
})

// 401 interceptor: clear token (router guard handles redirect)
api.interceptors.response.use(
  response => response,
  error => {
    if (error.response?.status === 401) {
      localStorage.removeItem('devbox_token')
      setToken('')
    }
    return Promise.reject(error)
  }
)

export function setToken(t: string) {
  api.defaults.headers.common['Authorization'] = `Bearer ${t}`
}

export function initAuth() {
  const t = localStorage.getItem('devbox_token')
  if (t) setToken(t)
}

export function logout() {
  localStorage.removeItem('devbox_token')
  setToken('')
  router.push('/login')
}

export function isLoggedIn(): boolean {
  return !!localStorage.getItem('devbox_token')
}

export async function login(t: string) {
  const res = await api.post('/auth/login', { token: t })
  localStorage.setItem('devbox_token', t)
  setToken(t)
  return res.data
}

export async function checkAuthRequired() {
  try {
    await api.get('/status')
    return false // no auth required (got through without token or token is valid)
  } catch (e: any) {
    if (e.response?.status === 401) return true
    return false
  }
}

export async function getStatus() {
  return api.get('/status').then(r => r.data)
}

export async function getTraffic(from?: string, to?: string, granularity?: string) {
  const params: Record<string, string> = {}
  if (from) params.from = from
  if (to) params.to = to
  if (granularity) params.granularity = granularity
  return api.get('/stats/traffic', { params }).then(r => r.data)
}

export async function getRecentLogs(limit?: number) {
  const params: Record<string, string> = {}
  if (limit) params.limit = String(limit)
  return api.get('/stats/logs', { params }).then(r => r.data)
}

export async function getMirrorConfig() {
  return api.get('/config/mirrors').then(r => r.data)
}

export async function updateMirrorConfig(name: string, enabled: boolean, upstream?: string) {
  return api.put('/config/mirrors', { name, enabled, upstream }).then(r => r.data)
}

export async function getPublicConfig() {
  return api.get('/config/public').then(r => r.data)
}

export async function searchMirrors(q: string, registry?: string) {
  const params: Record<string, string> = { q }
  if (registry) params.registry = registry
  return api.get('/search', { params }).then(r => r.data)
}