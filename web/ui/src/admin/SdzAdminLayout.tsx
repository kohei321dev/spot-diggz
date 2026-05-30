import { Link, Outlet } from 'react-router-dom';

// 管理画面の共通レイアウト
export function SdzAdminLayout() {
  return (
    <div style={{ maxWidth: 960, margin: '0 auto', padding: 16 }}>
      <nav style={{ marginBottom: 16, display: 'flex', gap: 16, alignItems: 'center' }}>
        <Link to="/admin" style={{ fontWeight: 600, textDecoration: 'none' }}>
          管理画面
        </Link>
        <Link to="/admin/spots/new" style={{ textDecoration: 'none' }}>
          + スポット作成
        </Link>
        <Link to="/" style={{ marginLeft: 'auto', textDecoration: 'none', fontSize: 14 }}>
          ← サイトに戻る
        </Link>
      </nav>
      <Outlet />
    </div>
  );
}
