import { api } from './api'

// Reactive state using Svelte 5 runes-compatible module-level $state
// We use plain objects + callback pattern for store since this is a .ts file

type Task = {
  id: string
  difficulty: number
  description: string
  hints: string[]
  on_pass_note: string
}

type Level = {
  key: string
  name: string
  tasks: Task[]
}

type GameState = {
  sessionID: string
  task: Task | null
  levels: Level[]
  verifyResult: any | null
  loading: boolean
  message: string
}

let state: GameState = {
  sessionID: '',
  task: null,
  levels: [],
  verifyResult: null,
  loading: false,
  message: '',
}

let listeners: Array<(s: GameState) => void> = []

function notify() {
  listeners.forEach(fn => fn(state))
}

export function subscribe(fn: (s: GameState) => void) {
  listeners.push(fn)
  fn(state)
  return () => {
    listeners = listeners.filter(l => l !== fn)
  }
}

export function getState() {
  return state
}

export async function initSession() {
  state = { ...state, loading: true }
  notify()
  try {
    const sess = await api.getSession()
    state = { ...state, sessionID: sess.id, loading: false }

    // Session cookie is now set — terminal can connect WS
    window.dispatchEvent(new CustomEvent('session-ready'))

    const levels = await api.getLevels()
    state = { ...state, levels }

    // If session has an active task, restore it
    if (sess.task_id) {
      const task = levels
        .flatMap((l: Level) => l.tasks)
        .find((t: Task) => t.id === sess.task_id)
      if (task) {
        state = { ...state, task }
      }
    }
    notify()
  } catch (e: any) {
    state = { ...state, loading: false, message: e.message }
    notify()
  }
}

export async function startNextTask() {
  state = { ...state, loading: true, verifyResult: null, message: '' }
  notify()
  try {
    const res = await api.nextTask()
    if (res.done) {
      // All topics mastered — success terminal state
      state = { ...state, loading: false, task: null, message: res.message || 'All topics mastered!' }
    } else {
      state = { ...state, task: res.task, loading: false }
      // Notify terminal to sync CWD/prompt after setup changed session
      window.dispatchEvent(new CustomEvent('task-started'))
    }
    notify()
  } catch (e: any) {
    // HTTP errors (4xx/5xx) land here via api.ts res.ok check
    state = { ...state, loading: false, message: e.message }
    notify()
  }
}

export async function verifyTask() {
  state = { ...state, loading: true, message: '' }
  notify()
  try {
    const result = await api.verifyTask()
    state = { ...state, verifyResult: result, loading: false }
    if (result.passed) {
      state = { ...state, message: '' }
      // Auto-advance after brief delay
    }
    notify()
  } catch (e: any) {
    state = { ...state, loading: false, message: e.message }
    notify()
  }
}

export async function startSpecificTask(taskId: string) {
  // Find the task from levels
  const task = state.levels
    .flatMap(l => l.tasks)
    .find(t => t.id === taskId)
  if (!task) return

  state = { ...state, loading: true, verifyResult: null, message: '' }
  notify()
  try {
    const res = await api.nextTask()
    if (res.done) {
      state = { ...state, loading: false, task: null, message: res.message || 'All topics mastered!' }
    } else {
      state = { ...state, task: res.task, loading: false }
      window.dispatchEvent(new CustomEvent('task-started'))
    }
    notify()
  } catch (e: any) {
    state = { ...state, loading: false, message: e.message }
    notify()
  }
}

export function clearResult() {
  state = { ...state, verifyResult: null, task: null }
  notify()
}

export async function resetProgress() {
  state = { ...state, loading: true, message: '' }
  notify()
  try {
    await api.reset()
    state = { ...state, loading: false, task: null, verifyResult: null, message: '' }
    window.dispatchEvent(new CustomEvent('task-started'))
    notify()
  } catch (e: any) {
    state = { ...state, loading: false, message: e.message }
    notify()
  }
}
