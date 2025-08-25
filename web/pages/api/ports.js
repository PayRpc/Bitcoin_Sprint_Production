import net from 'net'

function checkPort(host, port, timeoutMs = 1000) {
  return new Promise((resolve) => {
    const socket = new net.Socket()
    const onResult = (open, error) => {
      try { socket.destroy() } catch {}
      resolve({ host, port, open, error })
    }
    socket.setTimeout(timeoutMs)
    socket.once('error', (err) => onResult(false, (err && err.message) || 'error'))
    socket.once('timeout', () => onResult(false, 'timeout'))
    socket.connect(port, host, () => onResult(true))
  })
}

export default async function handler(req, res) {
  const q = req.query || {}
  const targetsParam = Array.isArray(q.targets) ? q.targets[0] : q.targets
  const targets = targetsParam ? String(targetsParam).split(',') : []

  const defaults = [ 'localhost:8080', '127.0.0.1:8080' ]
  const all = (targets.length ? targets : defaults)
    .map(s => s.trim())
    .filter(Boolean)

  const checks = await Promise.all(all.map(t => {
    const [host, portStr] = t.split(':')
    const port = Number(portStr)
    if (!host || !port || Number.isNaN(port)) {
      return Promise.resolve({ host: host || '', port: port || 0, open: false, error: 'invalid_target' })
    }
    return checkPort(host, port, 1000)
  }))

  res.status(200).json({ ok: true, checks })
}
