import React, { createContext, useState, useContext, useEffect } from 'react';
import { jwtDecode } from 'jwt-decode';

interface AuthContextType {
  isLoggedIn: boolean;
  token: string | null;
  loading: boolean;
  isAdmin: boolean;
  login: (token: string) => void;
  logout: () => void;
}

interface JwtPayload {
  email: string;
  role: string;
  exp: number;
}

const AuthContext = createContext<AuthContextType>({
  isLoggedIn: false,
  token: null,
  loading: true,
  isAdmin: false,
  login: () => { },
  logout: () => { },
});

export const useAuth = () => useContext(AuthContext);

export const AuthProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const [token, setToken] = useState<string | null>(null);
  const [isLoggedIn, setIsLoggedIn] = useState(false);
  const [loading, setLoading] = useState(true);
  const [isAdmin, setIsAdmin] = useState(false);

  useEffect(() => {
    const storedToken = localStorage.getItem('token');
    if (storedToken) {
      try {
        const claims = jwtDecode<JwtPayload>(storedToken);
        if (claims && claims.exp * 1000 > Date.now()) {
          setToken(storedToken);
          setIsLoggedIn(true);
          setIsAdmin(claims.role === 'admin');
        } else {
          localStorage.removeItem('token');
        }
      } catch (e) {
        localStorage.removeItem('token');
      }
    }
    setLoading(false);
  }, []);

  const login = (newToken: string) => {
    localStorage.setItem('token', newToken);
    setToken(newToken);
    setIsLoggedIn(true);
    
    try {
      const claims = jwtDecode<JwtPayload>(newToken);
      setIsAdmin(claims?.role === 'admin');
    } catch (e) {
      setIsAdmin(false);
    }
  };

  const logout = () => {
    localStorage.removeItem('token');
    setToken(null);
    setIsLoggedIn(false);
    setIsAdmin(false);
  };

  const value = {
    token,
    isLoggedIn,
    loading,
    isAdmin,
    login,
    logout,
  };

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
};

export default AuthContext; 
