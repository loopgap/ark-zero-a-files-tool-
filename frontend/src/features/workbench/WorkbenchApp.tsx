import { lazy, Suspense, useEffect, useMemo, useRef, useState, type MouseEvent as ReactMouseEvent } from 'react';
import type { LucideIcon } from 'lucide-react';
import {
	Archive,
	CheckCircle2,
	ChevronDown,
	ChevronRight,
	CircleHelp,
	Clock3,
	ExternalLink,
	FilePlus2,
	FileSearch,
	FileText,
	Folder,
	FolderOpen,
	FolderPlus,
	FolderTree,
	PencilLine,
	Plus,
	RefreshCw,
	Search,
	Settings,
	Star,
	Trash2,
	MoreHorizontal
} from 'lucide-react';
import { bootstrapDesktop, buildAssetUrl, pickDirectory, rpc } from '../../lib/desktop';
import type { ArchiveBrowseFile, ArchiveBrowseResponse, AutoCategory, HelpDoc, OpenTab, SearchHit, TreeNode, VirtualFolder, WorkbenchState, WorkspaceRoot } from '../../lib/types';
import {
	applyTheme,
	classifyExtension,
	describeError,
	extensionFromPath,
	normalizeTheme,
	normalizeWorkbenchState,
	renamePath,
	toOpenTab,
	type SourceMode,
	type ThemeName,
	type ToastMessage
} from '../../lib/workbench';
import { ToastViewport } from './components/ToastViewport';
import { TreeBrowserRow } from './components/TreeBrowserRow';
import { WorkbenchBrowserContent } from './components/WorkbenchBrowserContent';
import { ArchiveExplorerView } from './components/ArchiveExplorerView';
import { VirtualList } from './components/VirtualList';
import {
	archiveGroupViewFromLegacy,
	archiveSearchModeFromLegacy,
	archiveSortByFromLegacy,
	archiveSortDirectionFromLegacy,
	archiveSourceKey,
	createArchiveBrowseRequest,
	createArchiveExplorerState,
	type ArchiveSourceRef
} from './archiveExplorer';
import type { ArchiveCommittedSearch, ArchiveDirectoryEntry, ArchiveExplorerState, ArchiveMatchField, ArchiveSuggestion } from './types';

const CodeEditor = lazy(async () => ({ default: (await import('./components/CodeEditor')).CodeEditor }));
const CodePreview = lazy(async () => ({ default: (await import('./components/CodePreview')).CodePreview }));
const FilePreview = lazy(async () => ({ default: (await import('./components/FilePreview')).FilePreview }));
const SpreadsheetEditor = lazy(async () => ({ default: (await import('./components/SpreadsheetEditor')).SpreadsheetEditor }));
const HelpContent = lazy(async () => ({ default: (await import('./components/HelpContent')).HelpContent }));
const ModalDialog = lazy(async () => ({ default: (await import('./components/ModalDialog')).ModalDialog }));
const SettingsDialog = lazy(async () => ({ default: (await import('./components/SettingsDialog')).SettingsDialog }));

type DialogState =
	| { kind: 'hidden' }
	| {
			kind: 'input';
			title: string;
			description: string;
			confirmLabel: string;
			initialValue: string;
			placeholder: string;
			submitting: boolean;
			action: (value: string) => Promise<boolean>;
	  }
	| {
			kind: 'confirm';
			title: string;
			description: string;
			confirmLabel: string;
			danger: boolean;
			submitting: boolean;
			action: () => Promise<boolean>;
	  };

type TabMenuState =
	| { open: false }
	| {
			open: true;
			tabId: string;
			x: number;
			y: number;
	  };

const SOURCE_ITEMS: Array<{ id: SourceMode; label: string; icon: LucideIcon }> = [
	{ id: 'workspace', label: 'Workspace', icon: FolderTree },
	{ id: 'archives', label: 'Archives', icon: Archive },
	{ id: 'recent', label: 'Recent', icon: Clock3 },
	{ id: 'help', label: 'Help', icon: CircleHelp }
];

function lastItem<T>(items: T[]) {
	return items.length ? items[items.length - 1] : undefined;
}

function splitListValue(value: string) {
	return value
		.split(/\r?\n|,/)
		.map((item) => item.trim())
		.filter(Boolean);
}

function clamp(value: number, min: number, max: number) {
	return Math.min(Math.max(value, min), max);
}

function readStoredNumber(key: string, fallback: number) {
	if (typeof window === 'undefined') return fallback;
	const raw = window.localStorage.getItem(key);
	if (!raw) return fallback;
	const parsed = Number.parseInt(raw, 10);
	return Number.isFinite(parsed) ? parsed : fallback;
}

function isSubsequence(text: string, query: string) {
	if (!query) return true;
	let cursor = 0;
	for (const char of text) {
		if (char === query[cursor]) {
			cursor += 1;
			if (cursor === query.length) return true;
		}
	}
	return false;
}

function normalizeSearchValue(value: string, caseSensitive: boolean) {
	return caseSensitive ? value : value.toLowerCase();
}

function normalizeArchiveFileType(value: string) {
	const normalized = value.trim().toLowerCase();
	if (!normalized) return '';
	return normalized.startsWith('.') ? normalized : `.${normalized}`;
}

function archiveDirectory(path: string) {
	const normalized = path.replace(/\\/g, '/');
	const boundary = normalized.lastIndexOf('/');
	return boundary >= 0 ? normalized.slice(0, boundary) : '';
}

function archiveJoinPath(base: string, segment: string) {
	return base ? `${base}/${segment}` : segment;
}

function archiveBasename(path: string) {
	const normalized = path.replace(/\\/g, '/');
	const boundary = normalized.lastIndexOf('/');
	return boundary >= 0 ? normalized.slice(boundary + 1) : normalized;
}

function normalizePathKey(path: string) {
	return path.replace(/\\/g, '/').replace(/\/+/g, '/').replace(/\/$/, '');
}

function joinPathSegments(base: string, segment: string) {
	return `${normalizePathKey(base)}/${segment}`;
}

function relativePathSegments(rootPath: string, targetPath: string) {
	const normalizedRoot = normalizePathKey(rootPath);
	const normalizedTarget = normalizePathKey(targetPath);
	if (normalizedTarget === normalizedRoot) return [];
	if (!normalizedTarget.startsWith(`${normalizedRoot}/`)) return [];
	return normalizedTarget
		.slice(normalizedRoot.length + 1)
		.split('/')
		.filter(Boolean);
}

function archiveVirtualPath(item: SearchHit, roots: WorkspaceRoot[]) {
	const normalizedPath = normalizePathKey(item.path);
	const root = roots.find((candidate) => candidate.id === item.rootId);
	if (!root) return archiveBasename(normalizedPath);
	const rootPath = normalizePathKey(root.path);
	let relativePath = '';
	if (normalizedPath === rootPath) {
		relativePath = root.label;
	} else if (normalizedPath.startsWith(`${rootPath}/`)) {
		relativePath = normalizedPath.slice(rootPath.length + 1);
	} else {
		relativePath = archiveBasename(normalizedPath);
	}
	return roots.length > 1 ? archiveJoinPath(root.label, relativePath) : relativePath;
}

function archiveVirtualDirectory(item: SearchHit, roots: WorkspaceRoot[]) {
	return archiveDirectory(archiveVirtualPath(item, roots));
}

function matchKindLabel(matchKind: SearchHit['matchKind']) {
	switch (matchKind) {
		case 'directory':
			return '目录';
		case 'type':
			return '类型';
		case 'path':
			return '路径';
		case 'content':
			return '内容';
		default:
			return '名称';
	}
}

function cappedListHeight(count: number, rowHeight: number, minHeight = 140, maxHeight = 520) {
	if (!count) return minHeight;
	return Math.max(minHeight, Math.min(count * rowHeight, maxHeight));
}

function localArchiveMatchKind(
	query: string,
	item: SearchHit,
	matchField: ArchiveMatchField,
	caseSensitive: boolean,
	fileTypeFilter: string,
	roots: WorkspaceRoot[]
) {
	const normalizedType = normalizeArchiveFileType(fileTypeFilter);
	const itemExtension = (item.extension || extensionFromPath(item.path)).toLowerCase();
	if (normalizedType && itemExtension !== normalizedType) return null;

	const normalizedQuery = query.trim();
	if (!normalizedQuery) {
		if (normalizedType) return 'type' as const;
		return matchField === 'directory' || matchField === 'type' || matchField === 'content' ? matchField : item.matchKind;
	}

	const needle = normalizeSearchValue(normalizedQuery, caseSensitive);
	const name = normalizeSearchValue(item.name, caseSensitive);
	const directory = normalizeSearchValue(archiveVirtualDirectory(item, roots), caseSensitive);
	const type = normalizeSearchValue(itemExtension.replace(/^\./, ''), false);
	const exactType = normalizeSearchValue(itemExtension, false);

	const matchName =
		name === needle || name.startsWith(needle) || name.includes(needle) || isSubsequence(name, needle);
	const matchDirectory =
		directory.startsWith(needle) || directory.includes(needle) || isSubsequence(directory, needle);
	const matchType = exactType === normalizeArchiveFileType(normalizedQuery) || type === needle || type.includes(needle);

	switch (matchField) {
		case 'name':
			return matchName ? 'name' : null;
		case 'directory':
			return matchDirectory ? 'directory' : null;
		case 'type':
			return matchType ? 'type' : null;
		case 'content':
			return null;
		default:
			if (matchName) return 'name';
			if (matchDirectory) return 'directory';
			if (matchType) return 'type';
			return null;
	}
}

function scoreLocalArchiveItem(
	query: string,
	item: SearchHit,
	matchField: ArchiveMatchField,
	caseSensitive: boolean,
	fileTypeFilter: string,
	roots: WorkspaceRoot[]
) {
	const matchKind = localArchiveMatchKind(query, item, matchField, caseSensitive, fileTypeFilter, roots);
	if (!matchKind) return { score: -1, matchKind: null as SearchHit['matchKind'] | null };

	const normalizedQuery = query.trim();
	if (!normalizedQuery) {
		return { score: matchKind === 'type' ? 30 : 0, matchKind };
	}

	const needle = normalizeSearchValue(normalizedQuery, caseSensitive);
	const name = normalizeSearchValue(item.name, caseSensitive);
	const directory = normalizeSearchValue(archiveVirtualDirectory(item, roots), caseSensitive);
	const itemExtension = (item.extension || extensionFromPath(item.path)).toLowerCase();
	const type = itemExtension.replace(/^\./, '');
	let score = 0;

	switch (matchKind) {
		case 'name':
			if (name === needle) score = 240;
			else if (name.startsWith(needle)) score = 180;
			else if (name.includes(needle)) score = 120;
			else if (isSubsequence(name, needle)) score = 90;
			break;
		case 'directory':
			if (directory.startsWith(needle)) score = 130;
			else if (directory.includes(needle)) score = 88;
			else if (isSubsequence(directory, needle)) score = 52;
			break;
		case 'type':
			if (itemExtension === normalizeArchiveFileType(normalizedQuery) || type === needle) score = 160;
			else if (type.startsWith(needle)) score = 110;
			else if (type.includes(needle)) score = 70;
			break;
		default:
			score = 0;
	}

	if (fileTypeFilter && itemExtension === normalizeArchiveFileType(fileTypeFilter)) {
		score += 18;
	}

	return { score, matchKind };
}

const DEFAULT_SOURCE_SIDEBAR_WIDTH = 252;
const DEFAULT_BROWSER_PANE_WIDTH = 420;
const DEFAULT_PREVIEW_PANE_SIZE = 420;
const TAB_MENU_WIDTH = 168;
const TAB_MENU_HEIGHT = 156;

function ContentLoadingState() {
	return <div className="content-loading-state">正在载入视图...</div>;
}

function toggleNodeExpanded(nodes: TreeNode[], path: string): TreeNode[] {
	return nodes.map((node) => {
		if (node.path === path) {
			return { ...node, expanded: !node.expanded };
		}
		if (!node.children?.length) return node;
		return { ...node, children: toggleNodeExpanded(node.children, path) };
	});
}

function replaceNodeChildren(nodes: TreeNode[], path: string, children: TreeNode[], expanded: boolean): TreeNode[] {
	return nodes.map((node) => {
		if (node.path === path) {
			return { ...node, children, expanded };
		}
		if (!node.children?.length) return node;
		return { ...node, children: replaceNodeChildren(node.children, path, children, expanded) };
	});
}

function setNodeExpanded(nodes: TreeNode[], path: string, expanded: boolean): TreeNode[] {
	return nodes.map((node) => {
		if (node.path === path) {
			return { ...node, expanded };
		}
		if (!node.children?.length) return node;
		return { ...node, children: setNodeExpanded(node.children, path, expanded) };
	});
}

