import axios from 'axios'

const api = axios.create({
  baseURL: '/api',
})

export function setToken(t: string) {
  api.defaults.headers.common['Authorization'] = `Bearer ${t}`
}

export async function login(t: string) {
  const res = await api.post('/auth/login', { token: t })
  setToken(t)
  return res.data
}

export async function getStatus() {
  return api.get('/status').then(r => r.data)
}

export async function getTraffic(from?: string, to?: string) {
  const params: Record<string, string> = {}
  if (from) params.from = from
  if (to) params.to = to
  return api.get('/stats/traffic', { params }).then(r => r.data)
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