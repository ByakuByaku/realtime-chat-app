const API_PREFIX = '/api/v1';

class ApiClient {
  private accessToken: string | null = null;
  private refreshToken: string | null = null;
  private refreshPromise: Promise<boolean> | null = null;

  constructor() {
    this.accessToken = window.localStorage.getItem('access_token');
    this.refreshToken = window.localStorage.getItem('refresh_token');
  }

  setTokens(accessToken: string | null, refreshToken: string | null) {
    this.accessToken = accessToken;
    this.refreshToken = refreshToken;
    if (accessToken) {
      window.localStorage.setItem('access_token', accessToken);
    } else {
      window.localStorage.removeItem('access_token');
    }
    if (refreshToken) {
      window.localStorage.setItem('refresh_token', refreshToken);
    } else {
      window.localStorage.removeItem('refresh_token');
    }
  }

  getAccessToken() {
    return this.accessToken;
  }

  getRefreshToken() {
    return this.refreshToken;
  }

  clear() {
    this.accessToken = null;
    this.refreshToken = null;
    this.refreshPromise = null;
    window.localStorage.removeItem('access_token');
    window.localStorage.removeItem('refresh_token');
  }

  async request<T>(path: string, init: RequestInit = {}, retry = true): Promise<T> {
    const headers = new Headers(init.headers);
    if (this.accessToken) {
      headers.set('Authorization', `Bearer ${this.accessToken}`);
    }
    if (!headers.has('Content-Type') && init.body && typeof init.body === 'string') {
      headers.set('Content-Type', 'application/json');
    }

    const response = await fetch(`${API_PREFIX}${path}`, {
      ...init,
      headers,
    });

    if (response.status === 401 && retry) {
      const refreshed = await this.refreshAccessToken();
      if (refreshed) {
        return this.request<T>(path, init, false);
      }
    }

    if (!response.ok) {
      const payload = await response.json().catch(() => null);
      const message = payload?.error || payload?.details || response.statusText || 'Ошибка запроса';
      throw new Error(message);
    }

    if (response.status === 204) {
      return undefined as T;
    }

    return response.json() as Promise<T>;
  }

  private async refreshAccessToken(): Promise<boolean> {
    if (!this.refreshToken) {
      this.clear();
      return false;
    }
    if (this.refreshPromise) {
      return this.refreshPromise.catch(() => false);
    }

    this.refreshPromise = (async () => {
      try {
        const response = await fetch(`${API_PREFIX}/auth/refresh`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ refresh_token: this.refreshToken }),
        });
        if (!response.ok) {
          throw new Error('refresh failed');
        }
        const payload = (await response.json()) as { access_token: string; refresh_token: string };
        this.setTokens(payload.access_token, payload.refresh_token);
        return true;
      } catch {
        this.clear();
        return false;
      } finally {
        this.refreshPromise = null;
      }
    })();

    return this.refreshPromise.catch(() => false);
  }

  async authRegister(login: string, password: string) {
    const payload = await this.request<{ user: any; access_token: string; refresh_token: string }>('/auth/register', {
      method: 'POST',
      body: JSON.stringify({ login, password }),
    }, false);
    this.setTokens(payload.access_token, payload.refresh_token);
    return payload;
  }

  async authLogin(login: string, password: string) {
    const payload = await this.request<{ user: any; access_token: string; refresh_token: string }>('/auth/login', {
      method: 'POST',
      body: JSON.stringify({ login, password }),
    }, false);
    this.setTokens(payload.access_token, payload.refresh_token);
    return payload;
  }

  async authLogout() {
    if (!this.refreshToken) {
      this.clear();
      return;
    }
    try {
      await this.request('/auth/logout', {
        method: 'POST',
        body: JSON.stringify({ refresh_token: this.refreshToken }),
      }, false);
    } finally {
      this.clear();
    }
  }

  async getChats() {
    return this.request<{ items: any[] }>('/chats');
  }

  async createChat(type: 'direct' | 'group', title?: string) {
    return this.request<any>('/chats', {
      method: 'POST',
      body: JSON.stringify({ type, title: title || null }),
    });
  }

  async addMember(chatId: string, userId: string, role: 'member' | 'admin' = 'member') {
    return this.request<any>(`/chats/${chatId}/members`, {
      method: 'POST',
      body: JSON.stringify({ user_id: userId, role }),
    });
  }

  async getMembers(chatId: string) {
    return this.request<{ items: any[] }>(`/chats/${chatId}/members`);
  }

  async removeMember(chatId: string, userId: string) {
    return this.request<any>(`/chats/${chatId}/members/${userId}`, {
      method: 'DELETE',
    });
  }

  async deleteChat(chatId: string) {
    return this.request<any>(`/chats/${chatId}`, {
      method: 'DELETE',
    });
  }

  async getMessages(chatId: string, limit = 50, offset = 0) {
    return this.request<any>(`/chats/${chatId}/messages?limit=${limit}&offset=${offset}`);
  }

  async sendMessage(chatId: string, body: string, clientMsgId?: string) {
    return this.request<any>(`/chats/${chatId}/messages`, {
      method: 'POST',
      body: JSON.stringify({ body, client_msg_id: clientMsgId }),
    });
  }

  async searchMessages(chatId: string, query: string) {
    return this.request<any>(`/chats/${chatId}/search?q=${encodeURIComponent(query)}`);
  }
}

export default new ApiClient();
