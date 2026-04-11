import { useEffect, useRef, useState } from 'react';

type ModalDialogProps = {
	open: boolean;
	mode?: 'confirm' | 'input';
	title: string;
	description?: string;
	confirmLabel?: string;
	cancelLabel?: string;
	initialValue?: string;
	placeholder?: string;
	danger?: boolean;
	submitting?: boolean;
	onCancel: () => void;
	onConfirm: (value: string) => void;
};

export function ModalDialog({
	open,
	mode = 'confirm',
	title,
	description = '',
	confirmLabel = '确认',
	cancelLabel = '取消',
	initialValue = '',
	placeholder = '',
	danger = false,
	submitting = false,
	onCancel,
	onConfirm
}: ModalDialogProps) {
	const [value, setValue] = useState('');
	const inputRef = useRef<HTMLInputElement | null>(null);

	useEffect(() => {
		if (!open) return;
		setValue(initialValue);
		if (mode === 'input') {
			queueMicrotask(() => inputRef.current?.focus());
		}
	}, [initialValue, mode, open]);

	if (!open) return null;

	return (
		<div className="modal-overlay">
			<button
				aria-label="关闭对话框"
				className="modal-backdrop"
				onClick={onCancel}
				type="button"
			/>
			<div aria-labelledby="modal-title" aria-modal="true" className="modal-card" role="dialog">
				<div className="modal-header">
					<div>
						<h2 id="modal-title">{title}</h2>
						{description ? <p>{description}</p> : null}
					</div>
					<button className="icon-ghost" onClick={onCancel} type="button">
						×
					</button>
				</div>
				{mode === 'input' ? (
					<div className="modal-body">
						<input
							onChange={(event) => setValue(event.target.value)}
							onKeyDown={(event) => event.key === 'Enter' && onConfirm(value)}
							placeholder={placeholder}
							ref={inputRef}
							value={value}
						/>
					</div>
				) : null}
				<div className="modal-footer">
					<button onClick={onCancel} type="button">
						{cancelLabel}
					</button>
					<button
						className={danger ? 'danger' : 'primary'}
						disabled={submitting}
						onClick={() => onConfirm(value)}
						type="button"
					>
						{submitting ? '处理中...' : confirmLabel}
					</button>
				</div>
			</div>
		</div>
	);
}
