import { describe, it, expect, vi, beforeEach } from 'vitest';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import App from './App';
import { MemoryRouter } from 'react-router-dom';

describe('App', () => {
  beforeEach(() => {
    if (!window.localStorage || typeof window.localStorage.clear !== 'function') {
      const store = new Map<string, string>();
      Object.defineProperty(window, 'localStorage', {
        value: {
          getItem: (key: string) => store.get(key) ?? null,
          setItem: (key: string, value: string) => store.set(key, value),
          removeItem: (key: string) => store.delete(key),
          clear: () => store.clear(),
        },
        configurable: true,
      });
    }
    window.localStorage.clear();
  });

  it('renders map heading', () => {
    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => [],
    } as Response);
    render(
      <MemoryRouter>
        <App />
      </MemoryRouter>,
    );
    return waitFor(() => {
      expect(screen.getByRole('heading', { name: /マップでスポットを探す/i })).toBeInTheDocument();
    });
  });

  it('toggles favorites in list', async () => {
    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => [
        {
          spotId: 'spot-1',
          name: 'Favorite Spot',
          description: null,
          location: { lat: 35.0, lng: 139.0 },
          tags: ['smoke'],
          visibility: 'public',
          images: [],
          trustLevel: 'unverified',
          trustSources: [],
          userId: 'user-1',
          createdAt: '2025-01-01T00:00:00Z',
          updatedAt: '2025-01-01T00:00:00Z',
        },
      ],
    } as Response);

    render(
      <MemoryRouter>
        <App />
      </MemoryRouter>,
    );

    await waitFor(() => {
      expect(screen.getAllByText('Favorite Spot').length).toBeGreaterThan(0);
    });

    const addButton = screen.getByRole('button', { name: '☆' });
    fireEvent.click(addButton);
    expect(screen.getByRole('button', { name: '★' })).toBeInTheDocument();
  });

  it('shows favorites page', async () => {
    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => [
        {
          spotId: 'spot-1',
          name: 'Favorite Spot',
          description: null,
          location: { lat: 35.0, lng: 139.0 },
          tags: ['smoke'],
          visibility: 'public',
          images: [],
          trustLevel: 'unverified',
          trustSources: [],
          userId: 'user-1',
          createdAt: '2025-01-01T00:00:00Z',
          updatedAt: '2025-01-01T00:00:00Z',
        },
      ],
    } as Response);

    window.localStorage.setItem('sdzFavorites', JSON.stringify(['spot-1']));

    render(
      <MemoryRouter>
        <App />
      </MemoryRouter>,
    );

    await waitFor(() => {
      expect(screen.getAllByText('Favorite Spot').length).toBeGreaterThan(0);
    });

    fireEvent.click(screen.getByRole('button', { name: 'Favorites' }));

    await waitFor(() => {
      expect(screen.getByRole('heading', { name: 'お気に入り一覧' })).toBeInTheDocument();
    });
  });

  it('filters spots client-side using the Go API list contract', async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => [
        {
          spotId: 'spot-park',
          name: 'Ledge Plaza',
          description: 'Park with ledges',
          location: { lat: 35.0, lng: 139.0 },
          tags: ['park', 'ledge'],
          visibility: 'public',
          createdAt: '2025-01-01T00:00:00Z',
          updatedAt: '2025-01-01T00:00:00Z',
        },
        {
          spotId: 'spot-street',
          name: 'Street Rail',
          description: 'Street rail by the station',
          location: { lat: 35.1, lng: 139.1 },
          tags: ['street', 'rail'],
          visibility: 'public',
          createdAt: '2025-01-02T00:00:00Z',
          updatedAt: '2025-01-02T00:00:00Z',
        },
      ],
    } as Response);
    globalThis.fetch = fetchMock;

    render(
      <MemoryRouter>
        <App />
      </MemoryRouter>,
    );

    await waitFor(() => {
      expect(screen.getAllByText('Ledge Plaza').length).toBeGreaterThan(0);
      expect(screen.getAllByText('Street Rail').length).toBeGreaterThan(0);
    });
    expect(fetchMock.mock.calls[0][0]).toBe('http://localhost:8080/sdz/spots?visibility=public');

    fireEvent.change(screen.getByLabelText('キーワード'), { target: { value: 'rail' } });

    await waitFor(() => {
      expect(screen.queryByText('Ledge Plaza')).not.toBeInTheDocument();
      expect(screen.getAllByText('Street Rail').length).toBeGreaterThan(0);
    });

    fireEvent.change(screen.getByLabelText('キーワード'), { target: { value: '' } });
    fireEvent.change(screen.getByLabelText('種別'), { target: { value: 'park' } });

    await waitFor(() => {
      expect(screen.getAllByText('Ledge Plaza').length).toBeGreaterThan(0);
      expect(screen.queryByText('Street Rail')).not.toBeInTheDocument();
    });
  });
});
