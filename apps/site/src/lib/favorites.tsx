import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  type PropsWithChildren,
} from 'react';
import type { ExternalSkill } from './skills-directory-api';
import { useAuth } from './auth';

const STORAGE_KEY = 'skill-dock:favorites:v1';

export interface FavoriteSnapshot {
  name?: string;
  source?: string;
  installs?: number;
  isOfficial?: boolean;
  installCommand?: string;
}

export interface StoredFavorite {
  refKey: string;
  refType: 'catalog' | 'external';
  packageName?: string;
  externalProvider?: string;
  externalSource?: string;
  externalSkillId?: string;
  snapshot: FavoriteSnapshot;
  createdAt?: string;
}

interface FavoriteTarget {
  refKey: string;
  refType: 'catalog' | 'external';
  packageName?: string;
  externalProvider?: string;
  externalSource?: string;
  externalSkillId?: string;
  snapshot: FavoriteSnapshot;
}

interface FavoritesState {
  favorites: StoredFavorite[];
  count: number;
  isFavorite: (refKey: string) => boolean;
  toggle: (target: FavoriteTarget) => Promise<void>;
  remove: (refKey: string) => Promise<void>;
  signedIn: boolean;
}

const fallback: FavoritesState = {
  favorites: [],
  count: 0,
  isFavorite: () => false,
  toggle: async () => {},
  remove: async () => {},
  signedIn: false,
};

const FavoritesContext = createContext<FavoritesState>(fallback);

export function useFavorites(): FavoritesState {
  return useContext(FavoritesContext);
}

export function externalRefKey(skill: Pick<ExternalSkill, 'source' | 'skillId'>): string {
  return `external:skills.sh:${skill.source}:${skill.skillId}`;
}

export function catalogRefKey(packageName: string): string {
  return `catalog:${packageName}`;
}

export function favoriteFromExternal(skill: ExternalSkill | ExternalSkillLike): FavoriteTarget {
  return {
    refKey: externalRefKey(skill),
    refType: 'external',
    externalProvider: 'skills.sh',
    externalSource: skill.source,
    externalSkillId: skill.skillId,
    snapshot: {
      name: skill.name,
      source: skill.source,
      installs: skill.installs,
      isOfficial: skill.isOfficial,
      installCommand: skill.installCommand,
    },
  };
}

interface ExternalSkillLike {
  source: string;
  skillId: string;
  name: string;
  installs?: number;
  isOfficial?: boolean;
  installCommand?: string;
}

