import { lazy, Suspense, useMemo } from 'react';
import {
	Archive,
	ArrowDownAZ,
	ArrowUpAZ,
	ChevronLeft,
	ChevronsLeftRightEllipsis,
	Clock3,
	ExternalLink,
	FileSearch,
	FileText,
	Folder,
	FolderOpen,
	ListFilter,
	PanelLeftClose,
	Shapes,
	TextSearch
} from 'lucide-react';
import type { ArchiveBrowseFile, ArchiveBrowseFolder, AutoCategory, TreeNode, VirtualFolder, WorkbenchState } from '../../../lib/types';
import { classifyExtension } from '../../../lib/workbench';
import { VirtualList } from './VirtualList';
import type {
	ArchiveExplorerGroupView,
	ArchiveExplorerNavTab,
	ArchiveExplorerSearchMode,
	ArchiveExplorerSortBy,
	ArchiveExplorerSortDirection,
	ArchiveExplorerState
} from '../types';

type ArchiveExplorerViewProps = {
	baseUrl: string;
	workbench: WorkbenchState;
	explorerState: ArchiveExplorerState;
	selectedArchiveId: string;
	selectedAutoCategoryId: string;
	selectedArchive: VirtualFolder | null;
	selectedAutoCategory: AutoCategory | null;
	browseFolders: ArchiveBrowseFolder[];
	browseFiles: ArchiveBrowseFile[];
	totalFiles: number;
	totalFolders: number;
	nextCursor: number;
	currentFolderPath: string;
	loading: boolean;
	error: string;
	selectedFile: ArchiveBrowseFile | null;
	prefetchedContent?: string | null;
	onSelectNavTab: (tab: ArchiveExplorerNavTab) => void;
	onSelectAutoCategory: (category: AutoCategory) => void | Promise<void>;
	onSelectArchive: (folder: VirtualFolder) => void | Promise<void>;
	onSetFolderPath: (path: string) => void;
	onSetQuery: (value: string) => void;
	onSetSearchMode: (mode: ArchiveExplorerSearchMode) => void;
	onSetSortBy: (value: ArchiveExplorerSortBy) => void;
	onSetSortDirection: (value: ArchiveExplorerSortDirection) => void;
	onSetGroupView: (value: ArchiveExplorerGroupView) => void;
	onSelectFile: (file: ArchiveBrowseFile) => void;
	onOpenFile: (file: ArchiveBrowseFile) => void | Promise<void>;
	onOpenExternal: (path: string) => void | Promise<void>;
	onCollapse: () => void;
	onPrevPage: () => void;
	onNextPage: () => void;
};

type ArchiveExplorerRow =
	| { kind: 'group'; id: string; label: string; count: number }
	| { kind: 'file'; id: string; file: ArchiveBrowseFile };

const NAV_TABS: Array<{ id: ArchiveExplorerNavTab; label: string }> = [
	{ id: 'categories', label: '分类' },
	{ id: 'folders', label: '归档' },
	{ id: 'directory', label: '目录' }
];

const GROUP_OPTIONS: Array<{ id: ArchiveExplorerGroupView; label: string }> = [
	{ id: 'list', label: '列表' },
	{ id: 'alpha', label: '首字母' },
	{ id: 'directory', label: '目录' },
	{ id: 'type', label: '类型' }
];

const CodePreview = lazy(async () => ({ default: (await import('./CodePreview')).CodePreview }));
const FilePreview = lazy(async () => ({ default: (await import('./FilePreview')).FilePreview }));

const SORT_OPTIONS: Array<{ id: ArchiveExplorerSortBy; label: string }> = [
	{ id: 'name', label: '名称' },
	{ id: 'modified', label: '最近修改' },
	{ id: 'lastOpened', label: '最近打开' },
	{ id: 'type', label: '类型' },
	{ id: 'directory', label: '目录' }
];

function archiveRootLabel(workbench: WorkbenchState, rootId: string) {
	return workbench.workspace.roots.find((root) => root.id === rootId)?.label || '未设置默认根目录';
}

function formatTime(value: number) {
	if (!value) return '未记录';
	return new Intl.DateTimeFormat('zh-CN', {
		month: '2-digit',
		day: '2-digit',
		hour: '2-digit',
		minute: '2-digit'
	}).format(value * 1000);
}

