const BASE = ''

async function request(method: string, path: string, body?: any) {
  const opts: RequestInit = {
    method,
    credentials: 'include',
    headers: { 'Content-Type': 'application/json' },
  }
  if (body) opts.body = JSON.stringify(body)
  const res = await fetch(BASE + path, opts)
  return res.json()
}

export const api = {
  getSession: () => request('GET', '/api/session'),
  nextTask: () => request('POST', '/api/task/next'),
  verifyTask: () => request('POST', '/api/task/verify'),
  getProgress: () => request('GET', '/api/progress'),
  getLevels: () => request('GET', '/api/levels'),
}
