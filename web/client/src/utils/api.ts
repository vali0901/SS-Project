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

export const getMediaUrl = (path: string) => {
  if (!path) return '';
  if (path.startsWith('http')) return path;
  if (path.startsWith(API_BASE_URL)) return path;
  return buildUrl(path);
};

// Middleware/Interceptor types
type RequestMiddleware = (init: RequestInit) => RequestInit | Promise<RequestInit>;
type ResponseMiddleware = (response: Response) => Response | Promise<Response>;

class ApiClient {
  private requestMiddleware: RequestMiddleware[] = [];
  private responseMiddleware: ResponseMiddleware[] = [];

  useRequest(middleware: RequestMiddleware) {
    this.requestMiddleware.push(middleware);
  }

  useResponse(middleware: ResponseMiddleware) {
    this.responseMiddleware.push(middleware);
  }

  async fetch(path: string, init: RequestInit = {}): Promise<Response> {
    let currentInit = { ...init };

    // Run request middleware
    for (const middleware of this.requestMiddleware) {
      currentInit = await middleware(currentInit);
    }

    let response = await fetch(buildUrl(path), currentInit);

    // Run response middleware
    for (const middleware of this.responseMiddleware) {
      response = await middleware(response);
    }

    return response;
  }
}

export const apiClient = new ApiClient();

// Add default JWT auth middleware
apiClient.useRequest((init) => {
  const token = localStorage.getItem('token');
  if (token) {
    const headers = new Headers(init.headers);
    if (!headers.has('Authorization')) {
      headers.set('Authorization', `Bearer ${token}`);
    }
    return { ...init, headers };
  }
  return init;
});

// Add default JSON content-type middleware
apiClient.useRequest((init) => {
  const headers = new Headers(init.headers);
  if (!headers.has('Content-Type') && !(init.body instanceof FormData)) {
    headers.set('Content-Type', 'application/json');
  }
  return { ...init, headers };
});

// Add 401 handling middleware
apiClient.useResponse((response) => {
  if (response.status === 401) {
    localStorage.removeItem('token');
    // We can't easily trigger a redirect here without a global event or store,
    // but clearing the token will cause ProtectedRoute to redirect on next render.
    window.dispatchEvent(new Event('auth-unauthorized'));
  }
  return response;
});

// Export apiFetch for backward compatibility, now using the client
export const apiFetch = (path: string, init?: RequestInit) => apiClient.fetch(path, init);
