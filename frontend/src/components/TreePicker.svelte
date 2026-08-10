<script>
  import TreeNode from './TreeNode.svelte';
  import { t } from '../i18n';

  export let entries = [];
  export let selectedPaths = [];
  export let selectionStats = { fileCount: 0, size: 0 };
  export let allSelected = false;

  let expandedPaths = [''];
  let searchTerm = '';

  $: filteredEntries = filterEntries(entries, searchTerm);
  $: tree = buildTree(filteredEntries, $t('tree.repository'));
  $: fileCount = entries.filter((entry) => entry.kind === 'file').length;
  $: selectedFileCount = allSelected ? fileCount : entries.filter((entry) => entry.kind === 'file' && isSelected(entry.path)).length;
  $: selectedSize = entries
    .filter((entry) => entry.kind === 'file' && isSelected(entry.path))
    .reduce((total, entry) => total + (entry.size || 0), 0);
  $: selectionStats = { fileCount: selectedFileCount, size: selectedSize };

  function buildTree(items, repositoryName) {
    const root = { path: '', name: repositoryName, kind: 'dir', children: [] };
    const nodes = new Map([['', root]]);
    for (const item of items) {
      const parts = item.path.split('/').filter(Boolean);
      let parent = root;
      let currentPath = '';
      parts.forEach((part, index) => {
        currentPath = currentPath ? `${currentPath}/${part}` : part;
        let node = nodes.get(currentPath);
        if (!node) {
          const isLeaf = index === parts.length - 1;
          node = {
            path: currentPath,
            name: part,
            kind: isLeaf ? item.kind : 'dir',
            size: isLeaf ? item.size : undefined,
            children: [],
          };
          nodes.set(currentPath, node);
          parent.children.push(node);
        }
        parent = node;
      });
    }
    sortNodes(root);
    return root;
  }

  function sortNodes(node) {
    node.children.sort((left, right) => {
      if (left.kind !== right.kind) return left.kind === 'dir' ? -1 : 1;
      return left.name.localeCompare(right.name);
    });
    node.children.forEach(sortNodes);
  }

  function filterEntries(items, query) {
    const normalized = query.trim().toLowerCase();
    if (!normalized) return items;

    const matches = items.filter((item) =>
      item.path.toLowerCase().includes(normalized)
      || item.name.toLowerCase().includes(normalized)
    );
    const visiblePaths = new Set();
    for (const match of matches) {
      if (match.kind === 'dir') {
        for (const item of items) {
          if (item.path === match.path || item.path.startsWith(match.path + '/')) visiblePaths.add(item.path);
        }
      } else {
        visiblePaths.add(match.path);
      }
      let parent = match.path;
      while (parent.includes('/')) {
        parent = parent.slice(0, parent.lastIndexOf('/'));
        visiblePaths.add(parent);
      }
    }
    return items.filter((item) => visiblePaths.has(item.path));
  }

  function toggle(node) {
    if (allSelected) {
      allSelected = false;
      selectedPaths = entries.filter((entry) => entry.kind === 'file').map((entry) => entry.path);
    }
    const descendantFiles = collectFiles(node);
    const nodeFullySelected = descendantFiles.length > 0 && descendantFiles.every((file) => isSelected(file.path));

    if (node.kind === 'dir') {
      selectedPaths = selectedPaths.filter((selected) =>
        selected !== node.path && !selected.startsWith(node.path + '/')
      );
      if (!nodeFullySelected) selectedPaths = [...selectedPaths, node.path];
      return;
    }

    const ancestors = selectedPaths.filter((selected) => node.path.startsWith(selected + '/'));
    if (ancestors.length) {
      const affected = entries.filter((entry) =>
        entry.kind === 'file' && ancestors.some((ancestor) =>
          entry.path === ancestor || entry.path.startsWith(ancestor + '/')
        )
      );
      selectedPaths = selectedPaths.filter((selected) => !ancestors.includes(selected));
      selectedPaths = [
        ...selectedPaths,
        ...affected.filter((entry) => entry.path !== node.path).map((entry) => entry.path),
      ];
      return;
    }

    selectedPaths = selectedPaths.includes(node.path)
      ? selectedPaths.filter((selected) => selected !== node.path)
      : [...selectedPaths, node.path];
  }

  function isSelected(path) {
    return allSelected || selectedPaths.some((selected) => path === selected || path.startsWith(selected + '/'));
  }

  function collectFiles(node) {
    if (node.kind === 'file') return [node];
    return node.children.flatMap(collectFiles);
  }

  function formatSize(size) {
    if (size < 1024) return size + ' B';
    if (size < 1024 * 1024) return (size / 1024).toFixed(1) + ' KB';
    if (size < 1024 * 1024 * 1024) return (size / 1024 / 1024).toFixed(1) + ' MB';
    return (size / 1024 / 1024 / 1024).toFixed(2) + ' GB';
  }

  function toggleExpand(path) {
    expandedPaths = expandedPaths.includes(path)
      ? expandedPaths.filter((expanded) => expanded !== path)
      : [...expandedPaths, path];
  }

  function setSearchTerm(value) {
    searchTerm = value;
    expandedPaths = searchTerm.trim() ? allDirectoryPaths(tree) : [''];
  }

  function allDirectoryPaths(node) {
    if (node.kind !== 'dir') return [];
    return [node.path, ...node.children.flatMap(allDirectoryPaths)];
  }

  function expandAll() {
    expandedPaths = allDirectoryPaths(tree);
  }

  function collapseAll() {
    expandedPaths = [''];
  }

  function selectAll() {
    allSelected = true;
    selectedPaths = [];
  }

  function clearSelection() {
    allSelected = false;
    selectedPaths = [];
  }
