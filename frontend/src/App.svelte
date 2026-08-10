<script>
  import { onMount } from 'svelte';
  import InputForm from './components/InputForm.svelte';
  import FilterConfig from './components/FilterConfig.svelte';
  import ProgressBar from './components/ProgressBar.svelte';
  import TreePicker from './components/TreePicker.svelte';
  import OutputConfig from './components/OutputConfig.svelte';
  import { locale, t, theme, toggleLocale, toggleTheme } from './i18n';
  import { downloadStatus, errorMessage, fileCount, progress } from './stores/download';
  import { downloadFiles, exportToHost, getApiError, isGitHubAuthError, loadCapabilities, loadTree } from './api/download';

  let url = '';
  let githubToken = '';
  let accessToken = '';
  let tokenStorageReady = false;
  let ignorePatterns = ['node_modules/', '.git/', 'dist/', 'build/', '.DS_Store', '.env'];
  let includeOnly = [];
  let isLoading = false;
  let isLoadingTree = false;
  let treeLoaded = false;
  let treeEntries = [];
  let selectedPaths = [];
  let selectionStats = { fileCount: 0, size: 0 };
  let urlHistory = [];
  let showTokenHelp = false;
  let outputMode = 'browser';
  let hostPath = '';
  let hostFormat = 'folder';
  let hostExportEnabled = false;
  let serviceAuthRequired = false;
  let allSelected = false;
  let successMessage = '';

  onMount(() => {
    githubToken = window.localStorage.getItem('zzz.githubToken') || '';
    accessToken = window.localStorage.getItem('zzz.accessToken') || '';
    try {
      urlHistory = JSON.parse(window.localStorage.getItem('zzz.urlHistory') || '[]');
    } catch {
      urlHistory = [];
    }
    loadCapabilities().then((capabilities) => {
      hostExportEnabled = capabilities.host_export;
      serviceAuthRequired = capabilities.auth_required;
    }).catch(() => {
      hostExportEnabled = false;
    });
    tokenStorageReady = true;
  });

  $: if (tokenStorageReady) {
    if (githubToken.trim()) window.localStorage.setItem('zzz.githubToken', githubToken.trim());
    else window.localStorage.removeItem('zzz.githubToken');
    if (accessToken.trim()) window.localStorage.setItem('zzz.accessToken', accessToken.trim());
    else window.localStorage.removeItem('zzz.accessToken');
  }

  function recordUrl() {
    const value = url.trim();
    if (!value) return;
    urlHistory = [value, ...urlHistory.filter((item) => item !== value)].slice(0, 8);
    window.localStorage.setItem('zzz.urlHistory', JSON.stringify(urlHistory));
  }

  function useHistory(value) {
    url = value;
    invalidateTree();
    errorMessage.set('');
  }

  function invalidateTree() {
    treeLoaded = false;
    treeEntries = [];
    selectedPaths = [];
    allSelected = false;
    showTokenHelp = false;
    successMessage = '';
  }

  async function handleLoadTree() {
    if (!url.trim() || isLoadingTree) return;
    isLoadingTree = true;
    showTokenHelp = false;
    errorMessage.set('');
    successMessage = '';
    recordUrl();
    try {
      treeEntries = await loadTree(url.trim(), ignorePatterns, includeOnly, githubToken, accessToken);
      selectedPaths = [];
      treeLoaded = true;
      downloadStatus.set('idle');
      if (!treeEntries.length) errorMessage.set($t('errors.noTree'));
    } catch (error) {
      treeLoaded = false;
      showTokenHelp = isGitHubAuthError(error);
      errorMessage.set(getApiError(error, $locale));
      downloadStatus.set('error');
    } finally {
      isLoadingTree = false;
    }
  }

  async function handleDownload() {
    if (!url.trim() || isLoading) return;
    if (treeLoaded && selectedPaths.length === 0 && !allSelected) {
      errorMessage.set($t('errors.selectRequired'));
      downloadStatus.set('error');
      return;
    }
    isLoading = true;
    showTokenHelp = false;
    successMessage = '';
    recordUrl();
    downloadStatus.set('collecting');
    progress.set(10);
    fileCount.set(selectionStats.fileCount);
    errorMessage.set('');

    try {
      if (outputMode === 'host') {
        const result = await exportToHost({
          url: url.trim(),
          ignore: ignorePatterns,
          include: includeOnly,
          select: treeLoaded ? selectedPaths : undefined,
          selectAll: treeLoaded ? allSelected : undefined,
          token: githubToken,
          accessToken,
          destination: hostPath,
          format: hostFormat,
        });
        progress.set(100);
        downloadStatus.set('done');
        successMessage = $t('download.hostSaved', { path: result.path });
      } else {
        const result = await downloadFiles({
          url: url.trim(),
          ignore: ignorePatterns,
          include: includeOnly,
          select: treeLoaded ? selectedPaths : undefined,
          selectAll: treeLoaded ? allSelected : undefined,
          token: githubToken,
          accessToken,
        });
        progress.set(100);
        downloadStatus.set('done');
        const blob = new Blob([result.data], { type: 'application/zip' });
        const link = document.createElement('a');
        const objectUrl = URL.createObjectURL(blob);
        link.href = objectUrl;
        link.download = result.filename || 'zzz-download.zip';
        link.click();
        setTimeout(() => URL.revokeObjectURL(objectUrl), 1000);
      }
    } catch (error) {
      showTokenHelp = isGitHubAuthError(error);
      errorMessage.set(getApiError(error, $locale));
      downloadStatus.set('error');
    } finally {
      isLoading = false;
    }
  }
