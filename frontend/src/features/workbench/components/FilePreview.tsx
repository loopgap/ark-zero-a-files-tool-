import { useEffect, useMemo, useRef, useState } from 'react';
import type { TreeNode } from '../../../lib/types';
import {
	describeError,
	isDocxExtension,
	isIframePreviewExtension,
	isImagePreviewExtension
} from '../../../lib/workbench';

type FilePreviewProps = {
	baseUrl: string;
	file: TreeNode;
	onError: (message: string) => void;
};

export function FilePreview({ baseUrl, file, onError }: FilePreviewProps) {
	const fileUrl = useMemo(() => `${baseUrl}/file/${encodeURIComponent(file.path)}`, [baseUrl, file.path]);
	const [docHtml, setDocHtml] = useState('');
	const [loading, setLoading] = useState(false);
	const onErrorRef = useRef(onError);

	useEffect(() => {
		onErrorRef.current = onError;
	}, [onError]);

	useEffect(() => {
		if (!isDocxExtension(file.extension)) {
			setDocHtml('');
			setLoading(false);
			return;
		}

		let cancelled = false;
		setLoading(true);
		setDocHtml('');

		void fetch(fileUrl)
			.then((response) => {
				if (!response.ok) {
					throw new Error(`文档预览失败: ${response.status}`);
				}
				return response.arrayBuffer();
			})
			.then(async (buffer) => {
				const mammoth = await import('mammoth');
				return mammoth.convertToHtml({ arrayBuffer: buffer });
			})
			.then((result) => {
				if (cancelled) return;
				setDocHtml(result.value || '<p>文档没有可渲染内容。</p>');
			})
			.catch((error) => {
				if (cancelled) return;
				onErrorRef.current(describeError(error, 'DOCX 预览失败。'));
			})
			.finally(() => {
				if (!cancelled) {
					setLoading(false);
				}
			});

		return () => {
			cancelled = true;
		};
	}, [file.extension, fileUrl]);

	if (isImagePreviewExtension(file.extension)) {
		return (
			<div className="preview-surface image">
				<img alt={file.name} src={fileUrl} />
			</div>
		);
	}

	if (isIframePreviewExtension(file.extension)) {
		return (
			<div className="preview-surface iframe">
				<iframe src={fileUrl} title={`Preview ${file.name}`} />
			</div>
		);
	}

	if (isDocxExtension(file.extension)) {
		return (
			<div className="preview-surface docx">
				{loading ? (
					<div className="preview-placeholder">正在解析文档...</div>
				) : (
					<article className="docx-article" dangerouslySetInnerHTML={{ __html: docHtml }} />
				)}
			</div>
		);
	}

	return (
		<div className="preview-surface placeholder">
			<div className="preview-placeholder">当前文件暂不支持内置预览。</div>
		</div>
	);
}
