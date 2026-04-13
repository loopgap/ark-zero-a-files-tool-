import { lazy, Suspense, useEffect } from 'react';

const WorkbenchApp = lazy(async () => ({
	default: (await import('./features/workbench/WorkbenchApp')).WorkbenchApp
}));

function resolvePlatform() {
	if (typeof navigator === 'undefined') return 'other';
	const agent = navigator.userAgent.toLowerCase();
	if (agent.includes('windows')) return 'windows';
	if (agent.includes('linux') || agent.includes('x11')) return 'linux';
	return 'other';
}

export default function App() {
	useEffect(() => {
		const root = document.documentElement;
		root.setAttribute('data-platform', resolvePlatform());
		return () => {
			root.removeAttribute('data-platform');
		};
	}, []);

	return (
		<Suspense
			fallback={
				<div className="startup-shell">
					<div className="startup-copy">
						<span className="eyebrow">ArkKB</span>
						<h1>正在载入工作台</h1>
						<p>准备文件树、编辑区和索引视图。</p>
					</div>
				</div>
			}
		>
			<WorkbenchApp />
		</Suspense>
	);
}
