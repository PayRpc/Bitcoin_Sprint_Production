import net from 'net'

// Security controls
const DEFAULT_ALLOWED_HOSTS = ['localhost', '127.0.0.1', '::1']
const ALLOWED_HOSTS = (() => {
  const raw = process.env.PRECHECK_ALLOWED_HOSTS
  if (!raw) return new Set(DEFAULT_ALLOWED_HOSTS)
  return new Set(String(raw).split(',').map(s => s.trim()).filter(Boolean))
})()
const MAX_PORT_TARGETS = Number(process.env.PRECHECK_MAX_PORTS || 8)
const MAX_URL_TARGETS = Number(process.env.PRECHECK_MAX_URLS || 8)
const MAX_QUERY_LEN = Number(process.env.PRECHECK_MAX_QUERY_LEN || 1024)
const MIN_TIMEOUT_MS = 100
const MAX_TIMEOUT_MS = 5000

// Parse host:port with support for IPv6 in bracket notation, e.g. [::1]:8080
function parseHostPort(target) {
  if (!target) return { host: '', port: NaN }
  const s = String(target).trim()
  if (s.startsWith('[')) {
    const end = s.indexOf(']')
    if (end > 0) {
      const host = s.slice(1, end)
      const rest = s.slice(end + 1)
      const colon = rest.lastIndexOf(':')
      const portStr = colon >= 0 ? rest.slice(colon + 1) : ''
      return { host, port: Number(portStr) }
    }
  }
  const lastColon = s.lastIndexOf(':')
  if (lastColon === -1) return { host: s, port: NaN }
  return { host: s.slice(0, lastColon), port: Number(s.slice(lastColon + 1)) }
}

function checkPort(host, port, timeoutMs = 1000) {
  return new Promise((resolve) => {
    const socket = new net.Socket()
    const done = (open, error) => {
      try { socket.destroy() } catch {}
      resolve({ host, port, open, error })
    }
    socket.setTimeout(timeoutMs)
    socket.once('error', (err) => done(false, normalizeError(err)))
    socket.once('timeout', () => done(false, 'timeout'))
    socket.connect(port, host, () => done(true))
  })
}

async function checkHttp(url, timeoutMs = 1500) {
  try {
    const controller = new AbortController()
    const id = setTimeout(() => controller.abort(), timeoutMs)
    const resp = await fetch(url, { signal: controller.signal })
    clearTimeout(id)
    return { url, ok: resp.ok, status: resp.status }
  } catch (e) {
    return { url, ok: false, error: normalizeError(e) }
  }
}

function normalizeError(err) {
  const code = (err && (err.code || err.name || '').toString().toUpperCase()) || ''
  if (code.includes('TIMEOUT') || code === 'ABORTERROR') return 'timeout'
  if (code.includes('ENOTFOUND') || code.includes('EAI_AGAIN')) return 'dns_error'
  if (code.includes('ECONNREFUSED')) return 'refused'
  if (code.includes('EHOSTUNREACH') || code.includes('ENETUNREACH')) return 'unreachable'
  return 'error'
}

function isAllowedHost(host) {
  if (!host) return false
  // Exact hostname/IP match only
  return ALLOWED_HOSTS.has(host)
}

export default async function handler(req, res) {
  if (req.method && req.method.toUpperCase() !== 'GET') {
    try { res.setHeader('Allow', 'GET') } catch {}
    return res.status(405).json({ ok: false, error: 'method_not_allowed' })
  }
  const q = req.query || {}
  const portsParam = Array.isArray(q.ports) ? q.ports[0] : q.ports
  const urlsParam = Array.isArray(q.urls) ? q.urls[0] : q.urls
  const timeoutParam = Array.isArray(q.timeout) ? q.timeout[0] : q.timeout

  // Clamp timeouts
  let portTimeout = 1000
  let httpTimeout = 1500
  if (timeoutParam) {
    const t = Number(timeoutParam)
    if (!Number.isNaN(t)) {
      const clamped = Math.max(MIN_TIMEOUT_MS, Math.min(MAX_TIMEOUT_MS, t))
      portTimeout = clamped
      httpTimeout = Math.max(MIN_TIMEOUT_MS, Math.min(MAX_TIMEOUT_MS, Math.floor(clamped * 1.5)))
    }
  }

  const safePortsRaw = String(portsParam || 'localhost:8080').slice(0, MAX_QUERY_LEN)
  const safeUrlsRaw = String(urlsParam || 'http://localhost:8080/status').slice(0, MAX_QUERY_LEN)
  const portsList = safePortsRaw.split(',').map(s => s.trim()).filter(Boolean).slice(0, MAX_PORT_TARGETS)
  const urlsList = safeUrlsRaw.split(',').map(s => s.trim()).filter(Boolean).slice(0, MAX_URL_TARGETS)

  const portChecks = await Promise.all(portsList.map(t => {
    const { host, port } = parseHostPort(t)
    if (!host || !port || Number.isNaN(port)) {
      return Promise.resolve({ host: host || '', port: port || 0, open: false, error: 'invalid_target' })
    }
    if (!isAllowedHost(host)) {
      return Promise.resolve({ host, port, open: false, error: 'disallowed_host' })
    }
    if (port < 1 || port > 65535) {
      return Promise.resolve({ host, port, open: false, error: 'invalid_port' })
    }
    return checkPort(host, port, portTimeout)
  }))

  const httpChecks = await Promise.all(urlsList.map(u => {
    try {
      const url = new URL(u)
      if (!/^https?:$/.test(url.protocol)) {
        return Promise.resolve({ url: u, ok: false, error: 'invalid_scheme' })
      }
      if (!isAllowedHost(url.hostname)) {
        return Promise.resolve({ url: u, ok: false, error: 'disallowed_host' })
      }
      return checkHttp(u, httpTimeout)
    } catch {
      return Promise.resolve({ url: u, ok: false, error: 'invalid_url' })
    }
  }))

  const allPortsOk = portChecks.length === 0 ? true : portChecks.every(p => p && p.open === true)
  const allUrlsOk = httpChecks.length === 0 ? true : httpChecks.every(u => u && u.ok === true)
  const ok = allPortsOk && allUrlsOk

  try {
    res.setHeader('Cache-Control', 'no-store, no-cache, must-revalidate')
  } catch {}
  const limited = (String(portsParam || '').length > MAX_QUERY_LEN) || (String(urlsParam || '').length > MAX_QUERY_LEN)
    || (Array.isArray(portsList) && portsList.length >= MAX_PORT_TARGETS)
    || (Array.isArray(urlsList) && urlsList.length >= MAX_URL_TARGETS)
  res.status(200).json({ ok, ports: portChecks, urls: httpChecks, timestamp: Date.now(), limited })
}
