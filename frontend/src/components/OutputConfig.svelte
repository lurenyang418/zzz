<script>
  import { t } from '../i18n';

  export let mode = 'browser';
  export let hostPath = '';
  export let hostFormat = 'folder';
  export let hostExportEnabled = false;
</script>

<section class="mt-6 rounded-2xl border border-slate-200 bg-slate-50/80 p-4 dark:border-slate-800 dark:bg-slate-950/60">
  <div class="mb-3">
    <h2 class="text-sm font-bold text-slate-700 dark:text-slate-200">{$t('output.summary')}</h2>
  </div>

  <div class="grid grid-cols-2 gap-2 rounded-xl bg-slate-200/70 p-1 dark:bg-slate-800">
    <button
      type="button"
      class={`rounded-lg px-3 py-2 text-sm font-semibold transition ${mode === 'browser' ? 'bg-white text-indigo-700 shadow-sm dark:bg-slate-700 dark:text-indigo-300' : 'text-slate-500 dark:text-slate-400'}`}
      on:click={() => (mode = 'browser')}
    >{$t('output.browser')}</button>
    <button
      type="button"
      disabled={!hostExportEnabled}
      class={`rounded-lg px-3 py-2 text-sm font-semibold transition disabled:cursor-not-allowed disabled:opacity-50 ${mode === 'host' ? 'bg-white text-indigo-700 shadow-sm dark:bg-slate-700 dark:text-indigo-300' : 'text-slate-500 dark:text-slate-400'}`}
      on:click={() => hostExportEnabled && (mode = 'host')}
    >{$t('output.host')}</button>
  </div>

  {#if !hostExportEnabled}
    <p class="mt-3 text-xs text-slate-400 dark:text-slate-500">{$t('output.hostUnavailable')}</p>
  {:else if mode === 'host'}
    <div class="mt-4 space-y-3">
      <div>
        <label for="host-path" class="mb-1.5 block text-xs font-semibold text-slate-600 dark:text-slate-300">{$t('output.hostPathLabel')}</label>
        <input
          id="host-path"
          bind:value={hostPath}
          placeholder={$t('output.hostPathPlaceholder')}
          autocomplete="off"
          spellcheck="false"
          class="w-full rounded-xl border border-slate-300 bg-white px-3 py-2.5 text-sm outline-none transition focus:border-indigo-500 focus:ring-4 focus:ring-indigo-100 dark:border-slate-700 dark:bg-slate-900 dark:text-slate-100 dark:placeholder:text-slate-600 dark:focus:ring-indigo-950"
        />
        <p class="mt-1.5 text-xs leading-5 text-slate-400 dark:text-slate-500">{$t('output.hostPathHelp')}</p>
      </div>
      <div>
        <p class="mb-1.5 text-xs font-semibold text-slate-600 dark:text-slate-300">{$t('output.format')}</p>
        <div class="flex gap-2">
          <button
            type="button"
            class={`rounded-lg border px-3 py-2 text-xs font-semibold transition ${hostFormat === 'folder' ? 'border-indigo-500 bg-indigo-50 text-indigo-700 dark:bg-indigo-950 dark:text-indigo-300' : 'border-slate-200 text-slate-500 dark:border-slate-700 dark:text-slate-400'}`}
            on:click={() => (hostFormat = 'folder')}
          >{$t('output.folder')}</button>
          <button
            type="button"
            class={`rounded-lg border px-3 py-2 text-xs font-semibold transition ${hostFormat === 'zip' ? 'border-indigo-500 bg-indigo-50 text-indigo-700 dark:bg-indigo-950 dark:text-indigo-300' : 'border-slate-200 text-slate-500 dark:border-slate-700 dark:text-slate-400'}`}
            on:click={() => (hostFormat = 'zip')}
          >{$t('output.zip')}</button>
        </div>
        <p class="mt-1.5 text-xs leading-5 text-slate-400 dark:text-slate-500">
          {hostFormat === 'folder' ? $t('output.folderHelp') : $t('output.zipHelp')}
        </p>
      </div>
    </div>
  {/if}
</section>
