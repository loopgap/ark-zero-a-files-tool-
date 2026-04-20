import type { ArchiveBrowseFile, OpenTab, SearchHit, TreeNode, WorkbenchState } from './types';

export type SourceMode = 'workspace' | 'archives' | 'recent' | 'help' | 'search';
export type ThemeName = 'minimal-dark' | 'minimal-light';

export interface ToastMessage {
	id: string;
	message: string;
	tone: 'info' | 'success' | 'error';
}

export const TEXT_EXTENSIONS = [
	'.bash',
	'.bat',
	'.c',
	'.cc',
	'.cjs',
	'.clj',
	'.conf',
	'.cs',
	'.cpp',
	'.css',
	'.dart',
	'.env',
	'.erl',
	'.ex',
	'.exs',
	'.fs',
	'.fsharp',
	'.gql',
	'.go',
	'.gradle',
	'.graphql',
	'.groovy',
	'.h',
	'.hpp',
	'.html',
	'.htm',
	'.ini',
	'.java',
	'.js',
	'.jsx',
	'.json',
	'.kts',
	'.kt',
	'.less',
	'.lua',
	'.log',
	'.m',
	'.md',
	'.mk',
	'.php',
	'.pl',
	'.pm',
	'.properties',
	'.proto',
	'.py',
	'.r',
	'.rb',
	'.rs',
	'.sass',
	'.scala',
	'.scss',
	'.sh',
	'.sol',
	'.sql',
	'.svg',
	'.swift',
	'.svelte',
	'.tex',
	'.toml',
	'.ts',
	'.tsx',
	'.txt',
	'.vue',
	'.xml',
	'.yaml',
	'.yml',
	'.zsh'
];

export const TEXT_FILENAMES = ['dockerfile', 'makefile', 'justfile', 'cmakelists.txt'];
export const SPREADSHEET_EXTENSIONS = ['.csv', '.tsv', '.xls', '.xlsx'];
export const PREVIEW_EXTENSIONS = ['.docx', '.gif', '.jpeg', '.jpg', '.pdf', '.png'];

export function normalizeTheme(theme?: string): ThemeName {
	switch ((theme || '').trim().toLowerCase()) {
		case 'light':
		case 'minimal-light':
			return 'minimal-light';
		default:
			return 'minimal-dark';
	}
}

export function applyTheme(theme?: string) {
	document.documentElement.setAttribute('data-theme', normalizeTheme(theme));
}

export function extensionFromPath(path: string) {
	const index = path.lastIndexOf('.');
	return index === -1 ? '' : path.slice(index).toLowerCase();
}

export function filenameFromPath(path: string) {
	return path.replace(/\\/g, '/').split('/').pop()?.toLowerCase() || '';
}

export function classifyExtension(extension?: string): OpenTab['kind'] {
	const ext = (extension || '').toLowerCase();
	if (['.html', '.htm', '.svg'].includes(ext)) return 'text';
	if (SPREADSHEET_EXTENSIONS.includes(ext)) return 'spreadsheet';
	if (PREVIEW_EXTENSIONS.includes(ext)) return 'preview';
	if (TEXT_EXTENSIONS.includes(ext)) return 'text';
	return 'binary';
}

type OpenableWorkbenchItem = Pick<TreeNode, 'path' | 'name' | 'rootId' | 'extension'> & { virtualFolderIds?: string[] };

export function toOpenTab(item: OpenableWorkbenchItem | ArchiveBrowseFile | SearchHit): OpenTab {
	const extension = item.extension || extensionFromPath(item.path);
	const filename = filenameFromPath(item.path);
	return {
		id: item.path,
		path: item.path,
		name: item.name,
		rootId: item.rootId,
		extension,
		kind: TEXT_FILENAMES.includes(filename) ? 'text' : classifyExtension(extension),
		virtualFolderIds: 'virtualFolderIds' in item ? item.virtualFolderIds || [] : [],
		dirty: false
	};
}

export function isMarkdownFile(path: string) {
	return extensionFromPath(path) === '.md';
}

export function isSpreadsheetExtension(extension?: string) {
	return SPREADSHEET_EXTENSIONS.includes((extension || '').toLowerCase());
}

export function isDocxExtension(extension?: string) {
	return (extension || '').toLowerCase() === '.docx';
}

export function isIframePreviewExtension(extension?: string) {
	return ['.htm', '.html', '.pdf'].includes((extension || '').toLowerCase());
}

export function isImagePreviewExtension(extension?: string) {
	return ['.gif', '.jpeg', '.jpg', '.png', '.svg'].includes((extension || '').toLowerCase());
}

