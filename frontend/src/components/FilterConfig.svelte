<script>
  import { createEventDispatcher } from 'svelte';
  import { t } from '../i18n';

  export let ignorePatterns = [];
  export let includeOnly = [];
  const dispatch = createEventDispatcher();

  let ignoreDraft = '';
  let includeDraft = '';
  let activeTab = 'ignore';

  function addPatterns(kind, value) {
    const patterns = value.split(',').map((item) => item.trim()).filter(Boolean);
    if (!patterns.length) return;
    const current = kind === 'ignore' ? ignorePatterns : includeOnly;
    const merged = [...current, ...patterns.filter((pattern) => !current.includes(pattern))];
    if (kind === 'ignore') ignorePatterns = merged;
    else includeOnly = merged;
    dispatch('change');
  }

  function removePattern(kind, index) {
    if (kind === 'ignore') ignorePatterns = ignorePatterns.filter((_, itemIndex) => itemIndex !== index);
    else includeOnly = includeOnly.filter((_, itemIndex) => itemIndex !== index);
    dispatch('change');
  }

  function commitDraft(kind) {
    const draft = kind === 'ignore' ? ignoreDraft : includeDraft;
    addPatterns(kind, draft);
    if (kind === 'ignore') ignoreDraft = '';
    else includeDraft = '';
  }

  function updateDraft(kind, event) {
    const value = event.currentTarget.value;
    const parts = value.split(',');
    const draft = parts.pop() || '';
    if (parts.length) addPatterns(kind, parts.join(','));
    if (kind === 'ignore') ignoreDraft = draft;
    else includeDraft = draft;
  }

  function handleKeydown(kind, event) {
    const draft = kind === 'ignore' ? ignoreDraft : includeDraft;
    if (event.key === 'Enter' || event.key === ',') {
      event.preventDefault();
      commitDraft(kind);
    } else if (event.key === 'Backspace' && !draft) {
      const patterns = kind === 'ignore' ? ignorePatterns : includeOnly;
      if (patterns.length) removePattern(kind, patterns.length - 1);
    }
  }
</script>

<details class="mt-6 border-t border-slate-100 pt-5 dark:border-slate-800">
  <summary class="cursor-pointer text-sm font-semibold text-slate-600 dark:text-slate-300">{$t('filter.summary')}</summary>
  <div class="mt-4 rounded-xl border border-slate-200 bg-slate-100/70 p-1 dark:border-slate-800 dark:bg-slate-950/70" role="tablist" aria-label={$t('filter.summary')}>
    <div class="grid grid-cols-2 gap-1">
      <button
        type="button"
        role="tab"
        aria-selected={activeTab === 'ignore'}
        class={`rounded-lg px-3 py-2 text-left text-xs font-semibold transition ${activeTab === 'ignore' ? 'bg-white text-indigo-700 shadow-sm dark:bg-slate-800 dark:text-indigo-300' : 'text-slate-500 hover:text-slate-700 dark:text-slate-400 dark:hover:text-slate-200'}`}
        on:click={() => (activeTab = 'ignore')}
      >{$t('filter.ignore')}</button>
      <button
        type="button"
        role="tab"
        aria-selected={activeTab === 'include'}
        class={`rounded-lg px-3 py-2 text-left text-xs font-semibold transition ${activeTab === 'include' ? 'bg-white text-indigo-700 shadow-sm dark:bg-slate-800 dark:text-indigo-300' : 'text-slate-500 hover:text-slate-700 dark:text-slate-400 dark:hover:text-slate-200'}`}
        on:click={() => (activeTab = 'include')}
      >{$t('filter.include')}</button>
    </div>
    <div class="p-2 pb-1">
      {#if activeTab === 'ignore'}
        <label for="ignore-patterns" class="mb-1 block text-xs font-semibold uppercase tracking-wide text-slate-500 dark:text-slate-400">{$t('filter.ignore')}</label>
        <div class="flex min-h-10 flex-wrap items-center gap-1.5 rounded-lg border border-slate-300 bg-white px-2 py-1.5 transition focus-within:border-indigo-500 focus-within:ring-4 focus-within:ring-indigo-100 dark:border-slate-700 dark:bg-slate-950 dark:focus-within:ring-indigo-950">
          {#each ignorePatterns as pattern, index (pattern)}
            <span class="inline-flex items-center gap-1 rounded-md bg-indigo-50 px-2 py-1 text-xs font-medium text-indigo-700 dark:bg-indigo-950/70 dark:text-indigo-300">
              {pattern}
              <button type="button" class="rounded px-0.5 text-indigo-400 hover:bg-indigo-100 hover:text-indigo-700 dark:hover:bg-indigo-900 dark:hover:text-indigo-200" aria-label={`${$t('filter.remove')} ${pattern}`} on:click={() => removePattern('ignore', index)}>×</button>
            </span>
          {/each}
          <input id="ignore-patterns" value={ignoreDraft} on:input={(event) => updateDraft('ignore', event)} on:keydown={(event) => handleKeydown('ignore', event)} on:blur={() => commitDraft('ignore')} class="min-w-[8rem] flex-1 border-0 bg-transparent px-1 py-1 text-sm outline-none dark:text-slate-100 dark:placeholder:text-slate-600" placeholder="node_modules/, *.log, .env" />
        </div>
        <p class="mt-1 text-xs text-slate-400 dark:text-slate-500">{$t('filter.globHelp')}</p>
      {:else}
        <label for="include-patterns" class="mb-1 block text-xs font-semibold uppercase tracking-wide text-slate-500 dark:text-slate-400">{$t('filter.include')}</label>
        <div class="flex min-h-10 flex-wrap items-center gap-1.5 rounded-lg border border-slate-300 bg-white px-2 py-1.5 transition focus-within:border-indigo-500 focus-within:ring-4 focus-within:ring-indigo-100 dark:border-slate-700 dark:bg-slate-950 dark:focus-within:ring-indigo-950">
          {#each includeOnly as pattern, index (pattern)}
            <span class="inline-flex items-center gap-1 rounded-md bg-indigo-50 px-2 py-1 text-xs font-medium text-indigo-700 dark:bg-indigo-950/70 dark:text-indigo-300">
              {pattern}
              <button type="button" class="rounded px-0.5 text-indigo-400 hover:bg-indigo-100 hover:text-indigo-700 dark:hover:bg-indigo-900 dark:hover:text-indigo-200" aria-label={`${$t('filter.remove')} ${pattern}`} on:click={() => removePattern('include', index)}>×</button>
            </span>
          {/each}
          <input id="include-patterns" value={includeDraft} on:input={(event) => updateDraft('include', event)} on:keydown={(event) => handleKeydown('include', event)} on:blur={() => commitDraft('include')} class="min-w-[8rem] flex-1 border-0 bg-transparent px-1 py-1 text-sm outline-none dark:text-slate-100 dark:placeholder:text-slate-600" placeholder="src/**, *.go, package.json" />
        </div>
        <p class="mt-1 text-xs text-slate-400 dark:text-slate-500">{$t('filter.includeHelp')}</p>
      {/if}
    </div>
  </div>
</details>
