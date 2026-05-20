import { createContext, useContext } from 'react';

export interface AuthContextType {
  isLoggedIn: boolean;
  token: string | null;
  loading: boolean;
  isAdmin: boolean;
  login: (token: string) => void;
  logout: () => void;
}

export interface JwtPayload {
  email: string;
  role: string;
  exp: number;
}

export const AuthContext = createContext<AuthContextType>({
  isLoggedIn: false,
  token: null,
  loading: true,
  isAdmin: false,
  login: () => { },
  logout: () => { },
});

export const useAuth = () => useContext(AuthContext);
