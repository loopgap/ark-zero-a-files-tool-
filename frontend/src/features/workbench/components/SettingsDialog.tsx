import type { WorkbenchState } from '../../../lib/types';
import type { ThemeName } from '../../../lib/workbench';

type SettingsDialogProps = {
	open: boolean;
	theme: ThemeName;
	directoryAllowlist: string;
	directoryBlocklist: string;
	fileTypeAllowlist: string;
	fileTypeBlocklist: string;
	maxIndexedFileSize: string;
	loading: boolean;
	saving: boolean;
	errorMessage: string;
	workbench: WorkbenchState | null;
	onClose: () => void;
	onThemeChange: (value: ThemeName) => void;
	onDirectoryAllowlistChange: (value: string) => void;
	onDirectoryBlocklistChange: (value: string) => void;
	onFileTypeAllowlistChange: (value: string) => void;
	onFileTypeBlocklistChange: (value: string) => void;
	onMaxIndexedFileSizeChange: (value: string) => void;
	onSave: () => void;
};

export function SettingsDialog(props: SettingsDialogProps) {
	const {
		open,
		theme,
		directoryAllowlist,
		directoryBlocklist,
		fileTypeAllowlist,
		fileTypeBlocklist,
		maxIndexedFileSize,
		loading,
		saving,
		errorMessage,
		workbench,
		onClose,
		onThemeChange,
		onDirectoryAllowlistChange,
		onDirectoryBlocklistChange,
		onFileTypeAllowlistChange,
		onFileTypeBlocklistChange,
		onMaxIndexedFileSizeChange,
		onSave
	} = props;

	if (!open) return null;

	const defaultRoot =
		workbench?.workspace.roots.find((root) => root.id === workbench.workspace.defaultRootId)?.label ||
		'未设置';

	return (
		<div className="modal-overlay">
			<button aria-label="关闭设置" className="modal-backdrop" onClick={onClose} type="button" />
			<div className="settings-card" role="dialog" aria-modal="true" aria-labelledby="settings-title">
				<div className="modal-header">
					<div>
						<h2 id="settings-title">偏好设置</h2>
						<p>只保留当前可生效的配置。</p>
					</div>
					<button className="icon-ghost" onClick={onClose} type="button">
						×
					</button>
				</div>
				<div className="settings-body">
					{errorMessage ? <div className="inline-banner error">{errorMessage}</div> : null}
					{loading ? (
						<div className="help-placeholder">正在读取设置...</div>
					) : (
						<>
							<section className="settings-section">
								<div className="settings-copy">
									<h3>Appearance</h3>
									<p>主题会在保存后立即刷新。</p>
								</div>
								<label className="field">
									<span>Theme</span>
									<select value={theme} onChange={(event) => onThemeChange(event.target.value as ThemeName)}>
										<option value="minimal-dark">Minimal Dark</option>
										<option value="minimal-light">Minimal Light</option>
									</select>
								</label>
							</section>

							<section className="settings-section">
								<div className="settings-copy">
									<h3>Workspace</h3>
									<p>根目录管理保留在主界面，设置只展示摘要。</p>
								</div>
								<div className="summary-grid">
									<div className="summary-card">
										<span>当前工作区</span>
										<strong>{workbench?.workspace.name || 'ArkKB Workspace'}</strong>
									</div>
									<div className="summary-card">
										<span>根目录数量</span>
										<strong>{workbench?.workspace.roots.length ?? 0}</strong>
									</div>
									<div className="summary-card wide">
										<span>默认根目录</span>
										<strong>{defaultRoot}</strong>
									</div>
								</div>
							</section>

							<section className="settings-section">
								<div className="settings-copy">
									<h3>Indexing</h3>
									<p>目录与类型规则决定默认树和索引范围。</p>
								</div>
								<div className="field-grid">
									<label className="field">
										<span>Directory Allowlist</span>
										<textarea value={directoryAllowlist} onChange={(event) => onDirectoryAllowlistChange(event.target.value)} />
									</label>
									<label className="field">
										<span>Directory Blocklist</span>
										<textarea value={directoryBlocklist} onChange={(event) => onDirectoryBlocklistChange(event.target.value)} />
									</label>
									<label className="field">
										<span>File Type Allowlist</span>
										<input value={fileTypeAllowlist} onChange={(event) => onFileTypeAllowlistChange(event.target.value)} />
									</label>
									<label className="field">
										<span>File Type Blocklist</span>
										<input value={fileTypeBlocklist} onChange={(event) => onFileTypeBlocklistChange(event.target.value)} />
									</label>
									<label className="field wide">
										<span>Max Indexed File Size</span>
										<input value={maxIndexedFileSize} onChange={(event) => onMaxIndexedFileSizeChange(event.target.value)} />
									</label>
								</div>
							</section>

							<section className="settings-section">
								<div className="settings-copy">
									<h3>LSP</h3>
									<p>本轮不暴露未接通的安装或路径设置。</p>
								</div>
								<div className="summary-card wide">当前只保留 LSP 接口层，不提供额外设置项。</div>
							</section>
						</>
					)}
				</div>
				<div className="modal-footer">
					<button onClick={onClose} type="button">
						取消
					</button>
					<button className="primary" disabled={loading || saving} onClick={onSave} type="button">
						{saving ? '保存中...' : '保存'}
					</button>
				</div>
			</div>
		</div>
	);
}