</script>

<section class="mt-6 rounded-2xl border border-indigo-100 bg-indigo-50/40 p-3 dark:border-indigo-950 dark:bg-indigo-950/30">
  <div class="mb-2 flex flex-wrap items-center justify-between gap-2 px-2">
    <div>
      <h2 class="text-sm font-bold text-slate-700 dark:text-slate-200">{$t('tree.title')}</h2>
      <p class="text-xs text-slate-400 dark:text-slate-500">{$t('tree.help')}</p>
    </div>
    <div class="flex flex-wrap items-center gap-2 text-xs">
      <span class="text-slate-400 dark:text-slate-500">{$t('tree.count', { selected: selectedFileCount, files: fileCount })}</span>
      {#if selectedFileCount > 0}<span class="text-slate-400 dark:text-slate-500">· {$t('tree.size', { size: formatSize(selectedSize) })}</span>{/if}
      <button type="button" class="font-semibold text-indigo-600 hover:text-indigo-800 dark:text-indigo-400 dark:hover:text-indigo-300" on:click={selectAll}>{$t('tree.selectAll')}</button>
      <button type="button" class="font-semibold text-slate-500 hover:text-slate-700 dark:text-slate-400 dark:hover:text-slate-200" on:click={clearSelection}>{$t('tree.clear')}</button>
    </div>
  </div>
  <div class="mb-2 flex flex-wrap gap-2 px-2">
    <input
      value={searchTerm}
      on:input={(event) => setSearchTerm(event.currentTarget.value)}
      placeholder={$t('tree.searchPlaceholder')}
      class="min-w-[12rem] flex-1 rounded-lg border border-slate-200 bg-white px-3 py-2 text-sm outline-none focus:border-indigo-500 dark:border-slate-700 dark:bg-slate-950 dark:text-slate-100 dark:placeholder:text-slate-600"
    />
    <button type="button" class="rounded-lg border border-slate-200 px-3 py-2 text-xs font-semibold text-slate-600 hover:bg-white dark:border-slate-700 dark:text-slate-300 dark:hover:bg-slate-900" on:click={expandAll}>{$t('tree.expandAll')}</button>
    <button type="button" class="rounded-lg border border-slate-200 px-3 py-2 text-xs font-semibold text-slate-600 hover:bg-white dark:border-slate-700 dark:text-slate-300 dark:hover:bg-slate-900" on:click={collapseAll}>{$t('tree.collapseAll')}</button>
  </div>
  <div class="max-h-80 overflow-auto rounded-xl bg-white p-2 dark:bg-slate-950">
    {#if !filteredEntries.length}
      <p class="px-2 py-8 text-center text-sm text-slate-400 dark:text-slate-500">{$t('tree.searchEmpty')}</p>
    {:else}
      <TreeNode
        node={tree}
        {selectedPaths}
        {allSelected}
        {expandedPaths}
        on:toggle={(event) => toggle(event.detail)}
        on:expand={(event) => toggleExpand(event.detail)}
      />
    {/if}
  </div>
</section>
