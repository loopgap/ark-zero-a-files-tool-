import { useEffect, useRef } from 'react';
import { Compartment, EditorState } from '@codemirror/state';
import { basicSetup, EditorView } from 'codemirror';
import { keymap } from '@codemirror/view';
import type { TreeNode } from '../../../lib/types';
import { rpc } from '../../../lib/desktop';
import { describeError } from '../../../lib/workbench';
import { loadLanguageExtension } from './languageSupport';

type CodeEditorProps = {
	file: TreeNode;
	onDirtyChange: (dirty: boolean) => void;
	onSaved: () => void;
	onError: (message: string) => void;
};

const languageCompartment = new Compartment();

export function CodeEditor({ file, onDirtyChange, onSaved, onError }: CodeEditorProps) {
	const containerRef = useRef<HTMLDivElement | null>(null);
	const viewRef = useRef<EditorView | null>(null);
	const loadedPathRef = useRef('');
	const initialContentRef = useRef('');
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
		if (!containerRef.current) return;

		async function saveContent() {
			const view = viewRef.current;
			if (!view) return;
			try {
				const content = view.state.doc.toString();
				await rpc<boolean>('fs.save', { path: file.path, content });
				initialContentRef.current = content;
				onDirtyChangeRef.current(false);
				onSavedRef.current();
			} catch (error) {
				onErrorRef.current(describeError(error, '保存失败。'));
			}
		}

		const requestSave = () => void saveContent();
		window.addEventListener('arkkb:save', requestSave);

		viewRef.current = new EditorView({
			state: EditorState.create({
				doc: '',
				extensions: [
					basicSetup,
					languageCompartment.of([]),
					keymap.of([
						{
							key: 'Mod-s',
							run: () => {
								void saveContent();
								return true;
							},
							preventDefault: true
						}
					]),
					EditorView.updateListener.of((update) => {
						const view = viewRef.current;
						if (!update.docChanged || !view) return;
						onDirtyChangeRef.current(view.state.doc.toString() !== initialContentRef.current);
					}),
					EditorView.lineWrapping,
					EditorView.theme({
						'&': {
							height: '100%',
							backgroundColor: 'var(--content-bg)',
							color: 'var(--content-text)'
						},
						'.cm-scroller': {
							fontFamily: 'var(--mono)'
						},
						'.cm-gutters': {
							backgroundColor: 'var(--content-bg)',
							border: 'none',
							color: 'var(--content-text-tertiary)'
						},
						'.cm-activeLineGutter': { backgroundColor: 'var(--surface-hover)' },
						'.cm-activeLine': { backgroundColor: 'var(--surface-hover)' },
						'.cm-content': {
							caretColor: 'var(--accent)',
							fontSize: 'var(--font-size-body)',
							lineHeight: '1.75',
							color: 'var(--content-text)'
						},
						'.cm-selectionBackground': {
							backgroundColor: 'var(--selection-bg) !important'
						},
						'&.cm-focused .cm-selectionBackground': {
							backgroundColor: 'var(--selection-bg) !important'
						},
						'&.cm-focused .cm-cursor': { borderLeftColor: 'var(--accent)' },
						'&.cm-focused': {
							outline: 'none'
						},
						'.cm-searchMatch': {
							backgroundColor: 'rgba(255, 211, 105, 0.22)',
							outline: '1px solid rgba(255, 211, 105, 0.34)',
							color: 'var(--content-text)'
						},
						'.cm-searchMatch.cm-searchMatch-selected': {
							backgroundColor: 'rgba(255, 211, 105, 0.34)',
							outline: '1px solid rgba(255, 211, 105, 0.42)',
							color: 'var(--content-text)'
						}
					})
				]
			}),
			parent: containerRef.current
		});

		return () => {
			window.removeEventListener('arkkb:save', requestSave);
			viewRef.current?.destroy();
			viewRef.current = null;
		};
	}, [file.path]);

	useEffect(() => {
		const view = viewRef.current;
		if (!view || !file.path) return;

		// 如果路径相同且已加载，不重复加载
		if (loadedPathRef.current === file.path) return;

		// 使用 ref 来追踪取消状态，防止内存泄漏
		let isCancelled = false;

		Promise.all([rpc<string>('fs.read', { path: file.path }), loadLanguageExtension(file.path)])
			.then(([content, languageExtension]) => {
				// 双重检查：组件是否已卸载或路径已变更
				if (isCancelled || !viewRef.current) return;
				if (loadedPathRef.current === file.path) return; // 防止重复加载

				loadedPathRef.current = file.path;
				initialContentRef.current = content ?? '';
				viewRef.current.dispatch({
					effects: languageCompartment.reconfigure(languageExtension),
					changes: {
						from: 0,
						to: viewRef.current.state.doc.length,
						insert: initialContentRef.current
					}
				});
				onDirtyChangeRef.current(false);
			})
			.catch((error) => {
				if (!isCancelled) {
					onErrorRef.current(describeError(error, '文件读取失败。'));
				}
			});

		return () => {
			isCancelled = true;
		};
	}, [file.path]);

	return <div className="editor-surface" ref={containerRef} />;
}
