import { QueryClient } from "@tanstack/react-query";

/**
 * The app's single React Query cache.
 *
 * It lives here rather than next to the provider that installs it because the
 * places that have to *empty* it — signing out, and the API client's forced
 * sign-out on a refused refresh — sit underneath the provider in the import
 * graph. Reaching up from `services/api` to `app/providers` would close a
 * cycle: the provider tree imports the router, which imports every page, which
 * imports the services.
 *
 * Built on first use rather than at import. Every service module now reaches
 * this one, so constructing a cache at import time would stand a real client up
 * inside any test that merely touches a service — including the many that
 * replace `@tanstack/react-query` wholesale and have no `QueryClient` to
 * offer.
 */
let client: QueryClient | null = null;

export function getQueryClient(): QueryClient {
  if (!client) {
    client = new QueryClient({
      defaultOptions: {
        queries: {
          staleTime: 1000 * 60 * 5, // 5 minutes
          retry: 1,
          refetchOnWindowFocus: false,
        },
      },
    });
  }
  return client;
}
