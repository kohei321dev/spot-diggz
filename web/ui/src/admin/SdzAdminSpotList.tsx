import { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import type { SdzSpot } from '../types/spot';

const sdzApiUrl = import.meta.env.VITE_SDZ_API_URL || 'http://localhost:8080';

// 管理画面: スポット一覧
export function SdzAdminSpotList() {
  const [sdzSpots, setSdzSpots] = useState<SdzSpot[]>([]);
  const [sdzLoading, setSdzLoading] = useState(true);
  const [sdzError, setSdzError] = useState<string | null>(null);

  useEffect(() => {
    fetch(`${sdzApiUrl}/sdz/spots`)
      .then((res) => {
        if (!res.ok) throw new Error(`HTTP ${res.status}`);
        return res.json() as Promise<SdzSpot[]>;
      })
      .then(setSdzSpots)
      .catch((err) => setSdzError((err as Error).message))
      .finally(() => setSdzLoading(false));
  }, []);

  if (sdzLoading) return <p>読み込み中...</p>;
  if (sdzError) return <div className="sdz-error">エラー: {sdzError}</div>;

  return (
    <div className="sdz-card">
      <h2>スポット管理</h2>
      <p className="sdz-meta">{sdzSpots.length}件のスポット</p>

      <div style={{ display: 'flex', flexDirection: 'column', gap: 8, marginTop: 12 }}>
        {sdzSpots.map((spot) => (
          <Link
            key={spot.spotId}
            to={`/admin/spots/${spot.spotId}/edit`}
            style={{
              display: 'flex',
              justifyContent: 'space-between',
              alignItems: 'center',
              padding: '8px 12px',
              border: '1px solid #e0e0e0',
              borderRadius: 6,
              textDecoration: 'none',
              color: 'inherit',
            }}
          >
            <div>
              <div style={{ fontWeight: 500 }}>{spot.name}</div>
              <div style={{ fontSize: 12, color: '#888' }}>
                {spot.parkAttributes ? 'パーク' : spot.streetAttributes ? 'ストリート' : '未分類'}{' '}
                ・ {spot.tags.join(', ') || 'タグなし'}
              </div>
            </div>
            <span style={{ fontSize: 12, color: '#999' }}>編集 →</span>
          </Link>
        ))}
      </div>
    </div>
  );
}
