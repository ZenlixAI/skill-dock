import { Link } from 'react-router-dom';
import { RegistryHttpError } from '../api/httpClient';
import { StatusPill } from './StatusPill';
import { SurfaceCard } from './SurfaceCard';
import { buildAuditLink } from '../lib/auditLink';

export type MutationResultOutcome = 'succeeded' | 'failed' | 'forbidden' | 'unauthorized';

export type MutationResult = {
  action: string;
  outcome: MutationResultOutcome;
  message: string;
  requestId?: string | null;
  packageName?: string | null;
  version?: string | null;
  actor?: string | null;
};

type MutationResultPanelProps = {
  result: MutationResult | null;
};

const QUERY_KEYS = {
  action: 'mutation',
  outcome: 'mutationOutcome',
  message: 'mutationMessage',
  requestId: 'mutationRequestId',
  packageName: 'mutationPackage',
  version: 'mutationVersion',
  actor: 'mutationActor',
} as const;

function normalizeValue(value?: string | null) {
  const trimmed = value?.trim();
  return trimmed ? trimmed : null;
}

export function inferMutationFailureOutcome(error: unknown): Exclude<MutationResultOutcome, 'succeeded'> {
  if (error instanceof RegistryHttpError) {
    if (error.status === 401) {
      return 'unauthorized';
    }
    if (error.status === 403) {
      return 'forbidden';
    }
  }

  return 'failed';
}

export function readMutationResultSearchParams(
  searchParams: URLSearchParams,
  defaults: Pick<MutationResult, 'packageName' | 'version' | 'actor'> = {
    packageName: null,
    version: null,
    actor: null,
  }
): MutationResult | null {
  const action = normalizeValue(searchParams.get(QUERY_KEYS.action));
  const outcome = searchParams.get(QUERY_KEYS.outcome);
  const message = normalizeValue(searchParams.get(QUERY_KEYS.message));

  if (
    !action ||
    !message ||
    (outcome !== 'succeeded' && outcome !== 'failed' && outcome !== 'forbidden' && outcome !== 'unauthorized')
  ) {
    return null;
  }

  return {
    action,
    outcome,
    message,
    requestId: normalizeValue(searchParams.get(QUERY_KEYS.requestId)),
    packageName: normalizeValue(searchParams.get(QUERY_KEYS.packageName)) ?? defaults.packageName ?? null,
    version: normalizeValue(searchParams.get(QUERY_KEYS.version)) ?? defaults.version ?? null,
    actor: normalizeValue(searchParams.get(QUERY_KEYS.actor)) ?? defaults.actor ?? null,
  };
}

export function writeMutationResultSearchParams(searchParams: URLSearchParams, result: MutationResult) {
  const next = new URLSearchParams(searchParams);

  next.set(QUERY_KEYS.action, result.action);
  next.set(QUERY_KEYS.outcome, result.outcome);
  next.set(QUERY_KEYS.message, result.message);

  const requestId = normalizeValue(result.requestId);
  const packageName = normalizeValue(result.packageName);
  const version = normalizeValue(result.version);
  const actor = normalizeValue(result.actor);

  if (requestId) {
    next.set(QUERY_KEYS.requestId, requestId);
  } else {
    next.delete(QUERY_KEYS.requestId);
  }

  if (packageName) {
    next.set(QUERY_KEYS.packageName, packageName);
  } else {
    next.delete(QUERY_KEYS.packageName);
  }

  if (version) {
    next.set(QUERY_KEYS.version, version);
  } else {
    next.delete(QUERY_KEYS.version);
  }

  if (actor) {
    next.set(QUERY_KEYS.actor, actor);
  } else {
    next.delete(QUERY_KEYS.actor);
  }

  return next;
}

export function MutationResultPanel({ result }: MutationResultPanelProps) {
  if (!result) {
    return null;
  }

  const isSuccess = result.outcome === 'succeeded';
  const statusLabel =
    result.outcome === 'succeeded'
      ? 'success'
      : result.outcome === 'failed'
        ? 'failure'
        : result.outcome;

  const auditLink = buildAuditLink({
    packageName: result.packageName,
    version: result.version,
    outcome: result.outcome,
    actor: result.actor,
  });

  return (
    <SurfaceCard
      className={!isSuccess ? 'surface-card--cool' : undefined}
      description="Latest write attempt on this page, persisted in the URL for refresh-safe follow-up."
      title="Mutation Result"
      variant="rail"
    >
      <div className="pill-row">
        <StatusPill tone="accent">{result.action}</StatusPill>
        <StatusPill tone={isSuccess ? 'good' : 'bad'}>{statusLabel}</StatusPill>
        {result.packageName ? <StatusPill>{result.packageName}</StatusPill> : null}
        {result.version ? <StatusPill>{result.version}</StatusPill> : null}
      </div>
      {isSuccess ? (
        <div className="empty-box empty-box--success">{result.message}</div>
      ) : (
        <div className="error-box">{result.message}</div>
      )}
      {result.requestId ? <p className="support-copy">Request ID: {result.requestId}</p> : null}
      <div className="list-card__actions">
        <Link className="button button--secondary" to={auditLink}>
          Open audit trail
        </Link>
      </div>
    </SurfaceCard>
  );
}