function findNodeByPath(nodes: TreeNode[], path: string): TreeNode | null {
	const normalizedTarget = normalizePathKey(path);
	for (const node of nodes) {
		if (normalizePathKey(node.path) === normalizedTarget) {
			return node;
		}
		if (node.children?.length) {
			const nested = findNodeByPath(node.children, path);
			if (nested) return nested;
		}
	}
	return null;
}

export function WorkbenchApp() {
	const [workbench, setWorkbench] = useState<WorkbenchState | null>(null);
	const [settingsConfig, setSettingsConfig] = useState<any | null>(null);
	const [sourceMode, setSourceMode] = useState<SourceMode>('workspace');
	const [tabs, setTabs] = useState<OpenTab[]>([]);
	const [activeTabId, setActiveTabId] = useState('');
	const [previewTabIds, setPreviewTabIds] = useState<string[]>([]);
	const [searchQuery, setSearchQuery] = useState('');
	const [searchRootId, setSearchRootId] = useState('');
	const [searchMatchField, setSearchMatchField] = useState<ArchiveMatchField>('all');
	const [searchCaseSensitive, setSearchCaseSensitive] = useState(false);
	const [searchFileTypeFilter, setSearchFileTypeFilter] = useState('');
	const [searchArchiveId, setSearchArchiveId] = useState('');
	const [searchResults, setSearchResults] = useState<SearchHit[]>([]);
	const [archiveQuery, setArchiveQuery] = useState('');
	const [archiveMatchFilter, setArchiveMatchFilter] = useState<ArchiveMatchField>('all');
	const [archiveCaseSensitive, setArchiveCaseSensitive] = useState(false);
	const [archiveFileTypeFilter, setArchiveFileTypeFilter] = useState('');
	const [archiveCommittedSearch, setArchiveCommittedSearch] = useState<ArchiveCommittedSearch>({
		query: '',
		matchField: 'all',
		caseSensitive: false,
		fileType: ''
	});
	const [archiveBaseItems, setArchiveBaseItems] = useState<SearchHit[]>([]);
	const [archiveResultCursor, setArchiveResultCursor] = useState(-1);
	const [archiveFolderPath, setArchiveFolderPath] = useState('');
	const [selectedArchiveId, setSelectedArchiveId] = useState('');
	const [selectedAutoCategoryId, setSelectedAutoCategoryId] = useState('');
	const [archiveItems, setArchiveItems] = useState<SearchHit[]>([]);
	const [selectedHelpDoc, setSelectedHelpDoc] = useState('');
	const [helpContent, setHelpContent] = useState('');
	const [helpLoading, setHelpLoading] = useState(false);
	const [helpError, setHelpError] = useState('');
	const [loadedHelpDocId, setLoadedHelpDocId] = useState('');
	const [baseUrl, setBaseUrl] = useState('');
	const [statusText, setStatusText] = useState('准备就绪');
	const [errorText, setErrorText] = useState('');
	const [pendingCount, setPendingCount] = useState(0);
	const [toasts, setToasts] = useState<ToastMessage[]>([]);
	const [dialog, setDialog] = useState<DialogState>({ kind: 'hidden' });
	const [showSettings, setShowSettings] = useState(false);
	const [settingsLoading, setSettingsLoading] = useState(false);
	const [settingsSaving, setSettingsSaving] = useState(false);
	const [settingsError, setSettingsError] = useState('');
	const [theme, setTheme] = useState<ThemeName>('minimal-dark');
	const [previewLayout, setPreviewLayout] = useState<'right' | 'bottom'>('right');
	const [sourceSidebarWidth, setSourceSidebarWidth] = useState(() =>
		readStoredNumber('arkkb:source-sidebar-width', DEFAULT_SOURCE_SIDEBAR_WIDTH)
	);
	const [browserPaneWidth, setBrowserPaneWidth] = useState(() =>
		readStoredNumber('arkkb:browser-pane-width', DEFAULT_BROWSER_PANE_WIDTH)
	);
	const [previewPaneSize, setPreviewPaneSize] = useState(() =>
		readStoredNumber('arkkb:preview-pane-size', DEFAULT_PREVIEW_PANE_SIZE)
	);
	const [tabMenu, setTabMenu] = useState<TabMenuState>({ open: false });
	const [revealedWorkspacePath, setRevealedWorkspacePath] = useState('');
	const [archiveSuggestionCursor, setArchiveSuggestionCursor] = useState(-1);
	const [archiveExplorerStates, setArchiveExplorerStates] = useState<Record<string, ArchiveExplorerState>>({});
	const [archiveBrowseResponse, setArchiveBrowseResponse] = useState<ArchiveBrowseResponse | null>(null);
	const [archiveBrowseLoading, setArchiveBrowseLoading] = useState(false);
	const [archiveBrowseError, setArchiveBrowseError] = useState('');
	const [archivePreviewCacheTick, setArchivePreviewCacheTick] = useState(0);
	const [directoryAllowlist, setDirectoryAllowlist] = useState('');
	const [directoryBlocklist, setDirectoryBlocklist] = useState('');
	const [fileTypeAllowlist, setFileTypeAllowlist] = useState('');
	const [fileTypeBlocklist, setFileTypeBlocklist] = useState('');
	const [maxIndexedFileSize, setMaxIndexedFileSize] = useState('1048576');
	const lastFocusSyncAtRef = useRef(0);
	const focusSyncTimerRef = useRef<number | null>(null);
	const archiveSearchInputRef = useRef<HTMLInputElement | null>(null);
	const archiveLoadRequestRef = useRef(0);
	const archiveBrowseRequestRef = useRef(0);
	const archivePreviewContentRef = useRef<Map<string, string>>(new Map());
	const searchRequestRef = useRef(0);
	const helpRequestRef = useRef(0);
	const workbenchLoadRequestRef = useRef(0);

	const activeTab = useMemo(() => tabs.find((tab) => tab.id === activeTabId) ?? null, [activeTabId, tabs]);
	const selectedWorkspacePath = revealedWorkspacePath || activeTab?.path || '';
	const workspaceRoots = workbench?.workspace.roots ?? [];
	const activeRoot = useMemo(
		() => workbench?.workspace.roots.find((root) => root.id === workbench.workspace.activeRootId) ?? null,
		[workbench]
	);
	const selectedArchive = useMemo(
		() => workbench?.virtualFolders.find((folder) => folder.id === selectedArchiveId) ?? null,
		[selectedArchiveId, workbench]
	);
	const selectedAutoCategory = useMemo(
		() => workbench?.autoCategories.find((item) => item.id === selectedAutoCategoryId) ?? null,
		[selectedAutoCategoryId, workbench]
	);
	const currentArchiveSource = useMemo<ArchiveSourceRef | null>(() => {
		if (selectedArchiveId) return { kind: 'virtual_folder', id: selectedArchiveId };
		if (selectedAutoCategoryId) return { kind: 'auto_category', id: selectedAutoCategoryId };
		return null;
	}, [selectedArchiveId, selectedAutoCategoryId]);
	const currentArchiveSourceKey = useMemo(() => archiveSourceKey(currentArchiveSource), [currentArchiveSource]);
	const currentArchiveExplorerState = useMemo(
		() => (currentArchiveSource ? archiveExplorerStates[currentArchiveSourceKey] ?? createArchiveExplorerState(currentArchiveSource.kind) : null),
		[archiveExplorerStates, currentArchiveSource, currentArchiveSourceKey]
	);
	const selectedArchiveBrowseFile = useMemo(() => {
		if (!archiveBrowseResponse || !currentArchiveExplorerState) return null;
		const targetPath = currentArchiveExplorerState.selectedPath;
		return archiveBrowseResponse.files.find((item) => item.path === targetPath) ?? archiveBrowseResponse.files[0] ?? null;
	}, [archiveBrowseResponse, currentArchiveExplorerState]);
	const selectedArchivePrefetchedContent = useMemo(() => {
		if (!selectedArchiveBrowseFile) return null;
		return archivePreviewContentRef.current.get(selectedArchiveBrowseFile.path) ?? null;
	}, [archivePreviewCacheTick, selectedArchiveBrowseFile]);
	const normalizedArchiveFileType = useMemo(() => normalizeArchiveFileType(archiveFileTypeFilter), [archiveFileTypeFilter]);
	const archiveQueryNormalized = archiveQuery.trim();
	const isCommittedArchiveSearch =
		archiveCommittedSearch.query === archiveQueryNormalized &&
		archiveCommittedSearch.matchField === archiveMatchFilter &&
		archiveCommittedSearch.caseSensitive === archiveCaseSensitive &&
		archiveCommittedSearch.fileType === normalizedArchiveFileType;
	const displayedArchiveItems = useMemo(() => {
		const hasFilter = Boolean(archiveQueryNormalized || normalizedArchiveFileType);
		if (!hasFilter) {
			return archiveBaseItems;
		}

		if (isCommittedArchiveSearch) {
			if (archiveMatchFilter === 'all') {
				return archiveItems;
			}
			return archiveItems.filter((item) => item.matchKind === archiveMatchFilter);
		}

		if (archiveMatchFilter === 'content') {
			return [];
		}

		const scoredItems: Array<SearchHit & { _score: number }> = [];
		for (const item of archiveBaseItems) {
			const result = scoreLocalArchiveItem(
				archiveQueryNormalized,
				item,
				archiveMatchFilter,
				archiveCaseSensitive,
				normalizedArchiveFileType,
				workspaceRoots
			);
			if (result.score < 0 || !result.matchKind) continue;
			scoredItems.push({ ...item, matchKind: result.matchKind, _score: result.score });
		}
		scoredItems.sort((left, right) => {
			if (left._score === right._score) {
				return left.name.localeCompare(right.name);
			}
			return right._score - left._score;
		});
		return scoredItems.map(({ _score, ...item }) => item);
	}, [
		archiveBaseItems,
		archiveCaseSensitive,
		archiveItems,
		archiveMatchFilter,
		archiveQueryNormalized,
		isCommittedArchiveSearch,
		normalizedArchiveFileType,
		workspaceRoots
	]);
	const archiveBreadcrumbs = useMemo(() => {
		if (!archiveFolderPath) return [];
		const parts = archiveFolderPath.split('/').filter(Boolean);
		return parts.map((part, index) => ({
			label: part,
			path: parts.slice(0, index + 1).join('/')
		}));
	}, [archiveFolderPath]);
	const archiveDirectoryEntries = useMemo<ArchiveDirectoryEntry[]>(() => {
		const dirMap = new Map<string, number>();
		const fileEntries: Array<Extract<ArchiveDirectoryEntry, { kind: 'file' }>> = [];
		for (const item of displayedArchiveItems) {
			const normalizedPath = archiveVirtualPath(item, workspaceRoots);
			const segments = normalizedPath.split('/').filter(Boolean);
			const folderSegments = segments.slice(0, -1);
			if (archiveFolderPath) {
				const currentSegments = archiveFolderPath.split('/').filter(Boolean);
				const matchesPrefix = currentSegments.every((segment, index) => folderSegments[index] === segment);
				if (!matchesPrefix) continue;
				const remainder = folderSegments.slice(currentSegments.length);
				if (remainder.length > 0) {
					const nextDirPath = archiveJoinPath(archiveFolderPath, remainder[0]);
					dirMap.set(nextDirPath, (dirMap.get(nextDirPath) ?? 0) + 1);
					continue;
				}
			} else if (folderSegments.length > 0) {
				const nextDirPath = folderSegments[0];
				dirMap.set(nextDirPath, (dirMap.get(nextDirPath) ?? 0) + 1);
				continue;
			}
			fileEntries.push({ kind: 'file', item });
		}

		const directoryEntries: ArchiveDirectoryEntry[] = Array.from(dirMap.entries())
			.map(([path, count]) => ({
				kind: 'dir' as const,
				name: archiveBasename(path),
				path,
				count
			}))
			.sort((left, right) => left.name.localeCompare(right.name));

		const orderedFiles = [...fileEntries].sort((left, right) => left.item.name.localeCompare(right.item.name));
		return [...directoryEntries, ...orderedFiles];
	}, [archiveFolderPath, displayedArchiveItems, workspaceRoots]);
	const archiveDirectoryCount = useMemo(
		() => archiveDirectoryEntries.filter((entry) => entry.kind === 'dir').length,
		[archiveDirectoryEntries]
	);
	const archiveFileCount = useMemo(
		() => archiveDirectoryEntries.filter((entry) => entry.kind === 'file').length,
		[archiveDirectoryEntries]
	);
	const archiveSuggestions = useMemo<ArchiveSuggestion[]>(() => {
		const query = archiveQueryNormalized;
		if (!query) return [];

		const suggestions: ArchiveSuggestion[] = [];
		const seen = new Set<string>();
		const lowerQuery = normalizeSearchValue(query, archiveCaseSensitive);

		for (const entry of archiveDirectoryEntries) {
			if (entry.kind !== 'dir') continue;
			const label = entry.path;
			const haystack = normalizeSearchValue(label, archiveCaseSensitive);
			if (!(haystack.startsWith(lowerQuery) || haystack.includes(lowerQuery) || isSubsequence(haystack, lowerQuery))) continue;
			const id = `dir:${entry.path}`;
			if (seen.has(id)) continue;
			seen.add(id);
			suggestions.push({
				id,
				kind: 'directory',
				label: entry.name,
				description: entry.path,
				path: entry.path
			});
			if (suggestions.length >= 8) return suggestions;
		}

		for (const item of displayedArchiveItems) {
			const label = archiveVirtualPath(item, workspaceRoots);
			const haystack = normalizeSearchValue(label, archiveCaseSensitive);
			if (!(haystack.startsWith(lowerQuery) || haystack.includes(lowerQuery) || isSubsequence(haystack, lowerQuery))) continue;
			const id = `file:${item.path}`;
			if (seen.has(id)) continue;
			seen.add(id);
			suggestions.push({
				id,
				kind: 'file',
				label: item.name,
				description: label,
				item
			});
			if (suggestions.length >= 8) return suggestions;
		}

		const types = Array.from(new Set(archiveBaseItems.map((item) => (item.extension || extensionFromPath(item.path)).toLowerCase()).filter(Boolean))).sort();
		for (const extension of types) {
			const normalizedExtension = extension.replace(/^\./, '');
			const haystack = normalizeSearchValue(normalizedExtension, false);
			const normalizedNeedle = normalizeSearchValue(query, false).replace(/^\./, '');
			if (!(haystack.startsWith(normalizedNeedle) || haystack.includes(normalizedNeedle))) continue;
			const id = `type:${extension}`;
			if (seen.has(id)) continue;
			seen.add(id);
			suggestions.push({
				id,
				kind: 'type',
				label: extension,
				description: `筛选 ${extension} 文件`,
				value: extension
			});
			if (suggestions.length >= 8) return suggestions;
		}

		return suggestions;
	}, [archiveBaseItems, archiveCaseSensitive, archiveDirectoryEntries, archiveQueryNormalized, displayedArchiveItems, workspaceRoots]);
	const selectedHelpMeta = useMemo(
		() => workbench?.helpDocs.find((doc) => doc.id === selectedHelpDoc) ?? null,
		[selectedHelpDoc, workbench]
	);
	const archiveCursorEntry =
		archiveResultCursor >= 0 && archiveResultCursor < archiveDirectoryEntries.length
			? archiveDirectoryEntries[archiveResultCursor]
			: null;
	const hasRoots = Boolean(workbench?.workspace.roots.length);
	const currentFile = activeTab
		? ({
				id: activeTab.id,
				name: activeTab.name,
				path: activeTab.path,
				kind: 'file',
				rootId: activeTab.rootId,
				extension: activeTab.extension,
				virtualFolderIds: activeTab.virtualFolderIds
			} as TreeNode)
		: null;
	const markdownPreview = activeTab ? previewTabIds.includes(activeTab.id) : false;
	const renderUrl = activeTab && baseUrl ? buildAssetUrl('render', activeTab.path) : '';
	const sourceLabel =
		sourceMode === 'search' ? 'Search' : SOURCE_ITEMS.find((item) => item.id === sourceMode)?.label ?? 'Workspace';

	function pushToast(message: string, tone: ToastMessage['tone'] = 'info') {
		const id = `${Date.now()}-${Math.random().toString(16).slice(2)}`;
		setToasts((current) => [...current, { id, message, tone }]);
		window.setTimeout(() => {
			setToasts((current) => current.filter((item) => item.id !== id));
		}, 3600);
	}

	function dismissToast(id: string) {
		setToasts((current) => current.filter((item) => item.id !== id));
	}

	function fail(message: string) {
		setErrorText(message);
		setStatusText('操作失败');
		pushToast(message, 'error');
	}

	async function runTask<T>(task: () => Promise<T>) {
		setErrorText('');
		setPendingCount((count) => count + 1);
		try {
			return await task();
		} catch (error) {
			fail(describeError(error));
			return null;
		} finally {
			setPendingCount((count) => Math.max(0, count - 1));
		}
	}

	function currentArchiveSearch(query = archiveQuery): ArchiveCommittedSearch {
		return {
			query: query.trim(),
			matchField: archiveMatchFilter,
			caseSensitive: archiveCaseSensitive,
			fileType: normalizedArchiveFileType
		};
	}

	function updateArchiveExplorerState(source: ArchiveSourceRef, updater: Partial<ArchiveExplorerState> | ((current: ArchiveExplorerState) => ArchiveExplorerState)) {
		setArchiveExplorerStates((current) => {
			const key = archiveSourceKey(source);
			const base = current[key] ?? createArchiveExplorerState(source.kind);
			const next = typeof updater === 'function' ? updater(base) : { ...base, ...updater };
			return { ...current, [key]: next };
		});
	}

	function openArchiveExplorer() {
		if (!currentArchiveSource) return;
		updateArchiveExplorerState(currentArchiveSource, (current) => ({
			...current,
			folderPath: archiveFolderPath || current.folderPath,
			query: archiveQuery.trim() || current.query,
			searchMode: archiveSearchModeFromLegacy(archiveMatchFilter),
			sortBy: archiveSortByFromLegacy(),
			sortDirection: archiveSortDirectionFromLegacy(),
			groupView: archiveGroupViewFromLegacy(),
			cursor: 0,
			expanded: true
		}));
		setSourceMode('archives');
	}

	function collapseArchiveExplorer() {
		if (!currentArchiveSource) return;
		updateArchiveExplorerState(currentArchiveSource, { expanded: false });
	}

	function setCurrentArchiveExplorerState(updater: Partial<ArchiveExplorerState> | ((current: ArchiveExplorerState) => ArchiveExplorerState)) {
		if (!currentArchiveSource) return;
		updateArchiveExplorerState(currentArchiveSource, updater);
	}

	async function fetchArchiveBrowse(source: ArchiveSourceRef, state: ArchiveExplorerState) {
		const requestId = ++archiveBrowseRequestRef.current;
		setArchiveBrowseLoading(true);
		setArchiveBrowseError('');
		try {
			const request = createArchiveBrowseRequest(source, state) as unknown as Record<string, unknown>;
			const response = await rpc<ArchiveBrowseResponse>('archive.browse', request);
			if (requestId !== archiveBrowseRequestRef.current) return;
			setArchiveBrowseResponse(response);
			setArchiveBrowseError('');
			setArchiveExplorerStates((current) => {
				const key = archiveSourceKey(source);
				const existing = current[key] ?? state;
				const hasCurrentSelection = response.files.some((item) => item.path === existing.selectedPath);
				return {
					...current,
					[key]: {
						...existing,
						selectedPath: hasCurrentSelection ? existing.selectedPath : response.files[0]?.path || ''
					}
				};
			});
			setStatusText('归档浏览已更新');
		} catch (error) {
			if (requestId !== archiveBrowseRequestRef.current) return;
			const message = describeError(error, '归档浏览当前不可用。');
			setArchiveBrowseError(message);
			setStatusText('归档浏览加载失败');
		} finally {
			if (requestId === archiveBrowseRequestRef.current) {
				setArchiveBrowseLoading(false);
			}
		}
	}

	function setArchiveExplorerQuery(value: string) {
		setCurrentArchiveExplorerState((current) => ({ ...current, query: value, cursor: 0 }));
	}

	function setArchiveExplorerSearchMode(mode: 'quick' | 'content') {
		setCurrentArchiveExplorerState((current) => ({ ...current, searchMode: mode, cursor: 0 }));
	}

	function setArchiveExplorerSortBy(sortBy: ArchiveExplorerState['sortBy']) {
		setCurrentArchiveExplorerState((current) => ({ ...current, sortBy, cursor: 0 }));
	}

	function setArchiveExplorerSortDirection(sortDirection: ArchiveExplorerState['sortDirection']) {
		setCurrentArchiveExplorerState((current) => ({ ...current, sortDirection, cursor: 0 }));
	}

	function setArchiveExplorerGroupView(groupView: ArchiveExplorerState['groupView']) {
		setCurrentArchiveExplorerState({ groupView });
	}

	function setArchiveExplorerFolderPath(folderPath: string) {
		setCurrentArchiveExplorerState((current) => ({ ...current, folderPath, cursor: 0 }));
	}

	function setArchiveExplorerSelectedPath(selectedPath: string) {
		setCurrentArchiveExplorerState({ selectedPath });
	}

	function setArchiveExplorerNavTab(leftTab: ArchiveExplorerState['leftTab']) {
		setCurrentArchiveExplorerState({ leftTab });
	}

	function goToNextArchiveBrowsePage() {
		if (!currentArchiveExplorerState || archiveBrowseResponse?.nextCursor == null || archiveBrowseResponse.nextCursor < 0) return;
		setCurrentArchiveExplorerState((current) => ({ ...current, cursor: archiveBrowseResponse.nextCursor, selectedPath: '' }));
	}

	function goToPreviousArchiveBrowsePage() {
		if (!currentArchiveExplorerState) return;
		setCurrentArchiveExplorerState((current) => ({ ...current, cursor: Math.max(0, current.cursor - 120), selectedPath: '' }));
	}
	function shouldRunArchiveIndexSearch(request: ArchiveCommittedSearch) {
		return Boolean(request.query || request.fileType);
	}

	async function loadArchive(folderId: string, request: ArchiveCommittedSearch = currentArchiveSearch('')) {
		const requestId = ++archiveLoadRequestRef.current;
		searchRequestRef.current += 1;
		helpRequestRef.current += 1;
		setSelectedArchiveId(folderId);
		setSelectedAutoCategoryId('');
		setArchiveFolderPath('');
		setSearchArchiveId(folderId);
		setSourceMode('archives');
		updateArchiveExplorerState({ kind: 'virtual_folder', id: folderId }, (current) => ({
			...current,
			leftTab: 'folders',
			expanded: currentArchiveExplorerState?.expanded ?? current.expanded
		}));
		setArchiveQuery(request.query);
		const response = await runTask(async () => {
			if (shouldRunArchiveIndexSearch(request)) {
				const [baseItems, items] = await Promise.all([
					rpc<SearchHit[]>('archive.list', { folderId }),
					rpc<SearchHit[]>('search.query', {
						keyword: request.query,
						rootId: searchRootId || activeRoot?.id || '',
						virtualFolderId: folderId,
						autoCategory: '',
						matchField: request.matchField,
						caseSensitive: request.caseSensitive,
						fileType: request.fileType
					})
				]);
				return { baseItems, items };
			}
			const baseItems = await rpc<SearchHit[]>('archive.list', { folderId });
			return { baseItems, items: baseItems };
		});
		if (!response || requestId !== archiveLoadRequestRef.current) return;
		setArchiveBaseItems(response.baseItems);
		if (!shouldRunArchiveIndexSearch(request)) {
			setArchiveCommittedSearch({ query: '', matchField: 'all', caseSensitive: false, fileType: '' });
		} else {
			setArchiveCommittedSearch(request);
		}
		setArchiveItems(response.items);
		setArchiveResultCursor(response.items.length ? 0 : -1);
		setStatusText(
			shouldRunArchiveIndexSearch(request)
				? `归档搜索完成，共 ${response.items.length} 个结果`
				: `归档已载入，共 ${response.items.length} 个文件`
		);
	}

	async function loadAutoCategory(category: AutoCategory, request: ArchiveCommittedSearch = currentArchiveSearch('')) {
		const requestId = ++archiveLoadRequestRef.current;
		searchRequestRef.current += 1;
		helpRequestRef.current += 1;
		setSelectedAutoCategoryId(category.id);
		setSelectedArchiveId('');
		setArchiveFolderPath('');
		setSearchArchiveId('');
		setSourceMode('archives');
		updateArchiveExplorerState({ kind: 'auto_category', id: category.id }, (current) => ({
			...current,
			leftTab: 'categories',
			expanded: currentArchiveExplorerState?.expanded ?? current.expanded
		}));
		setArchiveQuery(request.query);
		setStatusText(shouldRunArchiveIndexSearch(request) ? `正在搜索 ${category.label}...` : `正在载入 ${category.label}...`);
		const response = await runTask(async () => {
			const baseRequest = {
				keyword: '',
				rootId: searchRootId || activeRoot?.id || '',
				virtualFolderId: '',
				autoCategory: category.extension
			};
			if (shouldRunArchiveIndexSearch(request)) {
				const [baseItems, items] = await Promise.all([
					rpc<SearchHit[]>('search.query', baseRequest),
					rpc<SearchHit[]>('search.query', {
						keyword: request.query,
						rootId: searchRootId || activeRoot?.id || '',
						virtualFolderId: '',
						autoCategory: category.extension,
						matchField: request.matchField,
						caseSensitive: request.caseSensitive,
						fileType: request.fileType
					})
				]);
				return { baseItems, items };
			}
			const baseItems = await rpc<SearchHit[]>('search.query', baseRequest);
			return { baseItems, items: baseItems };
		});
		if (!response || requestId !== archiveLoadRequestRef.current) return;
		setArchiveBaseItems(response.baseItems);
		if (!shouldRunArchiveIndexSearch(request)) {
			setArchiveCommittedSearch({ query: '', matchField: 'all', caseSensitive: false, fileType: '' });
		} else {
			setArchiveCommittedSearch(request);
		}
		setArchiveItems(response.items);
		setArchiveResultCursor(response.items.length ? 0 : -1);
		setStatusText(
			shouldRunArchiveIndexSearch(request) ? `${category.label} 搜索完成，共 ${response.items.length} 个结果` : `${category.label} 已载入`
		);
	}

	async function openArchiveFolderExplorer(folder: VirtualFolder) {
		updateArchiveExplorerState({ kind: 'virtual_folder', id: folder.id }, (current) => ({
			...current,
			leftTab: 'folders',
			cursor: 0,
			expanded: true
		}));
		await loadArchive(folder.id, { query: '', matchField: 'all', caseSensitive: false, fileType: '' });
	}

	async function openArchiveCategoryExplorer(category: AutoCategory) {
		updateArchiveExplorerState({ kind: 'auto_category', id: category.id }, (current) => ({
			...current,
			leftTab: 'categories',
			cursor: 0,
			expanded: true
		}));
		await loadAutoCategory(category, { query: '', matchField: 'all', caseSensitive: false, fileType: '' });
	}
	async function runArchiveSearch(keyword = archiveQuery) {
		const request = currentArchiveSearch(keyword);
		setArchiveQuery(request.query);
		setSourceMode('archives');
		if (!selectedArchiveId && !selectedAutoCategoryId) {
			setArchiveItems([]);
			setArchiveBaseItems([]);
			setArchiveCommittedSearch({ query: '', matchField: 'all', caseSensitive: false, fileType: '' });
			setArchiveResultCursor(-1);
			setStatusText('先选择一个自动分类或手动归档，再开始搜索。');
			return;
		}
		if (selectedArchiveId) {
			await loadArchive(selectedArchiveId, request);
			return;
		}
		if (selectedAutoCategoryId) {
			const category = workbench?.autoCategories.find((item) => item.id === selectedAutoCategoryId);
			if (category) {
				await loadAutoCategory(category, request);
				return;
			}
		}
	}

	async function applyArchiveSuggestion(suggestion: ArchiveSuggestion) {
		if (suggestion.kind === 'directory') {
			setArchiveFolderPath(suggestion.path);
			setArchiveSuggestionCursor(-1);
			setStatusText(`已进入 ${suggestion.path}`);
			return;
		}
		if (suggestion.kind === 'type') {
			setArchiveFileTypeFilter(suggestion.value);
			setArchiveSuggestionCursor(-1);
			await runArchiveSearch(archiveQuery);
			return;
		}
		setArchiveSuggestionCursor(-1);
		await openDocument(suggestion.item);
	}

	async function runSearch(keyword = searchQuery) {
		archiveLoadRequestRef.current += 1;
		helpRequestRef.current += 1;
		const requestId = ++searchRequestRef.current;
		const normalized = keyword.trim();
		setSearchQuery(keyword);
		setSourceMode('search');
		const normalizedSearchType = normalizeArchiveFileType(searchFileTypeFilter);
		if (!normalized && !normalizedSearchType) {
			setSearchResults([]);
			setStatusText('搜索条件为空');
			return;
		}
		const results = await runTask(() =>
			rpc<SearchHit[]>('search.query', {
				keyword: normalized,
				rootId: searchRootId,
				virtualFolderId: sourceMode === 'archives' ? searchArchiveId : '',
				autoCategory: '',
				matchField: searchMatchField,
				caseSensitive: searchCaseSensitive,
				fileType: normalizedSearchType
			})
		);
		if (!results || requestId !== searchRequestRef.current) return;
		setSearchResults(results);
		setStatusText(`找到 ${results.length} 个结果`);
	}

	async function loadWorkbench(resetTabs = false, reloadContext = true): Promise<WorkbenchState | null> {
		archiveLoadRequestRef.current += 1;
		searchRequestRef.current += 1;
		helpRequestRef.current += 1;
		const requestId = ++workbenchLoadRequestRef.current;
		const nextWorkbench = await runTask(() => rpc<WorkbenchState>('workspace.get'));
		if (!nextWorkbench || requestId !== workbenchLoadRequestRef.current) return null;
		const normalized = normalizeWorkbenchState(nextWorkbench);
		setWorkbench(normalized);
		applyTheme(normalized.theme);
		setTheme(normalizeTheme(normalized.theme));

		setSearchRootId((current) =>
			normalized.workspace.roots.some((root) => root.id === current)
				? current
				: normalized.workspace.activeRootId || ''
		);
		setSelectedArchiveId((current) => {
			if (!current) return normalized.virtualFolders[0]?.id || '';
			return normalized.virtualFolders.some((item) => item.id === current) ? current : '';
		});
		setSelectedAutoCategoryId((current) => {
			if (!current) return normalized.autoCategories[0]?.id || '';
			return normalized.autoCategories.some((item) => item.id === current) ? current : normalized.autoCategories[0]?.id || '';
		});
		setSelectedHelpDoc((current) => {
			if (!current) return normalized.helpDocs[0]?.id || '';
			return normalized.helpDocs.some((item) => item.id === current) ? current : normalized.helpDocs[0]?.id || '';
		});

		if (resetTabs) {
			setTabs([]);
			setActiveTabId('');
			setPreviewTabIds([]);
		}

		if (!reloadContext) {
			return normalized;
		}

		if (selectedArchiveId && normalized.virtualFolders.some((item) => item.id === selectedArchiveId)) {
			await loadArchive(selectedArchiveId, archiveCommittedSearch.query || archiveCommittedSearch.fileType ? archiveCommittedSearch : currentArchiveSearch(''));
		} else if (selectedAutoCategoryId && normalized.autoCategories.some((item) => item.id === selectedAutoCategoryId)) {
			const category = normalized.autoCategories.find((item) => item.id === selectedAutoCategoryId);
			if (category) {
				await loadAutoCategory(category, archiveCommittedSearch.query || archiveCommittedSearch.fileType ? archiveCommittedSearch : currentArchiveSearch(''));
			}
		} else if (!normalized.virtualFolders.length && !normalized.autoCategories.length) {
			setArchiveItems([]);
			setSearchArchiveId('');
		}

		if (sourceMode === 'search' && searchQuery.trim()) {
			await runSearch(searchQuery);
		}
		return normalized;
	}

	async function refreshSnapshot(resetTabs = false) {
		return loadWorkbench(resetTabs, false);
	}

	async function refreshVisibleState(resetTabs = false) {
		const next = await refreshSnapshot(resetTabs);
		await refreshCurrentContext(next);
		return next;
	}

	async function refreshCurrentContext(snapshotOverride?: WorkbenchState | null) {
		const snapshot = snapshotOverride ?? workbench;
		if (sourceMode === 'archives') {
			if (selectedArchiveId) {
				await loadArchive(selectedArchiveId, archiveCommittedSearch);
				return;
			}
			if (selectedAutoCategoryId) {
				const category = snapshot?.autoCategories.find((item) => item.id === selectedAutoCategoryId);
				if (category) {
					await loadAutoCategory(category, archiveCommittedSearch);
				}
				return;
			}
		}
		if (sourceMode === 'search' && searchQuery.trim()) {
			await runSearch(searchQuery);
		}
	}

	
	async function loadHelpDoc(docId: string, force = false) {
		const requestId = ++helpRequestRef.current;
		if (!docId) {
			setLoadedHelpDocId('');
			setHelpContent('');
			setHelpError('');
			setHelpLoading(false);
			return;
		}
		if (!force && loadedHelpDocId === docId && helpContent) return;

		setHelpLoading(true);
		setHelpError('');
		try {
			const content = await rpc<string>('help.read', { docId });
			if (requestId !== helpRequestRef.current) return;
			setHelpContent(content || '');
			setLoadedHelpDocId(docId);
			setStatusText('帮助内容已加载');
		} catch (error) {
			if (requestId !== helpRequestRef.current) return;
			const message = describeError(error, '帮助文档当前不可读。');
			setHelpContent('');
			setLoadedHelpDocId('');
			setHelpError(message);
			pushToast(message, 'error');
		} finally {
			if (requestId === helpRequestRef.current) {
				setHelpLoading(false);
			}
		}
	}

	async function openDocument(item: TreeNode | SearchHit) {
		const next = toOpenTab(item);
		setTabs((current) => (current.some((tab) => tab.id === next.id) ? current : [...current, next]));
		setActiveTabId(next.id);
		setRevealedWorkspacePath(next.path);
		setStatusText(`已打开 ${next.name}`);
		void rpc<boolean>('recent.record', { path: next.path }).catch(() => undefined);
	}

	async function ensureWorkspacePathVisible(targetPath: string, rootId: string, snapshotOverride?: WorkbenchState | null) {
		let snapshot = snapshotOverride ?? workbench;
		if (!snapshot) return;
		const root = snapshot.workspace.roots.find((item) => item.id === rootId);
		if (!root) return;

		const segments = relativePathSegments(root.path, targetPath);
		const ancestorSegments = segments.slice(0, -1);
		const targetKey = normalizePathKey(targetPath);
		let currentPath = root.path;

		snapshot = { ...snapshot, physicalRoots: setNodeExpanded(snapshot.physicalRoots, currentPath, true) };
		setWorkbench(snapshot);

		for (const segment of ancestorSegments) {
			let currentNode = findNodeByPath(snapshot.physicalRoots, currentPath);
			if (!currentNode) return;
			const currentNodePath = currentNode.path;

			let nextPath = joinPathSegments(currentPath, segment);
			let nextNode = currentNode.children?.find((child) => normalizePathKey(child.path) === normalizePathKey(nextPath));

			if (!nextNode) {
				const children = await runTask(() => rpc<TreeNode[]>('fs.listDir', { path: currentNodePath, rootId }));
				if (children === null) return;
				snapshot = {
					...snapshot,
					physicalRoots: replaceNodeChildren(snapshot.physicalRoots, currentNodePath, children, true)
				};
				setWorkbench(snapshot);
				currentNode = findNodeByPath(snapshot.physicalRoots, currentPath);
				if (!currentNode) return;
				nextNode = currentNode.children?.find((child) => normalizePathKey(child.path) === normalizePathKey(nextPath));
			}

			if (!nextNode) return;
			if (!nextNode.expanded) {
				snapshot = { ...snapshot, physicalRoots: setNodeExpanded(snapshot.physicalRoots, nextNode.path, true) };
				setWorkbench(snapshot);
			}
			currentPath = nextNode.path;
		}

		const parentNode = findNodeByPath(snapshot.physicalRoots, currentPath);
		const hasVisibleTarget = parentNode?.children?.some((child) => normalizePathKey(child.path) === targetKey);
		if (!hasVisibleTarget && parentNode) {
			const children = await runTask(() => rpc<TreeNode[]>('fs.listDir', { path: parentNode.path, rootId }));
			if (children === null) return;
			snapshot = {
				...snapshot,
				physicalRoots: replaceNodeChildren(snapshot.physicalRoots, parentNode.path, children, true)
			};
			setWorkbench(snapshot);
		}
	}

	async function revealInWorkspace(item: Pick<SearchHit, 'path' | 'rootId' | 'name'>) {
		if (!workbench) return;
		let snapshot = workbench;
		if (item.rootId && item.rootId !== workbench.workspace.activeRootId) {
			const ok = await runTask(() => rpc<boolean>('workspace.setActive', { rootId: item.rootId }));
			if (ok === null) return;
			setSearchRootId(item.rootId);
			const next = await refreshSnapshot();
			if (!next) return;
			snapshot = next;
		}
		setSourceMode('workspace');
		await ensureWorkspacePathVisible(item.path, item.rootId || snapshot.workspace.activeRootId, snapshot);
		await openDocument({
			path: item.path,
			name: item.name,
			rootId: item.rootId,
			virtualFolderIds: [],
			matchKind: 'path',
			extension: extensionFromPath(item.path)
		});
		setStatusText(`已切换到工作区并定位 ${item.name}`);
	}

	function closeTab(id: string) {
		setTabs((current) => {
			const next = current.filter((tab) => tab.id !== id);
			if (activeTabId === id) {
				setActiveTabId(lastItem(next)?.id || '');
			}
			return next;
		});
		setPreviewTabIds((current) => current.filter((item) => item !== id));
	}

	function closeOtherTabs(id: string) {
		setTabs((current) => current.filter((tab) => tab.id === id));
		setPreviewTabIds((current) => current.filter((item) => item === id));
		setActiveTabId(id);
	}

	function closeTabsToRight(id: string) {
		const index = tabs.findIndex((tab) => tab.id === id);
		if (index === -1) return;
		const allowed = new Set(tabs.slice(0, index + 1).map((tab) => tab.id));
		const nextTabs = tabs.filter((tab) => allowed.has(tab.id));
		setTabs(nextTabs);
		setPreviewTabIds((current) => current.filter((item) => allowed.has(item)));
		if (!nextTabs.some((tab) => tab.id === activeTabId)) {
			setActiveTabId(id);
		}
	}

	function closeAllTabs() {
		setTabs([]);
		setPreviewTabIds([]);
		setActiveTabId('');
	}

	function resetSourceSidebarWidth() {
		setSourceSidebarWidth(DEFAULT_SOURCE_SIDEBAR_WIDTH);
	}

	function resetBrowserPaneWidth() {
		setBrowserPaneWidth(DEFAULT_BROWSER_PANE_WIDTH);
	}

	function resetPreviewPaneSize() {
		setPreviewPaneSize(DEFAULT_PREVIEW_PANE_SIZE);
	}

	function closeTabsForPath(path: string) {
		const prefix = `${path}/`;
		setTabs((current) => {
			const next = current.filter((tab) => tab.path !== path && !tab.path.startsWith(prefix));
			if (activeTabId === path || activeTabId.startsWith(prefix)) {
				setActiveTabId(lastItem(next)?.id || '');
			}
			return next;
		});
		setPreviewTabIds((current) => current.filter((tabId) => tabId !== path && !tabId.startsWith(prefix)));
	}

	function updateDirty(path: string, dirty: boolean) {
		setTabs((current) => current.map((tab) => (tab.path === path ? { ...tab, dirty } : tab)));
	}

	function updateTabPath(oldPath: string, nextPath: string, nextName: string) {
		const nextExtension = extensionFromPath(nextPath);
		setTabs((current) =>
			current.map((tab) =>
				tab.path === oldPath
					? { ...tab, id: nextPath, path: nextPath, name: nextName, extension: nextExtension, kind: classifyExtension(nextExtension) }
					: tab
			)
		);
		setPreviewTabIds((current) => current.map((tabId) => (tabId === oldPath ? nextPath : tabId)));
		if (activeTabId === oldPath) {
			setActiveTabId(nextPath);
		}
	}

	async function openWorkspace() {
		const selected = await runTask(() => pickDirectory(activeRoot?.path || ''));
		if (!selected) return;
		const ok = await runTask(() => rpc<boolean>('workspace.open', { path: selected }));
		if (ok === null) return;
		await refreshSnapshot(true);
		setSourceMode('workspace');
		pushToast('工作区已切换', 'success');
		setStatusText('工作区已切换');
	}

	async function openRecentWorkspace(path: string) {
		const ok = await runTask(() => rpc<boolean>('workspace.open', { path }));
		if (ok === null) return;
		await refreshSnapshot(true);
		setSourceMode('workspace');
		pushToast('最近工作区已打开', 'success');
		setStatusText('最近工作区已打开');
	}

	async function addRoot() {
		const selected = await runTask(() => pickDirectory(activeRoot?.path || ''));
		if (!selected) return;
		const ok = await runTask(() => rpc<boolean>('workspace.addRoot', { path: selected }));
		if (ok === null) return;
		await refreshVisibleState();
		pushToast('根目录已添加', 'success');
		setStatusText('根目录已添加');
	}

	function openHelpCenter() {
		setSourceMode('help');
		if (workbench?.helpDocs[0]?.id) {
			setSelectedHelpDoc(workbench.helpDocs[0].id);
		}
	}

	async function setActiveRoot(rootId: string) {
		const ok = await runTask(() => rpc<boolean>('workspace.setActive', { rootId }));
		if (ok === null) return;
		setSearchRootId(rootId);
		const next = await refreshSnapshot();
		const nextRoot = next?.workspace.roots.find((root) => root.id === rootId) ?? null;
		setSourceMode('workspace');
		if (nextRoot) {
			setRevealedWorkspacePath(nextRoot.path);
			await ensureWorkspacePathVisible(nextRoot.path, rootId, next);
		}
		pushToast('活动根目录已更新', 'success');
		setStatusText('活动根目录已更新');
	}

	async function setDefaultRoot(rootId: string) {
		const ok = await runTask(() => rpc<boolean>('workspace.setDefault', { rootId }));
		if (ok === null) return;
		await refreshVisibleState();
		pushToast('默认根目录已更新', 'success');
		setStatusText('默认根目录已更新');
	}

	function openInputDialog(config: Extract<DialogState, { kind: 'input' }>) {
		setDialog(config);
	}

	function openConfirmDialog(config: Extract<DialogState, { kind: 'confirm' }>) {
		setDialog(config);
	}

	function requestCreatePhysical(kind: 'file' | 'folder', parentPath = '') {
		if (!hasRoots) {
			fail('请先打开工作区或添加根目录，再继续新建。');
			return;
		}
		openInputDialog({
			kind: 'input',
			title: kind === 'file' ? '新建文件' : '新建目录',
			description: kind === 'file' ? '输入文件名后立即创建。' : '输入目录名后立即创建。',
			confirmLabel: '创建',
			initialValue: '',
			placeholder: kind === 'file' ? 'notes.md' : 'docs',
			submitting: false,
			action: async (value) => {
				const name = value.trim();
				if (!name) {
					fail('名称不能为空。');
					return false;
				}
				const ok = await runTask(() =>
					kind === 'file'
						? rpc<boolean>('fs.createFile', { parentDir: parentPath, name })
						: rpc<boolean>('fs.createFolder', { parentDir: parentPath, name })
				);
				if (ok === null) return false;
				await refreshVisibleState();
				pushToast(kind === 'file' ? '文件已创建' : '目录已创建', 'success');
				setStatusText(kind === 'file' ? '文件已创建' : '目录已创建');
				return true;
			}
		});
	}

	function requestRenameNode(node: TreeNode) {
		openInputDialog({
			kind: 'input',
			title: '重命名',
			description: `更新 ${node.name} 的名称。`,
			confirmLabel: '保存',
			initialValue: node.name,
			placeholder: node.name,
			submitting: false,
			action: async (value) => {
				const nextName = value.trim();
				if (!nextName || nextName === node.name) return true;
				const nextPath = renamePath(node.path, nextName);
				const ok = await runTask(() => rpc<boolean>('fs.rename', { path: node.path, name: nextName }));
				if (ok === null) return false;
				updateTabPath(node.path, nextPath, nextName);
				await refreshVisibleState();
				pushToast('名称已更新', 'success');
				setStatusText('名称已更新');
				return true;
			}
		});
	}

	function requestDeleteNode(node: TreeNode) {
		openConfirmDialog({
			kind: 'confirm',
			title: '删除项目',
			description: `确认删除 ${node.name}。该操作会进入软删除流程。`,
			confirmLabel: '删除',
			danger: true,
			submitting: false,
			action: async () => {
				const ok = await runTask(() => rpc<boolean>('fs.delete', { path: node.path }));
				if (ok === null) return false;
				closeTabsForPath(node.path);
				await refreshVisibleState();
				pushToast('项目已删除', 'success');
				setStatusText('项目已删除');
				return true;
			}
		});
	}

	function requestRemoveRoot(rootId: string, label: string) {
		openConfirmDialog({
			kind: 'confirm',
			title: '移除根目录',
			description: `移除后不会删除磁盘内容，但会从当前工作区中移出 ${label}。`,
			confirmLabel: '移除',
			danger: true,
			submitting: false,
			action: async () => {
				const ok = await runTask(() => rpc<boolean>('workspace.removeRoot', { rootId }));
				if (ok === null) return false;
				await refreshSnapshot(true);
				pushToast('根目录已移除', 'success');
				setStatusText('根目录已移除');
				return true;
			}
		});
	}

	function requestCreateArchive() {
		openInputDialog({
			kind: 'input',
			title: '新建归档',
			description: '归档用于组织视图，不会复制物理文件。',
			confirmLabel: '创建',
			initialValue: '',
			placeholder: 'Design Notes',
			submitting: false,
			action: async (value) => {
				const name = value.trim();
				if (!name) {
					fail('归档名称不能为空。');
					return false;
				}
				const folder = await runTask(() => rpc<{ id: string }>('archive.create', { name }));
				if (!folder) return false;
				await refreshVisibleState();
				if (folder.id) {
					await loadArchive(folder.id, { query: '', matchField: 'all', caseSensitive: false, fileType: '' });
				}
				pushToast('归档已创建', 'success');
				setStatusText('归档已创建');
				return true;
			}
		});
	}

	function requestRenameArchive(folderId: string, currentName: string) {
		openInputDialog({
			kind: 'input',
			title: '重命名归档',
			description: '只修改归档名称，不影响原文件位置。',
			confirmLabel: '保存',
			initialValue: currentName,
			placeholder: currentName,
			submitting: false,
			action: async (value) => {
				const name = value.trim();
				if (!name || name === currentName) return true;
				const ok = await runTask(() => rpc<boolean>('archive.rename', { folderId, name }));
				if (ok === null) return false;
				await refreshVisibleState();
				pushToast('归档名称已更新', 'success');
				setStatusText('归档名称已更新');
				return true;
			}
		});
	}

	function requestDeleteArchive(folderId: string, name: string) {
		openConfirmDialog({
			kind: 'confirm',
			title: '删除归档',
			description: `归档 ${name} 会被移除，但不会删除物理文件。`,
			confirmLabel: '删除',
			danger: true,
			submitting: false,
			action: async () => {
				const ok = await runTask(() => rpc<boolean>('archive.delete', { folderId }));
				if (ok === null) return false;
				if (selectedArchiveId === folderId) {
					setSelectedArchiveId('');
					setArchiveItems([]);
				}
				await refreshVisibleState();
				pushToast('归档已删除', 'success');
				setStatusText('归档已删除');
				return true;
			}
		});
	}

	function requestCreateArchiveFile() {
		if (!selectedArchiveId) {
			fail('请先选择一个归档，再创建归档文件。');
			return;
		}
		if (!hasRoots) {
			fail('当前没有活动根目录，无法创建归档文件。');
			return;
		}
		openInputDialog({
			kind: 'input',
			title: '新建归档文件',
			description: '归档文件会落到活动根目录，并自动附加到当前归档。',
			confirmLabel: '创建',
			initialValue: '',
			placeholder: 'summary.md',
			submitting: false,
			action: async (value) => {
				const name = value.trim();
				if (!name) {
					fail('文件名不能为空。');
					return false;
				}
				const item = await runTask(() =>
					rpc<SearchHit>('archive.createFile', {
						folderId: selectedArchiveId,
						name,
						parentPath: selectedArchive?.preferredParentPath || activeRoot?.path || '',
						preferredRootId: selectedArchive?.preferredRootId || activeRoot?.id || ''
					})
				);
				if (!item) return false;
				await refreshSnapshot();
				await loadArchive(selectedArchiveId, currentArchiveSearch(archiveQuery));
				await openDocument(item);
				pushToast('归档文件已创建', 'success');
				setStatusText('归档文件已创建');
				return true;
			}
		});
	}

	async function attachActiveToArchive() {
		if (!activeTab || !selectedArchiveId) {
			fail('请先选中文件和归档。');
			return;
		}
		const ok = await runTask(() =>
			rpc<boolean>('archive.attach', { path: activeTab.path, folderId: selectedArchiveId })
		);
		if (ok === null) return;
		await loadArchive(selectedArchiveId, currentArchiveSearch(archiveQuery));
		await refreshSnapshot();
		pushToast('文件已附加到归档', 'success');
		setStatusText('文件已附加到归档');
	}

	async function detachFromArchive(path: string) {
		if (!selectedArchiveId) {
			fail('请先选择一个归档。');
			return;
		}
		const ok = await runTask(() =>
			rpc<boolean>('archive.detach', { path, folderId: selectedArchiveId })
		);
		if (ok === null) return;
		await loadArchive(selectedArchiveId, currentArchiveSearch(archiveQuery));
		await refreshSnapshot();
		pushToast('文件已从归档移除', 'success');
		setStatusText('文件已从归档移除');
	}

	async function attachSearchResultToSelectedArchive(item: SearchHit) {
		if (!selectedArchiveId) {
			fail('请先选择一个归档，再附加搜索结果。');
			return;
		}
		const ok = await runTask(() => rpc<boolean>('archive.attach', { path: item.path, folderId: selectedArchiveId }));
		if (ok === null) return;
		await loadArchive(selectedArchiveId, currentArchiveSearch(archiveQuery));
		pushToast('搜索结果已附加到归档', 'success');
		setStatusText('搜索结果已附加到归档');
	}

	async function openExternal(path: string) {
		const ok = await runTask(() => rpc<boolean>('fs.openExternal', { path }));
		if (ok === null) return;
		pushToast('已交给系统应用打开', 'success');
		setStatusText('已交给系统应用打开');
	}

	async function toggleTreeNode(node: TreeNode) {
		if (node.kind === 'file') {
			await openDocument(node);
			return;
		}
		if (node.expanded) {
			setWorkbench((current) =>
				current ? { ...current, physicalRoots: toggleNodeExpanded(current.physicalRoots, node.path) } : current
			);
			return;
		}
		if (node.children && node.children.length > 0) {
			setWorkbench((current) =>
				current ? { ...current, physicalRoots: toggleNodeExpanded(current.physicalRoots, node.path) } : current
			);
			return;
		}
		setStatusText(`正在展开 ${node.name}...`);
		const children = await runTask(() =>
			rpc<TreeNode[]>('fs.listDir', { path: node.path, rootId: node.rootId })
		);
		if (children === null) return;
		setWorkbench((current) =>
			current
				? { ...current, physicalRoots: replaceNodeChildren(current.physicalRoots, node.path, children, true) }
				: current
		);
		setStatusText(`已展开 ${node.name}`);
	}

	async function loadSettings() {
		setSettingsLoading(true);
		setSettingsError('');
		try {
			const config = await rpc<any>('settings.get');
			setSettingsConfig(config);
			const nextTheme = normalizeTheme(config?.viewSettings?.theme || workbench?.theme);
			setTheme(nextTheme);
			setDirectoryAllowlist((config?.policy?.directoryAllowlist ?? []).join('\n'));
			setDirectoryBlocklist((config?.policy?.directoryBlocklist ?? []).join('\n'));
			setFileTypeAllowlist((config?.policy?.fileTypeAllowlist ?? []).join(', '));
			setFileTypeBlocklist((config?.policy?.fileTypeBlocklist ?? []).join(', '));
			setMaxIndexedFileSize(String(config?.policy?.maxIndexedFileSize ?? 1048576));
		} catch (error) {
			setSettingsError(describeError(error, '设置读取失败。'));
		} finally {
			setSettingsLoading(false);
		}
	}

	async function saveSettings() {
		setSettingsSaving(true);
		setSettingsError('');
		try {
			const baseConfig = settingsConfig ?? (await rpc<any>('settings.get'));
			const nextConfig = {
				...baseConfig,
				viewSettings: { ...(baseConfig?.viewSettings ?? {}), theme },
				policy: {
					...(baseConfig?.policy ?? {}),
					directoryAllowlist: splitListValue(directoryAllowlist),
					directoryBlocklist: splitListValue(directoryBlocklist),
					fileTypeAllowlist: splitListValue(fileTypeAllowlist),
					fileTypeBlocklist: splitListValue(fileTypeBlocklist),
					maxIndexedFileSize: Math.max(1, Number.parseInt(maxIndexedFileSize, 10) || 1048576)
				}
			};
			const ok = await rpc<boolean>('settings.save', { config: nextConfig });
			if (!ok) {
				throw new Error('设置保存失败。');
			}
			setSettingsConfig(nextConfig);
			applyTheme(theme);
			await refreshVisibleState();
			setShowSettings(false);
			pushToast('设置已保存', 'success');
			setStatusText('设置已保存');
		} catch (error) {
			setSettingsError(describeError(error, '设置保存失败。'));
		} finally {
			setSettingsSaving(false);
		}
	}

	function startSourceSidebarResize(event: ReactMouseEvent<HTMLDivElement>) {
		event.preventDefault();
		const startX = event.clientX;
		const initialWidth = sourceSidebarWidth;
		function onMove(moveEvent: MouseEvent) {
			setSourceSidebarWidth(clamp(initialWidth + (moveEvent.clientX - startX), 208, 480));
		}
		function onUp() {
			window.removeEventListener('mousemove', onMove);
			window.removeEventListener('mouseup', onUp);
		}
		window.addEventListener('mousemove', onMove);
		window.addEventListener('mouseup', onUp);
	}

	function startBrowserPaneResize(event: ReactMouseEvent<HTMLDivElement>) {
		event.preventDefault();
		const startX = event.clientX;
		const initialWidth = browserPaneWidth;
		function onMove(moveEvent: MouseEvent) {
			setBrowserPaneWidth(clamp(initialWidth + (moveEvent.clientX - startX), 280, 860));
		}
		function onUp() {
			window.removeEventListener('mousemove', onMove);
			window.removeEventListener('mouseup', onUp);
		}
		window.addEventListener('mousemove', onMove);
		window.addEventListener('mouseup', onUp);
	}

	function startPreviewPaneResize(event: ReactMouseEvent<HTMLDivElement>) {
		event.preventDefault();
		const split = event.currentTarget.parentElement;
		if (!split) return;
		const rect = split.getBoundingClientRect();
		const layout = previewLayout;
		function onMove(moveEvent: MouseEvent) {
			if (layout === 'right') {
				setPreviewPaneSize(clamp(rect.right - moveEvent.clientX, 280, rect.width - 220));
				return;
			}
			setPreviewPaneSize(clamp(rect.bottom - moveEvent.clientY, 220, rect.height - 180));
		}
		function onUp() {
			window.removeEventListener('mousemove', onMove);
			window.removeEventListener('mouseup', onUp);
		}
		window.addEventListener('mousemove', onMove);
		window.addEventListener('mouseup', onUp);
	}

	function openTabMenu(event: ReactMouseEvent, tabId: string) {
		event.preventDefault();
		setTabMenu({
			open: true,
			tabId,
			x: clamp(event.clientX, 8, window.innerWidth - TAB_MENU_WIDTH),
			y: clamp(event.clientY, 8, window.innerHeight - TAB_MENU_HEIGHT)
		});
	}

	async function submitDialog(value: string) {
		if (dialog.kind === 'hidden' || dialog.submitting) return;
		const current = dialog;
		setDialog({ ...current, submitting: true });
		const shouldClose = current.kind === 'input' ? await current.action(value) : await current.action();
		if (shouldClose) {
			setDialog({ kind: 'hidden' });
			return;
		}
		setDialog({ ...current, submitting: false });
	}

	useEffect(() => {
		let cancelled = false;
		void bootstrapDesktop()
			.then((payload) => {
				if (cancelled) return;
				setBaseUrl(payload.baseUrl);
				return loadWorkbench(false, false);
			})
			.catch((error) => {
				if (cancelled) return;
				fail(describeError(error, '桌面服务启动失败。'));
			});
		return () => {
			cancelled = true;
		};
	}, []);

	useEffect(() => {
		if (!showSettings) return;
		void loadSettings();
	}, [showSettings]);

	useEffect(() => {
		window.localStorage.setItem('arkkb:source-sidebar-width', String(sourceSidebarWidth));
	}, [sourceSidebarWidth]);

	useEffect(() => {
		window.localStorage.setItem('arkkb:browser-pane-width', String(browserPaneWidth));
	}, [browserPaneWidth]);

	useEffect(() => {
		window.localStorage.setItem('arkkb:preview-pane-size', String(previewPaneSize));
	}, [previewPaneSize]);

	useEffect(() => {
		setArchiveResultCursor(archiveDirectoryEntries.length ? 0 : -1);
	}, [
		archiveDirectoryEntries.length,
		archiveCaseSensitive,
		archiveCommittedSearch,
		archiveFileTypeFilter,
		archiveMatchFilter,
		archiveQuery,
		selectedArchiveId,
		selectedAutoCategoryId
	]);

	useEffect(() => {
		setArchiveSuggestionCursor(archiveSuggestions.length ? 0 : -1);
	}, [archiveSuggestions.length, archiveCaseSensitive, archiveFileTypeFilter, archiveMatchFilter, archiveQuery, archiveFolderPath]);

	useEffect(() => {
		if (!currentArchiveSource) {
			setArchiveBrowseResponse(null);
			setArchiveBrowseError('');
			setArchiveBrowseLoading(false);
			return;
		}
		if (!currentArchiveExplorerState?.expanded) return;
		const delay = currentArchiveExplorerState.searchMode === 'content' ? 420 : 180;
		const timer = window.setTimeout(() => {
			void fetchArchiveBrowse(currentArchiveSource, currentArchiveExplorerState);
		}, delay);
		return () => window.clearTimeout(timer);
	}, [
		currentArchiveExplorerState?.cursor,
		currentArchiveExplorerState?.expanded,
		currentArchiveExplorerState?.folderPath,
		currentArchiveExplorerState?.query,
		currentArchiveExplorerState?.searchMode,
		currentArchiveExplorerState?.sortBy,
		currentArchiveExplorerState?.sortDirection,
		currentArchiveSource,
		currentArchiveSourceKey
	]);

	useEffect(() => {
		if (!currentArchiveExplorerState?.expanded || !selectedArchiveBrowseFile || !archiveBrowseResponse) return;
		let cancelled = false;
		const selectedIndex = archiveBrowseResponse.files.findIndex((item) => item.path === selectedArchiveBrowseFile.path);
		const targets = [selectedArchiveBrowseFile, archiveBrowseResponse.files[selectedIndex - 1], archiveBrowseResponse.files[selectedIndex + 1]].filter(Boolean) as ArchiveBrowseFile[];
		for (const file of targets) {
			const ext = (file.extension || '').toLowerCase();
			const kind = classifyExtension(ext);
			if (kind !== 'text' || ['.htm', '.html', '.svg', '.md'].includes(ext)) continue;
			if (archivePreviewContentRef.current.has(file.path)) continue;
			void rpc<string>('fs.read', { path: file.path })
				.then((content) => {
					if (cancelled) return;
					archivePreviewContentRef.current.set(file.path, content ?? '');
					setArchivePreviewCacheTick((value) => value + 1);
				})
				.catch(() => undefined);
		}
		return () => {
			cancelled = true;
		};
	}, [archiveBrowseResponse, currentArchiveExplorerState?.expanded, selectedArchiveBrowseFile]);
	useEffect(() => {
		if (!tabMenu.open) return;
		function closeMenu() {
			setTabMenu({ open: false });
		}
		window.addEventListener('click', closeMenu);
		window.addEventListener('blur', closeMenu);
		return () => {
			window.removeEventListener('click', closeMenu);
			window.removeEventListener('blur', closeMenu);
		};
	}, [tabMenu.open]);

	useEffect(() => {
		function handleKeyDown(event: KeyboardEvent) {
			const isAccel = event.ctrlKey || event.metaKey;
			const target = event.target as HTMLElement | null;
			const isTyping =
				target instanceof HTMLInputElement ||
				target instanceof HTMLTextAreaElement ||
				target?.isContentEditable;
			if (sourceMode === 'archives' && !isTyping && event.key === '/') {
				event.preventDefault();
				archiveSearchInputRef.current?.focus();
				archiveSearchInputRef.current?.select();
				return;
			}
			if (sourceMode === 'archives' && !isAccel && archiveDirectoryEntries.length) {
				if (event.key === 'ArrowDown') {
					event.preventDefault();
					setArchiveResultCursor((current) => Math.min(current + 1, archiveDirectoryEntries.length - 1));
					return;
				}
				if (event.key === 'ArrowUp') {
					event.preventDefault();
					setArchiveResultCursor((current) => Math.max(current - 1, 0));
					return;
				}
				if (event.key === 'Enter' && !isTyping && archiveCursorEntry) {
					event.preventDefault();
					if (archiveCursorEntry.kind === 'dir') {
						setArchiveFolderPath(archiveCursorEntry.path);
					} else {
						void openDocument(archiveCursorEntry.item);
					}
					return;
				}
			}
			if (!activeTabId) return;
			if (isAccel && !event.shiftKey && event.key.toLowerCase() === 'w') {
				event.preventDefault();
				closeTab(activeTabId);
				return;
			}
			if (event.ctrlKey && event.key === 'F4') {
				event.preventDefault();
				closeTab(activeTabId);
				return;
			}
			if (isAccel && event.shiftKey && event.key.toLowerCase() === 'w') {
				event.preventDefault();
				closeAllTabs();
			}
		}
		window.addEventListener('keydown', handleKeyDown);
		return () => window.removeEventListener('keydown', handleKeyDown);
	}, [activeTabId, archiveCursorEntry, archiveDirectoryEntries.length, sourceMode, tabs.length]);

	useEffect(() => {
		if (sourceMode !== 'help' || !selectedHelpDoc) return;
		void loadHelpDoc(selectedHelpDoc);
	}, [selectedHelpDoc, sourceMode]);

	useEffect(() => {
		function syncOnFocus() {
			const now = Date.now();
			if (now - lastFocusSyncAtRef.current < 10000) {
				return;
			}
			if (focusSyncTimerRef.current !== null) {
				window.clearTimeout(focusSyncTimerRef.current);
			}
			focusSyncTimerRef.current = window.setTimeout(() => {
				lastFocusSyncAtRef.current = Date.now();
				focusSyncTimerRef.current = null;
				void rpc<boolean>('app.focusSync')
					.then(() => loadWorkbench(false, false))
					.catch(() => undefined);
			}, 300);
		}
		window.addEventListener('focus', syncOnFocus);
		return () => {
			window.removeEventListener('focus', syncOnFocus);
			if (focusSyncTimerRef.current !== null) {
				window.clearTimeout(focusSyncTimerRef.current);
				focusSyncTimerRef.current = null;
			}
		};
	}, []);

	const browserContent = (
		<WorkbenchBrowserContent
			archiveBreadcrumbs={archiveBreadcrumbs}
			archiveCaseSensitive={archiveCaseSensitive}
			archiveDirectoryCount={archiveDirectoryCount}
			archiveDirectoryEntries={archiveDirectoryEntries}
			archiveFileCount={archiveFileCount}
			archiveFileTypeFilter={archiveFileTypeFilter}
			archiveFolderPath={archiveFolderPath}
			archiveMatchFilter={archiveMatchFilter}
			archiveQuery={archiveQuery}
			archiveQueryNormalized={archiveQueryNormalized}
			archiveResultCursor={archiveResultCursor}
			archiveSearchInputRef={archiveSearchInputRef}
			archiveSuggestionCursor={archiveSuggestionCursor}
			archiveSuggestions={archiveSuggestions}
			hasActiveTab={Boolean(activeTab)}
			formatArchiveDirectory={(item) => archiveVirtualDirectory(item, workspaceRoots)}
			formatArchivePath={(item) => archiveVirtualPath(item, workspaceRoots)}
			formatMatchKind={matchKindLabel}
			getListHeight={cappedListHeight}
			isCommittedArchiveSearch={isCommittedArchiveSearch}
			normalizedArchiveFileType={normalizedArchiveFileType}
			onAddRoot={addRoot}
			onApplyArchiveSuggestion={applyArchiveSuggestion}
			onAttachActiveToArchive={attachActiveToArchive}
			onAttachSearchResultToSelectedArchive={attachSearchResultToSelectedArchive}
			onDetachFromArchive={detachFromArchive}
			onLoadArchive={loadArchive}
			onLoadAutoCategory={loadAutoCategory}
			onOpenDocument={openDocument}
			onOpenRecentItem={(item) =>
				openDocument({
					...item,
					virtualFolderIds: [],
					matchKind: 'name',
					extension: extensionFromPath(item.path)
				})
			}
			onOpenRecentWorkspace={openRecentWorkspace}
			onOpenArchiveExplorer={openArchiveExplorer}
			onOpenWorkspace={openWorkspace}
			onRequestCreateArchive={requestCreateArchive}
			onRequestCreatePhysical={requestCreatePhysical}
			onRequestDeleteArchive={requestDeleteArchive}
			onRequestDeleteNode={requestDeleteNode}
			onRequestRemoveRoot={requestRemoveRoot}
			onRequestRenameArchive={requestRenameArchive}
			onRequestRenameNode={requestRenameNode}
			onRevealInWorkspace={revealInWorkspace}
			onRunArchiveSearch={runArchiveSearch}
			onSelectHelpDoc={(docId) => {
				setSelectedHelpDoc(docId);
				setLoadedHelpDocId('');
			}}
			onSetActiveRoot={setActiveRoot}
			onSetArchiveCaseSensitive={setArchiveCaseSensitive}
			onSetArchiveFileTypeFilter={setArchiveFileTypeFilter}
			onSetArchiveFolderPath={setArchiveFolderPath}
			onSetArchiveMatchFilter={setArchiveMatchFilter}
			onSetArchiveQuery={setArchiveQuery}
			onSetArchiveResultCursor={setArchiveResultCursor}
			onSetArchiveSuggestionCursor={setArchiveSuggestionCursor}
			onSetDefaultRoot={setDefaultRoot}
			onToggleTreeNode={toggleTreeNode}
			searchResults={searchResults}
			selectedArchive={selectedArchive}
			selectedArchiveId={selectedArchiveId}
			selectedAutoCategory={selectedAutoCategory}
			selectedAutoCategoryId={selectedAutoCategoryId}
			selectedHelpDoc={selectedHelpDoc}
			selectedWorkspacePath={selectedWorkspacePath}
			sourceMode={sourceMode}
			workbench={workbench}
		/>
	);

	const contentView = (() => {
		if (sourceMode === 'help') {
			return (
				<div className="help-view">
					<div className="content-header">
						<div>
							<h2>{selectedHelpMeta?.title || '帮助'}</h2>
							<p>{selectedHelpMeta ? selectedHelpMeta.path : '阅读内置帮助文档。'}</p>
						</div>
					</div>
					<div className="content-body">
						<Suspense fallback={<ContentLoadingState />}>
							<HelpContent content={helpContent} errorMessage={helpError} loading={helpLoading} />
						</Suspense>
					</div>
				</div>
			);
		}

		if (sourceMode === 'archives' && workbench && currentArchiveSource && currentArchiveExplorerState?.expanded) {
			return (
				<ArchiveExplorerView
					baseUrl={baseUrl}
					browseFiles={archiveBrowseResponse?.files ?? []}
					browseFolders={archiveBrowseResponse?.folders ?? []}
					currentFolderPath={archiveBrowseResponse?.currentFolderPath ?? currentArchiveExplorerState.folderPath}
					error={archiveBrowseError}
					explorerState={currentArchiveExplorerState}
					loading={archiveBrowseLoading}
					nextCursor={archiveBrowseResponse?.nextCursor ?? -1}
					onCollapse={collapseArchiveExplorer}
					onNextPage={goToNextArchiveBrowsePage}
					onOpenExternal={openExternal}
					onOpenFile={(file) => openDocument(file)}
					onPrevPage={goToPreviousArchiveBrowsePage}
					onSelectArchive={(folder) => openArchiveFolderExplorer(folder)}
					onSelectAutoCategory={(category) => openArchiveCategoryExplorer(category)}
					onSelectFile={(file) => setArchiveExplorerSelectedPath(file.path)}
					onSelectNavTab={setArchiveExplorerNavTab}
					onSetFolderPath={setArchiveExplorerFolderPath}
					onSetGroupView={setArchiveExplorerGroupView}
					onSetQuery={setArchiveExplorerQuery}
					onSetSearchMode={setArchiveExplorerSearchMode}
					onSetSortBy={setArchiveExplorerSortBy}
					onSetSortDirection={setArchiveExplorerSortDirection}
					prefetchedContent={selectedArchivePrefetchedContent}
					selectedArchive={selectedArchive}
					selectedArchiveId={selectedArchiveId}
					selectedAutoCategory={selectedAutoCategory}
					selectedAutoCategoryId={selectedAutoCategoryId}
					selectedFile={selectedArchiveBrowseFile}
					totalFiles={archiveBrowseResponse?.totalFiles ?? 0}
					totalFolders={archiveBrowseResponse?.totalFolders ?? 0}
					workbench={workbench}
				/>
			);
		}
		if (activeTab && currentFile) {
			const previewExtension = (activeTab.extension || '').toLowerCase();
			const canSidePreview = activeTab.kind === 'text';
			const showSidePreview = canSidePreview && markdownPreview;
			const sidePreviewMode =
				previewExtension === '.md'
					? 'markdown'
					: ['.htm', '.html', '.svg'].includes(previewExtension)
						? 'asset'
						: 'code';
			const sidePreviewTitle =
				sidePreviewMode === 'markdown' ? '文档预览' : sidePreviewMode === 'asset' ? '实时预览' : '只读预览';
			return (
				<div className="document-pane">
					<div className="content-tabs">
						{tabs.map((tab) => (
							<div
								className={`content-tab ${tab.id === activeTabId ? 'active' : ''}`}
								key={tab.id}
								onContextMenu={(event) => openTabMenu(event, tab.id)}
							>
								<button
									className="content-tab-button"
									onAuxClick={(event) => {
										if (event.button === 1) {
											closeTab(tab.id);
										}
									}}
									onClick={() => setActiveTabId(tab.id)}
									title={tab.path}
									type="button"
								>
									<span className="content-tab-label">{tab.name}</span>
									{tab.dirty ? <span className="content-tab-dirty" /> : null}
								</button>
								<button className="content-tab-close" onClick={() => closeTab(tab.id)} title="关闭" type="button">
									×
								</button>
							</div>
						))}
						{activeTab ? (
							<button className="content-tab-menu-trigger" onClick={(event) => openTabMenu(event, activeTab.id)} title="标签操作" type="button">
								<MoreHorizontal size={14} />
							</button>
						) : null}
					</div>
					<div className="content-header">
						<div>
							<h2>{activeTab.name}</h2>
							<p>{activeTab.path}</p>
						</div>
						<div className="content-actions">
							{canSidePreview ? (
								<>
									<button
										onClick={() =>
											setPreviewTabIds((current) =>
												current.includes(activeTab.id)
													? current.filter((item) => item !== activeTab.id)
													: [...current, activeTab.id]
											)
										}
										type="button"
									>
										{showSidePreview ? '关闭预览' : '打开预览'}
									</button>
									{showSidePreview ? (
										<button
											onClick={() => setPreviewLayout((current) => (current === 'right' ? 'bottom' : 'right'))}
											type="button"
										>
											{previewLayout === 'right' ? '上下分栏' : '左右分栏'}
										</button>
									) : null}
								</>
							) : null}
							{selectedArchiveId && activeTab.kind !== 'binary' ? (
								<button onClick={() => void attachActiveToArchive()} type="button">
									<Archive size={14} />
									附加到归档
								</button>
							) : null}
							<button onClick={() => window.dispatchEvent(new Event('arkkb:save'))} type="button">
								保存
							</button>
							<button onClick={() => void openExternal(activeTab.path)} type="button">
								<ExternalLink size={14} />
								外部打开
							</button>
						</div>
					</div>
					<div className={`content-body document-stage ${showSidePreview ? `split-${previewLayout}` : 'single'}`}>
						{showSidePreview ? (
							<div
								className={`document-split document-split-${previewLayout}`}
								style={{ ['--preview-pane-size' as string]: `${previewPaneSize}px` }}
							>
								<div className="document-split-pane editor-pane">
									<Suspense fallback={<ContentLoadingState />}>
										<CodeEditor
											file={currentFile}
											onDirtyChange={(dirty) => updateDirty(activeTab.path, dirty)}
											onError={fail}
											onSaved={() => {
												updateDirty(activeTab.path, false);
												pushToast('文件已保存', 'success');
												setStatusText('文件已保存');
											}}
										/>
									</Suspense>
								</div>
								<div
									aria-label="调整预览尺寸"
									className={`split-resizer split-resizer-${previewLayout}`}
									onDoubleClick={resetPreviewPaneSize}
									onMouseDown={startPreviewPaneResize}
									role="separator"
								/>
								<div className="document-split-pane preview-pane">
									<div className="document-preview-panel">
										<div className="document-preview-header">
											<strong>{sidePreviewTitle}</strong>
											<span>{previewLayout === 'right' ? '左右布局' : '上下布局'}</span>
										</div>
										{activeTab.extension === '.md' ? (
											<div className="preview-surface">
												<iframe src={renderUrl} title={`Render ${activeTab.name}`} />
											</div>
										) : sidePreviewMode === 'asset' ? (
											<Suspense fallback={<ContentLoadingState />}>
												<FilePreview baseUrl={baseUrl} file={currentFile} onError={fail} />
											</Suspense>
										) : (
											<Suspense fallback={<ContentLoadingState />}>
												<CodePreview file={currentFile} onError={fail} />
											</Suspense>
										)}
									</div>
								</div>
							</div>
						) : activeTab.kind === 'text' ? (
							<Suspense fallback={<ContentLoadingState />}>
								<CodeEditor
									file={currentFile}
									onDirtyChange={(dirty) => updateDirty(activeTab.path, dirty)}
									onError={fail}
									onSaved={() => {
										updateDirty(activeTab.path, false);
										pushToast('文件已保存', 'success');
										setStatusText('文件已保存');
									}}
								/>
							</Suspense>
						) : activeTab.kind === 'spreadsheet' ? (
							<Suspense fallback={<ContentLoadingState />}>
								<SpreadsheetEditor
									baseUrl={baseUrl}
									file={currentFile}
									onDirtyChange={(dirty) => updateDirty(activeTab.path, dirty)}
									onError={fail}
									onSaved={() => {
										updateDirty(activeTab.path, false);
										pushToast('表格已保存', 'success');
										setStatusText('表格已保存');
									}}
								/>
							</Suspense>
						) : activeTab.kind === 'preview' ? (
							<Suspense fallback={<ContentLoadingState />}>
								<FilePreview baseUrl={baseUrl} file={currentFile} onError={fail} />
							</Suspense>
						) : (
							<div className="empty-state compact">
								<div className="empty-copy">
									<h2>当前文件不提供内置编辑</h2>
									<p>二进制文件保留在系统应用中处理，工作台只负责定位和打开。</p>
								</div>
								<div className="empty-actions">
									<button className="primary" onClick={() => void openExternal(activeTab.path)} type="button">
										<ExternalLink size={15} />
										外部打开
									</button>
								</div>
							</div>
						)}
					</div>
				</div>
			);
		}

		if (!hasRoots) {
			return (
				<div className="empty-state empty-state-landing">
					<div className="empty-hero">
						<div className="empty-copy">
							<span className="eyebrow">ArkKB Workspace</span>
							<h1>从一个工作区开始</h1>
							<p>先连上真实目录，再继续编辑、检索、归档和预览。入口保持分区布局，避免在中文标签和高缩放下互相挤压。</p>
						</div>
						<div className="empty-highlights" role="list" aria-label="工作台能力摘要">
							<span className="pill subtle" role="listitem">
								多根工作区
							</span>
							<span className="pill subtle" role="listitem">
								内容搜索
							</span>
							<span className="pill subtle" role="listitem">
								虚拟归档
							</span>
							<span className="pill subtle" role="listitem">
								原生预览
							</span>
						</div>
					</div>
					<div className="empty-launchpad">
						<div className="empty-action-block">
							<div className="empty-action-copy">
								<span className="eyebrow">Workspace Entry</span>
								<strong>先接入你的文件源</strong>
								<p>主操作始终保持单行按钮，不依赖文案压缩来适配宽度。</p>
							</div>
							<div className="empty-actions">
								<button className="primary" onClick={() => void openWorkspace()} type="button">
									<FolderOpen size={15} />
									打开工作区
								</button>
								<button onClick={() => void addRoot()} type="button">
									<FolderPlus size={15} />
									添加根目录
								</button>
							</div>
						</div>
						<div className="empty-secondary-block">
							<div className="empty-action-copy">
								<span className="eyebrow">Need Context</span>
								<strong>先看一遍帮助与规则</strong>
								<p>帮助文档会解释索引范围、归档行为和默认创建目标。</p>
							</div>
							<button className="empty-secondary-action" onClick={openHelpCenter} type="button">
								<CircleHelp size={15} />
								打开帮助
							</button>
						</div>
					</div>
				</div>
			);
		}

		return (
			<div className="empty-state">
				<div className="empty-copy">
					<span className="eyebrow">{sourceLabel}</span>
					<h1>选择左侧列表中的一个项目</h1>
					<p>文件从浏览区打开，帮助在正文区阅读，设置保留在偏好窗口中。</p>
				</div>
			</div>
		);
	})();

	return (
		<div
			className="app-shell"
			style={{
				['--source-sidebar-width' as string]: `${sourceSidebarWidth}px`
			}}
		>
			<aside className="source-sidebar">
				<div className="source-sidebar-header">
					<div>
						<span className="eyebrow">ArkKB</span>
						<h1>{workbench?.workspace.name || 'Workspace'}</h1>
					</div>
				</div>
				<nav className="source-nav">
					{SOURCE_ITEMS.map((item) => {
						const Icon = item.icon;
						return (
							<button
								className={`source-nav-item ${sourceMode === item.id ? 'active' : ''}`}
								key={item.id}
								onClick={() => setSourceMode(item.id)}
								type="button"
							>
								<Icon size={16} />
								<span>{item.label}</span>
							</button>
						);
					})}
				</nav>
				<div className="source-sidebar-footer">
					<div className="source-summary">
						<span>活动根目录</span>
						<strong>{activeRoot?.label || '未设置'}</strong>
					</div>
					<div className="source-summary">
						<span>归档数</span>
						<strong>{workbench?.virtualFolders.length ?? 0}</strong>
					</div>
				</div>
			</aside>
			<div
				aria-label="调整导航栏宽度"
				className="pane-resizer shell-resizer"
				onDoubleClick={resetSourceSidebarWidth}
				onMouseDown={startSourceSidebarResize}
				role="separator"
			/>

			<div className="workspace-shell">
				<header className="topbar">
					<div className="topbar-title topbar-panel">
						<div className="topbar-panel-copy">
							<span className="eyebrow">{sourceLabel}</span>
							<h2>{activeRoot?.label || sourceLabel}</h2>
							<p>{activeRoot?.path || 'No Active Root'}</p>
						</div>
						<div className="topbar-title-meta">
							<span className="pill subtle">{workbench?.workspace.roots.length ?? 0} 个根目录</span>
							<span className="pill subtle">{workbench?.virtualFolders.length ?? 0} 个归档</span>
						</div>
					</div>
					<div className="topbar-rail">
						<section className="topbar-search topbar-panel" aria-label="工作区搜索">
							<div className="topbar-panel-copy">
								<span className="eyebrow">Search</span>
								<strong>搜索当前工作区</strong>
							</div>
							<div className="topbar-search-form">
								<div className="topbar-search-input">
									<Search size={15} />
									<input
										onChange={(event) => setSearchQuery(event.target.value)}
										onKeyDown={(event) => event.key === 'Enter' && void runSearch()}
										placeholder="搜索文件、目录或内容"
										value={searchQuery}
									/>
									<button className="primary" onClick={() => void runSearch()} type="button">
										搜索
									</button>
								</div>
								<div className="topbar-search-filters">
									<select onChange={(event) => setSearchRootId(event.target.value)} value={searchRootId}>
										<option value="">全部根目录</option>
										{workbench?.workspace.roots.map((root) => (
											<option key={root.id} value={root.id}>
												{root.label}
											</option>
										))}
									</select>
									<select onChange={(event) => setSearchMatchField(event.target.value as ArchiveMatchField)} value={searchMatchField}>
										<option value="all">全部字段</option>
										<option value="name">名称</option>
										<option value="directory">目录</option>
										<option value="type">类型</option>
										<option value="content">内容</option>
									</select>
									<select onChange={(event) => setSearchFileTypeFilter(event.target.value)} value={searchFileTypeFilter}>
										<option value="">全部类型</option>
										{workbench?.autoCategories.map((category) => (
											<option key={category.id} value={category.extension}>
												{category.label}
											</option>
										))}
									</select>
									<label className="topbar-check">
										<input
											checked={searchCaseSensitive}
											onChange={(event) => setSearchCaseSensitive(event.target.checked)}
											type="checkbox"
										/>
										<span>区分大小写</span>
									</label>
								</div>
							</div>
						</section>
						<section className="topbar-actions topbar-panel" aria-label="工作区操作">
							<div className="topbar-panel-copy">
								<span className="eyebrow">Workspace</span>
								<strong>打开与维护工作区</strong>
							</div>
							<div className="topbar-action-groups">
								<div className="topbar-action-group primary">
									<button className="primary" onClick={() => void openWorkspace()} type="button">
										<FolderOpen size={15} />
										打开工作区
									</button>
									<button onClick={() => void addRoot()} type="button">
										<FolderPlus size={15} />
										添加根目录
									</button>
								</div>
								<div className="topbar-action-group secondary">
									<button onClick={() => void loadWorkbench()} type="button">
										<RefreshCw size={15} />
										刷新
									</button>
									<button onClick={() => setShowSettings(true)} type="button">
										<Settings size={15} />
										设置
									</button>
								</div>
							</div>
						</section>
					</div>
				</header>

				<div
					className="workbench-panels"
					style={{
						['--browser-pane-width' as string]: `${browserPaneWidth}px`
					}}
				>
					<section className="browser-pane">
						<div className="pane-header">
							<div>
								<h3>{sourceLabel}</h3>
								<p>
									{sourceMode === 'workspace'
										? `${workbench?.workspace.roots.length ?? 0} 个根目录`
										: sourceMode === 'archives'
											? `${workbench?.virtualFolders.length ?? 0} 个归档`
											: sourceMode === 'recent'
												? '最近访问'
												: sourceMode === 'help'
													? '内置帮助'
													: `${searchResults.length} 个结果`}
								</p>
							</div>
							{sourceMode === 'workspace' ? (
								<div className="pane-header-actions">
									<button disabled={!hasRoots} onClick={() => requestCreatePhysical('file', activeRoot?.path || '')} type="button">
										<FilePlus2 size={14} />
									</button>
									<button disabled={!hasRoots} onClick={() => requestCreatePhysical('folder', activeRoot?.path || '')} type="button">
										<FolderPlus size={14} />
									</button>
								</div>
							) : null}
						</div>
						<div className="pane-body">{browserContent}</div>
					</section>
					<div
						aria-label="调整浏览区宽度"
						className="pane-resizer browser-resizer"
						onDoubleClick={resetBrowserPaneWidth}
						onMouseDown={startBrowserPaneResize}
						role="separator"
					/>

					<section className="content-pane">{contentView}</section>
				</div>

				<footer className="statusbar">
					<div className="statusbar-section">
						{pendingCount > 0 ? <RefreshCw className="spin" size={14} /> : <CheckCircle2 size={14} />}
						<span>{pendingCount > 0 ? '处理中...' : statusText}</span>
					</div>
					<div className="statusbar-section muted">
						<span>{sourceLabel}</span>
						<span>{activeRoot?.label || 'No Active Root'}</span>
						<span>{activeTab?.name || '无打开文档'}</span>
					</div>
				</footer>
			</div>

			{errorText ? <div className="inline-banner global error">{errorText}</div> : null}
			{tabMenu.open ? (
				<div
					className="tab-context-menu"
					onClick={(event) => event.stopPropagation()}
					style={{ left: tabMenu.x, top: tabMenu.y }}
				>
					<button
						onClick={() => {
							closeTab(tabMenu.tabId);
							setTabMenu({ open: false });
						}}
						type="button"
					>
						关闭
					</button>
					<button
						onClick={() => {
							closeOtherTabs(tabMenu.tabId);
							setTabMenu({ open: false });
						}}
						type="button"
					>
						关闭其他
					</button>
					<button
						onClick={() => {
							closeTabsToRight(tabMenu.tabId);
							setTabMenu({ open: false });
						}}
						type="button"
					>
						关闭右侧
					</button>
					<button
						onClick={() => {
							closeAllTabs();
							setTabMenu({ open: false });
						}}
						type="button"
					>
						关闭全部
					</button>
				</div>
			) : null}
			<ToastViewport items={toasts} onDismiss={dismissToast} />
			{dialog.kind !== 'hidden' ? (
				<Suspense fallback={null}>
					<ModalDialog
						cancelLabel="取消"
						confirmLabel={dialog.confirmLabel}
						danger={dialog.kind === 'confirm' ? dialog.danger : false}
						description={dialog.description}
						initialValue={dialog.kind === 'input' ? dialog.initialValue : ''}
						mode={dialog.kind === 'input' ? 'input' : 'confirm'}
						onCancel={() => setDialog({ kind: 'hidden' })}
						onConfirm={(value) => void submitDialog(value)}
						open
						placeholder={dialog.kind === 'input' ? dialog.placeholder : ''}
						submitting={dialog.submitting}
						title={dialog.title}
					/>
				</Suspense>
			) : null}
			{showSettings ? (
				<Suspense fallback={null}>
					<SettingsDialog
						directoryAllowlist={directoryAllowlist}
						directoryBlocklist={directoryBlocklist}
						errorMessage={settingsError}
						fileTypeAllowlist={fileTypeAllowlist}
						fileTypeBlocklist={fileTypeBlocklist}
						loading={settingsLoading}
						maxIndexedFileSize={maxIndexedFileSize}
						onClose={() => setShowSettings(false)}
						onDirectoryAllowlistChange={setDirectoryAllowlist}
						onDirectoryBlocklistChange={setDirectoryBlocklist}
						onFileTypeAllowlistChange={setFileTypeAllowlist}
						onFileTypeBlocklistChange={setFileTypeBlocklist}
						onMaxIndexedFileSizeChange={setMaxIndexedFileSize}
						onSave={() => void saveSettings()}
						onThemeChange={setTheme}
						open
						saving={settingsSaving}
						theme={theme}
						workbench={workbench}
					/>
				</Suspense>
			) : null}
		</div>
	);
}










