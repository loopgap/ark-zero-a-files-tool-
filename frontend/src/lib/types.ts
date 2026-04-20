export interface WorkspaceRoot {
	id: string;
	path: string;
	label: string;
}

export interface WorkspaceSession {
	id: string;
	name: string;
	roots: WorkspaceRoot[];
	activeRootId: string;
	defaultRootId: string;
}

export interface VirtualFolder {
	id: string;
	workspaceId: string;
	name: string;
	preferredRootId: string;
	preferredParentPath: string;
	createdAt: number;
	lastUsedAt: number;
}

export interface AutoCategory {
	id: string;
	label: string;
	extension: string;
	count: number;
}

export interface PolicyConfig {
	directoryAllowlist: string[];
	directoryBlocklist: string[];
	fileTypeAllowlist: string[];
	fileTypeBlocklist: string[];
	maxIndexedFileSize: number;
}

export interface RecentItem {
	path: string;
	name: string;
	rootId: string;
	lastAccessed: number;
}

export interface RecentWorkspace {
	path: string;
	label: string;
	lastOpened: number;
}

export interface TreeNode {
	id: string;
	name: string;
	path: string;
	kind: 'workspace-root' | 'dir' | 'file' | 'virtual-folder';
	rootId: string;
	children?: TreeNode[];
	expanded?: boolean;
	virtualFolderIds?: string[];
	extension?: string;
}

export interface SearchHit {
	path: string;
	name: string;
	rootId: string;
	virtualFolderIds: string[];
	matchKind: 'name' | 'path' | 'directory' | 'type' | 'content';
	extension: string;
}

export interface ArchiveBrowseFolder {
	name: string;
	path: string;
	count: number;
}

export interface ArchiveBrowseFile extends SearchHit {
	relativePath: string;
	directory: string;
	modifiedAt: number;
	lastOpenedAt: number;
}

export interface ArchiveBrowseRequest {
	sourceKind: 'virtual_folder' | 'auto_category';
	sourceId: string;
	folderPath: string;
	query: string;
	searchMode: 'quick' | 'content';
	sortBy: 'name' | 'modified' | 'lastOpened' | 'type' | 'directory';
	sortDirection: 'asc' | 'desc';
	pageSize: number;
	cursor: number;
}

export interface ArchiveBrowseResponse {
	folders: ArchiveBrowseFolder[];
	files: ArchiveBrowseFile[];
	totalFiles: number;
	totalFolders: number;
	nextCursor: number;
	currentFolderPath: string;
}

export interface HelpDoc {
	id: string;
	title: string;
	path: string;
}

export interface WorkbenchState {
	workspace: WorkspaceSession;
	physicalRoots: TreeNode[];
	virtualFolders: VirtualFolder[];
	autoCategories: AutoCategory[];
	policy: PolicyConfig;
	recentItems: RecentItem[];
	recentWorkspaces: RecentWorkspace[];
	helpDocs: HelpDoc[];
	language: string;
	theme: string;
}

export interface CreateTarget {
	rootId: string;
	path: string;
}

export interface OpenTab {
	id: string;
	path: string;
	name: string;
	rootId: string;
	extension?: string;
	kind: 'text' | 'preview' | 'spreadsheet' | 'binary';
	virtualFolderIds: string[];
	dirty: boolean;
}