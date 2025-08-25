import net from 'net';
import type { NextApiRequest, NextApiResponse } from 'next';

function checkPort(host: string, port: number, timeoutMs = 1000): Promise<{ host: string; port: number; open: boolean; error?: string }>{
  return new Promise((resolve) => {
    const socket = new net.Socket()
    const onResult = (open: boolean, error?: string) => {
      try { socket.destroy() } catch {}
      resolve({ host, port, open, error })
    }
    socket.setTimeout(timeoutMs)
    socket.once('error', (err) => onResult(false, err?.message || 'error'))
    socket.once('timeout', () => onResult(false, 'timeout'))
    socket.connect(port, host, () => onResult(true))
  })
}

export default async function handler(req: NextApiRequest, res: NextApiResponse) {
  const q = req.query
  const targetsParam = Array.isArray(q.targets) ? q.targets[0] : (q.targets as string | undefined)
  const targets = targetsParam ? targetsParam.split(',') : []

  // Default targets: local API and dashboard
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
