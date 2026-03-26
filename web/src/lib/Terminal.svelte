<script lang="ts">
  import { onMount } from 'svelte'
  import { Terminal } from '@xterm/xterm'
  import { FitAddon } from '@xterm/addon-fit'
  import '@xterm/xterm/css/xterm.css'

  let termEl: HTMLDivElement
  let term: Terminal
  let fitAddon: FitAddon
  let ws: WebSocket | null = null
  let inputBuffer = ''
  let cursorPos = 0
  let history: string[] = []
  let historyIdx = -1
  let currentPrompt = '$ '

  function connect() {
    const proto = location.protocol === 'https:' ? 'wss:' : 'ws:'
    ws = new WebSocket(`${proto}//${location.host}/ws`)

    ws.onmessage = (e) => {
      const msg = JSON.parse(e.data)
      switch (msg.type) {
        case 'output':
        case 'stderr':
          term.write(msg.data.replace(/\n/g, '\r\n'))
          break
        case 'error':
          term.write(`\x1b[31m${msg.data}\x1b[0m\r\n`)
          break
        case 'prompt':
          currentPrompt = msg.data
          term.write(`\x1b[32m${currentPrompt}\x1b[0m`)
          break
        case 'prompt-replace':
          currentPrompt = msg.data
          term.write(`\r\x1b[2K\x1b[32m${currentPrompt}\x1b[0m${inputBuffer}`)
          // Move cursor back if not at end
          if (cursorPos < inputBuffer.length) {
            term.write(`\x1b[${inputBuffer.length - cursorPos}D`)
          }
          break
      }
    }

    ws.onclose = () => {
      term.write('\r\n\x1b[33m[disconnected]\x1b[0m\r\n')
      setTimeout(connect, 2000)
    }

    ws.onerror = () => {
      ws?.close()
    }
  }

  function sendCommand(cmd: string) {
    if (ws && ws.readyState === WebSocket.OPEN) {
      ws.send(JSON.stringify({ type: 'cmd', data: cmd }))
    }
  }

  function syncPrompt() {
    if (ws && ws.readyState === WebSocket.OPEN) {
      ws.send(JSON.stringify({ type: 'sync', data: '' }))
    }
  }

  function onTaskStarted() {
    // Clear terminal and reset input for new task
    term.clear()
    inputBuffer = ''
    cursorPos = 0
    syncPrompt()
  }

  // Redraw the entire input line (prompt + buffer + position cursor)
  function redrawLine() {
    term.write(`\r\x1b[2K\x1b[32m${currentPrompt}\x1b[0m${inputBuffer}`)
    if (cursorPos < inputBuffer.length) {
      term.write(`\x1b[${inputBuffer.length - cursorPos}D`)
    }
  }

  onMount(() => {
    term = new Terminal({
      fontFamily: '"Cascadia Code", "Fira Code", "JetBrains Mono", Menlo, monospace',
      fontSize: 14,
      theme: {
        background: '#0d1117',
        foreground: '#c9d1d9',
        cursor: '#58a6ff',
        cursorAccent: '#0d1117',
        selectionBackground: '#264f78',
        black: '#484f58',
        red: '#ff7b72',
        green: '#3fb950',
        yellow: '#d29922',
        blue: '#58a6ff',
        magenta: '#bc8cff',
        cyan: '#39d353',
        white: '#b1bac4',
        brightBlack: '#6e7681',
        brightRed: '#ffa198',
        brightGreen: '#56d364',
        brightYellow: '#e3b341',
        brightBlue: '#79c0ff',
        brightMagenta: '#d2a8ff',
        brightCyan: '#56d364',
        brightWhite: '#f0f6fc',
      },
      cursorBlink: true,
      cursorStyle: 'bar',
      allowProposedApi: true,
    })

    fitAddon = new FitAddon()
    term.loadAddon(fitAddon)
    term.open(termEl)
    fitAddon.fit()

    term.write('\x1b[36m  Git Guider\x1b[0m\r\n')
    term.write('\x1b[90m  Type git commands to complete tasks.\x1b[0m\r\n\r\n')

    term.onKey(({ key, domEvent }) => {
      const code = domEvent.keyCode

      if (code === 13) { // Enter
        term.write('\r\n')
        const cmd = inputBuffer.trim()
        inputBuffer = ''
        cursorPos = 0
        if (cmd) {
          history.push(cmd)
          historyIdx = history.length
          sendCommand(cmd)
        } else {
          term.write(`\x1b[32m${currentPrompt}\x1b[0m`)
        }
      } else if (code === 8) { // Backspace
        if (cursorPos > 0) {
          inputBuffer = inputBuffer.slice(0, cursorPos - 1) + inputBuffer.slice(cursorPos)
          cursorPos--
          redrawLine()
        }
      } else if (code === 46) { // Delete
        if (cursorPos < inputBuffer.length) {
          inputBuffer = inputBuffer.slice(0, cursorPos) + inputBuffer.slice(cursorPos + 1)
          redrawLine()
        }
      } else if (code === 37) { // Left arrow
        if (cursorPos > 0) {
          cursorPos--
          term.write('\x1b[D')
        }
      } else if (code === 39) { // Right arrow
        if (cursorPos < inputBuffer.length) {
          cursorPos++
          term.write('\x1b[C')
        }
      } else if (code === 36) { // Home
        if (cursorPos > 0) {
          term.write(`\x1b[${cursorPos}D`)
          cursorPos = 0
        }
      } else if (code === 35) { // End
        if (cursorPos < inputBuffer.length) {
          term.write(`\x1b[${inputBuffer.length - cursorPos}C`)
          cursorPos = inputBuffer.length
        }
      } else if (code === 38) { // Up arrow
        if (historyIdx > 0) {
          historyIdx--
          inputBuffer = history[historyIdx]
          cursorPos = inputBuffer.length
          redrawLine()
        }
      } else if (code === 40) { // Down arrow
        if (historyIdx < history.length - 1) {
          historyIdx++
          inputBuffer = history[historyIdx]
        } else {
          historyIdx = history.length
          inputBuffer = ''
        }
        cursorPos = inputBuffer.length
        redrawLine()
      } else if (code === 76 && domEvent.ctrlKey) { // Ctrl+L
        term.clear()
        inputBuffer = ''
        cursorPos = 0
        term.write(`\x1b[32m${currentPrompt}\x1b[0m`)
      } else if (code === 65 && domEvent.ctrlKey) { // Ctrl+A → Home
        if (cursorPos > 0) {
          term.write(`\x1b[${cursorPos}D`)
          cursorPos = 0
        }
      } else if (code === 69 && domEvent.ctrlKey) { // Ctrl+E → End
        if (cursorPos < inputBuffer.length) {
          term.write(`\x1b[${inputBuffer.length - cursorPos}C`)
          cursorPos = inputBuffer.length
        }
      } else if (code === 85 && domEvent.ctrlKey) { // Ctrl+U → clear line
        inputBuffer = ''
        cursorPos = 0
        redrawLine()
      } else if (domEvent.ctrlKey || domEvent.metaKey) {
        // ignore other ctrl/meta
      } else if (key.length === 1) {
        inputBuffer = inputBuffer.slice(0, cursorPos) + key + inputBuffer.slice(cursorPos)
        cursorPos++
        if (cursorPos === inputBuffer.length) {
          // Appending at end — just write the char
          term.write(key)
        } else {
          // Inserting in middle — redraw from cursor
          redrawLine()
        }
      }
    })

    // Handle paste
    term.onData((data) => {
      if (data.length > 1 && !data.startsWith('\x1b')) {
        inputBuffer = inputBuffer.slice(0, cursorPos) + data + inputBuffer.slice(cursorPos)
        cursorPos += data.length
        redrawLine()
      }
    })

    const ro = new ResizeObserver(() => fitAddon.fit())
    ro.observe(termEl)

    function onSessionReady() {
      if (!ws) connect()
    }
    window.addEventListener('session-ready', onSessionReady)
    window.addEventListener('task-started', onTaskStarted)

    return () => {
      window.removeEventListener('session-ready', onSessionReady)
      window.removeEventListener('task-started', onTaskStarted)
      ro.disconnect()
      ws?.close()
      term.dispose()
    }
  })
</script>

<div class="terminal-wrapper" bind:this={termEl}></div>

<style>
  .terminal-wrapper {
    width: 100%;
    height: 100%;
    border-radius: 8px;
    overflow: hidden;
  }
  .terminal-wrapper :global(.xterm) {
    padding: 12px;
    height: 100%;
  }
</style>
