/**
 * The one address this bundle reports from.
 *
 * The same build also runs where nothing may be reported: inside the
 * prerenderer during the Docker build, and in the backend's headless Chrome,
 * which opens `http://frontend/print/resume` with a customer's resume on the
 * page. A build flag cannot tell those apart from a visitor; the address can.
 */
export const PRODUCTION_HOSTNAME = "cursopus.com";

export function isProductionHost(): boolean {
  return window.location.hostname === PRODUCTION_HOSTNAME;
}
