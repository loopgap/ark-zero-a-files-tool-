import type { ArchiveBrowseRequest } from '../../lib/types';
import type {
	ArchiveExplorerGroupView,
	ArchiveExplorerNavTab,
	ArchiveExplorerSearchMode,
	ArchiveExplorerSortBy,
	ArchiveExplorerSortDirection,
	ArchiveExplorerState
} from './types';

export type ArchiveSourceKind = 'virtual_folder' | 'auto_category';

export type ArchiveSourceRef = {
	kind: ArchiveSourceKind;
	id: string;
};

export const ARCHIVE_BROWSE_PAGE_SIZE = 120;

export function archiveSourceKey(source: ArchiveSourceRef | null) {
	return source ? `${source.kind}:${source.id}` : '';
}

export function archiveDefaultNavTab(kind: ArchiveSourceKind): ArchiveExplorerNavTab {
	return kind === 'auto_category' ? 'categories' : 'folders';
}

export function createArchiveExplorerState(kind: ArchiveSourceKind): ArchiveExplorerState {
	return {
		leftTab: archiveDefaultNavTab(kind),
		folderPath: '',
		query: '',
		searchMode: 'quick',
		sortBy: 'name',
		sortDirection: 'asc',
		groupView: 'list',
		selectedPath: '',
		cursor: 0,
		expanded: false
	};
}

export function createArchiveBrowseRequest(source: ArchiveSourceRef, state: ArchiveExplorerState): ArchiveBrowseRequest {
	return {
		sourceKind: source.kind,
		sourceId: source.id,
		folderPath: state.folderPath,
		query: state.query,
		searchMode: state.searchMode,
		sortBy: state.sortBy,
		sortDirection: state.sortDirection,
		pageSize: ARCHIVE_BROWSE_PAGE_SIZE,
		cursor: state.cursor
	};
}

export function archiveSearchModeFromLegacy(matchField: string): ArchiveExplorerSearchMode {
	return matchField === 'content' ? 'content' : 'quick';
}

export function archiveSortByFromLegacy(): ArchiveExplorerSortBy {
	return 'name';
}

export function archiveSortDirectionFromLegacy(): ArchiveExplorerSortDirection {
	return 'asc';
}

export function archiveGroupViewFromLegacy(): ArchiveExplorerGroupView {
	return 'list';
}