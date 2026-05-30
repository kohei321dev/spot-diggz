import { useCallback, useEffect, useRef, useState } from 'react';
import type { SdzPlaceResult } from '../types/spot';

interface SdzAdminMapPickerProps {
  lat: string;
  lng: string;
  onLocationChange: (lat: string, lng: string) => void;
  onPlaceSelect?: (place: SdzPlaceResult) => void;
}

const SDZ_GOOGLE_MAPS_API_KEY = import.meta.env.VITE_GOOGLE_MAPS_API_KEY || '';
const SDZ_DEFAULT_CENTER = { lat: 35.6812, lng: 139.7671 };
const SDZ_PLACE_FIELDS = [
  'place_id',
  'name',
  'formatted_address',
  'geometry',
  'url',
  'formatted_phone_number',
  'website',
  'opening_hours',
  'rating',
  'user_ratings_total',
  'types',
];

// Google Maps JS API を動的にロード
function sdzLoadGoogleMapsScript(): Promise<void> {
  return new Promise((resolve, reject) => {
    if (typeof google !== 'undefined' && google.maps) {
      resolve();
      return;
    }
    const existing = document.querySelector('script[src*="maps.googleapis.com"]');
    if (existing) {
      existing.addEventListener('load', () => resolve());
      return;
    }
    const script = document.createElement('script');
    script.src = `https://maps.googleapis.com/maps/api/js?key=${SDZ_GOOGLE_MAPS_API_KEY}&libraries=places&language=ja&region=JP`;
    script.async = true;
    script.onload = () => resolve();
    script.onerror = () => reject(new Error('Google Maps API の読み込みに失敗しました'));
    document.head.appendChild(script);
  });
}

// eslint-disable-next-line @typescript-eslint/no-explicit-any
declare const google: any;

export function SdzAdminMapPicker({
  lat,
  lng,
  onLocationChange,
  onPlaceSelect,
}: SdzAdminMapPickerProps) {
  const mapRef = useRef<HTMLDivElement>(null);
  const searchRef = useRef<HTMLInputElement>(null);
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const mapInstanceRef = useRef<any>(null);
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const markerRef = useRef<any>(null);
  const [sdzReady, setSdzReady] = useState(false);
  const [sdzError, setSdzError] = useState<string | null>(null);

  // Google Maps API をロード
  useEffect(() => {
    if (!SDZ_GOOGLE_MAPS_API_KEY) {
      setSdzError('VITE_GOOGLE_MAPS_API_KEY が設定されていません');
      return;
    }
    sdzLoadGoogleMapsScript()
      .then(() => setSdzReady(true))
      .catch((err) => setSdzError((err as Error).message));
  }, []);

  const handlePlaceChanged = useCallback(
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    (autocomplete: any) => {
      const place = autocomplete.getPlace();
      if (!place?.geometry?.location) return;

      const placeLat = place.geometry.location.lat();
      const placeLng = place.geometry.location.lng();

      onLocationChange(placeLat.toFixed(6), placeLng.toFixed(6));

      if (mapInstanceRef.current) {
        mapInstanceRef.current.panTo({ lat: placeLat, lng: placeLng });
        mapInstanceRef.current.setZoom(16);
      }
      if (markerRef.current) {
        markerRef.current.setPosition({ lat: placeLat, lng: placeLng });
      }

      if (onPlaceSelect) {
        const result: SdzPlaceResult = {
          placeId: place.place_id || '',
          name: place.name || '',
          address: place.formatted_address || '',
          lat: placeLat,
          lng: placeLng,
          googleMapsUrl: place.url,
          phoneNumber: place.formatted_phone_number,
          website: place.website,
          rating: place.rating,
          ratingCount: place.user_ratings_total,
          types: place.types,
          businessHours: place.opening_hours?.weekday_text,
        };
        onPlaceSelect(result);
      }
    },
    [onLocationChange, onPlaceSelect],
  );

  // 地図とAutocomplete を初期化
  useEffect(() => {
    if (!sdzReady || !mapRef.current) return;

    const center = {
      lat: parseFloat(lat) || SDZ_DEFAULT_CENTER.lat,
      lng: parseFloat(lng) || SDZ_DEFAULT_CENTER.lng,
    };

    const map = new google.maps.Map(mapRef.current, {
      center,
      zoom: 15,
      mapTypeControl: false,
      streetViewControl: false,
    });
    mapInstanceRef.current = map;

    const marker = new google.maps.Marker({
      position: center,
      map,
      draggable: true,
    });
    markerRef.current = marker;

    // マーカードラッグで座標更新
    marker.addListener('dragend', () => {
      const pos = marker.getPosition();
      if (pos) {
        onLocationChange(pos.lat().toFixed(6), pos.lng().toFixed(6));
      }
    });

    // 地図クリックで座標更新
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    map.addListener('click', (e: any) => {
      if (e.latLng) {
        marker.setPosition(e.latLng);
        onLocationChange(e.latLng.lat().toFixed(6), e.latLng.lng().toFixed(6));
      }
    });

    // Places Autocomplete を検索欄に接続
    if (searchRef.current) {
      const autocomplete = new google.maps.places.Autocomplete(searchRef.current, {
        fields: SDZ_PLACE_FIELDS,
        componentRestrictions: { country: 'jp' },
      });
      autocomplete.bindTo('bounds', map);
      autocomplete.addListener('place_changed', () => handlePlaceChanged(autocomplete));
    }

    // lat/lng の初期値がない場合、現在地を取得
    if (!lat && !lng && navigator.geolocation) {
      navigator.geolocation.getCurrentPosition(
        (pos) => {
          const userPos = { lat: pos.coords.latitude, lng: pos.coords.longitude };
          map.panTo(userPos);
          marker.setPosition(userPos);
          onLocationChange(userPos.lat.toFixed(6), userPos.lng.toFixed(6));
        },
        () => {
          // 位置情報取得に失敗してもデフォルト位置で表示
        },
      );
    }
    // 初期化は1回のみ
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [sdzReady]);

  if (sdzError) {
    return (
      <div
        style={{
          padding: 12,
          background: '#fff3cd',
          borderRadius: 6,
          fontSize: 14,
          marginBottom: 8,
        }}
      >
        ⚠️ {sdzError}
        <br />
        <span style={{ fontSize: 12, color: '#666' }}>
          緯度・経度を直接入力してスポットを登録できます。
        </span>
      </div>
    );
  }

  if (!sdzReady) {
    return <p style={{ color: '#888' }}>地図を読み込み中...</p>;
  }

  return (
    <div>
      <input
        ref={searchRef}
        type="text"
        placeholder="場所を検索（施設名・住所）..."
        style={{
          width: '100%',
          padding: '8px 12px',
          marginBottom: 8,
          border: '1px solid #ccc',
          borderRadius: 6,
          fontSize: 14,
        }}
      />
      <div ref={mapRef} style={{ width: '100%', height: 350, borderRadius: 6 }} />
    </div>
  );
}
