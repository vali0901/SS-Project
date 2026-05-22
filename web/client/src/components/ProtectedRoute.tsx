// ProtectedRoute enforces authentication for child routes.

import React from 'react';
import { Navigate, Outlet } from 'react-router-dom';
import { useAuth } from '../contexts/AuthContextState';

interface ProtectedRouteProps {
  authRequired: boolean;
  adminOnly?: boolean;
}

const ProtectedRoute: React.FC<ProtectedRouteProps> = ({ authRequired, adminOnly = false }) => {
  const { isLoggedIn, isAdmin, loading } = useAuth();

  // While auth state is loading, show a loading spinner
  if (loading) {
    return (
      <div className="flex justify-center items-center min-h-[60vh]">
        <div className="animate-spin rounded-full h-12 w-12 border-t-2 border-b-2 border-sky-500"></div>
      </div>
    );
  }

  // If auth is required and user is not logged in, redirect to login
  if (authRequired && !isLoggedIn) {
    return <Navigate to="/login" replace />;
  }

  // If admin access is required but user is not an admin, redirect to home
  if (adminOnly && !isAdmin) {
    return <Navigate to="/" replace />;
  }

  // If auth is not required (e.g. login/register pages) and user is logged in, redirect to root
  if (!authRequired && isLoggedIn) {
    return <Navigate to="/" replace />;
  }

  // Otherwise, render the children routes
  return <Outlet />;
};

export default ProtectedRoute; 
