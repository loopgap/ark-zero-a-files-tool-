import { useEffect, useRef } from 'react';
import { Compartment, EditorState } from '@codemirror/state';
import { basicSetup, EditorView } from 'codemirror';
import type { TreeNode } from '../../../lib/types';
import { rpc } from '../../../lib/desktop';
import { describeError } from '../../../lib/workbench';
import { loadLanguageExtension } from './languageSupport';

type CodePreviewProps = {
	file: TreeNode;
	onError: (message: string) => void;
	prefetchedContent?: string | null;
};

const languageCompartment = new Compartment();

export function CodePreview({ file, onError, prefetchedContent }: CodePreviewProps) {
	const containerRef = useRef<HTMLDivElement | null>(null);
	const viewRef = useRef<EditorView | null>(null);
	const loadedPathRef = useRef('');
	const loadedContentRef = useRef<string | null>(null);
	const onErrorRef = useRef(onError);

	useEffect(() => {
		onErrorRef.current = onError;
	}, [onError]);

	useEffect(() => {
		if (!containerRef.current) return;

		viewRef.current = new EditorView({
			state: EditorState.create({
				doc: '',
				extensions: [
					basicSetup,
					languageCompartment.of([]),
					EditorState.readOnly.of(true),
					EditorView.editable.of(false),
					EditorView.lineWrapping,
					EditorView.theme({
						'&': {
							height: '100%',
							backgroundColor: 'var(--bg-content-subtle)',
							color: 'var(--content-text)'
						},
						'.cm-scroller': {
							fontFamily: 'var(--mono)'
						},
						'.cm-gutters': {
							backgroundColor: 'var(--bg-content-subtle)',
							border: 'none',
							color: 'var(--content-text-tertiary)'
						},
						'.cm-activeLineGutter': { backgroundColor: 'transparent' },
						'.cm-activeLine': { backgroundColor: 'transparent' },
						'.cm-content': {
							fontSize: 'var(--font-size-body)',
							lineHeight: '1.75',
							color: 'var(--content-text)'
						},
						'&.cm-focused': {
							outline: 'none'
						}
					})
				]
			}),
			parent: containerRef.current
		});

		return () => {
			viewRef.current?.destroy();
			viewRef.current = null;
		};
	}, []);

	useEffect(() => {
		const view = viewRef.current;
		if (!view || !file.path) return;

		let isCancelled = false;
		const loadContent = prefetchedContent != null ? Promise.resolve(prefetchedContent) : rpc<string>('fs.read', { path: file.path });

		Promise.all([loadContent, loadLanguageExtension(file.path)])
			.then(([content, languageExtension]) => {
				if (isCancelled || !viewRef.current) return;
				if (loadedPathRef.current === file.path && loadedContentRef.current === (content ?? '')) return;

				loadedPathRef.current = file.path;
				loadedContentRef.current = content ?? '';
				viewRef.current.dispatch({
					effects: languageCompartment.reconfigure(languageExtension),
					changes: {
						from: 0,
						to: viewRef.current.state.doc.length,
						insert: content ?? ''
					}
				});
			})
			.catch((error) => {
				if (!isCancelled) {
					onErrorRef.current(describeError(error, '预览读取失败。'));
				}
			});

		return () => {
			isCancelled = true;
		};
	}, [file.path, prefetchedContent]);

	return <div className="code-preview-surface" ref={containerRef} />;
}