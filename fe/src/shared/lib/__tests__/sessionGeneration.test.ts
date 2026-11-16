import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const KEY = "cursopus-session-generation";

/** A freshly loaded tab: the module reads storage once, at import. */
async function loadTab(account: string | null = null) {
  vi.resetModules();
  const mod = await import("../sessionGeneration");
  mod.setTabAccountSource(() => account);
  const replaced = vi.fn();
  mod.onSessionReplaced(replaced);
  return { ...mod, replaced };
}

/** Another tab writing the shared value, as the browser reports it here. */
function writtenElsewhere(value: string | null, key: string | null = KEY) {
  if (value === null) localStorage.removeItem(KEY);
  else localStorage.setItem(KEY, value);
  window.dispatchEvent(new StorageEvent("storage", { key }));
}

beforeEach(() => localStorage.clear());
afterEach(() => {
  vi.restoreAllMocks();
  localStorage.clear();
});

describe("session generation", () => {
  it("names the account a sign-in generation belongs to", async () => {
    const { accountOfGeneration } = await loadTab();
    expect(accountOfGeneration(null)).toBeNull();
    expect(accountOfGeneration("6f1c")).toBeNull();
    expect(accountOfGeneration("6f1c:user-1")).toBe("user-1");
    expect(accountOfGeneration("6f1c:")).toBeNull();
  });

  // Nobody has a stored generation the first time this ships. Every tab must
  // stay as it is until someone actually signs in or out.
  it("treats the first deploy, with nothing stored, as one current session", async () => {
    const tab = await loadTab("a");

    expect(tab.sharedSessionGeneration()).toBeNull();
    expect(tab.isTabSessionCurrent()).toBe(true);
    expect(tab.isSameAccountSession(null, "a")).toBe(true);
    tab.syncTabSession();
    expect(tab.replaced).not.toHaveBeenCalled();
  });

  it("signs the tab out when another tab logs out, signalled by a storage event", async () => {
    const tab = await loadTab("a");

    writtenElsewhere("logout-1");

    expect(tab.replaced).toHaveBeenCalledOnce();
    expect(tab.tabSessionGeneration()).toBe("logout-1");
    expect(tab.isTabSessionCurrent()).toBe(true);
  });

  it("treats localStorage.clear() in another tab as a change", async () => {
    localStorage.setItem(KEY, "g1:a");
    const tab = await loadTab("a");

    localStorage.clear();
    window.dispatchEvent(new StorageEvent("storage", { key: null }));

    expect(tab.replaced).toHaveBeenCalledOnce();
    expect(tab.tabSessionGeneration()).toBeNull();
  });

  it("keeps the tab when the same account signs in again elsewhere", async () => {
    const tab = await loadTab("a");

    writtenElsewhere("g2:a");

    expect(tab.replaced).not.toHaveBeenCalled();
    expect(tab.tabSessionGeneration()).toBe("g2:a");
    expect(tab.isTabSessionCurrent()).toBe(true);
  });

  it("signs the tab out when a different account signs in elsewhere", async () => {
    const tab = await loadTab("a");

    writtenElsewhere("g2:b");

    expect(tab.replaced).toHaveBeenCalledOnce();
    expect(tab.tabSessionGeneration()).toBe("g2:b");
  });

  it("lets a request refresh only while the cookies are still its account's", async () => {
    localStorage.setItem(KEY, "g1:a");
    const tab = await loadTab("a");

    localStorage.setItem(KEY, "g2:a");
    expect(tab.isSameAccountSession("g1:a", "a")).toBe(true);
    localStorage.setItem(KEY, "g3");
    expect(tab.isSameAccountSession("g1:a", "a")).toBe(false);
    localStorage.setItem(KEY, "g4:b");
    expect(tab.isSameAccountSession("g1:a", "a")).toBe(false);
    expect(tab.isSameAccountSession("g4:b", null)).toBe(true);
  });

  // A frozen or back/forward-cached tab misses the events; it must not keep
  // showing the old account when it comes back.
  it.each([
    ["pageshow", () => window.dispatchEvent(new Event("pageshow"))],
    ["visibilitychange", () => document.dispatchEvent(new Event("visibilitychange"))],
  ])("catches up on %s after a missed change", async (_event, fire) => {
    const tab = await loadTab("a");
    localStorage.setItem(KEY, "logout-1");
    expect(tab.replaced).not.toHaveBeenCalled();

    fire();

    expect(tab.replaced).toHaveBeenCalledOnce();
    expect(tab.isTabSessionCurrent()).toBe(true);
  });

  it("does not catch up on visibilitychange while the tab is hidden", async () => {
    const tab = await loadTab("a");
    vi.spyOn(document, "visibilityState", "get").mockReturnValue("hidden");
    localStorage.setItem(KEY, "logout-1");

    document.dispatchEvent(new Event("visibilitychange"));

    expect(tab.replaced).not.toHaveBeenCalled();
  });

  // One failed write (quota, say) used to stop the tab reading storage for
  // good, cutting it off from other tabs. It now recovers on the next write.
  it("uses storage again after a failed write is followed by a good one", async () => {
    const tab = await loadTab();
    const write = Storage.prototype.setItem;
    const setItem = vi.spyOn(Storage.prototype, "setItem").mockImplementationOnce(() => {
      throw new DOMException("Quota exceeded", "QuotaExceededError");
    });

    const own = tab.startSessionGeneration("a");
    expect(localStorage.getItem(KEY)).toBeNull();
    expect(tab.sharedSessionGeneration()).toBe(own);

    setItem.mockImplementation(function (this: Storage, key: string, value: string) {
      write.call(this, key, value);
    });
    const next = tab.startSessionGeneration(null);
    expect(localStorage.getItem(KEY)).toBe(next);

    localStorage.setItem(KEY, "g9:b");
    expect(tab.sharedSessionGeneration()).toBe("g9:b");
  });

  it("starts sign-in generations with the account and logout ones without", async () => {
    const tab = await loadTab();

    const signIn = tab.startSessionGeneration("a");
    const logout = tab.startSessionGeneration();

    expect(tab.accountOfGeneration(signIn)).toBe("a");
    expect(tab.accountOfGeneration(logout)).toBeNull();
    expect(localStorage.getItem(KEY)).toBe(logout);
  });
});
