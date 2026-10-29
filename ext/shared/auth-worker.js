// Only the service worker mutates credentials. Panels share its refresh flight.
const JobberAuth = (() => {
  const API_BASE = "https://cursopus.com";
  const ALLOWED_ORIGINS = ["https://cursopus.com", "http://localhost:8080"];
  const PRIVATE_KEYS = ["accessToken", "refreshToken", "autofillProfile", "autofillSessionId", "autofillProfileRequest"];
  let writes = Promise.resolve();
  let flight = null;
  let retryAfter = 0;
  const profileRequests = new Map();
  let registrations = Promise.resolve();

  function safeBase(value) {
    try {
      const url = new URL(value);
      if (ALLOWED_ORIGINS.includes(url.origin) && url.href.replace(/\/$/, "") === url.origin) return url.origin;
    } catch { /* Missing/invalid persisted setting falls back to production. */ }
    return API_BASE;
  }

  function serialize(action) {
    const result = writes.then(action);
    writes = result.catch(() => {});
    return result;
  }

  async function read() {
    const data = await chrome.storage.local.get(["accessToken", "refreshToken", "authSessionId", "apiBase"]);
    return { ...data, apiBase: safeBase(data.apiBase) };
  }

  function snapshot() {
    return serialize(async () => {
      let data = await read();
      if (!data.authSessionId) {
        data = { ...data, authSessionId: crypto.randomUUID() };
        await chrome.storage.local.set({ authSessionId: data.authSessionId });
      }
      return data;
    });
  }

  function sameSession(current, original) {
    return current.authSessionId === original.authSessionId && current.apiBase === original.apiBase;
  }

  async function assertSession(original) {
    const current = await snapshot();
    if (!original?.accessToken || !current.accessToken || !sameSession(current, original)) throw new Error("AI_CANCELLED");
    return current;
  }

  function invalidate(expected) {
    return serialize(async () => {
      const current = await read();
      if (expected && !sameSession(current, expected)) return;
      // A new generation also invalidates refresh/login responses still in flight.
      await chrome.storage.local.set({ authSessionId: crypto.randomUUID() });
      await chrome.storage.local.remove(PRIVATE_KEYS);
      retryAfter = 0;
      return read();
    });
  }

  async function login(email, password) {
    const original = await invalidate();
    const response = await fetch(`${original.apiBase}/api/v1/auth/login`, {
      method: "POST", headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ email, password }),
    });
    if (!response.ok) {
      const error = await response.json().catch(() => null);
      throw new Error(error?.error_message || "Invalid credentials");
    }
    const data = await response.json();
    if (!data.tokens?.access_token || !data.tokens?.refresh_token) throw new Error("Invalid login response");
    return serialize(async () => {
      if (!sameSession(await read(), original)) throw new Error("AI_CANCELLED");
      await chrome.storage.local.set({ accessToken: data.tokens.access_token, refreshToken: data.tokens.refresh_token });
      return read();
    });
  }

  async function refresh(original) {
    const current = await assertSession(original);
    // Another caller already rotated the token used by this request.
    if (current.accessToken !== original.accessToken) return current;
    if (flight && sameSession(flight.session, current)) return flight.promise;
    if (!current.refreshToken || Date.now() < retryAfter) throw new Error("AUTH_RETRY_LATER");
    const promise = rotate(current);
    flight = { session: current, promise };
    try { return await promise; }
    finally { if (flight?.promise === promise) flight = null; }
  }

  async function rotate(original) {
    let response;
    try {
      response = await fetch(`${original.apiBase}/api/v1/auth/refresh`, {
        method: "POST", headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ refresh_token: original.refreshToken }),
      });
    } catch {
      await assertSession(original);
      retryAfter = Date.now() + 30000;
      throw new Error("AUTH_RETRY_LATER");
    }
    await assertSession(original);
    if (!response.ok) {
      if (response.status === 401 || response.status === 403) {
        await invalidate(original);
        throw new Error("SESSION_EXPIRED");
      }
      retryAfter = Date.now() + 30000;
      throw new Error("AUTH_RETRY_LATER");
    }
    const data = await response.json();
    if (!data.access_token || !data.refresh_token) {
      retryAfter = Date.now() + 30000;
      throw new Error("AUTH_RETRY_LATER");
    }
    return serialize(async () => {
      const current = await read();
      if (!sameSession(current, original) || current.refreshToken !== original.refreshToken) throw new Error("AI_CANCELLED");
      await chrome.storage.local.set({ accessToken: data.access_token, refreshToken: data.refresh_token });
      retryAfter = 0;
      return read();
    });
  }

  async function authedFetch(path, options = {}, original) {
    const session = await assertSession(original || await snapshot());
    const send = (token) => fetch(`${session.apiBase}${path}`, {
      ...options,
      headers: { "Content-Type": "application/json", ...options.headers, Authorization: `Bearer ${token}` },
    });
    let response = await send(session.accessToken);
    await assertSession(session);
    if (response.status === 401) {
      const fresh = await refresh(session);
      await assertSession(session);
      response = await send(fresh.accessToken);
      await assertSession(session);
    }
    return response;
  }

  function sameProfileRequest(left, right) {
    return left?.owner === right?.owner && left?.revision === right?.revision && left?.sessionId === right?.sessionId && left?.apiBase === right?.apiBase;
  }
  function profileKey(request, caller) { return `${caller}|${request?.owner}`; }
  function profileIsCurrent(request, caller) {
    const current = profileRequests.get(profileKey(request, caller));
    return current?.verified && !current.cancelled && sameProfileRequest(current, request);
  }
  async function latestRegistration(owner) {
    const stored = await chrome.storage.session.get(null);
    const prefix = `autofillPending:${owner}:`;
    return Object.entries(stored).filter(([key, value]) => key.startsWith(prefix) && value.owner === owner)
      .map(([, value]) => value).sort((left, right) => right.revision - left.revision)[0];
  }
  async function isProfileRevoked(request) {
    const key = `autofillRevoked:${request.owner}`;
    return (await chrome.storage.local.get(key))[key] >= request.revision;
  }
  function beginProfile(request, caller) {
    if (!request || typeof request.owner !== "string" || !Number.isSafeInteger(request.revision) || request.revision < 1 || !request.sessionId) throw new Error("AI_CANCELLED");
    const key = profileKey(request, caller);
    const previous = profileRequests.get(key);
    if (previous && previous.revision >= request.revision) {
      if (sameProfileRequest(previous, request) && !previous.cancelled) return previous.ready || true;
      throw new Error("AI_CANCELLED");
    }
    const persist = registrations.then(async () => {
      const latest = await latestRegistration(request.owner);
      if (latest && (latest.caller !== caller || latest.revision > request.revision || (latest.revision === request.revision && !sameProfileRequest(latest, request)))) throw new Error("AI_CANCELLED");
      if (await isProfileRevoked(request)) throw new Error("AI_CANCELLED");
      const active = profileRequests.get(key);
      if (!sameProfileRequest(active, request) || active.cancelled) throw new Error("AI_CANCELLED");
      // Immutable revision records cannot overwrite a newer registration. This
      // queue is separate from profile/auth writes, so cancellation never waits
      // behind a delayed profile GET/SET. HTTP starts only after this is durable.
      await chrome.storage.session.set({ [`autofillPending:${request.owner}:${request.revision}`]: { ...request, caller } });
      const current = profileRequests.get(key);
      if (!sameProfileRequest(current, request) || current.cancelled) {
        await chrome.storage.local.set({ [`autofillRevoked:${request.owner}`]: request.revision });
        throw new Error("AI_CANCELLED");
      }
      profileRequests.set(key, { ...current, verified: true });
      return true;
    });
    registrations = persist.catch(() => {});
    profileRequests.set(key, { ...request, caller, cancelled: false, verified: false, ready: persist });
    return persist;
  }
  async function restoreProfileRequest(request, caller) {
    if (!request) return false;
    const key = profileKey(request, caller);
    const active = profileRequests.get(key);
    if (active) {
      if (!sameProfileRequest(active, request) || active.cancelled) return false;
      if (active.ready) await active.ready;
      return profileIsCurrent(request, caller);
    }
    const latest = await latestRegistration(request.owner);
    if (!latest || latest.caller !== caller || !sameProfileRequest(latest, request) || await isProfileRevoked(request)) return false;
    // A begin/cancel may have arrived while storage was being read.
    const current = profileRequests.get(key);
    if (current) return profileIsCurrent(request, caller);
    profileRequests.set(key, { ...latest, verified: true, cancelled: false });
    return true;
  }
  async function removeOwnProfile(request) {
    const { autofillProfileRequest } = await chrome.storage.local.get("autofillProfileRequest");
    if (sameProfileRequest(autofillProfileRequest, request)) {
      await chrome.storage.local.remove(["autofillProfile", "autofillSessionId", "autofillProfileRequest"]);
    }
  }
  async function cancelProfile(request, caller) {
    const key = profileKey(request, caller);
    if (!profileRequests.has(key) && !await restoreProfileRequest(request, caller)) return true;
    const active = profileRequests.get(key);
    if (!sameProfileRequest(active, request) || active.cancelled) return true;
    profileRequests.set(key, { ...active, cancelled: true });
    if (!active.verified) return true; // begin will persist its revocation before acknowledging.
    // Invalidate immediately, outside the queue a delayed profile store owns.
    const persisted = chrome.storage.local.set({ [`autofillRevoked:${request.owner}`]: request.revision });
    void serialize(() => removeOwnProfile(request)).catch(() => {});
    return persisted.then(() => true);
  }
  function storeProfile(profile, request, caller) {
    return serialize(async () => {
      const current = await read();
      if (!current.accessToken || current.authSessionId !== request?.sessionId || current.apiBase !== request.apiBase) throw new Error("AI_CANCELLED");
      if (!await restoreProfileRequest(request, caller) || !profileIsCurrent(request, caller)) throw new Error("AI_CANCELLED");
      await chrome.storage.local.set({ autofillProfile: profile, autofillSessionId: request.sessionId, autofillProfileRequest: { ...request, caller } });
      if (!profileIsCurrent(request, caller)) {
        await removeOwnProfile(request);
        throw new Error("AI_CANCELLED");
      }
    });
  }
  function getAutofillProfile() {
    return serialize(async () => {
      const data = await chrome.storage.local.get(["accessToken", "authSessionId", "apiBase", "autofillProfile", "autofillSessionId", "autofillProfileRequest"]);
      const request = data.autofillProfileRequest;
      if (!data.accessToken || !request || data.authSessionId !== data.autofillSessionId || request.sessionId !== data.authSessionId || request.apiBase !== safeBase(data.apiBase)) return { profile: null };
      const key = `autofillRevoked:${request.owner}`;
      const revoked = await chrome.storage.local.get(key);
      const live = profileRequests.get(profileKey(request, request.caller));
      if (revoked[key] >= request.revision || (sameProfileRequest(live, request) && live.cancelled)) return { profile: null };
      return { profile: data.autofillProfile || null };
    });
  }

  function withSession(original, action) {
    return serialize(async () => {
      const current = await read();
      if (!current.accessToken || !sameSession(current, original)) throw new Error("AI_CANCELLED");
      return action();
    });
  }
  async function trackUrl(url) {
    const { importedUrls = [] } = await chrome.storage.local.get("importedUrls");
    if (url && !importedUrls.includes(url)) {
      await chrome.storage.local.set({ importedUrls: [...importedUrls, url].slice(-500) });
    }
    return true;
  }
  async function incrementCount() {
    const { savedCount = 0 } = await chrome.storage.session.get("savedCount");
    const count = savedCount + 1;
    await chrome.storage.session.set({ savedCount: count });
    return count;
  }
  function finishQuickSave(original, job, updateBadge, onSuccess) {
    return withSession(original, async () => {
      // Keep every read/write, badge update and the success response ordered
      // before a queued logout/login; none may resume under another account.
      await trackUrl(job.url);
      const count = await incrementCount();
      await updateBadge(count);
      const result = { ok: true, job };
      if (onSuccess) { onSuccess(result); return null; }
      return result;
    });
  }

  function publishPending(original, pending) {
    return serialize(async () => {
      const current = await read();
      if (!sameSession(current, original)) return;
      await chrome.storage.session.set({
        pendingParse: { ...pending, sessionId: current.accessToken ? current.authSessionId : null },
      });
    });
  }

  async function handle(message, caller) {
    switch (message.action) {
      case "authTakePending": return serialize(async () => {
        const current = await read();
        if (!current.accessToken || current.authSessionId !== message.sessionId) return null;
        const { pendingParse } = await chrome.storage.session.get("pendingParse");
        if (!pendingParse) return null;
        await chrome.storage.session.remove("pendingParse");
        if (Date.now() - pendingParse.timestamp > 10000 || (pendingParse.sessionId && pendingParse.sessionId !== current.authSessionId)) return null;
        return pendingParse;
      });
      case "authTrackImportedUrl": return withSession(message.session, () => trackUrl(message.url));
      case "authIncrementSavedCount": return withSession(message.session, incrementCount);
      case "authSnapshot": return snapshot();
      case "authLogin": return login(message.email, message.password);
      case "authLogout": return invalidate(message.session);
      case "authRefresh": return refresh(message.session);
      case "authBeginProfile": return beginProfile(message.request, caller);
      case "authCancelProfile": return cancelProfile(message.request, caller);
      case "authStoreProfile": return storeProfile(message.profile, message.request, caller);
      default: throw new Error("Unknown auth action");
    }
  }
  return { snapshot, assertSession, refresh, authedFetch, getAutofillProfile, finishQuickSave, publishPending, handle, safeBase };
})();
