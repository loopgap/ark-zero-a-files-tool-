import { useEffect, useState, type RefObject } from 'react';
import {
	Archive,
	CircleHelp,
	FileSearch,
	FileText,
	Folder,
	FolderOpen,
	FolderPlus,
	PencilLine,
	Plus,
	Search,
	Trash2
} from 'lucide-react';
import type { AutoCategory, HelpDoc, RecentItem, SearchHit, TreeNode, VirtualFolder, WorkbenchState } from '../../../lib/types';
import type { SourceMode } from '../../../lib/workbench';
import { TreeBrowserRow } from './TreeBrowserRow';
import { VirtualList } from './VirtualList';
import type { ArchiveCommittedSearch, ArchiveDirectoryEntry, ArchiveMatchField, ArchiveSuggestion } from '../types';

type ArchiveBreadcrumb = {
	label: string;
	path: string;
};

type ArchiveTargetTab = 'categories' | 'folders';

function archivePreferredRootLabel(workbench: WorkbenchState, rootId: string) {
	return workbench.workspace.roots.find((root) => root.id === rootId)?.label || '未设置默认根目录';
}

type WorkbenchBrowserContentProps = {
	sourceMode: SourceMode;
	workbench: WorkbenchState | null;
	selectedWorkspacePath: string;
	selectedArchive: VirtualFolder | null;
	selectedArchiveId: string;
	selectedAutoCategory: AutoCategory | null;
	selectedAutoCategoryId: string;
	selectedHelpDoc: string;
	searchResults: SearchHit[];
	archiveMatchFilter: ArchiveMatchField;
	archiveCaseSensitive: boolean;
	archiveFileTypeFilter: string;
	archiveQuery: string;
	archiveQueryNormalized: string;
	archiveSuggestions: ArchiveSuggestion[];
	archiveSuggestionCursor: number;
	archiveDirectoryEntries: ArchiveDirectoryEntry[];
	archiveResultCursor: number;
	archiveFolderPath: string;
	archiveBreadcrumbs: ArchiveBreadcrumb[];
	archiveDirectoryCount: number;
	archiveFileCount: number;
	isCommittedArchiveSearch: boolean;
	normalizedArchiveFileType: string;
	hasActiveTab: boolean;
	archiveSearchInputRef: RefObject<HTMLInputElement | null>;
	onOpenWorkspace: () => void | Promise<void>;
	onAddRoot: () => void | Promise<void>;
	onSetActiveRoot: (rootId: string) => void | Promise<void>;
	onSetDefaultRoot: (rootId: string) => void | Promise<void>;
	onRequestRemoveRoot: (rootId: string, label: string) => void;
	onRequestCreatePhysical: (kind: 'file' | 'folder', parentPath?: string) => void;
	onRequestDeleteNode: (node: TreeNode) => void;
	onRequestRenameNode: (node: TreeNode) => void;
	onToggleTreeNode: (node: TreeNode) => void | Promise<void>;
	onSetArchiveQuery: (value: string) => void;
	onRunArchiveSearch: (keyword?: string) => void | Promise<void>;
	onSetArchiveSuggestionCursor: (index: number) => void;
	onSetArchiveResultCursor: (index: number) => void;
	onApplyArchiveSuggestion: (suggestion: ArchiveSuggestion) => void | Promise<void>;
	onSetArchiveMatchFilter: (value: ArchiveMatchField) => void;
	onSetArchiveFileTypeFilter: (value: string) => void;
	onSetArchiveCaseSensitive: (checked: boolean) => void;
	onSetArchiveFolderPath: (path: string) => void;
	onAttachActiveToArchive: () => void | Promise<void>;
	onDetachFromArchive: (path: string) => void | Promise<void>;
	onRevealInWorkspace: (item: Pick<SearchHit, 'path' | 'rootId' | 'name'>) => void | Promise<void>;
	onLoadAutoCategory: (category: AutoCategory, request: ArchiveCommittedSearch) => void | Promise<void>;
	onLoadArchive: (folderId: string, request: ArchiveCommittedSearch) => void | Promise<void>;
	onRequestCreateArchive: () => void;
	onRequestRenameArchive: (folderId: string, currentName: string) => void;
	onRequestDeleteArchive: (folderId: string, name: string) => void;
	onOpenRecentWorkspace: (path: string) => void | Promise<void>;
	onSelectHelpDoc: (docId: string) => void;
	onOpenDocument: (item: SearchHit | TreeNode) => void | Promise<void>;
	onOpenRecentItem: (item: RecentItem) => void | Promise<void>;
	onAttachSearchResultToSelectedArchive: (item: SearchHit) => void | Promise<void>;
	onOpenArchiveExplorer: () => void;
	formatArchivePath: (item: SearchHit) => string;
	formatArchiveDirectory: (item: SearchHit) => string;
	formatMatchKind: (matchKind: SearchHit['matchKind']) => string;
	getListHeight: (count: number, rowHeight: number, minHeight?: number, maxHeight?: number) => number;
};

