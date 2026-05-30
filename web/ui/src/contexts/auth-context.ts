import { createContext } from 'react';

export type SdzBrowserUser = {
  uid: string;
  email?: string | null;
  displayName?: string | null;
  emailVerified?: boolean;
  providerData?: { providerId: string }[];
};

export type AuthContextValue = {
  user: SdzBrowserUser | null;
  idToken: string | null;
  loading: boolean;
  loginWithGoogle: () => Promise<void>;
  loginWithEmail: (email: string, password: string) => Promise<void>;
  signupWithEmail: (email: string, password: string) => Promise<void>;
  resendVerification: () => Promise<void>;
  logout: () => Promise<void>;
  sendPasswordReset: () => Promise<void>;
  updateDisplayName: (displayName: string) => Promise<void>;
};

const authNotConnected = async () => {
  throw new Error('Browser auth is not connected yet.');
};

export const AuthContext = createContext<AuthContextValue>({
  user: null,
  idToken: null,
  loading: false,
  loginWithGoogle: authNotConnected,
  loginWithEmail: authNotConnected,
  signupWithEmail: authNotConnected,
  resendVerification: authNotConnected,
  logout: async () => {},
  sendPasswordReset: authNotConnected,
  updateDisplayName: authNotConnected,
});
