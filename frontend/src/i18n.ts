import { derived, writable } from 'svelte/store';

export type Locale = 'zh' | 'en';
export type Theme = 'light' | 'dark';

const messages = {
  zh: {
    'app.eyebrow': 'GitHub archive tool',
    'app.subtitle': '选择 GitHub 文件或目录，按规则筛选后下载 ZIP。',
    'app.language': 'English',
    'app.themeLight': '切换到亮色主题',
    'app.themeDark': '切换到暗色主题',
    'app.themeLightShort': '亮色',
    'app.themeDarkShort': '暗色',
    'input.urlLabel': 'GitHub URL',
    'input.urlHelp': '支持仓库根路径、 blob 文件路径和 tree 目录路径。',
    'input.tokenSummary': 'GitHub Token（可选）',
    'input.tokenPlaceholder': '粘贴 ghp_... 或 github_pat_...',
    'input.hide': '隐藏',
    'input.show': '显示',
    'input.clear': '清除',
    'input.tokenHelp': 'Token 只保存在当前浏览器，并通过请求头发送给 zzz；建议使用 HTTPS。',
    'input.applyToken': '申请 Fine-grained Token',
    'input.accessTokenSummary': '服务访问密钥（可选）',
    'input.accessTokenPlaceholder': '粘贴服务端 ACCESS_TOKEN',
    'input.accessTokenHelp': '仅当管理员配置了 ACCESS_TOKEN 时需要填写；密钥只保存在当前浏览器。',
    'input.browse': '先浏览并选择目录',
    'input.loadingTree': '正在读取目录…',
    'filter.summary': '文件过滤设置',
    'filter.ignore': '忽略规则',
    'filter.include': '仅包含',
    'filter.globHelp': '逗号分隔，使用 glob 语法。',
    'filter.includeHelp': '留空表示包含所有未被忽略的文件。',
    'tree.title': '选择下载内容',
    'tree.help': '勾选目录会包含其下所有文件。',
    'tree.count': '{selected} 项 / {files} 个文件',
    'tree.size': '预计 {size}',
    'tree.selectAll': '全选',
    'tree.clear': '清空',
    'tree.expandAll': '展开全部',
    'tree.collapseAll': '折叠全部',
    'tree.searchPlaceholder': '搜索文件名或路径…',
    'tree.searchEmpty': '没有匹配的文件。',
    'tree.repository': '仓库',
    'tree.collapse': '收起目录',
    'tree.expand': '展开目录',
    'download.loading': '正在收集并打包…',
    'download.selected': '下载选中内容',
    'download.direct': '直接下载 ZIP',
    'download.selectedHelp': '将按当前勾选内容生成 ZIP。',
    'download.directHelp': '也可以先浏览目录，再选择需要下载的文件。',
    'download.hostSaved': '已保存到挂载目录：{path}',
    'output.summary': '输出方式',
    'output.browser': '浏览器下载',
    'output.host': '保存到主机',
    'output.hostUnavailable': '服务端未配置 DOWNLOAD_ROOT',
    'output.hostPathLabel': '主机相对路径',
    'output.hostPathHelp': '路径相对于 Docker 的 DOWNLOAD_ROOT，例如 downloads/ebooks。',
    'output.hostPathPlaceholder': '例如 ebooks/2025',
    'output.format': '保存格式',
    'output.folder': '文件夹',
    'output.zip': 'ZIP 文件',
    'output.folderHelp': '按原目录结构写入挂载目录。',
    'output.zipHelp': '在挂载目录中生成 ZIP 文件。',
    'status.collecting': '正在处理 GitHub 内容…',
    'status.done': '下载已开始',
    'status.error': '下载失败',
    'status.files': '{count} 个文件',
    'token.errorHelp': '可以前往',
    'token.errorHelpAfter': '，再将它配置到 zzz 服务端；建议只授予目标仓库的 Contents 只读权限。',
    'history.summary': '最近使用',
    'history.empty': '暂无历史记录',
    'history.clear': '清除历史',
    'errors.requestFailed': '请求失败',
    'errors.noTree': '没有可展示的文件。',
    'errors.selectRequired': '请至少勾选一个文件或目录，或重新输入 URL 直接下载。',
    'errors.github401': 'GitHub Token 无效或已过期，请检查后重试。',
    'errors.github403': 'GitHub 拒绝了请求，可能是 API 限流或仓库权限不足。',
    'errors.github429': 'GitHub API 已达到速率限制，请稍后重试。',
  },
  en: {
    'app.eyebrow': 'GitHub archive tool',
    'app.subtitle': 'Select GitHub files or directories, filter them, and download a ZIP.',
    'app.language': '中文',
    'app.themeLight': 'Switch to light theme',
    'app.themeDark': 'Switch to dark theme',
    'app.themeLightShort': 'Light',
    'app.themeDarkShort': 'Dark',
    'input.urlLabel': 'GitHub URL',
    'input.urlHelp': 'Supports repository roots, blob file paths, and tree directory paths.',
    'input.tokenSummary': 'GitHub Token (optional)',
    'input.tokenPlaceholder': 'Paste ghp_... or github_pat_...',
    'input.hide': 'Hide',
    'input.show': 'Show',
    'input.clear': 'Clear',
    'input.tokenHelp': 'The token stays in this browser and is sent in a request header; HTTPS is recommended.',
    'input.applyToken': 'Create a Fine-grained Token',
    'input.accessTokenSummary': 'Service access token (optional)',
    'input.accessTokenPlaceholder': 'Paste the server ACCESS_TOKEN',
    'input.accessTokenHelp': 'Only required when the administrator configured ACCESS_TOKEN; it stays in this browser.',
    'input.browse': 'Browse and select files',
    'input.loadingTree': 'Reading directory…',
    'filter.summary': 'File filters',
    'filter.ignore': 'Ignore patterns',
    'filter.include': 'Include only',
    'filter.globHelp': 'Comma-separated glob patterns.',
    'filter.includeHelp': 'Leave empty to include everything not ignored.',
    'tree.title': 'Select download contents',
    'tree.help': 'Selecting a directory includes all files underneath it.',
    'tree.count': '{selected} selected / {files} files',
    'tree.size': 'Estimated {size}',
    'tree.selectAll': 'Select all',
    'tree.clear': 'Clear',
    'tree.expandAll': 'Expand all',
    'tree.collapseAll': 'Collapse all',
    'tree.searchPlaceholder': 'Search file names or paths…',
    'tree.searchEmpty': 'No matching files.',
    'tree.repository': 'Repository',
    'tree.collapse': 'Collapse directory',
    'tree.expand': 'Expand directory',
    'download.loading': 'Collecting and packaging…',
    'download.selected': 'Download selected',
    'download.direct': 'Download ZIP',
    'download.selectedHelp': 'A ZIP will be generated from the current selection.',
    'download.directHelp': 'You can also browse the directory first and choose specific files.',
    'download.hostSaved': 'Saved to the mounted directory: {path}',
    'output.summary': 'Output',
    'output.browser': 'Browser download',
    'output.host': 'Save to host',
    'output.hostUnavailable': 'DOWNLOAD_ROOT is not configured on the server',
    'output.hostPathLabel': 'Host-relative path',
    'output.hostPathHelp': 'Path relative to Docker DOWNLOAD_ROOT, for example downloads/ebooks.',
    'output.hostPathPlaceholder': 'For example ebooks/2025',
    'output.format': 'Save as',
    'output.folder': 'Folder',
    'output.zip': 'ZIP file',
    'output.folderHelp': 'Write files to the mounted directory using the original structure.',
    'output.zipHelp': 'Generate a ZIP file inside the mounted directory.',
    'status.collecting': 'Processing GitHub contents…',
    'status.done': 'Download started',
    'status.error': 'Download failed',
    'status.files': '{count} files',
    'token.errorHelp': 'You can',
    'token.errorHelpAfter': ', then configure it for zzz; grant Contents read-only access to the target repository.',
    'history.summary': 'Recent URLs',
    'history.empty': 'No recent URLs',
    'history.clear': 'Clear history',
    'errors.requestFailed': 'Request failed',
    'errors.noTree': 'There are no files to display.',
    'errors.selectRequired': 'Select at least one file or directory, or enter a new URL for a direct download.',
    'errors.github401': 'The GitHub Token is invalid or expired. Check it and try again.',
    'errors.github403': 'GitHub rejected the request. The API may be rate-limited or the repository may be inaccessible.',
    'errors.github429': 'The GitHub API rate limit has been reached. Try again later.',
  },
} as const;

