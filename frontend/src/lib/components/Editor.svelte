<script lang='ts'>
    import { onMount, onDestroy } from 'svelte';
    import { EditorState, StateField, StateEffect } from '@codemirror/state';
    import { EditorView, keymap, Decoration, DecorationSet } from '@codemirror/view';
    import { basicSetup } from 'codemirror';
    import { autocompletion, type CompletionContext, type CompletionResult } from '@codemirror/autocomplete';
    import type { FileNode } from '../types';
    import { Bridge } from '../../wailsjs/go/models'; // This is a placeholder for Wails v3 bindings
    // Note: In Wails v3, services are typically accessed via import { Bridge } from 'wailsjs/go/bridge/Bridge'
    // But since I cannot run the wails generate, I will use a generic call approach if needed or assume standard v3 paths.
    import * as Br from '../../wailsjs/go/bridge/Bridge';

    let { file, fileDiagnostics = [] } = $props<{ file: FileNode, fileDiagnostics?: any[] }>();
    let editorContainer: HTMLElement;
    let view: EditorView | null = null;

    const diagnosticDecoration = Decoration.mark({ class: 'cm-lsp-diagnostic-error' });
    const warningDecoration = Decoration.mark({ class: 'cm-lsp-diagnostic-warning' });

    // Diagnostics Field
    const diagnosticsField = StateField.define<DecorationSet>({
        create() { return Decoration.none; },
        update(underlines, tr) {
            underlines = underlines.map(tr.changes);
            for (let e of tr.effects) {
                if (e.is(setDiagnostics)) underlines = e.value;
            }
            return underlines;
        },
        provide: f => EditorView.decorations.from(f)
    });

    const setDiagnostics = StateEffect.define<DecorationSet>();

    async function loadFileContent() {
        try {
            const content = await Br.ReadFile(file.path);
            if (view) {
                view.dispatch({
                    changes: { from: 0, to: view.state.doc.length, insert: content }
                });
                
                // LSP: didOpen
                await Br.LSPNotify('main', 'textDocument/didOpen', {
                    textDocument: {
                        uri: file.path,
                        languageId: getLanguageId(file.extension),
                        version: 1,
                        text: content
                    }
                });
            }
        } catch (e) {
            console.error('Failed to load file:', e);
        }
    }

    function getLanguageId(ext: string | undefined): string {
        switch (ext?.toLowerCase()) {
            case '.go': return 'go';
            case '.py': return 'python';
            case '.rs': return 'rust';
            case '.ts': return 'typescript';
            case '.js': return 'javascript';
            case '.json': return 'json';
            default: return 'plaintext';
        }
    }

    async function handleCompletion(context: CompletionContext): Promise<CompletionResult | null> {
        let word = context.matchBefore(/\w*/);
        if (!word || (word.from == word.to && !context.explicit)) return null;

        const pos = context.pos;
        const line = context.state.doc.lineAt(pos);
        
        try {
            const result: any = await Br.LSPCall('main', 'textDocument/completion', {
                textDocument: { uri: file.path },
                position: { line: line.number - 1, character: pos - line.from }
            });

            if (!result) return null;

            const items = Array.isArray(result) ? result : result.items;
            return {
                from: word.from,
                options: items.map((item: any) => ({
                    label: item.label,
                    type: mapLSPKindToCM(item.kind),
                    detail: item.detail,
                    apply: item.insertText || item.label
                }))
            };
        } catch (e) {
            return null;
        }
    }

    function mapLSPKindToCM(kind: number): string {
        const kinds = ['', 'text', 'method', 'function', 'constructor', 'field', 'variable', 'class', 'interface', 'module', 'property'];
        return kinds[kind] || 'variable';
    }

    onMount(async () => {
        if (!editorContainer) return;

        const startState = EditorState.create({
            doc: '',
            extensions: [
                basicSetup,
                diagnosticsField,
                autocompletion({ override: [handleCompletion] }),
                EditorView.theme({
                    '&': { height: '100%', backgroundColor: '#121212', color: '#d4d4d4' },
                    '.cm-content': { fontFamily: '"JetBrains Mono", "Fira Code", monospace', fontSize: '13px' },
                    '.cm-gutters': { backgroundColor: '#121212', color: '#454545', border: 'none' },
                    '.cm-lsp-diagnostic-error': { textDecoration: 'underline wavy #ff5f56' },
                    '.cm-lsp-diagnostic-warning': { textDecoration: 'underline wavy #ffbd2e' }
                }),
                EditorView.updateListener.of((update) => {
                    if (update.docChanged) {
                        Br.LSPNotify('main', 'textDocument/didChange', {
                            textDocument: { uri: file.path, version: Date.now() },
                            contentChanges: [{ text: update.state.doc.toString() }]
                        });
                    }
                })
            ]
        });

        view = new EditorView({
            state: startState,
            parent: editorContainer
        });

        await loadFileContent();
    });

    // Handle incoming diagnostics from parent
    $effect(() => {
        if (view && fileDiagnostics) {
            const deco = fileDiagnostics.map(d => {
                const from = view!.state.doc.line(d.range.start.line + 1).from + d.range.start.character;
                const to = view!.state.doc.line(d.range.end.line + 1).from + d.range.end.character;
                return (d.severity === 1 ? diagnosticDecoration : warningDecoration).range(from, to);
            });
            view.dispatch({
                effects: setDiagnostics.of(Decoration.set(deco, true))
            });
        }
    });

    // Reload when file changes
    $effect(() => {
        if (file) {
            loadFileContent();
        }
    });

    onDestroy(() => {
        if (view) view.destroy();
    });
</script>

<div class='editor-wrapper' bind:this={editorContainer}></div>

<style>
    .editor-wrapper {
        width: 100%;
        height: 100%;
        overflow: hidden;
        background-color: #121212;
    }

    :global(.cm-editor) {
        height: 100%;
    }

    :global(.cm-scroller) {
        scrollbar-width: thin;
        scrollbar-color: #333 transparent;
    }
</style>
