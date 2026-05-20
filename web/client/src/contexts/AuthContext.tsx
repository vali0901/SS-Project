import React, { useState, useEffect } from 'react';
import { jwtDecode } from 'jwt-decode';
import { AuthContext } from './AuthContextState';
import type { AuthContextType, JwtPayload } from './AuthContextState';

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
      } catch {
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
    } catch {
      setIsAdmin(false);
    }
  };

  const logout = () => {
    localStorage.removeItem('token');
    setToken(null);
    setIsLoggedIn(false);
    setIsAdmin(false);
  };

  const value: AuthContextType = {
    token,
    isLoggedIn,
    loading,
    isAdmin,
    login,
    logout,
  };

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
};
