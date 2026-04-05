<script lang='ts'>
    import type { FileNode } from '../types';

    let { onSelect } = \<{ onSelect: (file: FileNode) => void }>();

    // Ê¾ÀýÊý¾Ý
    const mockFiles: FileNode[] = [
        { name: 'example.txt', path: '/test/example.txt', type: 'file', extension: '.txt', size: 1024 },
        { name: 'heavy.log', path: '/test/heavy.log', type: 'file', extension: '.log', size: 10 * 1024 * 1024 }, // 10MB
        { name: 'document.pdf', path: '/test/document.pdf', type: 'file', extension: '.pdf' },
        { name: 'index.html', path: '/test/index.html', type: 'file', extension: '.html' },
        { name: 'archive.zip', path: '/test/archive.zip', type: 'file', extension: '.zip' },
    ];
</script>

<aside class='sidebar'>
    <div class='header'>ARK KB</div>
    <nav class='file-tree'>
        {#each mockFiles as file}
            <button class='file-item' onclick={() => onSelect(file)}>
                <span class='icon'>{file.type === 'dir' ? '??' : '??'}</span>
                <span class='name'>{file.name}</span>
            </button>
        {/each}
    </nav>
</aside>

<style>
    .sidebar {
        width: 240px;
        background: #1e1e1e;
        color: #cccccc;
        display: flex;
        flex-direction: column;
        border-right: 1px solid #333;
        height: 100vh;
    }

    .header {
        padding: 1rem;
        font-weight: bold;
        letter-spacing: 2px;
        border-bottom: 1px solid #333;
        text-align: center;
        color: #eee;
    }

    .file-tree {
        flex: 1;
        overflow-y: auto;
        padding: 0.5rem;
    }

    .file-item {
        width: 100%;
        display: flex;
        align-items: center;
        padding: 0.4rem 0.6rem;
        background: transparent;
        border: none;
        color: inherit;
        cursor: pointer;
        font-size: 0.9rem;
        border-radius: 4px;
        text-align: left;
    }

    .file-item:hover {
        background: #2a2d2e;
    }

    .icon {
        margin-right: 0.5rem;
        font-size: 1rem;
    }

    .name {
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
    }
</style>
