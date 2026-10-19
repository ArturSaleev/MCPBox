// Ignore invalid or stale IDs without losing the API order of newly added projects.
export function readProjectOrder(value: string | null): number[] {
  try {
    const parsed: unknown = JSON.parse(value ?? '[]');
    return Array.isArray(parsed)
      ? [...new Set(parsed.filter((id): id is number => Number.isSafeInteger(id) && id > 0))]
      : [];
  } catch {
    return [];
  }
}

export function reconcileProjectOrder(ids: number[], saved: number[]): number[] {
  const available = new Set(ids);
  return [...new Set([...saved.filter(id => available.has(id)), ...ids])];
}

export function moveProject(order: number[], source: number, target: number, after: boolean): number[] {
  if (source === target || !order.includes(source) || !order.includes(target)) return order;
  const next = order.filter(id => id !== source);
  next.splice(next.indexOf(target) + (after ? 1 : 0), 0, source);
  return next;
}
