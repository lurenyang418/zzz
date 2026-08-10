<script>
  import { createEventDispatcher } from 'svelte';
  import { t } from '../i18n';

  export let ignorePatterns = [];
  export let includeOnly = [];
  const dispatch = createEventDispatcher();

  let ignoreInput = ignorePatterns.join(', ');
  let includeInput = includeOnly.join(', ');

  function updateIgnore() {
    ignorePatterns = ignoreInput.split(',').map((value) => value.trim()).filter(Boolean);
    dispatch('change');
  }

  function updateInclude() {
    includeOnly = includeInput.split(',').map((value) => value.trim()).filter(Boolean);
    dispatch('change');
  }
</script>

<details class="mt-6 border-t border-slate-100 pt-5 dark:border-slate-800">
  <summary class="cursor-pointer text-sm font-semibold text-slate-600 dark:text-slate-300">{$t('filter.summary')}</summary>
  <div class="mt-4 space-y-4">
    <div>
      <label for="ignore-patterns" class="mb-1 block text-xs font-semibold uppercase tracking-wide text-slate-500 dark:text-slate-400">{$t('filter.ignore')}</label>
      <input id="ignore-patterns" bind:value={ignoreInput} on:change={updateIgnore} class="w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm outline-none focus:border-indigo-500 dark:border-slate-700 dark:bg-slate-950 dark:text-slate-100 dark:placeholder:text-slate-600 dark:focus:ring-indigo-950" placeholder="node_modules/, *.log, .env" />
      <p class="mt-1 text-xs text-slate-400 dark:text-slate-500">{$t('filter.globHelp')}</p>
    </div>
    <div>
      <label for="include-patterns" class="mb-1 block text-xs font-semibold uppercase tracking-wide text-slate-500 dark:text-slate-400">{$t('filter.include')}</label>
      <input id="include-patterns" bind:value={includeInput} on:change={updateInclude} class="w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm outline-none focus:border-indigo-500 dark:border-slate-700 dark:bg-slate-950 dark:text-slate-100 dark:placeholder:text-slate-600 dark:focus:ring-indigo-950" placeholder="src/**, *.go, package.json" />
      <p class="mt-1 text-xs text-slate-400 dark:text-slate-500">{$t('filter.includeHelp')}</p>
    </div>
  </div>
</details>