export function WorkbenchBrowserContent(props: WorkbenchBrowserContentProps) {
	const {
		sourceMode,
		workbench,
		selectedWorkspacePath,
		selectedArchive,
		selectedArchiveId,
		selectedAutoCategory,
		selectedAutoCategoryId,
		selectedHelpDoc,
		searchResults,
		archiveMatchFilter,
		archiveCaseSensitive,
		archiveFileTypeFilter,
		archiveQuery,
		archiveQueryNormalized,
		archiveSuggestions,
		archiveSuggestionCursor,
		archiveDirectoryEntries,
		archiveResultCursor,
		archiveFolderPath,
		archiveBreadcrumbs,
		archiveDirectoryCount,
		archiveFileCount,
		isCommittedArchiveSearch,
		normalizedArchiveFileType,
		hasActiveTab,
		archiveSearchInputRef,
		onOpenWorkspace,
		onAddRoot,
		onSetActiveRoot,
		onSetDefaultRoot,
		onRequestRemoveRoot,
		onRequestCreatePhysical,
		onRequestDeleteNode,
		onRequestRenameNode,
		onToggleTreeNode,
		onSetArchiveQuery,
		onRunArchiveSearch,
		onSetArchiveSuggestionCursor,
		onSetArchiveResultCursor,
		onApplyArchiveSuggestion,
		onSetArchiveMatchFilter,
		onSetArchiveFileTypeFilter,
		onSetArchiveCaseSensitive,
		onSetArchiveFolderPath,
		onAttachActiveToArchive,
		onDetachFromArchive,
		onRevealInWorkspace,
		onLoadAutoCategory,
		onLoadArchive,
		onRequestCreateArchive,
		onRequestRenameArchive,
		onRequestDeleteArchive,
		onOpenRecentWorkspace,
		onSelectHelpDoc,
		onOpenDocument,
		onOpenRecentItem,
		onAttachSearchResultToSelectedArchive,
		onOpenArchiveExplorer,
		formatArchivePath,
		formatArchiveDirectory,
		formatMatchKind,
		getListHeight
	} = props;

	const [archiveTargetTab, setArchiveTargetTab] = useState<ArchiveTargetTab>(selectedArchiveId ? 'folders' : 'categories');

	useEffect(() => {
		if (selectedArchiveId) {
			setArchiveTargetTab('folders');
			return;
		}
		if (selectedAutoCategoryId) {
			setArchiveTargetTab('categories');
		}
	}, [selectedArchiveId, selectedAutoCategoryId]);

	if (!workbench) {
		return <div className="browser-empty">正在连接工作台...</div>;
	}

	if (sourceMode === 'workspace') {
		if (!workbench.physicalRoots.length) {
			return <div className="browser-empty">当前还没有根目录。</div>;
		}

		return (
			<div className="browser-stack">
				<section className="browser-section">
					<header className="browser-section-header">
						<h3>工作区管理</h3>
						<div className="pane-header-actions">
							<button onClick={() => void onOpenWorkspace()} type="button">
								<FolderOpen size={14} />
							</button>
							<button onClick={() => void onAddRoot()} type="button">
								<FolderPlus size={14} />
							</button>
						</div>
					</header>
					<div className="workspace-root-cards">
						{workbench.workspace.roots.map((root) => (
							<div className={`workspace-root-card ${root.id === workbench.workspace.activeRootId ? 'active' : ''}`} key={root.id}>
								<div className="workspace-root-card-copy">
									<strong>{root.label}</strong>
									<span>{root.path}</span>
								</div>
								<div className="workspace-root-card-actions">
									<button onClick={() => void onSetActiveRoot(root.id)} type="button">活动</button>
									<button onClick={() => void onSetDefaultRoot(root.id)} type="button">默认</button>
									<button onClick={() => onRequestRemoveRoot(root.id, root.label)} type="button">移除</button>
								</div>
							</div>
						))}
					</div>
				</section>
				<section className="browser-section">
					<header className="browser-section-header">
						<h3>目录浏览</h3>
					</header>
					<div className="browser-tree">
						{workbench.physicalRoots.map((node) => (
							<TreeBrowserRow
								activeRootId={workbench.workspace.activeRootId}
								defaultRootId={workbench.workspace.defaultRootId}
								depth={0}
								key={node.path}
								node={node}
								selectedPath={selectedWorkspacePath}
								onCreateFile={(path) => onRequestCreatePhysical('file', path)}
								onCreateFolder={(path) => onRequestCreatePhysical('folder', path)}
								onDelete={onRequestDeleteNode}
								onRemoveRoot={onRequestRemoveRoot}
								onRename={onRequestRenameNode}
								onSetActiveRoot={onSetActiveRoot}
								onSetDefaultRoot={onSetDefaultRoot}
								onToggle={onToggleTreeNode}
							/>
						))}
					</div>
				</section>
			</div>
		);
	}

	if (sourceMode === 'archives') {
		const archiveTargetLabel = selectedAutoCategory?.label || selectedArchive?.name || '未选择归档';
		const archivePlaceholder = selectedArchiveId
			? '搜索当前归档的名称、目录、类型或内容'
			: selectedAutoCategoryId
				? '搜索当前分类的名称、目录、类型或内容'
				: '先选择一个归档或分类，再开始搜索';
		const needsDeepSearch = archiveMatchFilter === 'content' || archiveQueryNormalized.length > 0;
		const defaultArchiveRequest: ArchiveCommittedSearch = {
			query: '',
			matchField: archiveMatchFilter,
			caseSensitive: archiveCaseSensitive,
			fileType: normalizedArchiveFileType
		};
		const archiveScopeKind = selectedAutoCategoryId ? '自动分类' : selectedArchiveId ? '手动归档' : '未选中归档';
		const archiveScopeDescription =
			selectedArchiveId || selectedAutoCategoryId
				? `${archiveFolderPath || '根目录'} · ${archiveDirectoryCount} 个目录 · ${archiveFileCount} 个文件`
				: '先在下方选择自动分类或手动归档，再开始浏览';
		const selectedArchiveRoot = selectedArchive ? archivePreferredRootLabel(workbench, selectedArchive.preferredRootId) : '';

		return (
			<div className="browser-stack archive-browser-layout">
				<div className="archive-search-toolbar archive-search-toolbar-top">
					<div className="archive-search-meta">
						<div>
							<strong>归档搜索</strong>
							<span>{archiveTargetLabel}</span>
						</div>
						<span className="pill subtle">{selectedArchiveId || selectedAutoCategoryId ? '当前归档范围' : '未选中归档'}</span>
					</div>
					<div className="archive-search-input">
						<Search size={14} />
						<input
							onChange={(event) => onSetArchiveQuery(event.target.value)}
							onKeyDown={(event) => {
								if (event.key === 'ArrowDown' && archiveSuggestions.length) {
									event.preventDefault();
									onSetArchiveSuggestionCursor(0);
									return;
								}
								if (event.key === 'Enter') {
									event.preventDefault();
									void onRunArchiveSearch();
									return;
								}
								if (event.key === 'ArrowDown' && archiveDirectoryEntries.length) {
									event.preventDefault();
									onSetArchiveResultCursor(0);
								}
							}}
							placeholder={archivePlaceholder}
							ref={archiveSearchInputRef}
							value={archiveQuery}
						/>
						<button className="primary" disabled={!selectedArchiveId && !selectedAutoCategoryId} onClick={() => void onRunArchiveSearch()} type="button">深搜</button>
					</div>
					{archiveSuggestions.length ? (
						<div className="archive-suggestions">
							{archiveSuggestions.map((suggestion, index) => (
								<button className={archiveSuggestionCursor === index ? 'archive-suggestion active' : 'archive-suggestion'} key={suggestion.id} onClick={() => void onApplyArchiveSuggestion(suggestion)} type="button">
									<strong>{suggestion.label}</strong>
									<span>{suggestion.description}</span>
									<em>{suggestion.kind === 'directory' ? '进入目录' : suggestion.kind === 'type' ? '按类型过滤' : '打开文件'}</em>
								</button>
							))}
						</div>
					) : null}
					<div className="archive-search-options">
						<div className="archive-match-filters">
							{[
								['all', '全部'],
								['name', '名称'],
								['directory', '目录'],
								['type', '类型'],
								['content', '内容']
							].map(([id, label]) => (
								<button className={archiveMatchFilter === id ? 'active' : ''} key={id} onClick={() => onSetArchiveMatchFilter(id as ArchiveMatchField)} type="button">{label}</button>
							))}
						</div>
						<div className="archive-option-row">
							<label className="archive-option-input">
								<span>文件类型</span>
								<select value={archiveFileTypeFilter} onChange={(event) => onSetArchiveFileTypeFilter(event.target.value)}>
									<option value="">全部类型</option>
									{workbench.autoCategories.map((category) => (
										<option key={category.id} value={category.extension}>{category.label}</option>
									))}
								</select>
							</label>
							<label className="archive-check-option">
								<input checked={archiveCaseSensitive} onChange={(event) => onSetArchiveCaseSensitive(event.target.checked)} type="checkbox" />
								<span>区分大小写</span>
							</label>
							{archiveQuery || archiveFileTypeFilter ? (
								<button onClick={() => { onSetArchiveQuery(''); onSetArchiveFileTypeFilter(''); void onRunArchiveSearch(''); }} type="button">清空</button>
							) : null}
						</div>
					</div>
					<div className="archive-search-hint">
						<span>紧凑态里优先选范围，再浏览目录。</span>
						<span>名称 / 目录 / 类型会即时筛选，内容匹配按 <kbd>Enter</kbd> 或点“深搜”。</span>
						{archiveCaseSensitive ? <span>当前按大小写严格匹配。</span> : null}
						{archiveFileTypeFilter ? <span>当前仅显示 {archiveFileTypeFilter}。</span> : null}
						{needsDeepSearch && !isCommittedArchiveSearch ? <span>当前结果仍在本地预筛选，深搜后会补内容匹配。</span> : null}
					</div>
				</div>

				<div className="archive-compact-overview">
					<div className="archive-compact-scope">
						<span className="pill subtle">{archiveScopeKind}</span>
						<div className="archive-compact-scope-copy">
							<strong>{archiveTargetLabel}</strong>
							<span>{selectedArchiveId ? `${selectedArchiveRoot} · ${archiveScopeDescription}` : archiveScopeDescription}</span>
						</div>
					</div>
					<div className="pane-header-actions">
						{archiveFolderPath ? <button onClick={() => onSetArchiveFolderPath('')} type="button">返回根目录</button> : null}
						{selectedArchiveId || selectedAutoCategoryId ? <button onClick={onOpenArchiveExplorer} type="button">展开浏览</button> : null}
						{selectedArchiveId && hasActiveTab ? <button onClick={() => void onAttachActiveToArchive()} type="button">附加当前文件</button> : null}
					</div>
				</div>

				<section className="browser-section archive-browser-section">
					<header className="browser-section-header">
						<div>
							<h3>{selectedAutoCategory?.label || selectedArchive?.name || '归档内容'}</h3>
							<p className="browser-section-subtitle">
								{selectedArchiveId || selectedAutoCategoryId
									? '单击选择，双击打开文件；目录单击直接进入。'
									: '先在下方选择一个自动分类或手动归档，再开始浏览'}
							</p>
						</div>
					</header>
					{selectedAutoCategoryId || selectedArchiveId ? (
						archiveDirectoryEntries.length ? (
							<>
								<div className="archive-breadcrumbs">
									<button className={!archiveFolderPath ? 'active' : ''} onClick={() => onSetArchiveFolderPath('')} type="button">根目录</button>
									{archiveBreadcrumbs.map((crumb) => (
										<button className={crumb.path === archiveFolderPath ? 'active' : ''} key={crumb.path} onClick={() => onSetArchiveFolderPath(crumb.path)} type="button">{crumb.label}</button>
									))}
								</div>
								<VirtualList
									activeIndex={archiveResultCursor}
									className="browser-virtual-list archive-compact-list"
									height={getListHeight(archiveDirectoryEntries.length, 68, 180, 520)}
									itemKey={(entry) => entry.kind === 'dir' ? `dir:${entry.path}` : `${selectedAutoCategoryId || selectedArchiveId}:${entry.item.path}`}
									items={archiveDirectoryEntries}
									rowHeight={68}
									renderItem={(entry, index) => entry.kind === 'dir' ? (
										<div className="browser-row-card archive-compact-row">
											<button className="browser-list-button archive-compact-entry archive-compact-entry-directory" onClick={() => onSetArchiveFolderPath(entry.path)} title={entry.path} type="button">
												<Folder size={15} />
												<div className="archive-compact-entry-copy">
													<div className="archive-compact-entry-head">
														<strong>{entry.name}</strong>
														<span>{entry.count} 项</span>
													</div>
													<div className="archive-compact-entry-meta">
														<span className="match-kind match-kind-directory">目录</span>
														<span className="archive-result-path">{entry.path}</span>
													</div>
												</div>
											</button>
										</div>
									) : (
										<div className={archiveResultCursor === index ? 'browser-row-card archive-compact-row active' : 'browser-row-card archive-compact-row'}>
											<button className="browser-list-button archive-compact-entry" onClick={() => onSetArchiveResultCursor(index)} onDoubleClick={() => void onOpenDocument(entry.item)} title={formatArchivePath(entry.item)} type="button">
												<FileText size={15} />
												<div className="archive-compact-entry-copy">
													<div className="archive-compact-entry-head">
														<strong>{entry.item.name}</strong>
														<span>{formatArchiveDirectory(entry.item) || '根目录'}</span>
													</div>
													<div className="archive-compact-entry-meta">
														<span className={`match-kind match-kind-${entry.item.matchKind}`}>{formatMatchKind(entry.item.matchKind)}</span>
														{entry.item.extension ? <span className="match-kind match-kind-type">{entry.item.extension}</span> : null}
														<span className="archive-result-path">{formatArchivePath(entry.item)}</span>
													</div>
												</div>
											</button>
											<div className="browser-inline-actions">
												<button onClick={() => void onOpenDocument(entry.item)} type="button">打开</button>
												{selectedArchiveId ? <button onClick={() => void onDetachFromArchive(entry.item.path)} type="button">移除</button> : null}
												<button onClick={() => void onRevealInWorkspace(entry.item)} type="button">定位</button>
											</div>
										</div>
									)}
								/>
							</>
						) : (
							<div className="browser-empty">{archiveQuery || archiveFileTypeFilter ? archiveMatchFilter === 'content' && !isCommittedArchiveSearch ? '内容匹配需要执行深搜，当前还没有返回结果。' : '当前搜索条件下没有匹配文件或目录。' : '当前归档为空，或者索引里还没有可显示文件。'}</div>
						)
					) : (
						<div className="browser-empty">先在下方选择一个自动分类或手动归档。</div>
					)}
				</section>

				<section className="browser-section archive-targets-section archive-targets-combined">
					<header className="browser-section-header">
						<div>
							<h3>归档来源</h3>
							<p className="browser-section-subtitle">先选范围，再浏览目录或展开到主视图。</p>
						</div>
						{archiveTargetTab === 'folders' ? <button onClick={onRequestCreateArchive} type="button"><Plus size={14} />新建</button> : null}
					</header>
					<div className="archive-target-tabs">
						<button className={archiveTargetTab === 'categories' ? 'active' : ''} onClick={() => setArchiveTargetTab('categories')} type="button">自动分类</button>
						<button className={archiveTargetTab === 'folders' ? 'active' : ''} onClick={() => setArchiveTargetTab('folders')} type="button">手动归档</button>
					</div>
					{archiveTargetTab === 'categories' ? (
						workbench.autoCategories.length ? (
							<VirtualList
								className="browser-virtual-list compact archive-target-list"
								height={getListHeight(workbench.autoCategories.length, 60, 140, 300)}
								itemKey={(category) => category.id}
								items={workbench.autoCategories}
								rowHeight={60}
								renderItem={(category) => (
									<button className={selectedAutoCategoryId === category.id ? 'archive-target-row active' : 'archive-target-row'} onClick={() => void onLoadAutoCategory(category, defaultArchiveRequest)} title={`${category.label} (${category.count})`} type="button">
										<Archive size={15} />
										<div className="archive-target-row-copy">
											<strong>{category.label}</strong>
											<span>{category.extension || '自动分类'}</span>
										</div>
										<span className="archive-target-count">{category.count}</span>
									</button>
								)}
							/>
						) : <div className="browser-empty">索引完成后会在这里显示自动分类。</div>
					) : workbench.virtualFolders.length ? (
						<VirtualList
							className="browser-virtual-list compact archive-target-list"
							height={getListHeight(workbench.virtualFolders.length, 64, 150, 320)}
							itemKey={(folder) => folder.id}
							items={workbench.virtualFolders}
							rowHeight={64}
							renderItem={(folder) => (
								<div className={selectedArchiveId === folder.id ? 'archive-target-row-shell active' : 'archive-target-row-shell'}>
									<button className={selectedArchiveId === folder.id ? 'archive-target-row active' : 'archive-target-row'} onClick={() => void onLoadArchive(folder.id, defaultArchiveRequest)} title={folder.name} type="button">
										<FolderOpen size={15} />
										<div className="archive-target-row-copy">
											<strong>{folder.name}</strong>
											<span>{archivePreferredRootLabel(workbench, folder.preferredRootId)}</span>
										</div>
									</button>
									<div className="browser-inline-actions">
										<button onClick={() => onRequestRenameArchive(folder.id, folder.name)} type="button"><PencilLine size={14} /></button>
										<button onClick={() => onRequestDeleteArchive(folder.id, folder.name)} type="button"><Trash2 size={14} /></button>
									</div>
								</div>
							)}
						/>
					) : <div className="browser-empty">暂无归档。</div>}
				</section>
			</div>
		);
	}
	if (sourceMode === 'recent') {
		return (
			<div className="browser-stack">
				<section className="browser-section">
					<header className="browser-section-header"><h3>最近文件</h3></header>
					{workbench.recentItems.length ? (
						<VirtualList className="browser-virtual-list compact" height={getListHeight(workbench.recentItems.length, 72, 140, 320)} itemKey={(item) => item.path} items={workbench.recentItems} rowHeight={72} renderItem={(item) => (
							<button className="browser-list-button" onClick={() => void onOpenRecentItem(item)} title={item.path} type="button">
								<FileText size={15} />
								<div className="browser-row-copy"><strong>{item.name}</strong><span>{item.path}</span></div>
							</button>
						)} />
					) : <div className="browser-empty">暂无最近文件。</div>}
				</section>
				<section className="browser-section">
					<header className="browser-section-header"><h3>最近工作区</h3></header>
					{workbench.recentWorkspaces.length ? (
						<VirtualList className="browser-virtual-list compact" height={getListHeight(workbench.recentWorkspaces.length, 72, 140, 320)} itemKey={(item) => item.path} items={workbench.recentWorkspaces} rowHeight={72} renderItem={(item) => (
							<button className="browser-list-button" onClick={() => void onOpenRecentWorkspace(item.path)} title={item.path} type="button">
								<Folder size={15} />
								<div className="browser-row-copy"><strong>{item.label}</strong><span>{item.path}</span></div>
							</button>
						)} />
					) : <div className="browser-empty">暂无最近工作区。</div>}
				</section>
			</div>
		);
	}

	if (sourceMode === 'help') {
		return workbench.helpDocs.length ? (
			<div className="browser-stack">
				<VirtualList className="browser-virtual-list compact" height={getListHeight(workbench.helpDocs.length, 72, 140, 360)} itemKey={(doc: HelpDoc) => doc.id} items={workbench.helpDocs} rowHeight={72} renderItem={(doc: HelpDoc) => (
					<button className={`browser-list-button ${selectedHelpDoc === doc.id ? 'active' : ''}`} onClick={() => onSelectHelpDoc(doc.id)} title={doc.title} type="button">
						<CircleHelp size={15} />
						<div className="browser-row-copy"><strong>{doc.title}</strong><span>{doc.id}</span></div>
					</button>
				)} />
			</div>
		) : <div className="browser-empty">当前没有可读取的帮助文档。</div>;
	}

	return searchResults.length ? (
		<div className="browser-stack">
			<VirtualList className="browser-virtual-list search-results-list" height={getListHeight(searchResults.length, 96, 180, 560)} itemKey={(item) => `${item.rootId}:${item.path}`} items={searchResults} rowHeight={96} renderItem={(item) => (
				<div className="browser-row-card">
					<button className="browser-list-button" onClick={() => void onOpenDocument(item)} title={item.path} type="button">
						<FileSearch size={15} />
						<div className="browser-row-copy">
							<strong>{item.name}</strong>
							<span>{item.path}</span>
							<div className="archive-result-meta">
								<span className={`match-kind match-kind-${item.matchKind}`}>{formatMatchKind(item.matchKind)}</span>
								{item.extension ? <span className="match-kind match-kind-type">{item.extension}</span> : null}
							</div>
						</div>
					</button>
					<div className="browser-inline-actions">
						<button onClick={() => void onRevealInWorkspace(item)} type="button">定位</button>
						{selectedArchiveId ? <button onClick={() => void onAttachSearchResultToSelectedArchive(item)} type="button">归档</button> : null}
					</div>
				</div>
			)} />
		</div>
	) : <div className="browser-empty">没有匹配结果。</div>;
}