function readStored<T extends string>(key: string, fallback: T): T {
  if (typeof window === 'undefined') return fallback;
  return (window.localStorage.getItem(key) as T | null) || fallback;
}

function initialLocale(): Locale {
  const stored = readStored('zzz.locale', '');
  if (stored === 'zh' || stored === 'en') return stored;
  return typeof navigator !== 'undefined' && navigator.language.toLowerCase().startsWith('zh') ? 'zh' : 'en';
}

function initialTheme(): Theme {
  const stored = readStored('zzz.theme', '');
  if (stored === 'light' || stored === 'dark') return stored;
  return typeof window !== 'undefined' && window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
}

export const locale = writable<Locale>(initialLocale());
export const theme = writable<Theme>(initialTheme());

export const t = derived(locale, ($locale) => (key: keyof typeof messages.zh, values: Record<string, string | number> = {}) => {
  let value: string = messages[$locale][key] || messages.en[key] || key;
  for (const [name, replacement] of Object.entries(values)) value = value.replace(`{${name}}`, String(replacement));
  return value;
});

locale.subscribe((value) => {
  if (typeof document !== 'undefined') document.documentElement.lang = value === 'zh' ? 'zh-CN' : 'en';
  if (typeof window !== 'undefined') window.localStorage.setItem('zzz.locale', value);
});

theme.subscribe((value) => {
  if (typeof document !== 'undefined') document.documentElement.classList.toggle('dark', value === 'dark');
  if (typeof window !== 'undefined') window.localStorage.setItem('zzz.theme', value);
});

export function toggleLocale() {
  locale.update((value) => (value === 'zh' ? 'en' : 'zh'));
}

export function toggleTheme() {
  theme.update((value) => (value === 'light' ? 'dark' : 'light'));
}
