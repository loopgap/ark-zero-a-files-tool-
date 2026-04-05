<script lang='ts'>
    import type { FileNode } from '../types';
    let { file } = \<{ file: FileNode }>();
</script>

<div class='binary-alert'>
    <div class='icon'>📁</div>
    <h2>无法直接编辑此文件</h2>
    <p>这是一个二进制文件，或者是目前不支持的文件类型 (<strong>{file.extension || '未知'}</strong>)。</p>
    <button onclick={() => console.log('Opening with native app: ' + file.path)}>使用原生软件打开</button>
</div>

<style>
    .binary-alert {
        display: flex;
        flex-direction: column;
        align-items: center;
        justify-content: center;
        height: 100%;
        color: #888;
        background-color: #1e1e1e;
        padding: 2rem;
        text-align: center;
    }

    .icon {
        font-size: 3rem;
        margin-bottom: 1rem;
    }

    h2 {
        color: #eee;
        font-weight: 300;
        margin-bottom: 0.5rem;
    }

    button {
        margin-top: 1.5rem;
        padding: 0.6rem 1.2rem;
        background-color: #333;
        border: 1px solid #444;
        color: #ccc;
        cursor: pointer;
        border-radius: 4px;
        transition: all 0.2s;
    }

    button:hover {
        background-color: #444;
        color: #eee;
    }
</style>
