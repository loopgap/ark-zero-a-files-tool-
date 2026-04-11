import type { ToastMessage } from '../../../lib/workbench';

type ToastViewportProps = {
	items: ToastMessage[];
	onDismiss: (id: string) => void;
};

export function ToastViewport({ items, onDismiss }: ToastViewportProps) {
	if (!items.length) return null;

	return (
		<div aria-live="polite" className="toast-viewport">
			{items.map((item) => (
				<div className={`toast tone-${item.tone}`} key={item.id}>
					<span>{item.message}</span>
					<button aria-label="关闭提醒" onClick={() => onDismiss(item.id)} type="button">
						×
					</button>
				</div>
			))}
		</div>
	);
}
