import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import type { SdzSpot } from './types/spot';
import { useAuth } from './contexts/useAuth';
import { Route, Routes, useNavigate, useParams } from 'react-router-dom';
import { CircleMarker, MapContainer, TileLayer, Tooltip, useMap } from 'react-leaflet';
import L from 'leaflet';
import { SdzAdminGuard } from './admin/SdzAdminGuard';
import { SdzAdminLayout } from './admin/SdzAdminLayout';
import { SdzAdminSpotList } from './admin/SdzAdminSpotList';
import { SdzAdminSpotForm } from './admin/SdzAdminSpotForm';

const apiUrl = import.meta.env.VITE_SDZ_API_URL || 'http://localhost:8080';
const SDZ_FAVORITES_PAGE_SIZE = 8;
const SDZ_TYPE_TAGS = new Set(['パーク', 'ストリート', 'スケートパーク', 'スケートボードパーク']);
const SDZ_PARK_TAGS = new Set(['パーク', 'スケートパーク', 'スケートボードパーク']);
const SDZ_STREET_TAGS = new Set(['ストリート']);

function formatCoords(location?: SdzSpot['location']) {
  if (!location) return 'N/A';
  return `${location.lat.toFixed(4)}, ${location.lng.toFixed(4)}`;
}

function formatDateTime(value?: string) {
  if (!value) return '-';
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return date.toLocaleString('ja-JP', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  });
}

function paginate<T>(items: T[], page: number, perPage: number) {
  const totalPages = Math.max(1, Math.ceil(items.length / perPage));
  const safePage = Math.min(Math.max(page, 1), totalPages);
  const start = (safePage - 1) * perPage;
  return {
    page: safePage,
    totalPages,
    items: items.slice(start, start + perPage),
  };
}

function getInitialFavorites(): string[] {
  if (typeof window === 'undefined') return [];
  const raw = window.localStorage.getItem('sdzFavorites');
  if (!raw) return [];
  try {
    const parsed = JSON.parse(raw);
    return Array.isArray(parsed) ? parsed.filter((v) => typeof v === 'string') : [];
  } catch {
    return [];
  }
}

function getSpotTypeLabel(tags?: string[]) {
  const tagList = tags ?? [];
  if (tagList.some((tag) => SDZ_PARK_TAGS.has(tag))) return 'スケートパーク';
  if (tagList.some((tag) => SDZ_STREET_TAGS.has(tag))) return 'ストリート';
  return 'スポット';
}

function getSpotTone(tags?: string[]) {
  const typeLabel = getSpotTypeLabel(tags);
  if (typeLabel === 'スケートパーク') {
    return { label: typeLabel, color: '#32b990' };
  }
  if (typeLabel === 'ストリート') {
    return { label: typeLabel, color: '#ff7a45' };
  }
  return { label: typeLabel, color: '#7b8a94' };
}

type SdzMapCameraControllerProps = {
  selectedSpot?: SdzSpot;
  spots: SdzSpot[];
};

function SdzMapCameraController({ selectedSpot, spots }: SdzMapCameraControllerProps) {
  const map = useMap();

  useEffect(() => {
    if (selectedSpot?.location) {
      map.flyTo([selectedSpot.location.lat, selectedSpot.location.lng], 15, { duration: 0.6 });
      return;
    }
    const located = spots.filter((spot) => spot.location);
    if (located.length === 0) return;
    if (located.length === 1) {
      const { lat, lng } = located[0].location!;
      map.setView([lat, lng], 13);
      return;
    }
    const bounds = L.latLngBounds(
      located.map((spot) => [spot.location!.lat, spot.location!.lng] as [number, number]),
    );
    map.fitBounds(bounds, { padding: [80, 80], maxZoom: 15 });
  }, [map, selectedSpot, spots]);

  return null;
}

type SdzSpotMapProps = {
  spots: SdzSpot[];
  selectedSpotId: string | null;
  onSelect: (spotId: string) => void;
};

function SdzSpotMap({ spots, selectedSpotId, onSelect }: SdzSpotMapProps) {
  const fallbackCenter: [number, number] = [35.6812, 139.7671];
  const selectedSpot = spots.find((spot) => spot.spotId === selectedSpotId);
  const mapCenter: [number, number] = selectedSpot?.location
    ? [selectedSpot.location.lat, selectedSpot.location.lng]
    : fallbackCenter;

  return (
    <MapContainer center={mapCenter} zoom={13} className="sdz-map-canvas" scrollWheelZoom>
      <TileLayer
        url="https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png"
        attribution='&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a>'
      />
      <SdzMapCameraController selectedSpot={selectedSpot} spots={spots} />
      {spots.map((spot) => {
        if (!spot.location) return null;
        const { color, label } = getSpotTone(spot.tags);
        const isSelected = spot.spotId === selectedSpotId;
        return (
          <CircleMarker
            key={spot.spotId}
            center={[spot.location.lat, spot.location.lng]}
            radius={isSelected ? 12 : 6}
            pathOptions={{
              color: isSelected ? '#101820' : '#ffffff',
              weight: isSelected ? 3 : 1.5,
              fillColor: color,
              fillOpacity: 0.95,
            }}
            eventHandlers={{
              click: () => onSelect(spot.spotId),
            }}
          >
            {isSelected && (
              <Tooltip direction="top" offset={[0, -12]} opacity={1} permanent>
                <div className="sdz-map-tooltip">
                  <span className="sdz-map-tooltip-title">{spot.name}</span>
                  <span className="sdz-map-tooltip-tag">{label}</span>
                </div>
              </Tooltip>
            )}
          </CircleMarker>
        );
      })}
    </MapContainer>
  );
}

