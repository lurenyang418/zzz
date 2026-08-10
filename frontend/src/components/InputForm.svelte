<script>
  import { createEventDispatcher } from 'svelte';
  import { t } from '../i18n';

  export let url = '';
  export let githubToken = '';
  export let accessToken = '';
  export let serviceAuthRequired = false;
  export let loading = false;
  export let loadingTree = false;
  let showToken = false;
  let showAccessToken = false;
  const dispatch = createEventDispatcher();

  function handleSubmit(event) {
    event.preventDefault();
    dispatch('submit');
  }
</script>

<form on:submit={handleSubmit} class="space-y-5">
  <div>
    <label for="github-url" class="mb-2 block text-sm font-semibold text-slate-700 dark:text-slate-200">{$t('input.urlLabel')}</label>
    <input
      id="github-url"
      type="url"
      bind:value={url}
      placeholder="https://github.com/owner/repo/tree/main/path"
      autocomplete="url"
      required
      disabled={loading}
      on:input={() => dispatch('input')}
      class="w-full rounded-xl border border-slate-300 bg-white px-4 py-3 outline-none transition focus:border-indigo-500 focus:ring-4 focus:ring-indigo-100 disabled:bg-slate-100 dark:border-slate-700 dark:bg-slate-950 dark:text-slate-100 dark:placeholder:text-slate-600 dark:focus:ring-indigo-950 dark:disabled:bg-slate-800"
    />
    <p class="mt-2 text-xs text-slate-400 dark:text-slate-500">{$t('input.urlHelp')}</p>
  </div>

  <details open={Boolean(githubToken)} class="border-t border-slate-100 pt-4 dark:border-slate-800">
    <summary class="cursor-pointer text-sm font-semibold text-slate-600 dark:text-slate-300">{$t('input.tokenSummary')}</summary>
    <div class="mt-3">
      <div class="flex gap-2">
        <input
          type={showToken ? 'text' : 'password'}
          bind:value={githubToken}
          placeholder={$t('input.tokenPlaceholder')}
          autocomplete="off"
          spellcheck="false"
          disabled={loading}
          class="min-w-0 flex-1 rounded-xl border border-slate-300 bg-white px-4 py-3 font-mono text-sm outline-none transition focus:border-indigo-500 focus:ring-4 focus:ring-indigo-100 disabled:bg-slate-100 dark:border-slate-700 dark:bg-slate-950 dark:text-slate-100 dark:placeholder:text-slate-600 dark:focus:ring-indigo-950 dark:disabled:bg-slate-800"
        />
        <button
          type="button"
          on:click={() => (showToken = !showToken)}
          class="rounded-xl border border-slate-200 px-3 text-xs font-semibold text-slate-500 hover:bg-slate-50 dark:border-slate-700 dark:text-slate-400 dark:hover:bg-slate-800"
        >{showToken ? $t('input.hide') : $t('input.show')}</button>
        {#if githubToken}
          <button
            type="button"
            on:click={() => (githubToken = '')}
            class="rounded-xl border border-slate-200 px-3 text-xs font-semibold text-slate-500 hover:bg-slate-50 dark:border-slate-700 dark:text-slate-400 dark:hover:bg-slate-800"
          >{$t('input.clear')}</button>
        {/if}
      </div>
      <p class="mt-2 text-xs leading-5 text-slate-400 dark:text-slate-500">
        {$t('input.tokenHelp')}
        <a
          href="https://github.com/settings/personal-access-tokens/new"
          target="_blank"
          rel="noreferrer"
          class="font-semibold text-indigo-600 underline decoration-indigo-200 underline-offset-2 hover:text-indigo-800"
        >{$t('input.applyToken')}</a>。
      </p>
    </div>
  </details>

  {#if serviceAuthRequired}
    <details open={Boolean(accessToken)} class="border-t border-slate-100 pt-4 dark:border-slate-800">
      <summary class="cursor-pointer text-sm font-semibold text-slate-600 dark:text-slate-300">{$t('input.accessTokenSummary')}</summary>
      <div class="mt-3 flex gap-2">
        <input
          type={showAccessToken ? 'text' : 'password'}
          bind:value={accessToken}
          placeholder={$t('input.accessTokenPlaceholder')}
          autocomplete="off"
          spellcheck="false"
          disabled={loading}
          class="min-w-0 flex-1 rounded-xl border border-slate-300 bg-white px-4 py-3 font-mono text-sm outline-none transition focus:border-indigo-500 focus:ring-4 focus:ring-indigo-100 disabled:bg-slate-100 dark:border-slate-700 dark:bg-slate-950 dark:text-slate-100 dark:placeholder:text-slate-600 dark:focus:ring-indigo-950 dark:disabled:bg-slate-800"
        />
        <button type="button" on:click={() => (showAccessToken = !showAccessToken)} class="rounded-xl border border-slate-200 px-3 text-xs font-semibold text-slate-500 hover:bg-slate-50 dark:border-slate-700 dark:text-slate-400 dark:hover:bg-slate-800">
          {showAccessToken ? $t('input.hide') : $t('input.show')}
        </button>
        {#if accessToken}
          <button type="button" on:click={() => (accessToken = '')} class="rounded-xl border border-slate-200 px-3 text-xs font-semibold text-slate-500 hover:bg-slate-50 dark:border-slate-700 dark:text-slate-400 dark:hover:bg-slate-800">{$t('input.clear')}</button>
        {/if}
      </div>
      <p class="mt-2 text-xs leading-5 text-slate-400 dark:text-slate-500">{$t('input.accessTokenHelp')}</p>
    </details>
  {/if}

  <button
    type="button"
    disabled={!url.trim() || loading || loadingTree}
    on:click={() => dispatch('loadTree')}
    class="w-full rounded-xl border border-indigo-200 bg-indigo-50 px-4 py-3 font-semibold text-indigo-700 transition hover:bg-indigo-100 disabled:cursor-not-allowed disabled:opacity-50 dark:border-indigo-900 dark:bg-indigo-950/50 dark:text-indigo-300 dark:hover:bg-indigo-950"
  >
    {loadingTree ? $t('input.loadingTree') : $t('input.browse')}
  </button>
</form>
