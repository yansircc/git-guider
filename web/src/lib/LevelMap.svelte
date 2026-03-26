<script lang="ts">
  import { subscribe, getState } from '../stores/game'

  let state = $state(getState())
  const unsub = subscribe(s => { state = s })

  const layerNames: Record<string, string> = {
    '1': 'L1 Save',
    '2': 'L2 Branch',
    '3': 'L3 Remote',
    '4': 'L4 Fix',
    '5': 'L5 DAG',
  }

  function groupByLayer(levels: any[]) {
    const groups: Record<string, any[]> = {}
    for (const l of levels) {
      const layer = l.key[1] // "L1.1" → "1"
      if (!groups[layer]) groups[layer] = []
      groups[layer].push(l)
    }
    return groups
  }

  let layers = $derived(groupByLayer(state.levels))
</script>

<div class="level-map">
  {#each Object.entries(layers) as [layer, topics]}
    <div class="layer">
      <div class="layer-label" title={layerNames[layer] || 'L' + layer}>
        L{layer}
      </div>
      <div class="topics">
        {#each topics as topic}
          <div
            class="topic-dot"
            class:active={state.task && topic.tasks.some((t: any) => t.id === state.task?.id)}
            title={`${topic.key}: ${topic.name}`}
          >
            <span class="topic-key">{topic.key}</span>
          </div>
        {/each}
      </div>
    </div>
  {/each}
</div>

<style>
  .level-map {
    display: flex;
    gap: 4px;
    padding: 10px 16px;
    overflow-x: auto;
    align-items: center;
  }

  .layer {
    display: flex;
    align-items: center;
    gap: 4px;
  }

  .layer-label {
    color: #8b949e;
    font-size: 0.7rem;
    font-weight: 600;
    font-family: monospace;
    min-width: 22px;
  }

  .topics {
    display: flex;
    gap: 3px;
  }

  .topic-dot {
    width: 28px;
    height: 28px;
    border-radius: 6px;
    background: #21262d;
    border: 1px solid #30363d;
    display: flex;
    align-items: center;
    justify-content: center;
    cursor: default;
    transition: all 0.15s;
  }

  .topic-dot.active {
    background: #1f6feb;
    border-color: #58a6ff;
  }

  .topic-key {
    font-size: 0.55rem;
    color: #8b949e;
    font-family: monospace;
    font-weight: 600;
  }

  .topic-dot.active .topic-key {
    color: #fff;
  }
</style>
