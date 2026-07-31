import axios from "axios";

const api = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL || "/api",
});

// 401 interceptor: clear token and bounce to the login page.
// Uses a full page redirect (instead of importing the router) to avoid
// the circular import client.ts <-> router/index.ts.
api.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response?.status === 401) {
      localStorage.removeItem("devbox_token");
      setToken("");
      if (window.location.pathname !== "/login") {
        window.location.href =
          "/login?redirect=" +
          encodeURIComponent(window.location.pathname + window.location.search);
      }
    }
    return Promise.reject(error);
  },
);

export function setToken(t: string) {
  api.defaults.headers.common["Authorization"] = `Bearer ${t}`;
}

export function initAuth() {
  const t = localStorage.getItem("devbox_token");
  if (t) setToken(t);
}

export function logout() {
  localStorage.removeItem("devbox_token");
  setToken("");
}

export function isLoggedIn(): boolean {
  return !!localStorage.getItem("devbox_token");
}

export async function login(t: string) {
  const res = await api.post("/auth/login", { token: t });
  localStorage.setItem("devbox_token", t);
  setToken(t);
  return res.data;
}

export async function checkAuthRequired() {
  try {
    await api.get("/auth/check");
    return false;
  } catch (e: any) {
    if (e.response?.status === 401) return true;
    return false;
  }
}

export async function getStatus() {
  return api.get("/status").then((r) => r.data);
}

export async function getTraffic(
  from?: string,
  to?: string,
  granularity?: string,
) {
  const params: Record<string, string> = {};
  if (from) params.from = from;
  if (to) params.to = to;
  if (granularity) params.granularity = granularity;
  return api.get("/stats/traffic", { params }).then((r) => r.data);
}

export async function getRecentLogs(limit?: number) {
  const params: Record<string, string> = {};
  if (limit) params.limit = String(limit);
  return api.get("/stats/logs", { params }).then((r) => r.data);
}

export async function getMirrorConfig() {
  return api.get("/config/mirrors").then((r) => r.data);
}

export async function updateMirrorConfig(
  name: string,
  enabled: boolean,
  upstream?: string,
  cacheTTL?: string,
) {
  return api
    .put("/config/mirrors", { name, enabled, upstream, cacheTTL })
    .then((r) => r.data);
}

export async function getPublicConfig() {
  return api.get("/config/public").then((r) => r.data);
}

export async function getGitProxyConfig() {
  return api.get("/config/gitproxy").then((r) => r.data);
}

export async function updateGitProxyConfig(cacheTTL: string) {
  return api.put("/config/gitproxy", { cacheTTL }).then((r) => r.data);
}

export async function searchMirrors(
  q: string,
  registry?: string,
  page?: number,
  perPage?: number,
) {
  const params: Record<string, string> = { q };
  if (registry) params.registry = registry;
  if (page) params.page = String(page);
  if (perPage) params.per_page = String(perPage);
  return api.get("/search", { params }).then((r) => r.data);
}

export async function getRateLimitConfig() {
  return api.get("/config/ratelimit").then((r) => r.data);
}

export async function updateRateLimitConfig(config: {
  enabled: boolean;
  rate: number;
  interval: string;
  whitelist: string[];
  blacklist: string[];
}) {
  return api.put("/config/ratelimit", config).then((r) => r.data);
}

export interface ReleaseSource {
  id: number;
  name: string;
  owner: string;
  repo: string;
  assetName: string;
  createdAt: string;
  repositoryUrl: string;
  tagName?: string;
  releaseUrl?: string;
  publishedAt?: string;
  assetSize?: number;
  digest?: string;
  available: boolean;
  error?: string;
}

export async function getReleaseSources(
  refresh = false,
): Promise<ReleaseSource[]> {
  return api
    .get("/release-sources", { params: refresh ? { refresh: "1" } : undefined })
    .then((r) => r.data);
}

export async function createReleaseSource(source: {
  name: string;
  releaseUrl: string;
  assetName: string;
}): Promise<ReleaseSource> {
  return api.post("/release-sources", source).then((r) => r.data);
}

export async function deleteReleaseSource(id: number) {
  return api.delete(`/release-sources/${id}`).then((r) => r.data);
}

export async function createReleaseDownloadTicket(id: number): Promise<{
  ticket: string;
  fileName: string;
  tagName: string;
}> {
  return api.post(`/release-sources/${id}/download-ticket`).then((r) => r.data);
}

export function getReleaseDownloadUrl(ticket: string): string {
  const base = import.meta.env.VITE_API_BASE_URL || "/api";
  return `${base.replace(/\/$/, "")}/release-download?ticket=${encodeURIComponent(ticket)}`;
}