type SdzSpotDetailProps = {
  spots: SdzSpot[];
  apiUrl: string;
  favorites: string[];
  onToggleFavorite: (spotId: string) => void;
  idToken: string | null;
};

function SdzSpotDetailPage({
  spots,
  apiUrl,
  favorites,
  onToggleFavorite,
  idToken,
}: SdzSpotDetailProps) {
  const { spotId } = useParams();
  const navigate = useNavigate();
  const [sdzDetail, setSdzDetail] = useState<SdzSpot | null>(null);
  const [sdzDetailLoading, setSdzDetailLoading] = useState(false);
  const [sdzDetailError, setSdzDetailError] = useState<string | null>(null);

  useEffect(() => {
    if (!spotId) return;
    const sdzExisting = spots.find((spot) => spot.spotId === spotId);
    if (sdzExisting) {
      setSdzDetail(sdzExisting);
      setSdzDetailError(null);
      return;
    }
    let isActive = true;
    setSdzDetailLoading(true);
    setSdzDetailError(null);
    const headers = idToken ? { Authorization: `Bearer ${idToken}` } : undefined;
    fetch(`${apiUrl}/sdz/spots/${spotId}`, { headers })
      .then((res) => {
        if (res.status === 404) {
          return null;
        }
        if (!res.ok) {
          throw new Error(`スポットの取得に失敗しました (HTTP ${res.status})`);
        }
        return res.json() as Promise<SdzSpot>;
      })
      .then((data) => {
        if (!isActive) return;
        if (!data) {
          setSdzDetail(null);
          return;
        }
        setSdzDetail(data);
      })
      .catch((err) => {
        if (!isActive) return;
        setSdzDetailError((err as Error).message);
      })
      .finally(() => {
        if (!isActive) return;
        setSdzDetailLoading(false);
      });
    return () => {
      isActive = false;
    };
  }, [apiUrl, idToken, spotId, spots]);

  const sdzRelatedSpots = useMemo(() => {
    if (!sdzDetail) return [];
    const baseTags = new Set(sdzDetail.tags ?? []);
    const related = spots.filter((spot) => {
      if (spot.spotId === sdzDetail.spotId) return false;
      return spot.tags?.some((tag) => baseTags.has(tag));
    });
    return related.slice(0, 5);
  }, [sdzDetail, spots]);

  if (!spotId) {
    return (
      <div className="sdz-card sdz-empty">
        <p>スポットIDが指定されていません。</p>
        <button type="button" onClick={() => navigate('/')}>
          一覧へ戻る
        </button>
      </div>
    );
  }

  if (sdzDetailLoading) {
    return <p>スポット詳細を読み込み中...</p>;
  }

  if (sdzDetailError) {
    return (
      <div className="sdz-error">
        詳細取得エラー: {sdzDetailError}
        <div style={{ marginTop: 8 }}>
          <button type="button" onClick={() => navigate('/')}>
            一覧へ戻る
          </button>
        </div>
      </div>
    );
  }

  if (!sdzDetail) {
    return (
      <div className="sdz-card sdz-empty">
        <p>該当するスポットが見つかりませんでした。</p>
        <button type="button" onClick={() => navigate('/')}>
          一覧へ戻る
        </button>
      </div>
    );
  }

  return (
    <div className="sdz-detail-layout">
      <div className="sdz-card sdz-detail">
        <div className="sdz-detail-header">
          <div>
            <p className="sdz-eyebrow">Spot Detail</p>
            <h2>{sdzDetail.name}</h2>
            <div className="sdz-meta">
              投稿者: {sdzDetail.userId} / 位置: {formatCoords(sdzDetail.location)} / 作成:
              {formatDateTime(sdzDetail.createdAt)}
            </div>
          </div>
          <div className="sdz-detail-actions">
            <button
              type="button"
              className={`sdz-favorite ${favorites.includes(sdzDetail.spotId) ? 'is-active' : ''}`}
              onClick={() => onToggleFavorite(sdzDetail.spotId)}
            >
              {favorites.includes(sdzDetail.spotId) ? '★ 保存済み' : '☆ お気に入り'}
            </button>
            <button type="button" className="sdz-ghost" onClick={() => navigate('/')}>
              一覧へ戻る
            </button>
          </div>
        </div>
        {sdzDetail.description && <p>{sdzDetail.description}</p>}
        {sdzDetail.tags && sdzDetail.tags.length > 0 && (
          <div className="sdz-tags">
            {sdzDetail.tags.map((tag) => (
              <span key={tag} className="sdz-tag">
                {tag}
              </span>
            ))}
          </div>
        )}
        <div>
          <strong>写真</strong>
          {sdzDetail.images?.length ? (
            <div className="sdz-image-grid">
              {sdzDetail.images.map((imageUrl) => (
                <img
                  key={imageUrl}
                  src={imageUrl}
                  alt={`${sdzDetail.name}の画像`}
                  className="sdz-image"
                  loading="lazy"
                />
              ))}
            </div>
          ) : (
            <p className="sdz-meta">画像はまだ登録されていません。</p>
          )}
        </div>
      </div>
      <aside className="sdz-related">
        <div className="sdz-card">
          <h3>関連スポット</h3>
          <p className="sdz-meta">同じタグのスポットを優先表示</p>
          {sdzRelatedSpots.length === 0 ? (
            <p className="sdz-meta">まだ関連スポットがありません。</p>
          ) : (
            <div className="sdz-related-list">
              {sdzRelatedSpots.map((spot) => (
                <button
                  key={spot.spotId}
                  type="button"
                  className="sdz-related-item"
                  onClick={() => navigate(`/spots/${spot.spotId}`)}
                >
                  <span>{spot.name}</span>
                  <span className="sdz-meta">{formatCoords(spot.location)}</span>
                </button>
              ))}
            </div>
          )}
        </div>
      </aside>
    </div>
  );
}

