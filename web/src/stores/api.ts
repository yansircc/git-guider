const BASE = ''

async function request(method: string, path: string, body?: any) {
  const opts: RequestInit = {
    method,
    credentials: 'include',
    headers: { 'Content-Type': 'application/json' },
  }
  if (body) opts.body = JSON.stringify(body)
  const res = await fetch(BASE + path, opts)

  // Parse body safely — server should always return JSON, but proxy errors
  // or panics may return plain text. Never let res.json() throw unhandled.
  let data: any
  const text = await res.text()
  try {
    data = JSON.parse(text)
  } catch {
    // Non-JSON response — wrap the raw text as an error
    throw new Error(text.slice(0, 200) || `HTTP ${res.status}`)
  }

  if (!res.ok) {
    throw new Error(data.error || `HTTP ${res.status}`)
  }
  return data
}

export const api = {
  getSession: () => request('GET', '/api/session'),
  nextTask: () => request('POST', '/api/task/next'),
  verifyTask: () => request('POST', '/api/task/verify'),
  getProgress: () => request('GET', '/api/progress'),
  getLevels: () => request('GET', '/api/levels'),
  reset: () => request('POST', '/api/reset'),
}
