import { renderMarkdown } from '../../../lib/workbench';

type HelpContentProps = {
	content: string;
	errorMessage: string;
	loading: boolean;
};

export function HelpContent({ content, errorMessage, loading }: HelpContentProps) {
	if (loading) {
		return <div className="help-placeholder">正在加载帮助内容...</div>;
	}

	if (errorMessage) {
		return <div className="help-placeholder error">{errorMessage}</div>;
	}

	if (!content) {
		return <div className="help-placeholder">当前没有可展示的帮助内容。</div>;
	}

	return (
		<article
			className="help-article"
			dangerouslySetInnerHTML={{ __html: renderMarkdown(content) }}
		/>
	);
}
