import type {
  MutationResult,
  MutationResultOutcome,
} from '../../components/MutationResultPanel';
import { writeMutationResultSearchParams } from '../../components/MutationResultPanel';

export type CatalogMutationState<TLoading extends string | true = true> = {
  loading: TLoading | false;
  error: string | null;
  success: string | null;
};

export function createMutationState<TLoading extends string | true = true>(): CatalogMutationState<TLoading> {
  return {
    loading: false,
    error: null,
    success: null,
  };
}

export type CatalogMutationDefaults = Pick<MutationResult, 'packageName' | 'version' | 'actor'> & {
  requestId?: string | null;
};

export function buildCatalogMutationResult(
  action: string,
  outcome: MutationResultOutcome,
  message: string,
  defaults: CatalogMutationDefaults
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
  result: {
    action: string;
    outcome: MutationResultOutcome;
    message: string;
    requestId?: string | null;
  },
  defaults: CatalogMutationDefaults
) {
  return writeMutationResultSearchParams(
    searchParams,
    buildCatalogMutationResult(result.action, result.outcome, result.message, {
      ...defaults,
      requestId: result.requestId ?? null,
    })
  );
}
