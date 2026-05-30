const sdzApiUrl = import.meta.env.VITE_SDZ_API_URL || 'http://localhost:8080';

import type { SdzSpotParkAttributes, SdzStreetAttributes } from '../types/spot';

export interface SdzSpotPayload {
  name: string;
  description?: string;
  location?: { lat: number; lng: number };
  tags: string[];
  images: string[];
  parkAttributes?: SdzSpotParkAttributes;
  streetAttributes?: SdzStreetAttributes;
  instagramTag?: string;
  instagramLocationUrl?: string;
  instagramProfileUrl?: string;
  googlePlaceId?: string;
  googleMapsUrl?: string;
  address?: string;
  phoneNumber?: string;
  googleRating?: number;
  googleRatingCount?: number;
  googleTypes?: string[];
}

async function sdzAdminFetch(
  path: string,
  idToken: string,
  options?: RequestInit,
): Promise<Response> {
  const res = await fetch(`${sdzApiUrl}${path}`, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${idToken}`,
      ...options?.headers,
    },
  });
  if (!res.ok) {
    const text = await res.text().catch(() => '');
    throw new Error(`HTTP ${res.status}: ${text}`);
  }
  return res;
}

export async function sdzAdminCreateSpot(idToken: string, payload: SdzSpotPayload): Promise<void> {
  await sdzAdminFetch('/sdz/admin/spots', idToken, {
    method: 'POST',
    body: JSON.stringify(payload),
  });
}

export async function sdzAdminUpdateSpot(
  idToken: string,
  spotId: string,
  payload: Partial<SdzSpotPayload>,
): Promise<void> {
  await sdzAdminFetch(`/sdz/admin/spots/${spotId}`, idToken, {
    method: 'PATCH',
    body: JSON.stringify(payload),
  });
}

export async function sdzAdminGetUploadUrl(
  idToken: string,
  contentType: string,
): Promise<{ uploadUrl: string; objectUrl: string }> {
  const res = await sdzAdminFetch('/sdz/admin/spots/upload-url', idToken, {
    method: 'POST',
    body: JSON.stringify({ sdzContentType: contentType }),
  });
  return res.json();
}

export async function sdzAdminUploadImage(uploadUrl: string, file: File): Promise<void> {
  const res = await fetch(uploadUrl, {
    method: 'PUT',
    headers: { 'Content-Type': file.type },
    body: file,
  });
  if (!res.ok) {
    throw new Error(`画像アップロード失敗: HTTP ${res.status}`);
  }
}