</script>

<main class="min-h-screen bg-slate-50 px-4 py-6 text-slate-900 transition-colors dark:bg-slate-950 dark:text-slate-100 sm:py-10">
  <div class="mx-auto max-w-3xl">
    <header class="mb-4">
      <div class="flex items-start justify-between gap-4">
        <div>
          <p class="mb-1 text-xs font-semibold uppercase tracking-[0.25em] text-indigo-600">{$t('app.eyebrow')}</p>
          <h1 class="text-4xl font-black tracking-tight text-slate-950 dark:text-white">zzz</h1>
        </div>
        <div class="flex shrink-0 gap-2">
        <button
          type="button"
          on:click={toggleLocale}
          class="rounded-lg border border-slate-200 bg-white px-2.5 py-1.5 text-xs font-semibold text-slate-600 shadow-sm transition hover:border-indigo-300 hover:text-indigo-700 dark:border-slate-700 dark:bg-slate-900 dark:text-slate-300 dark:hover:border-indigo-500 dark:hover:text-indigo-300"
          aria-label={$t('app.language')}
        >{$t('app.language')}</button>
        <button
          type="button"
          on:click={toggleTheme}
          class="rounded-lg border border-slate-200 bg-white px-2.5 py-1.5 text-sm shadow-sm transition hover:border-indigo-300 dark:border-slate-700 dark:bg-slate-900 dark:text-slate-200 dark:hover:border-indigo-500"
          aria-label={$theme === 'dark' ? $t('app.themeLight') : $t('app.themeDark')}
          title={$theme === 'dark' ? $t('app.themeLight') : $t('app.themeDark')}
        >{$theme === 'dark' ? '☀︎' : '☾'}</button>
        </div>
      </div>
      <p class="mt-1 text-sm text-slate-500 dark:text-slate-400">{$t('app.subtitle')}</p>
    </header>

    <section class="rounded-3xl border border-slate-200 bg-white p-5 shadow-xl shadow-slate-200/50 transition-colors dark:border-slate-800 dark:bg-slate-900 dark:shadow-slate-950/50 sm:p-7">
      <InputForm
        bind:url
        bind:githubToken
        bind:accessToken
        {serviceAuthRequired}
        loading={isLoading}
        loadingTree={isLoadingTree}
        on:input={invalidateTree}
        on:submit={handleDownload}
        on:loadTree={handleLoadTree}
      />
      {#if urlHistory.length}
        <details class="mt-4 border-t border-slate-100 pt-4 dark:border-slate-800">
          <summary class="cursor-pointer text-sm font-semibold text-slate-600 dark:text-slate-300">{$t('history.summary')}</summary>
          <div class="mt-2 space-y-1">
            {#each urlHistory as recentUrl}
              <button
                type="button"
                class="block w-full truncate rounded-lg px-2 py-1.5 text-left text-xs text-slate-500 hover:bg-slate-50 hover:text-indigo-700 dark:text-slate-400 dark:hover:bg-slate-800 dark:hover:text-indigo-300"
                title={recentUrl}
                on:click={() => useHistory(recentUrl)}
              >{recentUrl}</button>
            {/each}
            <button
              type="button"
              class="mt-1 text-xs font-semibold text-slate-400 hover:text-rose-600 dark:text-slate-500 dark:hover:text-rose-400"
              on:click={() => { urlHistory = []; window.localStorage.removeItem('zzz.urlHistory'); }}
            >{$t('history.clear')}</button>
          </div>
        </details>
      {/if}
      <FilterConfig bind:ignorePatterns bind:includeOnly on:change={invalidateTree} />
      {#if treeLoaded}
        <TreePicker entries={treeEntries} bind:selectedPaths bind:selectionStats bind:allSelected />
      {/if}
      <OutputConfig bind:mode={outputMode} bind:hostPath bind:hostFormat {hostExportEnabled} />
      <div class="mt-6 border-t border-slate-100 pt-5 dark:border-slate-800">
        <button
          type="button"
          disabled={!url.trim() || isLoading}
          on:click={handleDownload}
          class="w-full rounded-xl bg-indigo-600 px-4 py-3 font-semibold text-white transition hover:bg-indigo-700 disabled:cursor-not-allowed disabled:opacity-50"
        >
          {isLoading ? $t('download.loading') : outputMode === 'host' ? $t('output.host') : treeLoaded ? $t('download.selected') : $t('download.direct')}
        </button>
        <p class="mt-2 text-center text-xs text-slate-400 dark:text-slate-500">
          {outputMode === 'host' ? $t('output.hostPathHelp') : treeLoaded ? $t('download.selectedHelp') : $t('download.directHelp')}
        </p>
      </div>
    </section>

    <ProgressBar
      status={$downloadStatus}
      progress={$progress}
      fileCount={$fileCount}
      error={$errorMessage}
      success={successMessage}
      {showTokenHelp}
    />
  </div>
</main>
