import { useEffect, useMemo, useRef, useState, type ReactNode, type UIEvent } from 'react';

type VirtualListProps<T> = {
	items: T[];
	itemKey: (item: T, index: number) => string;
	renderItem: (item: T, index: number) => ReactNode;
	rowHeight: number;
	height: number;
	overscan?: number;
	className?: string;
	emptyState?: ReactNode;
	activeIndex?: number;
};

export function VirtualList<T>({
	items,
	itemKey,
	renderItem,
	rowHeight,
	height,
	overscan = 4,
	className,
	emptyState,
	activeIndex = -1
}: VirtualListProps<T>) {
	const viewportRef = useRef<HTMLDivElement | null>(null);
	const [scrollTop, setScrollTop] = useState(0);
	const [viewportHeight, setViewportHeight] = useState(height);

	useEffect(() => {
		setViewportHeight(height);
	}, [height]);

	useEffect(() => {
		const viewport = viewportRef.current;
		if (!viewport || activeIndex < 0 || activeIndex >= items.length) return;

		const itemTop = activeIndex * rowHeight;
		const itemBottom = itemTop + rowHeight;
		const visibleTop = viewport.scrollTop;
		const visibleBottom = visibleTop + viewport.clientHeight;

		if (itemTop < visibleTop) {
			viewport.scrollTop = itemTop;
		} else if (itemBottom > visibleBottom) {
			viewport.scrollTop = itemBottom - viewport.clientHeight;
		}
	}, [activeIndex, items.length, rowHeight]);

	const range = useMemo(() => {
		if (!items.length) {
			return { start: 0, end: 0, offsetTop: 0, totalHeight: 0 };
		}
		const safeViewportHeight = Math.max(viewportHeight, rowHeight);
		const visibleCount = Math.ceil(safeViewportHeight / rowHeight);
		const start = Math.max(0, Math.floor(scrollTop / rowHeight) - overscan);
		const end = Math.min(items.length, start + visibleCount + overscan * 2);
		return {
			start,
			end,
			offsetTop: start * rowHeight,
			totalHeight: items.length * rowHeight
		};
	}, [items.length, overscan, rowHeight, scrollTop, viewportHeight]);

	function handleScroll(event: UIEvent<HTMLDivElement>) {
		setScrollTop(event.currentTarget.scrollTop);
	}

	if (!items.length) {
		return emptyState ? <>{emptyState}</> : null;
	}

	return (
		<div
			className={className ? `virtual-list ${className}` : 'virtual-list'}
			ref={viewportRef}
			onScroll={handleScroll}
			style={{ height: `${height}px` }}
		>
			<div className="virtual-list-inner" style={{ height: `${range.totalHeight}px` }}>
				<div className="virtual-list-window" style={{ transform: `translateY(${range.offsetTop}px)` }}>
					{items.slice(range.start, range.end).map((item, index) => (
						<div className="virtual-list-row" key={itemKey(item, range.start + index)}>
							{renderItem(item, range.start + index)}
						</div>
					))}
				</div>
			</div>
		</div>
	);
}
