<script>
  import { t } from '../i18n';

  export let status = 'idle';
  export let progress = 0;
  export let fileCount = 0;
  export let error = '';
  export let success = '';
  export let showTokenHelp = false;
</script>

  {#if status !== 'idle'}
  <div class="mt-5 rounded-2xl border border-slate-200 bg-white p-4 shadow-sm dark:border-slate-800 dark:bg-slate-900">
    <div class="flex items-center justify-between gap-4">
      <p class="text-sm font-semibold text-slate-700 dark:text-slate-200">
        {#if status === 'collecting'}{$t('status.collecting')}{:else if status === 'done'}{$t('status.done')}{:else}{$t('status.error')}{/if}
      </p>
      {#if fileCount > 0}<span class="text-xs text-slate-400 dark:text-slate-500">{$t('status.files', { count: fileCount })}</span>{/if}
    </div>
    {#if status !== 'error' && status !== 'done'}
      <div class="mt-3 h-2 overflow-hidden rounded-full bg-slate-100"><div class="h-full rounded-full bg-indigo-600 transition-all" style={`width: ${progress}%`}></div></div>
    {/if}
    {#if error}<p class="mt-3 whitespace-pre-wrap text-sm text-rose-600">{error}</p>{/if}
    {#if success}<p class="mt-3 whitespace-pre-wrap text-sm text-emerald-600 dark:text-emerald-400">{success}</p>{/if}
    {#if showTokenHelp}
      <p class="mt-3 text-xs leading-5 text-slate-500 dark:text-slate-400">
        {$t('token.errorHelp')}
        <a
          href="https://github.com/settings/personal-access-tokens/new"
          target="_blank"
          rel="noreferrer"
          class="font-semibold text-indigo-600 underline decoration-indigo-200 underline-offset-2 hover:text-indigo-800"
        >{$t('input.applyToken')}</a>{$t('token.errorHelpAfter')}
      </p>
    {/if}
  </div>
{/if}
