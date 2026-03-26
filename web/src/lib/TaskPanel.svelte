<script lang="ts">
  import { subscribe, verifyTask, startNextTask, clearResult, getState } from '../stores/game'

  let state = $state(getState())
  const unsub = subscribe(s => { state = s })

  function handleVerify() {
    verifyTask()
  }

  function handleNext() {
    clearResult()
    startNextTask()
  }

  let showHints = $state(false)
</script>

<div class="task-panel">
  {#if state.task}
    <div class="task-header">
      <span class="task-badge">{state.task.id}</span>
      <span class="difficulty">
        {#each Array(state.task.difficulty) as _}
          <span class="star">&#9733;</span>
        {/each}
      </span>
    </div>

    <p class="task-desc">{state.task.description}</p>

    {#if state.verifyResult}
      <div class="result" class:passed={state.verifyResult.passed} class:failed={!state.verifyResult.passed}>
        {#if state.verifyResult.passed}
          <div class="result-icon">&#10004;</div>
          <div class="result-text">
            <strong>Pass!</strong>
            {#if state.verifyResult.on_pass_note}
              <p class="note">{state.verifyResult.on_pass_note}</p>
            {/if}
          </div>
        {:else}
          <div class="result-icon">&#10008;</div>
          <div class="result-text">
            <strong>Not yet...</strong>
            <p class="note">Check the hints below</p>
          </div>
        {/if}
      </div>
    {/if}

    <div class="actions">
      {#if state.verifyResult?.passed}
        <button class="btn btn-primary" onclick={handleNext}>
          Next Task &rarr;
        </button>
      {:else}
        <button class="btn btn-primary" onclick={handleVerify} disabled={state.loading}>
          {state.loading ? 'Checking...' : 'Check'}
        </button>
      {/if}
    </div>

    {#if state.task.hints && state.task.hints.length > 0}
      <div class="hints-section">
        <button class="btn-link" onclick={() => showHints = !showHints}>
          {showHints ? 'Hide hints' : 'Show hints'}
        </button>
        {#if showHints || state.verifyResult && !state.verifyResult.passed}
          <ul class="hints">
            {#each state.task.hints as hint}
              <li><code>{hint}</code></li>
            {/each}
          </ul>
        {/if}
      </div>
    {/if}

  {:else}
    <div class="empty-state">
      <p>No active task</p>
      <button class="btn btn-primary" onclick={handleNext} disabled={state.loading}>
        {state.loading ? 'Loading...' : 'Start Next Task'}
      </button>
      {#if state.message}
        <p class="msg">{state.message}</p>
      {/if}
    </div>
  {/if}
</div>

<style>
  .task-panel {
    padding: 20px;
    height: 100%;
    overflow-y: auto;
    display: flex;
    flex-direction: column;
    gap: 16px;
  }

  .task-header {
    display: flex;
    align-items: center;
    gap: 10px;
  }

  .task-badge {
    background: #1f6feb;
    color: #fff;
    font-size: 0.75rem;
    font-weight: 600;
    padding: 2px 8px;
    border-radius: 12px;
    font-family: monospace;
  }

  .star {
    color: #d29922;
    font-size: 0.9rem;
  }

  .task-desc {
    color: #c9d1d9;
    line-height: 1.6;
    margin: 0;
  }

  .result {
    display: flex;
    gap: 12px;
    padding: 12px 16px;
    border-radius: 8px;
    align-items: flex-start;
  }
  .result.passed {
    background: rgba(63, 185, 80, 0.12);
    border: 1px solid rgba(63, 185, 80, 0.3);
  }
  .result.failed {
    background: rgba(255, 123, 114, 0.12);
    border: 1px solid rgba(255, 123, 114, 0.3);
  }
  .result-icon {
    font-size: 1.4rem;
    line-height: 1;
  }
  .passed .result-icon { color: #3fb950; }
  .failed .result-icon { color: #ff7b72; }
  .result-text strong {
    color: #c9d1d9;
  }
  .note {
    color: #8b949e;
    font-size: 0.85rem;
    margin: 4px 0 0;
    line-height: 1.5;
  }

  .actions {
    display: flex;
    gap: 8px;
  }

  .btn {
    padding: 8px 20px;
    border-radius: 6px;
    border: 1px solid transparent;
    font-size: 0.9rem;
    font-weight: 500;
    cursor: pointer;
    transition: all 0.15s;
  }
  .btn:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }
  .btn-primary {
    background: #238636;
    color: #fff;
    border-color: rgba(240, 246, 252, 0.1);
  }
  .btn-primary:hover:not(:disabled) {
    background: #2ea043;
  }

  .btn-link {
    background: none;
    border: none;
    color: #58a6ff;
    cursor: pointer;
    font-size: 0.85rem;
    padding: 0;
  }
  .btn-link:hover {
    text-decoration: underline;
  }

  .hints {
    list-style: none;
    padding: 0;
    margin: 8px 0 0;
    display: flex;
    flex-direction: column;
    gap: 6px;
  }
  .hints li {
    color: #8b949e;
    font-size: 0.85rem;
  }
  .hints code {
    background: rgba(110, 118, 129, 0.15);
    padding: 2px 6px;
    border-radius: 4px;
    font-size: 0.8rem;
    color: #c9d1d9;
  }

  .empty-state {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 16px;
    height: 100%;
    color: #8b949e;
  }

  .msg {
    color: #d29922;
    font-size: 0.85rem;
  }
</style>
