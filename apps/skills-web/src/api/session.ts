import type { RegistrySession } from './types';

const MUTATION_ACTIONS = new Set([
  'packaging.prepare',
  'packaging.validate',
  'packaging.pack',
  'catalog.release_record.create',
  'catalog.governance.skill.apply',
  'catalog.governance.version.apply',
]);

export function canMutateCatalog(session: RegistrySession | null) {
  return Boolean(session?.allowedActions.some((action) => MUTATION_ACTIONS.has(action)));
}

export function hasPackageScope(session: RegistrySession | null) {
  return Boolean(session && session.allowedPackages.length > 0);
}

export function canAccessPackage(session: RegistrySession | null, packageName?: string | null) {
  if (!session) {
    return false;
  }
  if (!hasPackageScope(session)) {
    return true;
  }

  const normalizedPackage = packageName?.trim();
  if (!normalizedPackage) {
    return false;
  }

  return session.allowedPackages.some((pattern) => matchesPackagePattern(pattern, normalizedPackage));
}

export function canPerformAction(session: RegistrySession | null, action: string, packageName?: string | null) {
  return Boolean(session?.allowedActions.includes(action)) && canAccessPackage(session, packageName);
}

function matchesPackagePattern(pattern: string, packageName: string) {
  const normalizedPattern = pattern.trim();
  if (normalizedPattern === '') {
    return false;
  }
  if (normalizedPattern.endsWith('*')) {
    return packageName.startsWith(normalizedPattern.slice(0, -1));
  }
  return packageName === normalizedPattern;
}

export function describeSessionPosture(session: RegistrySession | null) {
  if (!session) {
    return 'No authenticated Catalog session';
  }
  if (canMutateCatalog(session) && hasPackageScope(session)) {
    return 'Catalog write access limited to allowed packages';
  }
  if (canMutateCatalog(session)) {
    return 'Catalog write access';
  }
  if (hasPackageScope(session)) {
    return 'Catalog read-only access limited to allowed packages';
  }
  return 'Catalog read-only access';
}

export function describeSessionScope(session: RegistrySession | null) {
  if (!hasPackageScope(session) || !session) {
    return null;
  }
  if (session.allowedPackages.length === 1) {
    return `scope ${session.allowedPackages[0]}`;
  }
  return `scope ${session.allowedPackages.length} packages`;
}
