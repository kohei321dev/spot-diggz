import { Navigate, Outlet } from 'react-router-dom';
import { useAuth } from '../contexts/useAuth';

// 管理画面アクセスガード: 認証済みユーザーのみ許可
export function SdzAdminGuard() {
  const { user, loading } = useAuth();

  if (loading) {
    return <p>認証確認中...</p>;
  }

  if (!user) {
    return <Navigate to="/" replace />;
  }

  return <Outlet />;
}