function archiveDirectoryBreadcrumbs(folderPath: string) {
	if (!folderPath) return [] as Array<{ label: string; path: string }>;
	const parts = folderPath.split('/').filter(Boolean);
	return parts.map((part, index) => ({
		label: part,
		path: parts.slice(0, index + 1).join('/')
	}));
}

function archiveGroupRows(files: ArchiveBrowseFile[], groupView: ArchiveExplorerGroupView): ArchiveExplorerRow[] {
	if (!files.length) return [];
	if (groupView === 'list') {
		return files.map((file) => ({ kind: 'file' as const, id: file.path, file }));
	}

	const groups = new Map<string, ArchiveBrowseFile[]>();
	for (const file of files) {
		let label = '其他';
		switch (groupView) {
			case 'alpha': {
				const first = file.name.trim().charAt(0).toUpperCase();
				label = first && /[A-Z0-9]/.test(first) ? first : '#';
				break;
			}
			case 'directory':
				label = file.directory || '根目录';
				break;
			case 'type':
				label = file.extension || '无扩展';
				break;
			default:
				label = '文件';
		}
		const current = groups.get(label) ?? [];
		current.push(file);
		groups.set(label, current);
	}

	const labels = Array.from(groups.keys()).sort((left, right) => left.localeCompare(right, 'zh-CN'));
	const rows: ArchiveExplorerRow[] = [];
	for (const label of labels) {
		const groupedFiles = groups.get(label) ?? [];
		rows.push({ kind: 'group', id: `group:${label}`, label, count: groupedFiles.length });
		for (const file of groupedFiles) {
			rows.push({ kind: 'file', id: file.path, file });
		}
	}
	return rows;
}

function PreviewFallback() {
	return <div className="content-loading-state">正在载入预览...</div>;
}

