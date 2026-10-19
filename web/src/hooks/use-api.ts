"use client";

import {
  useQuery,
  useMutation,
  useQueryClient,
  type UseQueryOptions,
  type UseMutationOptions,
  type QueryKey,
} from "@tanstack/react-query";
import { toast } from "@/components/providers/toast-provider";
import type { ApiError } from "@/lib/api";

export interface ApiResponse<T> {
  data: T;
  message?: string;
}

export interface ApiErrorResponse {
  code: number;
  message: string;
  details?: unknown;
}

export function useApiQuery<TData, TError = ApiError>(
  queryKey: QueryKey,
  queryFn: () => Promise<TData>,
  options?: Omit<UseQueryOptions<TData, TError, TData>, "queryKey" | "queryFn">
) {
  return useQuery<TData, TError, TData>({
    queryKey,
    queryFn,
    ...options,
  });
}

export function useApiMutation<TData, TVariables, TError = ApiError>(
  mutationFn: (variables: TVariables) => Promise<TData>,
  options?: Omit<
    UseMutationOptions<TData, TError, TVariables>,
    "mutationFn"
  > & {
    successMessage?: string;
    invalidateKeys?: QueryKey[];
  }
) {
  const queryClient = useQueryClient();
  const { successMessage, invalidateKeys, ...mutationOptions } = options || {};

  return useMutation<TData, TError, TVariables>({
    mutationFn,
    ...mutationOptions,
    onSuccess: (data, variables, onMutateResult, context) => {
      if (successMessage) {
        toast.success(successMessage);
      }
      if (invalidateKeys) {
        invalidateKeys.forEach((key) => {
          queryClient.invalidateQueries({ queryKey: key });
        });
      }
      mutationOptions.onSuccess?.(data, variables, onMutateResult, context);
    },
    onError: (error, variables, onMutateResult, context) => {
      const errorMessage =
        (error as ApiErrorResponse).message || "操作失败，请稍后重试";
      toast.error(errorMessage);
      mutationOptions.onError?.(error, variables, onMutateResult, context);
    },
  });
}

export function useApiInvalidate() {
  const queryClient = useQueryClient();

  return {
    invalidateQueries: (queryKey: QueryKey) =>
      queryClient.invalidateQueries({ queryKey }),
    invalidateAll: () => queryClient.invalidateQueries(),
    resetQueries: (queryKey: QueryKey) =>
      queryClient.resetQueries({ queryKey }),
    removeQueries: (queryKey: QueryKey) =>
      queryClient.removeQueries({ queryKey }),
  };
}
