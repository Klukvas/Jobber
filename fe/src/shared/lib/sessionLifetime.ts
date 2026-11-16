/** A request belongs to one sign-in lifetime, even when the account ID is reused. */
let controller = new AbortController();

export function sessionSignal(): AbortSignal {
  return controller.signal;
}

export function changeSessionLifetime(active: boolean): void {
  controller.abort(new DOMException("Session changed", "AbortError"));
  controller = new AbortController();
  if (!active) controller.abort(new DOMException("Session ended", "AbortError"));
}
