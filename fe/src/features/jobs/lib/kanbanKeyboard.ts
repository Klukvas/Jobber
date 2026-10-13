import {
  defaultKeyboardCoordinateGetter,
  type KeyboardCoordinateGetter,
} from "@dnd-kit/core";

/**
 * Keyboard coordinates for the Kanban board. dnd-kit's default moves the drag
 * 25px per arrow press, which is roughly seven presses per column. Here
 * ArrowRight / ArrowLeft jump straight to the next / previous column (the
 * droppables), centring the card in it; ArrowUp / ArrowDown keep dnd-kit's
 * default nudge.
 */
export const kanbanCoordinateGetter: KeyboardCoordinateGetter = (
  event,
  args,
) => {
  if (event.code !== "ArrowRight" && event.code !== "ArrowLeft") {
    return defaultKeyboardCoordinateGetter(event, args);
  }

  const { collisionRect, droppableRects, droppableContainers } = args.context;
  if (!collisionRect) return undefined;

  const columns = droppableContainers
    .getEnabled()
    .map((container) => droppableRects.get(container.id))
    .filter((rect): rect is NonNullable<typeof rect> => Boolean(rect))
    .sort((a, b) => a.left - b.left);
  if (columns.length === 0) return undefined;

  // The column the card currently sits over: nearest by horizontal centre.
  const cardCenter = collisionRect.left + collisionRect.width / 2;
  let current = 0;
  let best = Infinity;
  columns.forEach((rect, i) => {
    const distance = Math.abs(rect.left + rect.width / 2 - cardCenter);
    if (distance < best) {
      best = distance;
      current = i;
    }
  });

  const target = current + (event.code === "ArrowRight" ? 1 : -1);
  if (target < 0 || target >= columns.length) return undefined;

  const rect = columns[target];
  return {
    x: rect.left + (rect.width - collisionRect.width) / 2,
    y: args.currentCoordinates.y,
  };
};
