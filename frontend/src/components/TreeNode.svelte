<script>
  import { createEventDispatcher } from 'svelte';
  import { t } from '../i18n';

  export let node;
  export let selectedPaths = [];
  export let expandedPaths = [];
  export let allSelected = false;

  const dispatch = createEventDispatcher();
  $: expanded = expandedPaths.includes(node.path);
  $: descendantFiles = collectFiles(node);
  $: checked = allSelected || (descendantFiles.length > 0 && descendantFiles.every((file) => isSelected(file.path)));
  $: indeterminate = node.kind === 'dir'
    && !allSelected
    && descendantFiles.some((file) => isSelected(file.path))
    && !checked;
  let checkbox;
  $: if (checkbox) checkbox.indeterminate = indeterminate;

  function toggleExpand() {
    dispatch('expand', node.path);
  }

  function toggleSelect() {
    dispatch('toggle', node);
  }

  function collectFiles(current) {
    if (current.kind === 'file') return [current];
    return current.children.flatMap(collectFiles);
  }

  function isSelected(path) {
    return allSelected || selectedPaths.some((selected) => path === selected || path.startsWith(selected + '/'));
  }

  function formatSize(size) {
    if (size == null) return '';
    if (size < 1024) return `${size} B`;
    if (size < 1024 * 1024) return `${(size / 1024).toFixed(1)} KB`;
    return `${(size / 1024 / 1024).toFixed(1)} MB`;
  }
</script>

<div>
  <div class="flex min-h-9 items-center gap-2 rounded-lg px-2 text-sm hover:bg-slate-50 dark:hover:bg-slate-900">
    {#if node.kind === 'dir'}
      <button type="button" class="grid h-6 w-6 place-items-center rounded text-slate-400 hover:bg-slate-200 hover:text-slate-700 dark:hover:bg-slate-800 dark:hover:text-slate-200" on:click={toggleExpand} aria-label={expanded ? $t('tree.collapse') : $t('tree.expand')}>
        <span class:rotate-90={expanded} class="transition-transform">›</span>
      </button>
    {:else}
      <span class="w-6"></span>
    {/if}
    <input bind:this={checkbox} type="checkbox" checked={checked} disabled={node.path === ''} on:change={toggleSelect} class="h-4 w-4 rounded border-slate-300 text-indigo-600 focus:ring-indigo-500 disabled:opacity-30" />
    <button type="button" class="flex min-w-0 flex-1 items-center gap-2 text-left" on:click={node.kind === 'dir' ? toggleExpand : toggleSelect}>
      <span>{node.kind === 'dir' ? (expanded ? '📂' : '📁') : '📄'}</span>
      <span class="truncate text-slate-700 dark:text-slate-300">{node.name}</span>
      {#if node.kind === 'file'}<span class="ml-auto text-xs text-slate-400 dark:text-slate-500">{formatSize(node.size)}</span>{/if}
    </button>
  </div>

  {#if node.kind === 'dir' && expanded}
    <div class="ml-6 border-l border-slate-100 pl-2 dark:border-slate-800">
      {#each node.children as child (child.path)}
        <svelte:self
          node={child}
          {selectedPaths}
          {allSelected}
          {expandedPaths}
          on:toggle={(event) => dispatch('toggle', event.detail)}
          on:expand={(event) => dispatch('expand', event.detail)}
        />
      {/each}
    </div>
  {/if}
</div>