export function encodeBase64(bytes: Uint8Array) {
	let binary = '';
	const chunkSize = 32768;
	for (let index = 0; index < bytes.length; index += chunkSize) {
		const chunk = bytes.subarray(index, index + chunkSize);
		for (let cursor = 0; cursor < chunk.length; cursor += 1) {
			binary += String.fromCharCode(chunk[cursor]);
		}
	}
	return btoa(binary);
}

export function normalizeWorkbenchState(state: WorkbenchState): WorkbenchState {
	return {
		...state,
		workspace: {
			...state.workspace,
			roots: state.workspace?.roots ?? []
		},
		physicalRoots: state.physicalRoots ?? [],
		virtualFolders: state.virtualFolders ?? [],
		autoCategories: state.autoCategories ?? [],
		recentItems: state.recentItems ?? [],
		recentWorkspaces: state.recentWorkspaces ?? [],
		helpDocs: state.helpDocs ?? [],
		theme: normalizeTheme(state.theme),
		policy: {
			...state.policy,
			directoryAllowlist: state.policy?.directoryAllowlist ?? [],
			directoryBlocklist: state.policy?.directoryBlocklist ?? [],
			fileTypeAllowlist: state.policy?.fileTypeAllowlist ?? [],
			fileTypeBlocklist: state.policy?.fileTypeBlocklist ?? [],
			maxIndexedFileSize: state.policy?.maxIndexedFileSize ?? 1048576
		}
	};
}

export function describeError(error: unknown, fallback = '操作失败，请重试。') {
	if (error instanceof Error && error.message) return error.message;
	if (typeof error === 'string' && error.trim()) return error;
	return fallback;
}

export function renamePath(path: string, nextName: string) {
	const parts = path.split('/');
	parts[parts.length - 1] = nextName;
	return parts.join('/');
}

export function renderMarkdown(markdown: string) {
	const lines = markdown.replace(/\r\n/g, '\n').split('\n');
	const blocks: string[] = [];
	let paragraph: string[] = [];
	let listItems: string[] = [];
	let codeLines: string[] = [];
	let inCodeBlock = false;

	const flushParagraph = () => {
		if (!paragraph.length) return;
		blocks.push(`<p>${parseInline(paragraph.join(' '))}</p>`);
		paragraph = [];
	};

	const flushList = () => {
		if (!listItems.length) return;
		blocks.push(`<ul>${listItems.map((item) => `<li>${parseInline(item)}</li>`).join('')}</ul>`);
		listItems = [];
	};

	const flushCode = () => {
		if (!codeLines.length) return;
		blocks.push(`<pre><code>${escapeHtml(codeLines.join('\n'))}</code></pre>`);
		codeLines = [];
	};

	for (const rawLine of lines) {
		const line = rawLine.trimEnd();
		if (line.startsWith('```')) {
			flushParagraph();
			flushList();
			if (inCodeBlock) {
				flushCode();
				inCodeBlock = false;
			} else {
				inCodeBlock = true;
			}
			continue;
		}
		if (inCodeBlock) {
			codeLines.push(rawLine);
			continue;
		}
		const trimmed = line.trim();
		if (!trimmed) {
			flushParagraph();
			flushList();
			continue;
		}
		if (/^#{1,6}\s/.test(trimmed)) {
			flushParagraph();
			flushList();
			const level = trimmed.match(/^#+/)?.[0].length ?? 1;
			blocks.push(`<h${level}>${parseInline(trimmed.slice(level).trim())}</h${level}>`);
			continue;
		}
		if (/^[-*]\s+/.test(trimmed)) {
			flushParagraph();
			listItems.push(trimmed.replace(/^[-*]\s+/, ''));
			continue;
		}
		if (/^---+$/.test(trimmed)) {
			flushParagraph();
			flushList();
			blocks.push('<hr />');
			continue;
		}
		paragraph.push(trimmed);
	}

	flushParagraph();
	flushList();
	flushCode();
	return blocks.join('');
}

function escapeHtml(value: string) {
	return value
		.replaceAll('&', '&amp;')
		.replaceAll('<', '&lt;')
		.replaceAll('>', '&gt;')
		.replaceAll('"', '&quot;')
		.replaceAll("'", '&#39;');
}

function parseInline(value: string) {
	return escapeHtml(value)
		.replace(/`([^`]+)`/g, '<code>$1</code>')
		.replace(/\*\*([^*]+)\*\*/g, '<strong>$1</strong>')
		.replace(/\*([^*]+)\*/g, '<em>$1</em>')
		.replace(/\[([^\]]+)\]\(([^)]+)\)/g, '<a href="$2" target="_blank" rel="noreferrer">$1</a>');
}
