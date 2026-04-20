import {
	CheckCircle2,
	ChevronDown,
	ChevronRight,
	FilePlus2,
	FileText,
	Folder,
	FolderOpen,
	FolderPlus,
	PencilLine,
	Star,
	Trash2
} from 'lucide-react';
import type { TreeNode } from '../../../lib/types';

type TreeBrowserRowProps = {
	node: TreeNode;
	depth: number;
	activeRootId: string;
	defaultRootId: string;
	selectedPath: string;
	onToggle: (node: TreeNode) => void;
	onSetActiveRoot: (rootId: string) => void;
	onSetDefaultRoot: (rootId: string) => void;
	onCreateFile: (path: string) => void;
	onCreateFolder: (path: string) => void;
	onRename: (node: TreeNode) => void;
	onDelete: (node: TreeNode) => void;
	onRemoveRoot: (rootId: string, label: string) => void;
};

function normalizePathKey(path: string) {
	return path.replace(/\\/g, '/').replace(/\/+/g, '/').replace(/\/$/, '');
}

export function TreeBrowserRow(props: TreeBrowserRowProps) {
	const {
		node,
		depth,
		activeRootId,
		defaultRootId,
		selectedPath,
		onToggle,
		onSetActiveRoot,
		onSetDefaultRoot,
		onCreateFile,
		onCreateFolder,
		onRename,
		onDelete,
		onRemoveRoot
	} = props;

	const isRoot = node.kind === 'workspace-root';
	const isDir = node.kind === 'dir' || node.kind === 'workspace-root';
	const isFile = node.kind === 'file';
	const isSelected = normalizePathKey(node.path) === normalizePathKey(selectedPath);

	return (
		<>
			<div className={`browser-tree-row${isSelected ? ' selected' : ''}`} style={{ paddingLeft: `${14 + depth * 14}px` }}>
				<button
					className="browser-tree-main"
					onClick={() => {
						if (isRoot && node.rootId !== activeRootId) {
							void onSetActiveRoot(node.rootId);
						}
						onToggle(node);
					}}
					title={node.path}
					type="button"
				>
					<span className="browser-tree-leading">
						{isDir ? (
							<>
								{node.children?.length ? (
									node.expanded ? (
										<ChevronDown size={14} />
									) : (
										<ChevronRight size={14} />
									)
								) : (
									<span className="browser-tree-spacer" />
								)}
								{node.expanded ? <FolderOpen size={15} /> : <Folder size={15} />}
							</>
						) : (
							<>
								<span className="browser-tree-spacer" />
								<FileText size={15} />
							</>
						)}
					</span>
					<span className="browser-tree-copy">
						<span className="browser-tree-label">{node.name}</span>
					</span>
					<span className="browser-tree-badges">
						{isRoot && node.rootId === activeRootId ? <span className="pill">活动</span> : null}
						{isRoot && node.rootId === defaultRootId ? <span className="pill subtle">默认</span> : null}
					</span>
				</button>
				<div className="browser-tree-actions">
					{isRoot ? (
						<>
							<button aria-label="设为活动根目录" className="tree-action-icon" onClick={() => onSetActiveRoot(node.rootId)} title="设为活动根目录" type="button">
								<CheckCircle2 size={14} />
							</button>
							<button aria-label="设为默认根目录" className="tree-action-icon" onClick={() => onSetDefaultRoot(node.rootId)} title="设为默认根目录" type="button">
								<Star size={14} />
							</button>
							<button aria-label="新建文件" className="tree-action-icon" onClick={() => onCreateFile(node.path)} title="新建文件" type="button">
								<FilePlus2 size={14} />
							</button>
							<button aria-label="新建目录" className="tree-action-icon" onClick={() => onCreateFolder(node.path)} title="新建目录" type="button">
								<FolderPlus size={14} />
							</button>
							<button aria-label="移除根目录" className="tree-action-icon" onClick={() => onRemoveRoot(node.rootId, node.name)} title="移除根目录" type="button">
								<Trash2 size={14} />
							</button>
						</>
					) : isDir ? (
						<>
							<button aria-label="新建文件" className="tree-action-icon" onClick={() => onCreateFile(node.path)} title="新建文件" type="button">
								<FilePlus2 size={14} />
							</button>
							<button aria-label="新建目录" className="tree-action-icon" onClick={() => onCreateFolder(node.path)} title="新建目录" type="button">
								<FolderPlus size={14} />
							</button>
							<button aria-label="重命名" className="tree-action-icon" onClick={() => onRename(node)} title="重命名" type="button">
								<PencilLine size={14} />
							</button>
							<button aria-label="删除" className="tree-action-icon" onClick={() => onDelete(node)} title="删除" type="button">
								<Trash2 size={14} />
							</button>
						</>
					) : isFile ? (
						<>
							<button aria-label="重命名" className="tree-action-icon" onClick={() => onRename(node)} title="重命名" type="button">
								<PencilLine size={14} />
							</button>
							<button aria-label="删除" className="tree-action-icon" onClick={() => onDelete(node)} title="删除" type="button">
								<Trash2 size={14} />
							</button>
						</>
					) : null}
				</div>
			</div>
			{node.expanded &&
				node.children?.map((child) => (
					<TreeBrowserRow
						activeRootId={activeRootId}
						defaultRootId={defaultRootId}
						depth={depth + 1}
						key={child.path}
						node={child}
						selectedPath={selectedPath}
						onCreateFile={onCreateFile}
						onCreateFolder={onCreateFolder}
						onDelete={onDelete}
						onRemoveRoot={onRemoveRoot}
						onRename={onRename}
						onSetActiveRoot={onSetActiveRoot}
						onSetDefaultRoot={onSetDefaultRoot}
						onToggle={onToggle}
					/>
				))}
		</>
	);
}
