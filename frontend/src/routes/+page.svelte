<script lang='ts'>
    import { onMount } from 'svelte';
    import Sidebar from '../lib/components/Sidebar.svelte';
    import Editor from '../lib/components/Editor.svelte';
    import Viewer from '../lib/components/Viewer.svelte';
    import BinaryAlert from '../lib/components/BinaryAlert.svelte';
    import type { FileNode } from '../lib/types';
    import { Events } from '@wailsio/runtime';

    let currentFile: FileNode | null = null;
    let diagnostics: Record<string, any[]> = {};
    let globalAlert: { message: string, type: 'error' | 'warning' } | null = null;

    // 三层路由分发协议实现
    const TEXT_EXTENSIONS = ['.txt', '.md', '.py', '.rs', '.go', '.c', '.cpp', '.ts', '.js', '.log', '.json', '.yaml', '.toml'];
    const PREVIEW_EXTENSIONS = ['.pdf', '.html', '.png', '.jpg', '.jpeg', '.svg', '.gif'];

    function handleFileSelect(file: FileNode) {
        currentFile = file;
    }

    function getFileType(file: FileNode | null): 'text' | 'preview' | 'binary' | null {
        if (!file) return null;
        const ext = file.extension?.toLowerCase() || '';
        if (TEXT_EXTENSIONS.includes(ext)) return 'text';
        if (PREVIEW_EXTENSIONS.includes(ext)) return 'preview';
        return 'binary';
    }

    $: fileType = getFileType(currentFile);

    onMount(() => {
        // LSP Notification Listener
        const unoff = Events.On('lsp:notification', (data: any) => {
            const { method, params } = data;
            if (method === 'textDocument/publishDiagnostics') {
                diagnostics[params.uri] = params.diagnostics;
                // If there are critical errors, show a temporary alert
                const critical = params.diagnostics.find((d: any) => d.severity === 1);
                if (critical) {
                    globalAlert = { message: `LSP Error: ${critical.message}`, type: 'error' };
                    setTimeout(() => { globalAlert = null; }, 5000);
                }
            }
        });

        return () => {
            unoff();
        };
    });
</script>

<div class='app-container'>
    <Sidebar onSelect={handleFileSelect} />

    <main class='main-content'>
        {#if globalAlert}
            <div class='global-alert {globalAlert.type}'>
                {globalAlert.message}
            </div>
        {/if}

        {#if !currentFile}
            <div class='placeholder'>选择一个文件开始编辑或预览</div>
        {:else if fileType === 'text'}
            <Editor file={currentFile} fileDiagnostics={diagnostics[currentFile.path] || []} />
        {:else if fileType === 'preview'}
            <Viewer file={currentFile} />
        {:else if fileType === 'binary'}
            <BinaryAlert file={currentFile} />
        {/if}
    </main>
</div>

<style>
    :global(body, html) {
        margin: 0;
        padding: 0;
        height: 100%;
        overflow: hidden;
        background-color: #0c0c0c;
        color: #d1d1d1;
        font-family: 'Inter', -apple-system, BlinkMacSystemFont, \"Segoe UI\", Roboto, Oxygen, Ubuntu, Cantarell, \"Fira Sans\", \"Droid Sans\", \"Helvetica Neue\", sans-serif;
    }

    .app-container {
        display: flex;
        height: 100vh;
        width: 100vw;
    }

    .main-content {
        flex: 1;
        overflow: hidden;
        display: flex;
        flex-direction: column;
        background-color: #121212;
        position: relative;
    }

    .placeholder {
        display: flex;
        align-items: center;
        justify-content: center;
        height: 100%;
        color: #444;
        font-size: 0.9rem;
        font-weight: 300;
        letter-spacing: 0.5px;
        text-transform: uppercase;
    }

    .global-alert {
        position: absolute;
        top: 20px;
        right: 20px;
        padding: 12px 20px;
        border-radius: 4px;
        font-size: 0.85rem;
        z-index: 1000;
        box-shadow: 0 4px 12px rgba(0,0,0,0.5);
        border-left: 4px solid;
        background: #1e1e1e;
        color: #eee;
        animation: slideIn 0.3s ease-out;
    }

    .global-alert.error {
        border-left-color: #ff5f56;
    }

    .global-alert.warning {
        border-left-color: #ffbd2e;
    }

    @keyframes slideIn {
        from { transform: translateX(100%); opacity: 0; }
        to { transform: translateX(0); opacity: 1; }
    }
</style>