export function ArchiveExplorerView(props: ArchiveExplorerViewProps) {
	const {
		baseUrl,
		workbench,
		explorerState,
		selectedArchiveId,
		selectedAutoCategoryId,
		selectedArchive,
		selectedAutoCategory,
		browseFolders,
		browseFiles,
		totalFiles,
		totalFolders,
		nextCursor,
		currentFolderPath,
		loading,
		error,
		selectedFile,
		prefetchedContent,
		onSelectNavTab,
		onSelectAutoCategory,
		onSelectArchive,
		onSetFolderPath,
		onSetQuery,
		onSetSearchMode,
		onSetSortBy,
		onSetSortDirection,
		onSetGroupView,
		onSelectFile,
		onOpenFile,
		onOpenExternal,
		onCollapse,
		onPrevPage,
		onNextPage
	} = props;

	const rows = useMemo(() => archiveGroupRows(browseFiles, explorerState.groupView), [browseFiles, explorerState.groupView]);
	const breadcrumbs = useMemo(() => archiveDirectoryBreadcrumbs(currentFolderPath), [currentFolderPath]);
	const selectedIndex = selectedFile ? browseFiles.findIndex((item) => item.path === selectedFile.path) : -1;
	const renderUrl = selectedFile && baseUrl ? `${baseUrl}/render/${encodeURIComponent(selectedFile.path)}` : '';
	const previewFile = selectedFile
		? ({
				id: selectedFile.path,
				name: selectedFile.name,
				path: selectedFile.path,
				kind: 'file',
				rootId: selectedFile.rootId,
				extension: selectedFile.extension,
				virtualFolderIds: selectedFile.virtualFolderIds
			} as TreeNode)
		: null;
	const previewKind = selectedFile ? classifyExtension(selectedFile.extension) : 'binary';
	const pageStart = browseFiles.length ? explorerState.cursor + 1 : 0;
	const pageEnd = browseFiles.length ? explorerState.cursor + browseFiles.length : 0;
	const activeScopeLabel = selectedAutoCategory?.label || selectedArchive?.name || '未选择归档范围';
	const activeScopeMeta = selectedAutoCategoryId
		? `${selectedAutoCategory?.extension || '自动分类'} · ${totalFiles} 个文件`
		: selectedArchiveId
			? `${archiveRootLabel(workbench, selectedArchive?.preferredRootId || '')} · ${totalFiles} 个文件`
			: '先选择一个分类或手动归档';
	const navHeading = explorerState.leftTab === 'categories'
		? '自动分类'
		: explorerState.leftTab === 'folders'
			? '手动归档'
			: '目录导航';
	const navDescription = explorerState.leftTab === 'categories'
		? '按类型快速切换归档范围。'
		: explorerState.leftTab === 'folders'
			? '优先显示可继续使用的手动归档。'
			: '目录只负责缩小当前浏览范围。';

	function handleFileListKeyDown(event: React.KeyboardEvent<HTMLDivElement>) {
		if (!browseFiles.length) return;
		if (event.key === 'ArrowDown') {
			event.preventDefault();
			const nextIndex = Math.min(browseFiles.length - 1, selectedIndex < 0 ? 0 : selectedIndex + 1);
			onSelectFile(browseFiles[nextIndex]);
			return;
		}
		if (event.key === 'ArrowUp') {
			event.preventDefault();
			const nextIndex = Math.max(0, selectedIndex <= 0 ? 0 : selectedIndex - 1);
			onSelectFile(browseFiles[nextIndex]);
			return;
		}
		if (event.key === 'Enter' && selectedFile) {
			event.preventDefault();
			void onOpenFile(selectedFile);
		}
	}

	return (
		<div className="archive-explorer-shell">
			<div className="archive-explorer-header">
				<div>
					<span className="eyebrow">Archive Explorer</span>
					<h2>{selectedAutoCategory?.label || selectedArchive?.name || '归档浏览'}</h2>
					<p>{currentFolderPath || '根目录'} · {totalFolders} 个目录 · {totalFiles} 个文件</p>
				</div>
				<div className="archive-explorer-header-actions">
					<span className="pill subtle">{selectedAutoCategoryId ? '自动分类' : selectedArchiveId ? '手动归档' : '未选择归档'}</span>
					<button onClick={onCollapse} type="button">
						<PanelLeftClose size={14} />
						收起
					</button>
				</div>
			</div>

			<div className="archive-explorer-grid">
				<aside className="archive-explorer-sidebar">
					<div className="archive-explorer-nav-tabs">
						{NAV_TABS.map((tab) => (
							<button className={explorerState.leftTab === tab.id ? 'active' : ''} key={tab.id} onClick={() => onSelectNavTab(tab.id)} type="button">{tab.label}</button>
						))}
					</div>
					<div className="archive-explorer-nav-summary">
						<span className="pill subtle">{selectedAutoCategoryId ? '自动分类' : selectedArchiveId ? '手动归档' : '未选择范围'}</span>
						<div className="archive-explorer-nav-summary-copy">
							<strong>{activeScopeLabel}</strong>
							<span>{activeScopeMeta}</span>
						</div>
					</div>
					<div className="archive-explorer-nav-body">
						<div className="archive-explorer-nav-heading">
							<strong>{navHeading}</strong>
							<span>{navDescription}</span>
						</div>
						{explorerState.leftTab === 'categories' ? (
							workbench.autoCategories.length ? (
								<VirtualList
									className="archive-explorer-list compact"
									height={360}
									itemKey={(item) => item.id}
									items={workbench.autoCategories}
									rowHeight={60}
									renderItem={(category) => (
										<button className={`archive-source-button ${selectedAutoCategoryId === category.id ? 'active' : ''}`} onClick={() => void onSelectAutoCategory(category)} type="button">
											<Archive size={15} />
											<div className="archive-source-button-copy">
												<div className="archive-source-button-head">
													<strong>{category.label}</strong>
													<span className="archive-source-count">{category.count}</span>
												</div>
												<div className="archive-source-button-meta">
													<span>自动分类</span>
													<span>{category.extension || '无扩展'}</span>
												</div>
											</div>
										</button>
									)}
								/>
							) : <div className="browser-empty">暂无自动分类。</div>
						) : explorerState.leftTab === 'folders' ? (
							workbench.virtualFolders.length ? (
								<VirtualList
									className="archive-explorer-list compact"
									height={360}
									itemKey={(item) => item.id}
									items={workbench.virtualFolders}
									rowHeight={64}
									renderItem={(folder) => (
										<button className={`archive-source-button ${selectedArchiveId === folder.id ? 'active' : ''}`} onClick={() => void onSelectArchive(folder)} type="button">
											<FolderOpen size={15} />
											<div className="archive-source-button-copy">
												<div className="archive-source-button-head">
													<strong>{folder.name}</strong>
													<span className="archive-source-count">归档</span>
												</div>
												<div className="archive-source-button-meta">
													<span>{archiveRootLabel(workbench, folder.preferredRootId)}</span>
													<span>最近使用 {formatTime(folder.lastUsedAt)}</span>
												</div>
											</div>
										</button>
									)}
								/>
							) : <div className="browser-empty">暂无手动归档。</div>
						) : (
							<div className="archive-explorer-directory-nav">
								<div className="archive-explorer-breadcrumbs">
									<button className={!currentFolderPath ? 'active' : ''} onClick={() => onSetFolderPath('')} type="button">根目录</button>
									{breadcrumbs.map((crumb) => (
										<button className={crumb.path === currentFolderPath ? 'active' : ''} key={crumb.path} onClick={() => onSetFolderPath(crumb.path)} type="button">{crumb.label}</button>
									))}
								</div>
								{browseFolders.length ? (
									<VirtualList
										className="archive-explorer-list compact"
										height={320}
										itemKey={(item) => item.path}
										items={browseFolders}
										rowHeight={60}
										renderItem={(folder) => (
											<button className="archive-source-button" onClick={() => onSetFolderPath(folder.path)} title={folder.path} type="button">
												<Folder size={15} />
												<div className="archive-source-button-copy">
													<div className="archive-source-button-head">
														<strong>{folder.name}</strong>
														<span className="archive-source-count">{folder.count}</span>
													</div>
													<div className="archive-source-button-meta">
														<span>目录</span>
														<span>{folder.path === folder.name ? '当前范围子目录' : folder.path}</span>
													</div>
												</div>
											</button>
										)}
									/>
								) : <div className="browser-empty">当前目录下没有更多子目录。</div>}
							</div>
						)}
					</div>
				</aside>

				<section className="archive-explorer-main">
					<div className="archive-explorer-toolbar">
						<div className="archive-explorer-searchbar">
							<FileSearch size={15} />
							<input onChange={(event) => onSetQuery(event.target.value)} placeholder={explorerState.searchMode === 'content' ? '全文内容检索' : '按名称、目录、路径或类型快找'} value={explorerState.query} />
						</div>
						<div className="archive-explorer-mode-switch">
							<button className={explorerState.searchMode === 'quick' ? 'active' : ''} onClick={() => onSetSearchMode('quick')} type="button">
								<ListFilter size={14} />
								快找
							</button>
							<button className={explorerState.searchMode === 'content' ? 'active' : ''} onClick={() => onSetSearchMode('content')} type="button">
								<TextSearch size={14} />
								内容
							</button>
						</div>
						<div className="archive-explorer-group-switch">
							{GROUP_OPTIONS.map((option) => (
								<button className={explorerState.groupView === option.id ? 'active' : ''} key={option.id} onClick={() => onSetGroupView(option.id)} type="button">{option.label}</button>
							))}
						</div>
						<div className="archive-explorer-sortbar">
							<select onChange={(event) => onSetSortBy(event.target.value as ArchiveExplorerSortBy)} value={explorerState.sortBy}>
								{SORT_OPTIONS.map((option) => <option key={option.id} value={option.id}>{option.label}</option>)}
							</select>
							<button onClick={() => onSetSortDirection(explorerState.sortDirection === 'asc' ? 'desc' : 'asc')} type="button">
								{explorerState.sortDirection === 'asc' ? <ArrowDownAZ size={14} /> : <ArrowUpAZ size={14} />}
								{explorerState.sortDirection === 'asc' ? '升序' : '降序'}
							</button>
						</div>
					</div>

					<div className="archive-explorer-statusbar">
						<span>{loading ? '正在更新归档结果...' : `${pageStart}-${pageEnd} / ${totalFiles}`}</span>
						{error ? <span className="archive-explorer-error">{error}</span> : null}
						{explorerState.searchMode === 'content' ? <span className="pill subtle">内容模式</span> : <span className="pill subtle">快找模式</span>}
					</div>

					<div className="archive-explorer-filelist" onKeyDown={handleFileListKeyDown} tabIndex={0}>
						{rows.length ? (
							<VirtualList
								activeIndex={selectedIndex >= 0 ? rows.findIndex((row) => row.kind === 'file' && row.file.path === selectedFile?.path) : -1}
								className="archive-explorer-list"
								height={520}
								itemKey={(row) => row.id}
								items={rows}
								rowHeight={84}
								renderItem={(row) => row.kind === 'group' ? (
									<div className="archive-explorer-group-row">
										<strong>{row.label}</strong>
										<span>{row.count} 个文件</span>
									</div>
								) : (
									<div className={`archive-explorer-file-row ${selectedFile?.path === row.file.path ? 'active' : ''}`}>
										<button className="archive-explorer-file-button" onClick={() => onSelectFile(row.file)} onDoubleClick={() => void onOpenFile(row.file)} title={row.file.relativePath} type="button">
											<FileText size={15} />
											<div className="archive-explorer-file-copy">
												<div className="archive-explorer-file-headline">
													<strong>{row.file.name}</strong>
													<span>{row.file.directory || '根目录'}</span>
												</div>
												<div className="archive-explorer-file-meta">
													<span>{row.file.extension || '无扩展'}</span>
													<span>修改 {formatTime(row.file.modifiedAt)}</span>
													<span>打开 {formatTime(row.file.lastOpenedAt)}</span>
													<span className={`match-kind match-kind-${row.file.matchKind}`}>{row.file.matchKind}</span>
												</div>
											</div>
										</button>
									</div>
								)}
							/>
						) : (
							<div className="browser-empty">{loading ? '正在载入文件列表...' : explorerState.query ? '当前条件下没有匹配文件。' : '当前目录没有文件。'}</div>
						)}
					</div>
					<div className="archive-explorer-pagination">
						<button disabled={explorerState.cursor <= 0} onClick={onPrevPage} type="button">
							<ChevronLeft size={14} />
							上一页
						</button>
						<button disabled={nextCursor < 0} onClick={onNextPage} type="button">
							下一页
							<ChevronsLeftRightEllipsis size={14} />
						</button>
					</div>
				</section>

				<aside className="archive-explorer-preview">
					<div className="archive-explorer-preview-header">
						<div>
							<h3>{selectedFile?.name || '快速预览'}</h3>
							<p>{selectedFile?.relativePath || '单击中列文件进行预览，双击或 Enter 打开正式标签。'}</p>
						</div>
						{selectedFile ? (
							<div className="archive-explorer-preview-actions">
								<button className="primary" onClick={() => void onOpenFile(selectedFile)} type="button">打开文件</button>
								<button onClick={() => void onOpenExternal(selectedFile.path)} type="button">
									<ExternalLink size={14} />
									外部打开
								</button>
							</div>
						) : null}
					</div>
					{selectedFile ? (
						<>
							<div className="archive-explorer-preview-meta-grid">
								<div><span>类型</span><strong>{selectedFile.extension || '无扩展'}</strong></div>
								<div><span>目录</span><strong>{selectedFile.directory || '根目录'}</strong></div>
								<div><span>最近修改</span><strong>{formatTime(selectedFile.modifiedAt)}</strong></div>
								<div><span>最近打开</span><strong>{formatTime(selectedFile.lastOpenedAt)}</strong></div>
							</div>
							<div className="archive-explorer-preview-surface">
								{previewFile ? (
									previewKind === 'text' ? (
										selectedFile.extension === '.md' ? (
											<div className="preview-surface">
												<iframe src={renderUrl} title={`Render ${selectedFile.name}`} />
											</div>
										) : ['.htm', '.html', '.svg'].includes((selectedFile.extension || '').toLowerCase()) ? (
											<Suspense fallback={<PreviewFallback />}>
												<FilePreview baseUrl={baseUrl} file={previewFile} onError={() => undefined} />
											</Suspense>
										) : (
											<Suspense fallback={<PreviewFallback />}>
												<CodePreview file={previewFile} onError={() => undefined} prefetchedContent={prefetchedContent} />
											</Suspense>
										)
									) : previewKind === 'preview' ? (
										<Suspense fallback={<PreviewFallback />}>
											<FilePreview baseUrl={baseUrl} file={previewFile} onError={() => undefined} />
										</Suspense>
									) : (
										<div className="empty-state compact archive-preview-empty">
											<div className="empty-copy">
												<h2>当前文件暂不支持快速预览</h2>
												<p>可以直接打开标签页，或交给系统默认应用处理。</p>
											</div>
										</div>
									)
								) : null}
							</div>
						</>
					) : (
						<div className="browser-empty">当前没有选中文件。</div>
					)}
				</aside>
			</div>
		</div>
	);
}
