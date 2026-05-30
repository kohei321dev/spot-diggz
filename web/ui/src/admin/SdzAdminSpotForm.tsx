import { useCallback, useEffect, useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import { useAuth } from '../contexts/useAuth';
import type { SdzSpot, SdzPlaceResult } from '../types/spot';
import {
  sdzAdminCreateSpot,
  sdzAdminUpdateSpot,
  sdzAdminGetUploadUrl,
  sdzAdminUploadImage,
} from '../lib/SdzAdminApi';
import { SdzAdminMapPicker } from './SdzAdminMapPicker';

const sdzApiUrl = import.meta.env.VITE_SDZ_API_URL || 'http://localhost:8080';

const SDZ_ALLOWED_URL_SCHEMES = ['https:', 'http:'];

function sdzValidateUrl(value: string): string | null {
  if (!value) return null;
  try {
    const url = new URL(value);
    if (!SDZ_ALLOWED_URL_SCHEMES.includes(url.protocol)) {
      return `許可されていないURLスキームです: ${url.protocol}`;
    }
    return null;
  } catch {
    return '無効なURL形式です';
  }
}

// Firestore に保存されるフィールドのマッピング定義
interface SdzSavedField {
  label: string;
  firestoreKey: string;
  type: string;
  value: string | number | string[] | undefined;
}

function sdzBuildSavedFieldsSummary(
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  payload: Record<string, any>,
): SdzSavedField[] {
  const mapping: { key: string; label: string; firestoreKey: string; type: string }[] = [
    { key: 'name', label: 'スポット名', firestoreKey: 'name', type: 'string' },
    { key: 'description', label: '説明', firestoreKey: 'description', type: 'string?' },
    {
      key: 'location',
      label: '位置情報',
      firestoreKey: 'location',
      type: '{lat: number, lng: number}?',
    },
    { key: 'tags', label: 'タグ', firestoreKey: 'tags', type: 'string[]' },
    { key: 'images', label: '画像', firestoreKey: 'images', type: 'string[]' },
    { key: 'parkAttributes', label: 'パーク属性', firestoreKey: 'parkAttributes', type: 'object?' },
    {
      key: 'instagramLocationUrl',
      label: 'Instagram URL',
      firestoreKey: 'instagramLocationUrl',
      type: 'string?',
    },
    {
      key: 'googlePlaceId',
      label: 'Google Place ID',
      firestoreKey: 'googlePlaceId',
      type: 'string?',
    },
    {
      key: 'googleMapsUrl',
      label: 'Google Maps URL',
      firestoreKey: 'googleMapsUrl',
      type: 'string?',
    },
    { key: 'address', label: '住所', firestoreKey: 'address', type: 'string?' },
    { key: 'phoneNumber', label: '電話番号', firestoreKey: 'phoneNumber', type: 'string?' },
    { key: 'googleRating', label: 'Google 評価', firestoreKey: 'googleRating', type: 'number?' },
    {
      key: 'googleRatingCount',
      label: 'レビュー件数',
      firestoreKey: 'googleRatingCount',
      type: 'number?',
    },
    { key: 'googleTypes', label: '施設タイプ', firestoreKey: 'googleTypes', type: 'string[]' },
  ];

  return mapping
    .filter((m) => {
      const v = payload[m.key];
      if (v == null) return false;
      if (Array.isArray(v) && v.length === 0) return false;
      return true;
    })
    .map((m) => {
      const v = payload[m.key];
      let displayValue: string;
      if (typeof v === 'object' && !Array.isArray(v)) {
        displayValue = JSON.stringify(v);
      } else if (Array.isArray(v)) {
        displayValue = v.length > 0 ? v.join(', ') : '(空)';
      } else {
        displayValue = String(v);
      }
      return { label: m.label, firestoreKey: m.firestoreKey, type: m.type, value: displayValue };
    });
}

export function SdzAdminSpotForm() {
  const { id } = useParams();
  const isEdit = id != null;
  const navigate = useNavigate();
  const { idToken } = useAuth();

  // 管理者が編集するフィールド
  const [sdzName, setSdzName] = useState('');
  const [sdzDescription, setSdzDescription] = useState('');
  const [sdzLat, setSdzLat] = useState('');
  const [sdzLng, setSdzLng] = useState('');
  const [sdzSpotType, setSdzSpotType] = useState('park');
  const [sdzTags, setSdzTags] = useState('');
  const [sdzImages, setSdzImages] = useState<string[]>([]);
  const [sdzOfficialUrl, setSdzOfficialUrl] = useState('');
  const [sdzBusinessHoursNote, setSdzBusinessHoursNote] = useState('');
  const [sdzAccessInfo, setSdzAccessInfo] = useState('');
  const [sdzInstagramLocationUrl, setSdzInstagramLocationUrl] = useState('');

  // Google Places から自動取得（読み取り専用）
  const [sdzGooglePlaceId, setSdzGooglePlaceId] = useState('');
  const [sdzGoogleMapsUrl, setSdzGoogleMapsUrl] = useState('');
  const [sdzAddress, setSdzAddress] = useState('');
  const [sdzPhoneNumber, setSdzPhoneNumber] = useState('');
  const [sdzGoogleRating, setSdzGoogleRating] = useState('');
  const [sdzGoogleRatingCount, setSdzGoogleRatingCount] = useState('');
  const [sdzGoogleTypes, setSdzGoogleTypes] = useState<string[]>([]);

  // UI状態
  const [sdzUploading, setSdzUploading] = useState(false);
  const [sdzSubmitting, setSdzSubmitting] = useState(false);
  const [sdzError, setSdzError] = useState<string | null>(null);
  const [sdzSuccess, setSdzSuccess] = useState<string | null>(null);
  const [sdzSavedFields, setSdzSavedFields] = useState<SdzSavedField[]>([]);
  const [sdzLoadingSpot, setSdzLoadingSpot] = useState(false);

  const handleLocationChange = useCallback((lat: string, lng: string) => {
    setSdzLat(lat);
    setSdzLng(lng);
  }, []);

  const handlePlaceSelect = useCallback(
    (place: SdzPlaceResult) => {
      if (!sdzName && place.name) setSdzName(place.name);
      if (!sdzOfficialUrl && place.website) setSdzOfficialUrl(place.website);
      if (place.placeId) setSdzGooglePlaceId(place.placeId);
      if (place.googleMapsUrl) setSdzGoogleMapsUrl(place.googleMapsUrl);
      if (place.address) setSdzAddress(place.address);
      if (place.phoneNumber) setSdzPhoneNumber(place.phoneNumber);
      if (place.rating != null) setSdzGoogleRating(String(place.rating));
      if (place.ratingCount != null) setSdzGoogleRatingCount(String(place.ratingCount));
      if (place.types) setSdzGoogleTypes(place.types);
      if (place.businessHours?.length) {
        setSdzBusinessHoursNote(place.businessHours.join('\n'));
      }
    },
    [sdzName, sdzOfficialUrl],
  );

  useEffect(() => {
    if (!isEdit) return;
    setSdzLoadingSpot(true);
    fetch(`${sdzApiUrl}/sdz/spots/${id}`)
      .then((res) => {
        if (!res.ok) throw new Error(`HTTP ${res.status}`);
        return res.json() as Promise<SdzSpot>;
      })
      .then((spot) => {
        setSdzName(spot.name);
        setSdzDescription(spot.description ?? '');
        setSdzLat(spot.location?.lat?.toString() ?? '');
        setSdzLng(spot.location?.lng?.toString() ?? '');
        setSdzSpotType(spot.parkAttributes ? 'park' : spot.streetAttributes ? 'street' : 'park');
        setSdzTags(spot.tags?.join(', ') ?? '');
        setSdzImages(spot.images ?? []);
        setSdzOfficialUrl(spot.parkAttributes?.officialUrl ?? '');
        setSdzBusinessHoursNote(spot.parkAttributes?.businessHours?.note ?? '');
        setSdzAccessInfo(spot.parkAttributes?.accessInfo ?? '');
        setSdzInstagramLocationUrl(spot.instagramLocationUrl ?? '');
        setSdzGooglePlaceId(spot.googlePlaceId ?? '');
        setSdzGoogleMapsUrl(spot.googleMapsUrl ?? '');
        setSdzAddress(spot.address ?? '');
        setSdzPhoneNumber(spot.phoneNumber ?? '');
        setSdzGoogleRating(spot.googleRating != null ? String(spot.googleRating) : '');
        setSdzGoogleRatingCount(
          spot.googleRatingCount != null ? String(spot.googleRatingCount) : '',
        );
        setSdzGoogleTypes(spot.googleTypes ?? []);
      })
      .catch((err) => setSdzError((err as Error).message))
      .finally(() => setSdzLoadingSpot(false));
  }, [id, isEdit]);

  const handleImageUpload = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const files = e.target.files;
    if (!files || !idToken) return;
    setSdzUploading(true);
    setSdzError(null);
    try {
      for (const file of Array.from(files)) {
        if (sdzImages.length >= 10) break;
        const result = await sdzAdminGetUploadUrl(idToken, file.type);
        await sdzAdminUploadImage(result.uploadUrl, file);
        setSdzImages((prev) => [...prev, result.objectUrl]);
      }
    } catch (err) {
      setSdzError((err as Error).message);
    } finally {
      setSdzUploading(false);
      e.target.value = '';
    }
  };

  const handleRemoveImage = (index: number) => {
    setSdzImages((prev) => prev.filter((_, i) => i !== index));
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!idToken) {
      setSdzError('認証トークンがありません');
      return;
    }
    setSdzSubmitting(true);
    setSdzError(null);
    setSdzSuccess(null);
    setSdzSavedFields([]);

    const sdzOfficialUrlError = sdzValidateUrl(sdzOfficialUrl);
    if (sdzOfficialUrlError) {
      setSdzError(`公式サイトURL: ${sdzOfficialUrlError}`);
      setSdzSubmitting(false);
      return;
    }
    const sdzInstagramUrlError = sdzValidateUrl(sdzInstagramLocationUrl);
    if (sdzInstagramUrlError) {
      setSdzError(`Instagram URL: ${sdzInstagramUrlError}`);
      setSdzSubmitting(false);
      return;
    }

    const parseSplit = (value: string) =>
      value
        .split(',')
        .map((s) => s.trim())
        .filter(Boolean);

    const parsedLat = parseFloat(sdzLat);
    const parsedLng = parseFloat(sdzLng);
    const location =
      sdzLat && sdzLng && Number.isFinite(parsedLat) && Number.isFinite(parsedLng)
        ? { lat: parsedLat, lng: parsedLng }
        : undefined;

    const hasParkData = sdzOfficialUrl || sdzBusinessHoursNote || sdzAccessInfo || sdzPhoneNumber;
    const parkAttributes =
      sdzSpotType === 'park' && hasParkData
        ? {
            officialUrl: sdzOfficialUrl || undefined,
            businessHours: sdzBusinessHoursNote
              ? {
                  scheduleType: 'manual' as const,
                  is24Hours: false,
                  sameAsWeekday: false,
                  note: sdzBusinessHoursNote,
                }
              : undefined,
            accessInfo: sdzAccessInfo || undefined,
            phoneNumber: sdzPhoneNumber || undefined,
          }
        : undefined;

    const payload = {
      name: sdzName,
      description: sdzDescription || undefined,
      location,
      tags: parseSplit(sdzTags),
      images: sdzImages,
      parkAttributes,
      instagramLocationUrl: sdzInstagramLocationUrl || undefined,
      googlePlaceId: sdzGooglePlaceId || undefined,
      googleMapsUrl: sdzGoogleMapsUrl || undefined,
      address: sdzAddress || undefined,
      phoneNumber: !parkAttributes ? sdzPhoneNumber || undefined : undefined,
      googleRating: sdzGoogleRating ? parseFloat(sdzGoogleRating) : undefined,
      googleRatingCount: sdzGoogleRatingCount ? parseInt(sdzGoogleRatingCount) : undefined,
      googleTypes: sdzGoogleTypes.length > 0 ? sdzGoogleTypes : undefined,
    };

    try {
      if (isEdit) {
        await sdzAdminUpdateSpot(idToken, id, payload);
        setSdzSuccess('スポットを更新しました');
      } else {
        await sdzAdminCreateSpot(idToken, payload);
        setSdzSuccess('スポットを作成しました');
      }
      setSdzSavedFields(sdzBuildSavedFieldsSummary(payload));
    } catch (err) {
      setSdzError((err as Error).message);
    } finally {
      setSdzSubmitting(false);
    }
  };

  if (sdzLoadingSpot) {
    return <p>スポット情報を読み込み中...</p>;
  }

  return (
    <div className="sdz-card">
      <h3>{isEdit ? 'スポット編集' : 'スポット新規作成'}</h3>

      {sdzError && (
        <div className="sdz-error" style={{ marginBottom: 12 }}>
          {sdzError}
        </div>
      )}

      {/* 保存完了サマリー */}
      {sdzSuccess && (
        <div style={{ marginBottom: 16 }}>
          <div style={{ color: '#4caf50', marginBottom: 8 }}>✅ {sdzSuccess}</div>
          {sdzSavedFields.length > 0 && (
            <div
              style={{
                background: '#f0faf0',
                border: '1px solid #c8e6c9',
                borderRadius: 8,
                padding: 12,
                fontSize: 13,
              }}
            >
              <div style={{ fontWeight: 600, marginBottom: 8 }}>
                Firestore に保存されたデータ ({sdzSavedFields.length} フィールド)
              </div>
              <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: 12 }}>
                <thead>
                  <tr style={{ borderBottom: '1px solid #c8e6c9', textAlign: 'left' }}>
                    <th style={{ padding: '4px 8px' }}>フィールド名</th>
                    <th style={{ padding: '4px 8px' }}>Firestore キー</th>
                    <th style={{ padding: '4px 8px' }}>データ型</th>
                    <th style={{ padding: '4px 8px' }}>値</th>
                  </tr>
                </thead>
                <tbody>
                  {sdzSavedFields.map((f) => (
                    <tr key={f.firestoreKey} style={{ borderBottom: '1px solid #e8f5e9' }}>
                      <td style={{ padding: '4px 8px' }}>{f.label}</td>
                      <td style={{ padding: '4px 8px', fontFamily: 'monospace', fontSize: 11 }}>
                        {f.firestoreKey}
                      </td>
                      <td
                        style={{
                          padding: '4px 8px',
                          fontFamily: 'monospace',
                          fontSize: 11,
                          color: '#666',
                        }}
                      >
                        {f.type}
                      </td>
                      <td
                        style={{
                          padding: '4px 8px',
                          maxWidth: 300,
                          overflow: 'hidden',
                          textOverflow: 'ellipsis',
                          whiteSpace: 'nowrap',
                        }}
                      >
                        {String(f.value)}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
              <div style={{ marginTop: 8 }}>
                <button
                  type="button"
                  className="sdz-ghost"
                  style={{ fontSize: 12 }}
                  onClick={() => navigate('/admin')}
                >
                  ← スポット一覧に戻る
                </button>
              </div>
            </div>
          )}
        </div>
      )}

      <form onSubmit={handleSubmit}>
        <div style={{ display: 'grid', gap: 12 }}>
          {/* ===== 位置情報・場所検索 ===== */}
          <fieldset style={{ border: '1px solid #e0e0e0', borderRadius: 8, padding: 12 }}>
            <legend style={{ fontWeight: 600, fontSize: 14 }}>📍 位置情報・場所検索</legend>
            <SdzAdminMapPicker
              lat={sdzLat}
              lng={sdzLng}
              onLocationChange={handleLocationChange}
              onPlaceSelect={handlePlaceSelect}
            />
            {(sdzLat || sdzLng) && (
              <div style={{ marginTop: 8, fontSize: 13, color: '#666' }}>
                緯度: {sdzLat || '—'} / 経度: {sdzLng || '—'}
              </div>
            )}
          </fieldset>

          {/* ===== Google Places 情報（自動取得・読み取り専用） ===== */}
          {sdzGooglePlaceId && (
            <fieldset
              style={{
                border: '1px solid #d0d0d0',
                borderRadius: 8,
                padding: 12,
                background: '#fafafa',
              }}
            >
              <legend style={{ fontWeight: 600, fontSize: 14 }}>
                🗺️ Google Places 情報（自動取得）
              </legend>
              <div style={{ display: 'grid', gap: 6, fontSize: 13 }}>
                <div>住所: {sdzAddress || '—'}</div>
                {sdzGoogleMapsUrl && (
                  <div>
                    Google Maps:{' '}
                    <a href={sdzGoogleMapsUrl} target="_blank" rel="noopener noreferrer">
                      {sdzGoogleMapsUrl}
                    </a>
                  </div>
                )}
                <div>電話番号: {sdzPhoneNumber || '—'}</div>
                {sdzGoogleRating && (
                  <div>
                    評価: ⭐ {sdzGoogleRating} ({sdzGoogleRatingCount} 件)
                  </div>
                )}
                {sdzGoogleTypes.length > 0 && (
                  <div>
                    施設タイプ:{' '}
                    {sdzGoogleTypes.map((t) => (
                      <span
                        key={t}
                        style={{
                          display: 'inline-block',
                          background: '#e0e0e0',
                          borderRadius: 4,
                          padding: '1px 6px',
                          marginRight: 4,
                          fontSize: 11,
                        }}
                      >
                        {t}
                      </span>
                    ))}
                  </div>
                )}
                <div style={{ fontSize: 11, color: '#999' }}>Place ID: {sdzGooglePlaceId}</div>
              </div>
            </fieldset>
          )}

          {/* ===== 基本情報（管理者編集） ===== */}
          <fieldset style={{ border: '1px solid #e0e0e0', borderRadius: 8, padding: 12 }}>
            <legend style={{ fontWeight: 600, fontSize: 14 }}>📋 基本情報</legend>
            <div style={{ display: 'grid', gap: 10 }}>
              <div>
                <label htmlFor="sdz-name">スポット名 *</label>
                <input
                  id="sdz-name"
                  type="text"
                  value={sdzName}
                  onChange={(e) => setSdzName(e.target.value)}
                  required
                  style={{ width: '100%' }}
                />
              </div>
              <div>
                <label>スポットタイプ</label>
                <div style={{ display: 'flex', gap: 16 }}>
                  <label>
                    <input
                      type="radio"
                      name="spotType"
                      value="park"
                      checked={sdzSpotType === 'park'}
                      onChange={(e) => setSdzSpotType(e.target.value)}
                    />{' '}
                    パーク
                  </label>
                  <label>
                    <input
                      type="radio"
                      name="spotType"
                      value="street"
                      checked={sdzSpotType === 'street'}
                      onChange={(e) => setSdzSpotType(e.target.value)}
                    />{' '}
                    ストリート
                  </label>
                </div>
              </div>
              <div>
                <label htmlFor="sdz-description">説明</label>
                <textarea
                  id="sdz-description"
                  value={sdzDescription}
                  onChange={(e) => setSdzDescription(e.target.value)}
                  rows={3}
                  style={{ width: '100%' }}
                />
              </div>
              <div>
                <label htmlFor="sdz-tags">タグ（カンマ区切り）</label>
                <input
                  id="sdz-tags"
                  type="text"
                  value={sdzTags}
                  onChange={(e) => setSdzTags(e.target.value)}
                  style={{ width: '100%' }}
                />
              </div>
            </div>
          </fieldset>

          {/* ===== 施設情報（管理者編集） ===== */}
          <fieldset style={{ border: '1px solid #e0e0e0', borderRadius: 8, padding: 12 }}>
            <legend style={{ fontWeight: 600, fontSize: 14 }}>🏢 施設情報</legend>
            <div style={{ display: 'grid', gap: 10 }}>
              <div>
                <label htmlFor="sdz-official-url">公式サイトURL</label>
                <input
                  id="sdz-official-url"
                  type="url"
                  value={sdzOfficialUrl}
                  onChange={(e) => setSdzOfficialUrl(e.target.value)}
                  style={{ width: '100%' }}
                />
              </div>
              <div>
                <label htmlFor="sdz-business-hours">営業時間メモ</label>
                <textarea
                  id="sdz-business-hours"
                  value={sdzBusinessHoursNote}
                  onChange={(e) => setSdzBusinessHoursNote(e.target.value)}
                  rows={3}
                  style={{ width: '100%' }}
                />
              </div>
              <div>
                <label htmlFor="sdz-access-info">アクセス情報</label>
                <input
                  id="sdz-access-info"
                  type="text"
                  value={sdzAccessInfo}
                  onChange={(e) => setSdzAccessInfo(e.target.value)}
                  style={{ width: '100%' }}
                />
              </div>
            </div>
          </fieldset>

          {/* ===== Instagram（管理者編集） ===== */}
          <fieldset style={{ border: '1px solid #e0e0e0', borderRadius: 8, padding: 12 }}>
            <legend style={{ fontWeight: 600, fontSize: 14 }}>📸 Instagram</legend>
            <div>
              <label htmlFor="sdz-instagram-location-url">Instagram 位置情報ページURL</label>
              <input
                id="sdz-instagram-location-url"
                type="url"
                value={sdzInstagramLocationUrl}
                onChange={(e) => setSdzInstagramLocationUrl(e.target.value)}
                placeholder="https://www.instagram.com/explore/locations/..."
                style={{ width: '100%' }}
              />
            </div>
          </fieldset>

          {/* ===== 画像 ===== */}
          <fieldset style={{ border: '1px solid #e0e0e0', borderRadius: 8, padding: 12 }}>
            <legend style={{ fontWeight: 600, fontSize: 14 }}>
              🖼️ 画像 ({sdzImages.length}/10)
            </legend>
            {sdzImages.length > 0 && (
              <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap', marginBottom: 8 }}>
                {sdzImages.map((url, i) => (
                  <div key={`${url}-${i}`} style={{ position: 'relative' }}>
                    <img
                      src={url}
                      alt={`画像${i + 1}`}
                      style={{ width: 120, height: 80, objectFit: 'cover', borderRadius: 4 }}
                    />
                    <button
                      type="button"
                      onClick={() => handleRemoveImage(i)}
                      style={{
                        position: 'absolute',
                        top: -4,
                        right: -4,
                        background: '#e53935',
                        color: '#fff',
                        border: 'none',
                        borderRadius: '50%',
                        width: 20,
                        height: 20,
                        cursor: 'pointer',
                        fontSize: 12,
                        lineHeight: '20px',
                        padding: 0,
                      }}
                    >
                      x
                    </button>
                  </div>
                ))}
              </div>
            )}
            {sdzImages.length < 10 && (
              <input
                type="file"
                accept="image/jpeg,image/png,image/webp,image/heic,image/heif"
                multiple
                onChange={handleImageUpload}
                disabled={sdzUploading}
              />
            )}
            {sdzUploading && <p className="sdz-meta">アップロード中...</p>}
          </fieldset>

          {/* ===== 送信 ===== */}
          <div style={{ display: 'flex', gap: 8, marginTop: 8 }}>
            <button type="submit" disabled={sdzSubmitting || !sdzName}>
              {sdzSubmitting ? '保存中...' : isEdit ? '更新' : '作成'}
            </button>
            <button type="button" className="sdz-ghost" onClick={() => navigate('/admin')}>
              キャンセル
            </button>
          </div>
        </div>
      </form>
    </div>
  );
}