export function FavoritesProvider({ children }: PropsWithChildren) {
  const auth = useAuth();
  const [favorites, setFavorites] = useState<StoredFavorite[]>(() => readLocal());

  useEffect(() => {
    writeLocal(favorites);
  }, [favorites]);

  useEffect(() => {
    if (!auth.enabled || !auth.authenticated) return;
    let cancelled = false;
    (async () => {
      try {
        const local = readLocal();
        if (local.length > 0) {
          await auth.authFetch(`${auth.apiBase}/me/favorites/merge`, {
            method: 'POST',
            headers: { 'content-type': 'application/json' },
            body: JSON.stringify({ favorites: local.map(toServerPayload) }),
          });
        }
        const response = await auth.authFetch(`${auth.apiBase}/me/favorites`);
        if (!response.ok) return;
        const payload = (await response.json()) as { items?: ServerFavorite[] };
        if (cancelled) return;
        const merged = (payload.items ?? []).map(fromServerFavorite).filter((item): item is StoredFavorite => Boolean(item));
        setFavorites(merged);
        writeLocal(merged);
      } catch {
        // The local mirror remains usable when the catalog is unavailable.
      }
    })();
    return () => {
      cancelled = true;
    };
    // The auth object is intentionally read at merge time.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [auth.authenticated, auth.activeOrganization?.id]);

  const isFavorite = useCallback(
    (refKey: string) => favorites.some((item) => item.refKey === refKey),
    [favorites]
  );

  const remove = useCallback(
    async (refKey: string) => {
      setFavorites((current) => current.filter((item) => item.refKey !== refKey));
      if (auth.enabled && auth.authenticated) {
        try {
          await auth.authFetch(`${auth.apiBase}/me/favorites?refKey=${encodeURIComponent(refKey)}`, {
            method: 'DELETE',
          });
        } catch {
          // Keep the optimistic local removal.
        }
      }
    },
    [auth]
  );

  const toggle = useCallback(
    async (target: FavoriteTarget) => {
      if (isFavorite(target.refKey)) {
        await remove(target.refKey);
        return;
      }
      const next: StoredFavorite = {
        refKey: target.refKey,
        refType: target.refType,
        packageName: target.packageName,
        externalProvider: target.externalProvider,
        externalSource: target.externalSource,
        externalSkillId: target.externalSkillId,
        snapshot: target.snapshot,
        createdAt: new Date().toISOString(),
      };
      setFavorites((current) => [next, ...current.filter((item) => item.refKey !== target.refKey)]);
      if (auth.enabled && auth.authenticated) {
        try {
          await auth.authFetch(`${auth.apiBase}/me/favorites`, {
            method: 'POST',
            headers: { 'content-type': 'application/json' },
            body: JSON.stringify(toServerPayload(next)),
          });
        } catch {
          // The local favorite remains; merge on the next sign-in.
        }
      }
    },
    [auth, isFavorite, remove]
  );

  const value = useMemo<FavoritesState>(
    () => ({
      favorites,
      count: favorites.length,
      isFavorite,
      toggle,
      remove,
      signedIn: auth.authenticated,
    }),
    [auth.authenticated, favorites, isFavorite, remove, toggle]
  );

  return <FavoritesContext.Provider value={value}>{children}</FavoritesContext.Provider>;
}

interface ServerFavorite {
  refKey: string;
  refType: string;
  packageName?: string;
  externalProvider?: string;
  externalSource?: string;
  externalSkillId?: string;
  snapshot?: FavoriteSnapshot;
  createdAt?: string;
  available?: boolean;
}

function toServerPayload(favorite: StoredFavorite) {
  return {
    ref: {
      refKey: favorite.refKey,
      refType: favorite.refType,
      packageName: favorite.packageName,
      externalProvider: favorite.externalProvider,
      externalSource: favorite.externalSource,
      externalSkillId: favorite.externalSkillId,
    },
    snapshot: favorite.snapshot,
  };
}

function fromServerFavorite(item: ServerFavorite): StoredFavorite | null {
  if (item.available === false) {
    return {
      refKey: item.refKey,
      refType: item.refType === 'catalog' ? 'catalog' : 'external',
      snapshot: {},
    };
  }
  if (item.refType === 'catalog' && !item.packageName) return null;
  return {
    refKey: item.refKey,
    refType: item.refType === 'catalog' ? 'catalog' : 'external',
    packageName: item.packageName,
    externalProvider: item.externalProvider,
    externalSource: item.externalSource,
    externalSkillId: item.externalSkillId,
    snapshot: item.snapshot ?? {},
    createdAt: item.createdAt,
  };
}

function readLocal(): StoredFavorite[] {
  if (typeof window === 'undefined') return [];
  try {
    const raw = window.localStorage.getItem(STORAGE_KEY);
    if (!raw) return [];
    const parsed = JSON.parse(raw) as StoredFavorite[];
    if (!Array.isArray(parsed)) return [];
    return parsed.filter((item) => item && typeof item.refKey === 'string');
  } catch {
    return [];
  }
}

function writeLocal(favorites: StoredFavorite[]) {
  if (typeof window === 'undefined') return;
  try {
    window.localStorage.setItem(STORAGE_KEY, JSON.stringify(favorites));
  } catch {
    // Ignore quota and privacy-mode failures.
  }
}
