<script lang="ts">
  import { onMount } from 'svelte'
  import Terminal from './lib/Terminal.svelte'
  import TaskPanel from './lib/TaskPanel.svelte'
  import LevelMap from './lib/LevelMap.svelte'
  import { initSession, resetProgress } from './stores/game'

  onMount(() => {
    initSession()
  })
</script>

<div class="app">
  <header class="topbar">
    <div class="brand">
      <span class="logo">&#x2387;</span>
      <span class="title">Git Guider</span>
    </div>
    <LevelMap />
    <button class="reset-btn" onclick={() => { if (confirm('Reset all progress?')) resetProgress() }}>Reset</button>
  </header>
  <main class="content">
    <div class="terminal-pane">
      <Terminal />
    </div>
    <aside class="task-pane">
      <TaskPanel />
    </aside>
  </main>
</div>

<style>
  .app {
    display: flex;
    flex-direction: column;
    height: 100vh;
    background: #0d1117;
    color: #c9d1d9;
  }

  .topbar {
    display: flex;
    align-items: center;
    gap: 16px;
    padding: 0 16px;
    height: 48px;
    background: #161b22;
    border-bottom: 1px solid #21262d;
    flex-shrink: 0;
  }

  .brand {
    display: flex;
    align-items: center;
    gap: 8px;
    flex-shrink: 0;
  }

  .logo {
    font-size: 1.4rem;
    color: #58a6ff;
  }

  .title {
    font-size: 0.95rem;
    font-weight: 600;
    color: #f0f6fc;
  }

  .content {
    display: flex;
    flex: 1;
    min-height: 0;
  }

  .terminal-pane {
    flex: 1;
    min-width: 0;
    padding: 8px;
    padding-right: 0;
  }

  .task-pane {
    width: 340px;
    flex-shrink: 0;
    border-left: 1px solid #21262d;
    background: #161b22;
  }

  .reset-btn {
    margin-left: auto;
    padding: 4px 12px;
    border-radius: 6px;
    border: 1px solid #30363d;
    background: transparent;
    color: #8b949e;
    font-size: 0.75rem;
    cursor: pointer;
    transition: all 0.15s;
  }
  .reset-btn:hover {
    color: #ff7b72;
    border-color: #ff7b72;
  }
</style>
