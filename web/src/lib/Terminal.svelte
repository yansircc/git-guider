<script lang="ts">
  import { onMount, onDestroy } from 'svelte'
  import { Terminal } from '@xterm/xterm'
  import { FitAddon } from '@xterm/addon-fit'
  import '@xterm/xterm/css/xterm.css'

  let termEl: HTMLDivElement
  let term: Terminal
  let fitAddon: FitAddon
  let ws: WebSocket | null = null
  let inputBuffer = ''
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
        if (cmd) {
          history.push(cmd)
          historyIdx = history.length
          sendCommand(cmd)
        } else {
          term.write(`\x1b[32m${currentPrompt}\x1b[0m`)
        }
      } else if (code === 8) { // Backspace
        if (inputBuffer.length > 0) {
          inputBuffer = inputBuffer.slice(0, -1)
          term.write('\b \b')
        }
      } else if (code === 38) { // Up arrow
        if (historyIdx > 0) {
          // Clear current input
          while (inputBuffer.length > 0) {
            term.write('\b \b')
            inputBuffer = inputBuffer.slice(0, -1)
          }
          historyIdx--
          inputBuffer = history[historyIdx]
          term.write(inputBuffer)
        }
      } else if (code === 40) { // Down arrow
        while (inputBuffer.length > 0) {
          term.write('\b \b')
          inputBuffer = inputBuffer.slice(0, -1)
        }
        if (historyIdx < history.length - 1) {
          historyIdx++
          inputBuffer = history[historyIdx]
          term.write(inputBuffer)
        } else {
          historyIdx = history.length
        }
      } else if (code === 76 && domEvent.ctrlKey) { // Ctrl+L
        term.clear()
        term.write(`\x1b[32m${currentPrompt}\x1b[0m`)
        term.write(inputBuffer)
      } else if (domEvent.ctrlKey || domEvent.metaKey) {
        // ignore other ctrl/meta
      } else if (key.length === 1) {
        inputBuffer += key
        term.write(key)
      }
    })

    // Handle paste
    term.onData((data) => {
      // onData fires for paste events with the full string
      // But onKey also fires for single chars, so only handle multi-char pastes
      if (data.length > 1 && !data.startsWith('\x1b')) {
        inputBuffer += data
        term.write(data)
      }
    })

    const ro = new ResizeObserver(() => fitAddon.fit())
    ro.observe(termEl)

    connect()

    return () => {
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
