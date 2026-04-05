<script lang='ts'>
    import type { FileNode } from '../types';
    let { file } = \<{ file: FileNode }>();
</script>

<div class='viewer-container'>
    <iframe title='Preview' src='wails://file\' sandbox='allow-same-origin allow-scripts'></iframe>
</div>

<style>
    .viewer-container {
        width: 100%;
        height: 100%;
        background: #fff;
    }
    iframe {
        width: 100%;
        height: 100%;
        border: none;
    }
</style>
