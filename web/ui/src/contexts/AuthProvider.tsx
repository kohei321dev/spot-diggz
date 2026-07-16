import { useMemo } from 'react';
import { AuthContext, type AuthContextValue } from './auth-context';

const authNotConnected = async () => {
  throw new Error('Browser auth is not connected yet.');
};

export const AuthProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const value = useMemo<AuthContextValue>(
    () => ({
      user: null,
      idToken: null,
      loading: false,
      loginWithGoogle: authNotConnected,
      loginWithEmail: authNotConnected,
      signupWithEmail: authNotConnected,
      resendVerification: authNotConnected,
      sendPasswordReset: authNotConnected,
      updateDisplayName: authNotConnected,
      logout: async () => {},
    }),
    [],
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
};
