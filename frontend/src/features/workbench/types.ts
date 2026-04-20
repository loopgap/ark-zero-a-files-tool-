export type ArchiveMatchField = 'all' | 'name' | 'directory' | 'type' | 'content';
export type ArchiveExplorerNavTab = 'categories' | 'folders' | 'directory';
export type ArchiveExplorerSearchMode = 'quick' | 'content';
export type ArchiveExplorerSortBy = 'name' | 'modified' | 'lastOpened' | 'type' | 'directory';
export type ArchiveExplorerSortDirection = 'asc' | 'desc';
export type ArchiveExplorerGroupView = 'list' | 'alpha' | 'directory' | 'type';

export type ArchiveCommittedSearch = {
	query: string;
	matchField: ArchiveMatchField;
	caseSensitive: boolean;
	fileType: string;
};

export type ArchiveDirectoryEntry =
	| {
			kind: 'dir';
			name: string;
			path: string;
			count: number;
	  }
	| {
			kind: 'file';
			item: import('../../lib/types').SearchHit;
	  };

export type ArchiveSuggestion =
	| {
			id: string;
			kind: 'directory';
			label: string;
			description: string;
			path: string;
	  }
	| {
			id: string;
			kind: 'file';
			label: string;
			description: string;
			item: import('../../lib/types').SearchHit;
	  }
	| {
			id: string;
			kind: 'type';
			label: string;
			description: string;
			value: string;
	  };

export type ArchiveExplorerState = {
	leftTab: ArchiveExplorerNavTab;
	folderPath: string;
	query: string;
	searchMode: ArchiveExplorerSearchMode;
	sortBy: ArchiveExplorerSortBy;
	sortDirection: ArchiveExplorerSortDirection;
	groupView: ArchiveExplorerGroupView;
	selectedPath: string;
	cursor: number;
	expanded: boolean;
};