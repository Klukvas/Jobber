import { useCallback, useRef } from "react";
import {
  useMutation,
  type MutateOptions,
  type UseMutationOptions,
  type UseMutationResult,
} from "@tanstack/react-query";

/**
 * `useMutation` whose `mutate` ignores calls while a previous one is in flight.
 *
 * Disabling the submit button on `isPending` is not enough: TanStack Query
 * publishes the pending state through its notify scheduler (a `setTimeout`), so
 * a double-click or Enter-then-click lands both events before the button ever
 * re-renders as disabled — and each one creates a record. The lock here is a
 * ref, set synchronously inside `mutate`, so the second call sees it at once.
 */
export function useSingleFlightMutation<
  TData = unknown,
  TError = Error,
  TVariables = void,
  TContext = unknown,
>(
  options: UseMutationOptions<TData, TError, TVariables, TContext>,
): UseMutationResult<TData, TError, TVariables, TContext> {
  const mutation = useMutation(options);
  const isInFlight = useRef(false);
  const { mutate: baseMutate } = mutation;

  const mutate = useCallback(
    (
      variables: TVariables,
      mutateOptions?: MutateOptions<TData, TError, TVariables, TContext>,
    ) => {
      if (isInFlight.current) return;
      isInFlight.current = true;
      baseMutate(variables, {
        ...mutateOptions,
        onSettled: (...args) => {
          isInFlight.current = false;
          return mutateOptions?.onSettled?.(...args);
        },
      });
    },
    [baseMutate],
  );

  return { ...mutation, mutate };
}
