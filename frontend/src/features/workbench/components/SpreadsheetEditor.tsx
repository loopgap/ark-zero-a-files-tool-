import { useEffect, useMemo, useRef, useState } from 'react';
import type { TreeNode } from '../../../lib/types';
import { rpc } from '../../../lib/desktop';
import { describeError, encodeBase64 } from '../../../lib/workbench';

type SpreadsheetSheet = {
	name: string;
	rows: string[][];
};

type SpreadsheetEditorProps = {
	baseUrl: string;
	file: TreeNode;
	onDirtyChange: (dirty: boolean) => void;
	onSaved: () => void;
	onError: (message: string) => void;
};

function cloneSheets(sheets: SpreadsheetSheet[]) {
	return sheets.map((sheet) => ({
		...sheet,
		rows: sheet.rows.map((row) => [...row])
	}));
}

function normalizeRows(rows: unknown[][]) {
	const width = rows.reduce((max, row) => Math.max(max, row.length), 0);
	const normalized = rows.map((row) =>
		Array.from({ length: Math.max(width, 1) }, (_, index) => {
			const value = row[index];
			return value == null ? '' : String(value);
		})
	);
	return normalized.length ? normalized : [['']];
}

function columnLabel(index: number) {
	let current = index + 1;
	let label = '';
	while (current > 0) {
		const remainder = (current - 1) % 26;
		label = String.fromCharCode(65 + remainder) + label;
		current = Math.floor((current - 1) / 26);
	}
	return label;
}

