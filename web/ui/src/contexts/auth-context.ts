import { User } from 'firebase/auth';
import { createContext } from 'react';

export type AuthContextValue = {
  user: User | null;
  idToken: string | null;
  loading: boolean;
  loginWithGoogle: () => Promise<void>;
  loginWithEmail: (email: string, password: string) => Promise<void>;
  signupWithEmail: (email: string, password: string) => Promise<void>;
  resendVerification: () => Promise<void>;
  sendPasswordReset: () => Promise<void>;
  updateDisplayName: (displayName: string) => Promise<void>;
  logout: () => Promise<void>;
};

export const AuthContext = createContext<AuthContextValue>({
  user: null,
  idToken: null,
  loading: true,
  loginWithGoogle: async () => {},
  loginWithEmail: async () => {},
  signupWithEmail: async () => {},
  resendVerification: async () => {},
  sendPasswordReset: async () => {},
  updateDisplayName: async () => {},
  logout: async () => {},
});
