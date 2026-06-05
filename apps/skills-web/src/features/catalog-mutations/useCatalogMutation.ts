import type {
  MutationResult,
  MutationResultOutcome,
} from '../../components/MutationResultPanel';
import { writeMutationResultSearchParams } from '../../components/MutationResultPanel';

export type CatalogMutationState<TLoading extends string | boolean = boolean> = {
  loading: TLoading | false;
  error: string | null;
  success: string | null;
};

export function createMutationState<TLoading extends string | boolean = boolean>(): CatalogMutationState<TLoading> {
  return {
    loading: false,
    error: null,
    success: null,
  };
}

export function buildCatalogMutationResult(
  action: string,
  outcome: MutationResultOutcome,
  message: string,
  defaults: Pick<MutationResult, 'packageName' | 'version' | 'actor'> & {
    requestId?: string | null;
  }
): MutationResult {
  return {
    action,
    outcome,
    message,
    requestId: defaults.requestId ?? null,
    packageName: defaults.packageName ?? null,
    version: defaults.version ?? null,
    actor: defaults.actor ?? null,
  };
}

export function persistCatalogMutation(
  searchParams: URLSearchParams,
  result: Parameters<typeof writeMutationResultSearchParams>[1]
) {
  return writeMutationResultSearchParams(searchParams, result);
}