export function SpreadsheetEditor({ baseUrl, file, onDirtyChange, onSaved, onError }: SpreadsheetEditorProps) {
	const fileUrl = useMemo(() => `${baseUrl}/file/${encodeURIComponent(file.path)}`, [baseUrl, file.path]);
	const [sheets, setSheets] = useState<SpreadsheetSheet[]>([]);
	const [activeSheet, setActiveSheet] = useState('');
	const [loading, setLoading] = useState(false);
	const [saving, setSaving] = useState(false);
	const [initialSnapshot, setInitialSnapshot] = useState('');
	const onDirtyChangeRef = useRef(onDirtyChange);
	const onSavedRef = useRef(onSaved);
	const onErrorRef = useRef(onError);

	useEffect(() => {
		onDirtyChangeRef.current = onDirtyChange;
	}, [onDirtyChange]);

	useEffect(() => {
		onSavedRef.current = onSaved;
	}, [onSaved]);

	useEffect(() => {
		onErrorRef.current = onError;
	}, [onError]);

	useEffect(() => {
		let cancelled = false;
		setLoading(true);
		setSheets([]);
		setActiveSheet('');

		void (async () => {
			try {
				const XLSX = await import('xlsx');
				const response = await fetch(fileUrl);
				if (!response.ok) {
					throw new Error(`表格读取失败: ${response.status}`);
				}
				const extension = (file.extension || '').toLowerCase();
				const workbook =
					extension === '.csv' || extension === '.tsv'
						? XLSX.read(await response.text(), {
								type: 'string',
								FS: extension === '.tsv' ? '\t' : ','
							})
						: XLSX.read(await response.arrayBuffer(), { type: 'array' });
				const parsedSheets = workbook.SheetNames.map((name) => ({
					name,
					rows: normalizeRows(XLSX.utils.sheet_to_json(workbook.Sheets[name], { header: 1, defval: '' }) as unknown[][])
				}));
				if (cancelled) return;
				const snapshot = JSON.stringify(parsedSheets);
				setSheets(parsedSheets);
				setActiveSheet(parsedSheets[0]?.name || '');
				setInitialSnapshot(snapshot);
				onDirtyChangeRef.current(false);
			} catch (error) {
				if (!cancelled) {
					onErrorRef.current(describeError(error, '表格读取失败。'));
				}
			} finally {
				if (!cancelled) {
					setLoading(false);
				}
			}
		})();

		return () => {
			cancelled = true;
		};
	}, [file.extension, fileUrl]);

	const activeSheetData = sheets.find((sheet) => sheet.name === activeSheet) ?? sheets[0] ?? null;

	function updateSheets(nextSheets: SpreadsheetSheet[]) {
		setSheets(nextSheets);
		onDirtyChangeRef.current(JSON.stringify(nextSheets) !== initialSnapshot);
	}

	function updateCell(rowIndex: number, columnIndex: number, value: string) {
		if (!activeSheetData) return;
		const nextSheets = cloneSheets(sheets).map((sheet) =>
			sheet.name !== activeSheetData.name
				? sheet
				: {
						...sheet,
						rows: sheet.rows.map((row, currentRowIndex) =>
							currentRowIndex !== rowIndex
								? row
								: row.map((cell, currentColumnIndex) => (currentColumnIndex === columnIndex ? value : cell))
						)
					}
		);
		updateSheets(nextSheets);
	}

	function addRow() {
		if (!activeSheetData) return;
		const width = activeSheetData.rows.reduce((max, row) => Math.max(max, row.length), 0) || 1;
		const nextSheets = cloneSheets(sheets).map((sheet) =>
			sheet.name !== activeSheetData.name
				? sheet
				: {
						...sheet,
						rows: [...sheet.rows, Array.from({ length: width }, () => '')]
					}
		);
		updateSheets(nextSheets);
	}

	function addColumn() {
		if (!activeSheetData) return;
		const nextSheets = cloneSheets(sheets).map((sheet) =>
			sheet.name !== activeSheetData.name
				? sheet
				: {
						...sheet,
						rows: sheet.rows.map((row) => [...row, ''])
					}
		);
		updateSheets(nextSheets);
	}

	async function saveSpreadsheet() {
		if (!sheets.length || saving) return;
		setSaving(true);
		try {
			const XLSX = await import('xlsx');
			const workbook = XLSX.utils.book_new();
			for (const sheet of sheets) {
				const worksheet = XLSX.utils.aoa_to_sheet(sheet.rows);
				XLSX.utils.book_append_sheet(workbook, worksheet, sheet.name);
			}

			const extension = (file.extension || '').toLowerCase();
			if (extension === '.csv' || extension === '.tsv') {
				const delimiter = extension === '.tsv' ? '\t' : ',';
				const text = XLSX.utils.sheet_to_csv(workbook.Sheets[sheets[0].name], { FS: delimiter });
				await rpc<boolean>('fs.save', { path: file.path, content: text });
			} else {
				const workbookBytes = XLSX.write(workbook, {
					bookType: extension === '.xls' ? 'xls' : 'xlsx',
					type: 'array'
				}) as ArrayBuffer;
				await rpc<boolean>('fs.saveBinary', {
					path: file.path,
					contentBase64: encodeBase64(new Uint8Array(workbookBytes))
				});
			}

			const snapshot = JSON.stringify(sheets);
			setInitialSnapshot(snapshot);
			onDirtyChangeRef.current(false);
			onSavedRef.current();
		} catch (error) {
			onErrorRef.current(describeError(error, '表格保存失败。'));
		} finally {
			setSaving(false);
		}
	}

	useEffect(() => {
		const requestSave = () => void saveSpreadsheet();
		window.addEventListener('arkkb:save', requestSave);
		return () => window.removeEventListener('arkkb:save', requestSave);
	}, [sheets, saving, initialSnapshot]);

	if (loading) {
		return <div className="spreadsheet-loading">正在载入表格...</div>;
	}

	if (!activeSheetData) {
		return <div className="spreadsheet-loading">当前表格没有可编辑内容。</div>;
	}

	return (
		<div className="spreadsheet-shell">
			<div className="spreadsheet-toolbar">
				<div className="spreadsheet-tabs">
					{sheets.map((sheet) => (
						<button
							className={`spreadsheet-tab ${sheet.name === activeSheetData.name ? 'active' : ''}`}
							key={sheet.name}
							onClick={() => setActiveSheet(sheet.name)}
							type="button"
						>
							{sheet.name}
						</button>
					))}
				</div>
				<div className="spreadsheet-actions">
					<button onClick={addRow} type="button">
						添加行
					</button>
					<button onClick={addColumn} type="button">
						添加列
					</button>
					<button className="primary" disabled={saving} onClick={() => void saveSpreadsheet()} type="button">
						{saving ? '保存中...' : '保存表格'}
					</button>
				</div>
			</div>
			<div className="spreadsheet-grid-wrap">
				<table className="spreadsheet-grid">
					<thead>
						<tr>
							<th>#</th>
							{activeSheetData.rows[0]?.map((_, columnIndex) => (
								<th key={`head-${columnIndex}`}>{columnLabel(columnIndex)}</th>
							))}
						</tr>
					</thead>
					<tbody>
						{activeSheetData.rows.map((row, rowIndex) => (
							<tr key={`row-${rowIndex}`}>
								<th>{rowIndex + 1}</th>
								{row.map((cell, columnIndex) => (
									<td key={`${rowIndex}-${columnIndex}`}>
										<input
											onChange={(event) => updateCell(rowIndex, columnIndex, event.target.value)}
											value={cell}
										/>
									</td>
								))}
							</tr>
						))}
					</tbody>
				</table>
			</div>
		</div>
	);
}