function App() {
  const {
    idToken,
    user,
    loginWithGoogle,
    loginWithEmail,
    signupWithEmail,
    resendVerification,
    logout,
    loading: authLoading,
    sendPasswordReset,
    updateDisplayName,
  } = useAuth();
  const [spots, setSpots] = useState<SdzSpot[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [authError, setAuthError] = useState<string | null>(null);
  const [loginEmail, setLoginEmail] = useState('');
  const [loginPassword, setLoginPassword] = useState('');
  const [sdzAuthMode, setSdzAuthMode] = useState<'login' | 'signup'>('login');
  const [sdzSearchText, setSdzSearchText] = useState('');
  const [sdzSelectedTags, setSdzSelectedTags] = useState<string[]>([]);
  const [sdzTagToAdd, setSdzTagToAdd] = useState('');
  const [sdzSelectedType, setSdzSelectedType] = useState('all');
  const [sdzFavorites, setSdzFavorites] = useState<string[]>(getInitialFavorites);
  const [sdzProfileMessage, setSdzProfileMessage] = useState<string | null>(null);
  const [sdzFavoritesPage, setSdzFavoritesPage] = useState(1);
  const [sdzDisplayName, setSdzDisplayName] = useState('');
  const [sdzSelectedSpotId, setSdzSelectedSpotId] = useState<string | null>(null);
  const sdzMapCardsContainerRef = useRef<HTMLDivElement | null>(null);
  const sdzMapCardElementRefs = useRef<Record<string, HTMLElement | null>>({});
  const sdzMapCardsScrollRafRef = useRef<number | null>(null);

  const isTestEnv = import.meta.env.MODE === 'test';
  const subtitle = `API base: ${apiUrl}（GET /sdz/spots を表示中）`;
  const isEmailPending = user && !user.emailVerified;
  const sdzAvailableTags = useMemo(() => {
    const tagSet = new Set<string>();
    spots.forEach((spot) => {
      spot.tags?.forEach((tag) => {
        if (!SDZ_TYPE_TAGS.has(tag)) {
          tagSet.add(tag);
        }
      });
    });
    return Array.from(tagSet).sort((a, b) => a.localeCompare(b));
  }, [spots]);
  const sdzFilteredSpots = useMemo(() => spots, [spots]);
  const sdzMapSpots = useMemo(
    () => sdzFilteredSpots.filter((spot) => spot.location),
    [sdzFilteredSpots],
  );
  const sdzSelectableTags = useMemo(
    () => sdzAvailableTags.filter((tag) => !sdzSelectedTags.includes(tag)),
    [sdzAvailableTags, sdzSelectedTags],
  );
  const sdzSearchRequest = useMemo(() => {
    return {
      query: sdzSearchText,
      type: sdzSelectedType,
      tags: sdzSelectedTags,
    };
  }, [sdzSearchText, sdzSelectedTags, sdzSelectedType]);

  const fetchSpots = useCallback(
    async (signal?: AbortSignal, options?: { query?: string; type?: string; tags?: string[] }) => {
      try {
        setLoading(true);
        setError(null);
        const headers = idToken ? { Authorization: `Bearer ${idToken}` } : undefined;
        const params = new URLSearchParams();
        const query = options?.query?.trim();
        if (query) params.set('q', query);
        const type = options?.type && options.type !== 'all' ? options.type : '';
        if (type) params.set('type', type);
        if (options?.tags && options.tags.length > 0) {
          params.set('tags', options.tags.join(','));
        }
        const queryString = params.toString();
        const url = queryString ? `${apiUrl}/sdz/spots?${queryString}` : `${apiUrl}/sdz/spots`;
        const res = await fetch(url, { signal, headers });
        if (!res.ok) {
          throw new Error(`Failed to fetch spots: ${res.status}`);
        }
        const data: SdzSpot[] = await res.json();
        setSpots(data);
      } catch (err) {
        if (err instanceof Error && err.name === 'AbortError') return;
        setError((err as Error).message);
      } finally {
        setLoading(false);
      }
    },
    [idToken],
  );

  useEffect(() => {
    const controller = new AbortController();
    const timer = window.setTimeout(() => {
      fetchSpots(controller.signal, sdzSearchRequest);
    }, 300);
    return () => {
      controller.abort();
      window.clearTimeout(timer);
    };
  }, [fetchSpots, sdzSearchRequest]);

  useEffect(() => {
    if (typeof window === 'undefined') return;
    window.localStorage.setItem('sdzFavorites', JSON.stringify(sdzFavorites));
  }, [sdzFavorites]);

  useEffect(() => {
    setSdzDisplayName(user?.displayName ?? '');
  }, [user]);

  useEffect(() => {
    setSdzFavoritesPage(1);
  }, [sdzFavorites.length]);

  useEffect(() => {
    if (sdzMapSpots.length === 0) {
      setSdzSelectedSpotId(null);
      return;
    }
    if (!sdzSelectedSpotId || !sdzMapSpots.some((spot) => spot.spotId === sdzSelectedSpotId)) {
      setSdzSelectedSpotId(sdzMapSpots[0].spotId);
    }
  }, [sdzMapSpots, sdzSelectedSpotId]);

  useEffect(() => {
    const spotIds = new Set(sdzMapSpots.map((spot) => spot.spotId));
    Object.keys(sdzMapCardElementRefs.current).forEach((spotId) => {
      if (!spotIds.has(spotId)) {
        delete sdzMapCardElementRefs.current[spotId];
      }
    });
  }, [sdzMapSpots]);

  useEffect(
    () => () => {
      if (sdzMapCardsScrollRafRef.current !== null) {
        window.cancelAnimationFrame(sdzMapCardsScrollRafRef.current);
      }
    },
    [],
  );

  const handleLoginWithGoogle = async () => {
    setAuthError(null);
    try {
      await loginWithGoogle();
    } catch (err) {
      setAuthError((err as Error).message);
    }
  };

  const handleLoginWithEmail = async (e: React.FormEvent) => {
    e.preventDefault();
    setAuthError(null);
    try {
      await loginWithEmail(loginEmail, loginPassword);
      setLoginEmail('');
      setLoginPassword('');
    } catch (err) {
      setAuthError((err as Error).message);
    }
  };

  const handleSignupWithEmail = async (e: React.FormEvent) => {
    e.preventDefault();
    setAuthError(null);
    try {
      await signupWithEmail(loginEmail, loginPassword);
      setLoginEmail('');
      setLoginPassword('');
    } catch (err) {
      if (
        err &&
        typeof err === 'object' &&
        'code' in err &&
        (err as { code?: string }).code === 'auth/email-already-in-use'
      ) {
        setAuthError('すでに登録済みです。ログインしてください。');
        setSdzAuthMode('login');
        return;
      }
      setAuthError((err as Error).message);
    }
  };

  const handleDisplayNameUpdate = async (e: React.FormEvent) => {
    e.preventDefault();
    setSdzProfileMessage(null);
    const trimmed = sdzDisplayName.trim();
    if (!trimmed) {
      setSdzProfileMessage('表示名を入力してください。');
      return;
    }
    try {
      await updateDisplayName(trimmed);
      setSdzProfileMessage('表示名を更新しました。');
    } catch (err) {
      setSdzProfileMessage((err as Error).message);
    }
  };

  const handleResendVerification = async () => {
    setAuthError(null);
    try {
      await resendVerification();
      setAuthError('確認メールを再送しました。受信トレイを確認してください。');
    } catch (err) {
      setAuthError((err as Error).message);
    }
  };

  const handleLogout = async () => {
    setAuthError(null);
    await logout();
  };

  const handleAuthSubmit = async (e: React.FormEvent) => {
    if (sdzAuthMode === 'login') {
      await handleLoginWithEmail(e);
      return;
    }
    await handleSignupWithEmail(e);
  };

  const handleToggleFavorite = (spotId: string) => {
    setSdzFavorites((prev) =>
      prev.includes(spotId) ? prev.filter((id) => id !== spotId) : [...prev, spotId],
    );
  };

  const handlePasswordReset = async () => {
    setSdzProfileMessage(null);
    try {
      await sendPasswordReset();
      setSdzProfileMessage('パスワード再設定メールを送信しました。');
    } catch (err) {
      setSdzProfileMessage((err as Error).message);
    }
  };

  const handleRefresh = () => fetchSpots(undefined, sdzSearchRequest);
  const handleResetFilters = () => {
    setSdzSearchText('');
    setSdzSelectedTags([]);
    setSdzTagToAdd('');
    setSdzSelectedType('all');
  };
  const handleAddTag = () => {
    if (!sdzTagToAdd) return;
    setSdzSelectedTags((prev) => (prev.includes(sdzTagToAdd) ? prev : [...prev, sdzTagToAdd]));
    setSdzTagToAdd('');
  };
  const handleRemoveTag = (tag: string) => {
    setSdzSelectedTags((prev) => prev.filter((item) => item !== tag));
  };
  const handleMapCardsScroll = useCallback(() => {
    if (sdzMapCardsScrollRafRef.current !== null) {
      window.cancelAnimationFrame(sdzMapCardsScrollRafRef.current);
    }
    sdzMapCardsScrollRafRef.current = window.requestAnimationFrame(() => {
      const container = sdzMapCardsContainerRef.current;
      if (!container) return;
      const cards = Array.from(container.querySelectorAll<HTMLElement>('[data-spot-id]'));
      if (cards.length === 0) return;
      const centerX = container.scrollLeft + container.clientWidth / 2;
      let nearestSpotId: string | null = null;
      let nearestDistance = Number.POSITIVE_INFINITY;

      cards.forEach((card) => {
        const spotId = card.dataset.spotId;
        if (!spotId) return;
        const cardCenterX = card.offsetLeft + card.offsetWidth / 2;
        const distance = Math.abs(cardCenterX - centerX);
        if (distance < nearestDistance) {
          nearestDistance = distance;
          nearestSpotId = spotId;
        }
      });

      if (nearestSpotId && nearestSpotId !== sdzSelectedSpotId) {
        setSdzSelectedSpotId(nearestSpotId);
      }
    });
  }, [sdzSelectedSpotId]);
  const setSdzMapCardRef = useCallback((spotId: string, element: HTMLElement | null) => {
    sdzMapCardElementRefs.current[spotId] = element;
  }, []);
  const navigate = useNavigate();
  const sdzSelectedSpot = useMemo(
    () => sdzMapSpots.find((spot) => spot.spotId === sdzSelectedSpotId) ?? null,
    [sdzMapSpots, sdzSelectedSpotId],
  );

  useEffect(() => {
    if (!sdzSelectedSpotId) return;
    const targetCard = sdzMapCardElementRefs.current[sdzSelectedSpotId];
    if (!targetCard) return;
    if (typeof targetCard.scrollIntoView !== 'function') return;
    targetCard.scrollIntoView({
      behavior: 'smooth',
      block: 'nearest',
      inline: 'center',
    });
  }, [sdzSelectedSpotId]);
  const sdzMySpots = useMemo(
    () => spots.filter((spot) => (user ? spot.userId === user.uid : false)),
    [spots, user],
  );
  const sdzFavoriteSpots = useMemo(
    () => spots.filter((spot) => sdzFavorites.includes(spot.spotId)),
    [sdzFavorites, spots],
  );
  const sdzFavoritesPaging = useMemo(
    () => paginate(sdzFavoriteSpots, sdzFavoritesPage, SDZ_FAVORITES_PAGE_SIZE),
    [sdzFavoriteSpots, sdzFavoritesPage],
  );
  const sdzHasPasswordProvider = useMemo(
    () => user?.providerData?.some((provider) => provider.providerId === 'password') ?? false,
    [user],
  );

  return (
    <div className="sdz-container">
      <header className="sdz-nav">
        <div className="sdz-brand">
          <span className="sdz-brand-mark">sdz</span>
          <span>spot-diggz</span>
        </div>
        <nav className="sdz-nav-links">
          <button type="button" className="sdz-ghost" onClick={() => navigate('/')}>
            Spots
          </button>
          <button type="button" className="sdz-ghost" onClick={() => navigate('/favorites')}>
            Favorites
          </button>
          <button type="button" className="sdz-ghost" onClick={() => navigate('/me')}>
            My Page
          </button>
        </nav>
      </header>

      {/* 認証セクション */}
      <div className="sdz-card sdz-auth-card">
        {authLoading ? (
          <p>認証状態を確認中...</p>
        ) : user ? (
          <div className="sdz-auth-info">
            <div>
              <strong>ログイン中:</strong> {user.email ?? user.uid}
              <div className="sdz-meta">uid: {user.uid}</div>
            </div>
            <button onClick={handleLogout}>ログアウト</button>
          </div>
        ) : (
          <div className="sdz-auth-actions">
            <div className="sdz-auth-row">
              <button type="button" onClick={handleLoginWithGoogle} disabled={authLoading}>
                Googleでログイン / 新規登録
              </button>
            </div>

            <div className="sdz-auth-row">
              <button
                type="button"
                className={`sdz-ghost ${sdzAuthMode === 'login' ? 'is-active' : ''}`}
                onClick={() => setSdzAuthMode('login')}
              >
                メールでログイン
              </button>
              <button
                type="button"
                className={`sdz-ghost ${sdzAuthMode === 'signup' ? 'is-active' : ''}`}
                onClick={() => setSdzAuthMode('signup')}
              >
                メールで新規登録
              </button>
            </div>

            <form onSubmit={handleAuthSubmit} className="sdz-auth-row">
              <input
                type="email"
                placeholder="メールアドレス"
                value={loginEmail}
                onChange={(e) => setLoginEmail(e.target.value)}
                required
              />
              <input
                type="password"
                placeholder="パスワード"
                value={loginPassword}
                onChange={(e) => setLoginPassword(e.target.value)}
                required
              />
              <button type="submit" disabled={authLoading}>
                {sdzAuthMode === 'login' ? 'ログイン' : '新規登録'}
              </button>
            </form>
            {authError && <div className="sdz-error">Authエラー: {authError}</div>}
          </div>
        )}
      </div>

      {/* 未認証ユーザー向けの案内（専用ビュー） */}
      {isEmailPending && (
        <div className="sdz-card" style={{ marginBottom: 16 }}>
          <h2>メール認証が必要です</h2>
          <p>
            {user?.email}{' '}
            宛に認証メールを送信しました。受信トレイ（迷惑メール含む）でリンクを開いて認証してください。
          </p>
          <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap', alignItems: 'center' }}>
            <button type="button" onClick={handleResendVerification}>
              認証メールを再送
            </button>
            <button type="button" onClick={() => window.location.reload()}>
              認証後は再読み込み
            </button>
            <button type="button" onClick={handleLogout}>
              ログアウト
            </button>
          </div>
          {authError && <div className="sdz-error">Authエラー: {authError}</div>}
        </div>
      )}

      {/* 認証済みユーザー専用コンテンツ */}
      {!isEmailPending && user && (
        /* 新規/更新ボタンプレースホルダ */
        <div className="sdz-card sdz-mode-card">
          <div className="sdz-mode-content">
            <div>
              <strong>閲覧専用モード</strong>
              <div className="sdz-meta">新規登録・画像登録はモバイルアプリからのみ行えます。</div>
            </div>
            <button type="button" onClick={handleRefresh}>
              再読み込み
            </button>
          </div>
        </div>
      )}

      {error && <div className="sdz-error">エラー: {error}</div>}
      {loading && <p>読み込み中...</p>}

      <Routes>
        <Route
          path="/"
          element={
            <>
              <section className="sdz-map-hero">
                <div className="sdz-map-header">
                  <div className="sdz-map-header-copy">
                    <p className="sdz-eyebrow">Spot Map</p>
                    <h1>マップでスポットを探す</h1>
                    <p className="sdz-hero-lead">
                      spot-diggzのスポットを、地図とカードで直感的に探索できます。気になる地点をタップして、すぐに詳細へ。
                    </p>
                    <div className="sdz-hero-footnote">{subtitle}</div>
                  </div>
                  <div className="sdz-map-stats">
                    <div>
                      <strong>{spots.length}</strong>
                      <span>登録スポット</span>
                    </div>
                    <div>
                      <strong>{sdzMapSpots.length}</strong>
                      <span>位置情報あり</span>
                    </div>
                    <div>
                      <strong>{sdzFavoriteSpots.length}</strong>
                      <span>お気に入り</span>
                    </div>
                  </div>
                </div>

                <div className="sdz-map-shell">
                  <div className="sdz-map-toolbar">
                    <div className="sdz-map-search">
                      <label htmlFor="sdz-search-map">キーワード</label>
                      <input
                        id="sdz-search-map"
                        type="text"
                        placeholder="スポット名・タグ・説明文で検索"
                        value={sdzSearchText}
                        onChange={(e) => setSdzSearchText(e.target.value)}
                      />
                    </div>
                    <div className="sdz-map-search">
                      <label htmlFor="sdz-type-map">種別</label>
                      <select
                        id="sdz-type-map"
                        value={sdzSelectedType}
                        onChange={(e) => setSdzSelectedType(e.target.value)}
                      >
                        <option value="all">すべて</option>
                        <option value="park">スケートパーク</option>
                        <option value="street">ストリート</option>
                      </select>
                    </div>
                    <div className="sdz-map-search">
                      <label htmlFor="sdz-tag-map">タグ</label>
                      <div className="sdz-tag-controls">
                        <select
                          id="sdz-tag-map"
                          value={sdzTagToAdd}
                          onChange={(e) => setSdzTagToAdd(e.target.value)}
                        >
                          <option value="">タグを追加</option>
                          {sdzSelectableTags.map((tag) => (
                            <option key={tag} value={tag}>
                              {tag}
                            </option>
                          ))}
                        </select>
                        <button type="button" onClick={handleAddTag} disabled={!sdzTagToAdd}>
                          追加
                        </button>
                      </div>
                    </div>
                    <div className="sdz-search-meta">
                      <span>
                        表示中 {sdzFilteredSpots.length} / {spots.length}
                      </span>
                      <button type="button" onClick={handleResetFilters}>
                        クリア
                      </button>
                    </div>
                  </div>

                  {sdzSelectedTags.length > 0 && (
                    <div className="sdz-tag-chips" aria-label="選択中のタグ">
                      {sdzSelectedTags.map((tag) => (
                        <button
                          key={tag}
                          type="button"
                          className="sdz-tag-chip"
                          onClick={() => handleRemoveTag(tag)}
                        >
                          {tag}
                          <span aria-hidden="true">×</span>
                        </button>
                      ))}
                    </div>
                  )}

                  <div className="sdz-map-legend">
                    <div className="sdz-map-legend-item">
                      <span className="sdz-map-legend-dot is-park" />
                      スケートパーク
                    </div>
                    <div className="sdz-map-legend-item">
                      <span className="sdz-map-legend-dot is-street" />
                      ストリート
                    </div>
                  </div>

                  <div className="sdz-map-canvas-wrap">
                    {isTestEnv ? (
                      <div className="sdz-map-canvas sdz-map-placeholder">
                        <div>
                          <strong>Map Preview</strong>
                          <p className="sdz-meta">テスト環境ではマップを簡易表示します。</p>
                        </div>
                      </div>
                    ) : (
                      <SdzSpotMap
                        spots={sdzMapSpots}
                        selectedSpotId={sdzSelectedSpotId}
                        onSelect={setSdzSelectedSpotId}
                      />
                    )}
                  </div>
                </div>

                {!loading && !error && spots.length === 0 && (
                  <div className="sdz-card sdz-empty">スポットがまだありません。</div>
                )}
                {!loading && !error && spots.length > 0 && sdzFilteredSpots.length === 0 && (
                  <div className="sdz-card sdz-empty">条件に一致するスポットがありません。</div>
                )}

                <div className="sdz-map-carousel">
                  {sdzMapSpots.length === 0 ? (
                    <div className="sdz-card sdz-empty">位置情報付きスポットがまだありません。</div>
                  ) : (
                    <div
                      className="sdz-map-cards"
                      ref={sdzMapCardsContainerRef}
                      onScroll={handleMapCardsScroll}
                    >
                      {sdzMapSpots.map((spot) => {
                        const tone = getSpotTone(spot.tags);
                        const isSelected = spot.spotId === sdzSelectedSpotId;
                        return (
                          <article
                            key={spot.spotId}
                            ref={(element) => setSdzMapCardRef(spot.spotId, element)}
                            data-spot-id={spot.spotId}
                            className={`sdz-map-card ${isSelected ? 'is-selected' : ''}`}
                            role="button"
                            tabIndex={0}
                            aria-label={`${spot.name} を選択`}
                            onClick={() => setSdzSelectedSpotId(spot.spotId)}
                            onKeyDown={(event) => {
                              if (event.key === 'Enter' || event.key === ' ') {
                                event.preventDefault();
                                setSdzSelectedSpotId(spot.spotId);
                              }
                            }}
                          >
                            <div className="sdz-map-card-header">
                              <span
                                className="sdz-map-card-type"
                                style={{ backgroundColor: tone.color }}
                              >
                                {tone.label}
                              </span>
                              <button
                                type="button"
                                className={`sdz-favorite ${
                                  sdzFavorites.includes(spot.spotId) ? 'is-active' : ''
                                }`}
                                onClick={(event) => {
                                  event.stopPropagation();
                                  handleToggleFavorite(spot.spotId);
                                }}
                              >
                                {sdzFavorites.includes(spot.spotId) ? '★' : '☆'}
                              </button>
                            </div>
                            <h3>{spot.name}</h3>
                            <div className="sdz-map-card-meta">
                              位置: {formatCoords(spot.location)}
                            </div>
                            {spot.description && (
                              <p className="sdz-map-card-desc">{spot.description}</p>
                            )}
                            {spot.tags && spot.tags.length > 0 && (
                              <div className="sdz-tags">
                                {spot.tags.map((tag) => (
                                  <span key={tag} className="sdz-tag">
                                    {tag}
                                  </span>
                                ))}
                              </div>
                            )}
                            <div className="sdz-map-card-actions">
                              <button
                                type="button"
                                className="sdz-ghost"
                                onClick={(event) => {
                                  event.stopPropagation();
                                  navigate(`/spots/${spot.spotId}`);
                                }}
                              >
                                詳細を見る
                              </button>
                            </div>
                          </article>
                        );
                      })}
                    </div>
                  )}
                </div>

                <div className="sdz-map-footer">
                  <div className="sdz-card sdz-app-box">
                    <h3>アプリでスポット登録</h3>
                    <p className="sdz-meta">
                      Webは閲覧と整理専用です。スポット登録や画像投稿はモバイルアプリから行います。
                    </p>
                    <div className="sdz-app-links">
                      <button type="button" className="sdz-ghost" disabled>
                        iOSアプリ（準備中）
                      </button>
                      <button type="button" className="sdz-ghost" disabled>
                        Androidアプリ（準備中）
                      </button>
                    </div>
                    <div className="sdz-meta">QR/リンクは後日追加予定</div>
                  </div>
                  {sdzSelectedSpot && (
                    <div className="sdz-card sdz-map-highlight">
                      <p className="sdz-eyebrow">Selected Spot</p>
                      <h3>{sdzSelectedSpot.name}</h3>
                      <p className="sdz-meta">{getSpotTypeLabel(sdzSelectedSpot.tags)}</p>
                      {sdzSelectedSpot.description && (
                        <p className="sdz-map-card-desc">{sdzSelectedSpot.description}</p>
                      )}
                      <button
                        type="button"
                        onClick={() => navigate(`/spots/${sdzSelectedSpot.spotId}`)}
                      >
                        詳細へ移動
                      </button>
                    </div>
                  )}
                </div>
              </section>
            </>
          }
        />
        <Route
          path="/spots/:spotId"
          element={
            <SdzSpotDetailPage
              spots={spots}
              apiUrl={apiUrl}
              favorites={sdzFavorites}
              onToggleFavorite={handleToggleFavorite}
              idToken={idToken}
            />
          }
        />
        <Route
          path="/favorites"
          element={
            <div className="sdz-card">
              <h2>お気に入り一覧</h2>
              <p className="sdz-meta">ローカル保存のお気に入りスポットです。</p>
              {sdzFavoriteSpots.length === 0 ? (
                <p className="sdz-meta">お気に入りがありません。</p>
              ) : (
                <>
                  <div className="sdz-related-list">
                    {sdzFavoritesPaging.items.map((spot) => (
                      <button
                        key={spot.spotId}
                        type="button"
                        className="sdz-related-item"
                        onClick={() => navigate(`/spots/${spot.spotId}`)}
                      >
                        <span>{spot.name}</span>
                        <span className="sdz-meta">{formatCoords(spot.location)}</span>
                      </button>
                    ))}
                  </div>
                  <div className="sdz-pagination">
                    <button
                      type="button"
                      className="sdz-ghost"
                      onClick={() => setSdzFavoritesPage((prev) => Math.max(1, prev - 1))}
                      disabled={sdzFavoritesPaging.page === 1}
                    >
                      前へ
                    </button>
                    <span>
                      {sdzFavoritesPaging.page} / {sdzFavoritesPaging.totalPages}
                    </span>
                    <button
                      type="button"
                      className="sdz-ghost"
                      onClick={() =>
                        setSdzFavoritesPage((prev) =>
                          Math.min(sdzFavoritesPaging.totalPages, prev + 1),
                        )
                      }
                      disabled={sdzFavoritesPaging.page === sdzFavoritesPaging.totalPages}
                    >
                      次へ
                    </button>
                  </div>
                </>
              )}
            </div>
          }
        />
        <Route
          path="/me"
          element={
            <div className="sdz-profile-grid">
              <div className="sdz-card">
                <h2>マイページ</h2>
                <p className="sdz-meta">ログインしたユーザーの情報と保存内容</p>
                {!user ? (
                  <p>ログインするとマイページを利用できます。</p>
                ) : (
                  <>
                    <div className="sdz-profile-summary">
                      <div>
                        <strong>{user.displayName || 'ユーザー名未設定'}</strong>
                        <div className="sdz-meta">{user.email ?? user.uid}</div>
                      </div>
                      <div className="sdz-profile-pill">
                        投稿 {sdzMySpots.length}件 / お気に入り {sdzFavoriteSpots.length}件
                      </div>
                    </div>
                    <form className="sdz-profile-form" onSubmit={handleDisplayNameUpdate}>
                      <label htmlFor="displayName">表示名</label>
                      <input
                        id="displayName"
                        type="text"
                        value={sdzDisplayName}
                        onChange={(event) => setSdzDisplayName(event.target.value)}
                        placeholder="表示名を入力"
                      />
                      <button type="submit" className="sdz-ghost">
                        表示名を更新
                      </button>
                    </form>
                    {sdzHasPasswordProvider ? (
                      <button type="button" className="sdz-ghost" onClick={handlePasswordReset}>
                        パスワード再設定メールを送信
                      </button>
                    ) : (
                      <div className="sdz-meta">
                        Googleログインの場合はGoogleアカウント側でパスワードを変更してください。
                      </div>
                    )}
                    {sdzProfileMessage && <p className="sdz-meta">{sdzProfileMessage}</p>}
                    <button
                      type="button"
                      className="sdz-ghost"
                      onClick={() => navigate('/favorites')}
                    >
                      お気に入りページへ
                    </button>
                  </>
                )}
              </div>
              <div className="sdz-card">
                <h3>お気に入り</h3>
                {sdzFavoriteSpots.length === 0 ? (
                  <p className="sdz-meta">お気に入りがありません。</p>
                ) : (
                  <div className="sdz-related-list">
                    {sdzFavoriteSpots.map((spot) => (
                      <button
                        key={spot.spotId}
                        type="button"
                        className="sdz-related-item"
                        onClick={() => navigate(`/spots/${spot.spotId}`)}
                      >
                        <span>{spot.name}</span>
                        <span className="sdz-meta">{formatCoords(spot.location)}</span>
                      </button>
                    ))}
                  </div>
                )}
              </div>
              <div className="sdz-card">
                <h3>自分のスポット</h3>
                {sdzMySpots.length === 0 ? (
                  <p className="sdz-meta">まだ投稿したスポットがありません。</p>
                ) : (
                  <div className="sdz-related-list">
                    {sdzMySpots.map((spot) => (
                      <button
                        key={spot.spotId}
                        type="button"
                        className="sdz-related-item"
                        onClick={() => navigate(`/spots/${spot.spotId}`)}
                      >
                        <span>{spot.name}</span>
                        <span className="sdz-meta">{formatCoords(spot.location)}</span>
                      </button>
                    ))}
                  </div>
                )}
              </div>
            </div>
          }
        />
        <Route element={<SdzAdminGuard />}>
          <Route element={<SdzAdminLayout />}>
            <Route path="/admin" element={<SdzAdminSpotList />} />
            <Route path="/admin/spots/new" element={<SdzAdminSpotForm />} />
            <Route path="/admin/spots/:id/edit" element={<SdzAdminSpotForm />} />
          </Route>
        </Route>
      </Routes>
    </div>
  );
}

export default App;
