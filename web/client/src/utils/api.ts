const DEFAULT_BASE_URL = '/api';

const normalize = (url: string) => url.replace(/\/+$/, '');

export const API_BASE_URL = normalize(import.meta.env.VITE_API_BASE_URL ?? DEFAULT_BASE_URL);

const buildUrl = (path: string) => {
  if (path.startsWith('http')) {
    return path;
  }
  const normalizedPath = path.startsWith('/') ? path : `/${path}`;
  return `${API_BASE_URL}${normalizedPath}`;
};

export const apiFetch = (path: string, init?: RequestInit) => fetch(buildUrl(path), init);

export const getMediaUrl = (path: string) => {
  if (!path) return '';
  if (path.startsWith('http')) return path;
  // If it already starts with the API_BASE_URL, return as is
  if (path.startsWith(API_BASE_URL)) return path;
  // Otherwise build the full URL through the proxy
  return buildUrl(path);
};

