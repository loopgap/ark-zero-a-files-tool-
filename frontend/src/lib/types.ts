export interface FileNode {
    path: string;
    name: string;
    type: 'file' | 'dir';
    extension?: string;
    size?: number;
}
