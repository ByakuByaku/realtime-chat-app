import { createContext, useCallback, useContext, useEffect, useMemo, useRef, useState } from 'react';
import { Navigate, Route, Routes, useNavigate, useParams } from 'react-router-dom';
import api from './api';
import type { Chat, ChatMember, Message, SocketMessageFrame, User } from './types';

type AuthContextValue = {
  user: User | null;
  isAuthenticated: boolean;
  login: (login: string, password: string) => Promise<void>;
  register: (login: string, password: string) => Promise<void>;
  logout: () => Promise<void>;
};

type ChatMeta = {
  label?: string;
  members: string[];
};

type ChatContextValue = {
  chats: Chat[];
  refreshChats: () => Promise<void>;
  refreshChatMembers: (chatId: string) => Promise<void>;
  createChat: (type: 'direct' | 'group', title?: string) => Promise<Chat>;
  chatMeta: Record<string, ChatMeta>;
  chatMembers: Record<string, ChatMember[]>;
  updateChatMeta: (chatId: string, patch: Partial<ChatMeta> | ((prev: ChatMeta) => ChatMeta)) => void;
};

type RichMessage = Message & {
  status?: 'pending' | 'sent' | 'failed';
};

const MESSAGE_PAGE_SIZE = 50;

const AuthContext = createContext<AuthContextValue>({
  user: null,
  isAuthenticated: false,
  login: async () => undefined,
  register: async () => undefined,
  logout: async () => undefined,
});

const ChatContext = createContext<ChatContextValue>({
  chats: [],
  refreshChats: async () => undefined,
  refreshChatMembers: async () => undefined,
  createChat: async () => {
    throw new Error('chat context is not ready');
  },
  chatMeta: {},
  chatMembers: {},
  updateChatMeta: () => undefined,
});

function useAuth() {
  return useContext(AuthContext);
}

function useChats() {
  return useContext(ChatContext);
}

function chatTitle(chat: Chat, members: ChatMember[] = [], currentUserId?: string, fallbackLabel?: string) {
  if (chat.type === 'group') {
    return `Группа: ${chat.title || fallbackLabel || 'Без названия'}`;
  }

  const other = members.find((member) => member.user_id !== currentUserId);
  const name = other?.login || fallbackLabel || 'Личный чат';
  return `Личный чат: ${name}`;
}

function memberLabel(member: ChatMember, currentUserId?: string) {
  const label = member.login || member.user_id;
  return member.user_id === currentUserId ? `${label} (вы)` : label;
}

function App() {
  const [user, setUser] = useState<User | null>(() => {
    const stored = window.localStorage.getItem('auth_user');
    return stored ? (JSON.parse(stored) as User) : null;
  });
  const [chats, setChats] = useState<Chat[]>([]);
  const [chatMembers, setChatMembers] = useState<Record<string, ChatMember[]>>({});
  const [chatMeta, setChatMeta] = useState<Record<string, ChatMeta>>(() => {
    const stored = window.localStorage.getItem('chat_meta');
    return stored ? (JSON.parse(stored) as Record<string, ChatMeta>) : {};
  });

  const refreshChatMembers = useCallback(async (chatId: string) => {
    try {
      const response = await api.getMembers(chatId);
      setChatMembers((current) => ({
        ...current,
        [chatId]: (response.items ?? []) as ChatMember[],
      }));
    } catch {
      setChatMembers((current) => current);
    }
  }, []);

  const refreshChats = useCallback(async () => {
    if (!user) {
      setChats([]);
      setChatMembers({});
      return;
    }
    try {
      const response = await api.getChats();
      const items = (response.items ?? []) as Chat[];
      setChats(items);
      const entries = await Promise.all(
        items.map(async (chat) => {
          try {
            const membersResponse = await api.getMembers(chat.id);
            return [chat.id, (membersResponse.items ?? []) as ChatMember[]] as const;
          } catch {
            return [chat.id, [] as ChatMember[]] as const;
          }
        }),
      );
      setChatMembers(Object.fromEntries(entries));
    } catch {
      setChats([]);
    }
  }, [user]);

  const createChat = useCallback(async (type: 'direct' | 'group', title?: string) => {
    const created = await api.createChat(type, title);
    await refreshChats();
    return created as Chat;
  }, [refreshChats]);

  const login = useCallback(async (loginValue: string, password: string) => {
    const payload = await api.authLogin(loginValue, password);
    setUser(payload.user as User);
    window.localStorage.setItem('auth_user', JSON.stringify(payload.user));
  }, []);

  const register = useCallback(async (loginValue: string, password: string) => {
    const payload = await api.authRegister(loginValue, password);
    setUser(payload.user as User);
    window.localStorage.setItem('auth_user', JSON.stringify(payload.user));
  }, []);

  const logout = useCallback(async () => {
    await api.authLogout();
    setUser(null);
    window.localStorage.removeItem('auth_user');
    setChats([]);
  }, []);

  useEffect(() => {
    if (!user) {
      return;
    }
    void refreshChats();
  }, [refreshChats, user]);

  const authValue = useMemo<AuthContextValue>(
    () => ({ user, isAuthenticated: Boolean(user), login, register, logout }),
    [login, logout, register, user],
  );

  const updateChatMeta = useCallback((chatId: string, patch: Partial<ChatMeta> | ((prev: ChatMeta) => ChatMeta)) => {
    setChatMeta((current) => {
      const prevMeta = current[chatId] ?? { members: [] };
      const nextMeta = typeof patch === 'function' ? patch(prevMeta) : { ...prevMeta, ...patch };
      const nextValue = { ...current, [chatId]: nextMeta };
      window.localStorage.setItem('chat_meta', JSON.stringify(nextValue));
      return nextValue;
    });
  }, []);

  const chatValue = useMemo<ChatContextValue>(
    () => ({ chats, refreshChats, refreshChatMembers, createChat, chatMeta, chatMembers, updateChatMeta }),
    [chats, chatMembers, chatMeta, createChat, refreshChatMembers, refreshChats, updateChatMeta],
  );

  return (
    <AuthContext.Provider value={authValue}>
      <ChatContext.Provider value={chatValue}>
        <Routes>
          <Route path="/login" element={<AuthPage mode="login" />} />
          <Route path="/register" element={<AuthPage mode="register" />} />
          <Route
            path="/"
            element={
              <ProtectedRoute>
                <HomePage />
              </ProtectedRoute>
            }
          />
          <Route
            path="/chats/:chatId"
            element={
              <ProtectedRoute>
                <ChatPage />
              </ProtectedRoute>
            }
          />
          <Route path="*" element={<Navigate to="/" replace />} />
        </Routes>
      </ChatContext.Provider>
    </AuthContext.Provider>
  );
}

function ProtectedRoute({ children }: { children: JSX.Element }) {
  const { isAuthenticated } = useAuth();
  return isAuthenticated ? children : <Navigate to="/login" replace />;
}

function AuthPage({ mode }: { mode: 'login' | 'register' }) {
  const { login, register } = useAuth();
  const navigate = useNavigate();
  const [loginValue, setLoginValue] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  const submit = async (event: React.FormEvent) => {
    event.preventDefault();
    setBusy(true);
    setError(null);
    try {
      if (mode === 'login') {
        await login(loginValue, password);
      } else {
        await register(loginValue, password);
      }
      navigate('/');
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Не удалось выполнить авторизацию');
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="auth-shell">
      <form className="card auth-card" onSubmit={submit}>
        <h1>{mode === 'login' ? 'Вход' : 'Регистрация'}</h1>
        <label>
          Логин
          <input value={loginValue} onChange={(event) => setLoginValue(event.target.value)} required />
        </label>
        <label>
          Пароль
          <input type="password" value={password} onChange={(event) => setPassword(event.target.value)} required />
        </label>
        {error ? <p className="error">{error}</p> : null}
        <button disabled={busy}>{busy ? 'Подождите...' : mode === 'login' ? 'Войти' : 'Зарегистрироваться'}</button>
        <p className="secondary-link">
          {mode === 'login' ? (
            <span>
              Нет аккаунта? <a href="/register">Создать</a>
            </span>
          ) : (
            <span>
              Уже есть аккаунт? <a href="/login">Войти</a>
            </span>
          )}
        </p>
      </form>
    </div>
  );
}

function HomePage() {
  const { user, logout } = useAuth();
  const { chats, refreshChats, refreshChatMembers, createChat, chatMeta, chatMembers, updateChatMeta } = useChats();
  const navigate = useNavigate();
  const [type, setType] = useState<'direct' | 'group'>('group');
  const [title, setTitle] = useState('');
  const [directTargetId, setDirectTargetId] = useState('');
  const [directLabel, setDirectLabel] = useState('');
  const [copyState, setCopyState] = useState('Копировать ID');
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  const handleCreate = async (event: React.FormEvent) => {
    event.preventDefault();
    setBusy(true);
    setError(null);
    try {
      if (type === 'direct') {
        if (!directTargetId.trim()) {
          throw new Error('Введите ID другого пользователя');
        }

        const created = await createChat('direct');
        const targetId = directTargetId.trim();
        await api.addMember(created.id, targetId, 'member');
        updateChatMeta(created.id, {
          label: directLabel.trim() || undefined,
          members: [user?.id, targetId].filter(Boolean) as string[],
        });
        await refreshChatMembers(created.id);
        await refreshChats();
        navigate(`/chats/${created.id}`);
        return;
      }

      const created = await createChat(type, title);
      updateChatMeta(created.id, {
        label: title || 'Групповой чат',
        members: [user?.id].filter(Boolean) as string[],
      });
      await refreshChats();
      navigate(`/chats/${created.id}`);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Не удалось создать чат');
    } finally {
      setBusy(false);
    }
  };

  const copyMyId = async () => {
    if (!user?.id) {
      return;
    }
    try {
      await navigator.clipboard.writeText(user.id);
      setCopyState('Скопировано!');
      window.setTimeout(() => setCopyState('Копировать ID'), 1500);
    } catch {
      setCopyState('Не удалось скопировать');
    }
  };

  return (
    <div className="shell">
      <header className="topbar">
        <div>
          <h1>Чаты</h1>
          <p>Добро пожаловать, {user?.login}</p>
        </div>
        <button onClick={() => void logout()}>Выйти</button>
      </header>

      <div className="layout">
        <section className="card">
          <h2>Ваш ID</h2>
          <p className="empty">Поделитесь этим ID с другом, чтобы он мог начать с вами личный чат.</p>
          <div className="id-box">
            <code>{user?.id}</code>
            <button type="button" className="secondary" onClick={() => void copyMyId()}>{copyState}</button>
          </div>

          <h2>Создать чат</h2>
          <form onSubmit={handleCreate} className="stack">
            <label>
              Тип
              <select value={type} onChange={(event) => setType(event.target.value as 'direct' | 'group')}>
                <option value="direct">Личный</option>
                <option value="group">Групповой</option>
              </select>
            </label>
            {type === 'group' ? (
              <label>
                Название
                <input value={title} onChange={(event) => setTitle(event.target.value)} />
              </label>
            ) : (
              <>
                <label>
                  ID другого пользователя
                  <input value={directTargetId} onChange={(event) => setDirectTargetId(event.target.value)} placeholder="UUID другого пользователя" />
                </label>
                <label>
                  Имя контакта
                  <input value={directLabel} onChange={(event) => setDirectLabel(event.target.value)} placeholder="Необязательное имя для этого чата" />
                </label>
              </>
            )}
            {error ? <p className="error">{error}</p> : null}
            <button disabled={busy}>{busy ? 'Создание...' : 'Создать чат'}</button>
          </form>
        </section>

        <section className="card list-card">
          <h2>Ваши чаты</h2>
          {chats.length === 0 ? (
            <p className="empty">Пока нет чатов.</p>
          ) : (
            <div className="chat-list">
              {chats.map((chat) => {
                const meta = chatMeta[chat.id];
                const label = chatTitle(chat, chatMembers[chat.id], user?.id, meta?.label);
                return (
                  <button key={chat.id} className="chat-item" onClick={() => navigate(`/chats/${chat.id}`)}>
                    <strong>{label}</strong>
                    <span>Участников: {chatMembers[chat.id]?.length ?? 0}</span>
                  </button>
                );
              })}
            </div>
          )}
        </section>
      </div>
    </div>
  );
}

function ChatPage() {
  const { chatId } = useParams();
  const { user } = useAuth();
  const { chats, refreshChats, refreshChatMembers, chatMeta, chatMembers, updateChatMeta } = useChats();
  const navigate = useNavigate();
  const [messages, setMessages] = useState<RichMessage[]>([]);
  const [messageText, setMessageText] = useState('');
  const [searchQuery, setSearchQuery] = useState('');
  const [searchResults, setSearchResults] = useState<Message[]>([]);
  const [memberUserId, setMemberUserId] = useState('');
  const [memberRole, setMemberRole] = useState<'member' | 'admin'>('member');
  const [status, setStatus] = useState('Подключение...');
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const [historyOffset, setHistoryOffset] = useState(0);
  const [hasMoreHistory, setHasMoreHistory] = useState(true);
  const [loadingMore, setLoadingMore] = useState(false);
  const socketRef = useRef<WebSocket | null>(null);
  const pendingRef = useRef<Map<string, RichMessage>>(new Map());
  const reconnectRef = useRef<number>(0);
  const retryTimersRef = useRef<Map<string, number>>(new Map());
  const latestSeqRef = useRef<number>(0);
  const messagesContainerRef = useRef<HTMLDivElement | null>(null);
  const pendingScrollRef = useRef(false);

  useEffect(() => {
    if (!pendingScrollRef.current) {
      return;
    }
    pendingScrollRef.current = false;
    const container = messagesContainerRef.current;
    if (container) {
      container.scrollTop = container.scrollHeight;
    }
  }, [messages]);

  const currentChat = chats.find((item) => item.id === chatId);
  const currentMeta = chatId ? (chatMeta[chatId] ?? { members: [] }) : { members: [] };
  const fallbackMembers = Array.from(new Set([
    currentChat?.created_by,
    ...(currentMeta.members ?? []),
  ].filter(Boolean) as string[])).map((memberId) => ({
    chat_id: chatId ?? '',
    user_id: memberId,
    role: memberId === currentChat?.created_by ? 'admin' as const : 'member' as const,
  }));
  const currentMembers = chatId && chatMembers[chatId]?.length ? chatMembers[chatId] : fallbackMembers;
  const accessToken = api.getAccessToken();

  const sortMessages = useCallback((items: RichMessage[]) => {
    return [...items].sort((left, right) => (left.seq ?? 0) - (right.seq ?? 0));
  }, []);

  const clearRetryTimer = useCallback((clientMsgId: string) => {
    const timerId = retryTimersRef.current.get(clientMsgId);
    if (timerId) {
      window.clearTimeout(timerId);
      retryTimersRef.current.delete(clientMsgId);
    }
  }, []);

  const scheduleRetry = useCallback((message: RichMessage) => {
    clearRetryTimer(message.client_msg_id!);
    const timerId = window.setTimeout(() => {
      if (message.status === 'pending' && socketRef.current?.readyState === WebSocket.OPEN) {
        const pendingMessage = pendingRef.current.get(message.client_msg_id!);
        if (pendingMessage?.status === 'pending') {
          socketRef.current.send(JSON.stringify({ body: message.body, client_msg_id: message.client_msg_id }));
        }
      }
      retryTimersRef.current.delete(message.client_msg_id!);
    }, 5000);
    retryTimersRef.current.set(message.client_msg_id!, timerId);
  }, [clearRetryTimer]);

  useEffect(() => {
    if (!chatId || !accessToken || !user?.id) {
      return;
    }

    const initialise = async () => {
      setLoading(true);
      setHistoryOffset(0);
      setHasMoreHistory(true);
      try {
        const response = await api.getMessages(chatId, MESSAGE_PAGE_SIZE, 0);
        const history = (response.items ?? []).map((item: Message) => ({ ...item, status: 'sent' as const }));
        pendingScrollRef.current = true;
        setMessages(sortMessages(history));
        latestSeqRef.current = history.length > 0 ? Math.max(...history.map((item: Message) => item.seq)) : 0;
        setHistoryOffset(history.length);
        setHasMoreHistory(history.length === MESSAGE_PAGE_SIZE);
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Не удалось загрузить историю чата');
      } finally {
        setLoading(false);
      }
    };

    void initialise();
    void refreshChatMembers(chatId);
  }, [accessToken, chatId, refreshChatMembers, sortMessages, user?.id]);

  const loadMoreHistory = useCallback(async () => {
    if (!chatId || loadingMore || !hasMoreHistory) {
      return;
    }
    setLoadingMore(true);
    try {
      const response = await api.getMessages(chatId, MESSAGE_PAGE_SIZE, historyOffset);
      const older = (response.items ?? []).map((item: Message) => ({ ...item, status: 'sent' as const }));
      setMessages((current) => {
        const existingIds = new Set(current.map((item) => item.id));
        const newOnes = older.filter((item) => !existingIds.has(item.id));
        return sortMessages([...newOnes, ...current]);
      });
      setHistoryOffset((prev) => prev + older.length);
      setHasMoreHistory(older.length === MESSAGE_PAGE_SIZE);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Не удалось загрузить историю чата');
    } finally {
      setLoadingMore(false);
    }
  }, [chatId, hasMoreHistory, historyOffset, loadingMore, sortMessages]);

  useEffect(() => {
    if (!chatId || !accessToken || !user?.id) {
      return;
    }

    const connect = () => {
      const afterSeq = latestSeqRef.current;
      const wsUrl = `ws://localhost:8080/api/v1/chats/${chatId}/ws?token=${encodeURIComponent(accessToken)}&after_seq=${afterSeq}`;
      const socket = new WebSocket(wsUrl);
      socketRef.current = socket;

      socket.onopen = () => {
        setStatus('Подключено');
        reconnectRef.current = 0;
        pendingRef.current.forEach((message) => {
          if (message.status === 'pending' && socket.readyState === WebSocket.OPEN) {
            socket.send(JSON.stringify({ body: message.body, client_msg_id: message.client_msg_id }));
          }
        });
      };

      socket.onmessage = (event) => {
        const frame = JSON.parse(event.data) as SocketMessageFrame;
        if (frame.type === 'ack' && frame.message?.client_msg_id) {
          latestSeqRef.current = Math.max(latestSeqRef.current, frame.message.seq);
          setMessages((current) => {
            const next = current.map((item) => {
              if (item.client_msg_id === frame.message?.client_msg_id) {
                return {
                  ...item,
                  ...frame.message,
                  status: 'sent' as const,
                };
              }
              return item;
            });
            return sortMessages(next);
          });
          const pending = pendingRef.current.get(frame.message.client_msg_id);
          if (pending) {
            pendingRef.current.delete(frame.message.client_msg_id);
            clearRetryTimer(frame.message.client_msg_id);
          }
          return;
        }

        if (frame.type === 'message' && frame.message) {
          const incoming = frame.message;
          setMessages((current) => {
            const existingIndex = current.findIndex((item) => {
              if (incoming.client_msg_id) {
                return item.client_msg_id === incoming.client_msg_id;
              }
              return item.id === incoming.id;
            });

            if (existingIndex >= 0) {
              const next = current.map((item) => {
                if (incoming.client_msg_id && item.client_msg_id === incoming.client_msg_id) {
                  return {
                    ...item,
                    ...incoming,
                    status: 'sent' as const,
                  };
                }
                if (!incoming.client_msg_id && item.id === incoming.id) {
                  return {
                    ...item,
                    ...incoming,
                    status: 'sent' as const,
                  };
                }
                return item;
              });
              return sortMessages(next);
            }

            if (incoming.id) {
              const next = [...current, { ...incoming, status: 'sent' as const }];
              return sortMessages(next);
            }

            return current;
          });

          if (incoming.id) {
            latestSeqRef.current = Math.max(latestSeqRef.current, incoming.seq);
          }
          return;
        }

        if (frame.type === 'error') {
          setError(frame.error ?? 'Не удалось отправить сообщение');
          setMessages((current) => current.map((item) => (item.status === 'pending' ? { ...item, status: 'failed' as const } : item)));
        }
      };

      socket.onerror = () => {
        setStatus('Ошибка соединения');
      };

      socket.onclose = () => {
        setStatus('Переподключение...');
        const delay = Math.min(1000 * 2 ** reconnectRef.current, 4000);
        reconnectRef.current += 1;
        window.setTimeout(() => connect(), delay);
      };
    };

    connect();
    return () => {
      socketRef.current?.close();
      socketRef.current = null;
    };
  }, [accessToken, chatId, clearRetryTimer, sortMessages, user?.id]);

  useEffect(() => {
    return () => {
      retryTimersRef.current.forEach((timer) => window.clearTimeout(timer));
      retryTimersRef.current.clear();
    };
  }, []);

  const handleSend = async (event: React.FormEvent) => {
    event.preventDefault();
    if (!messageText.trim() || !chatId || !user?.id) {
      return;
    }

    const clientMsgId = crypto.randomUUID();
    const optimistic: RichMessage = {
      id: clientMsgId,
      chat_id: chatId,
      sender_id: user.id,
      body: messageText.trim(),
      client_msg_id: clientMsgId,
      seq: latestSeqRef.current + 1,
      created_at: new Date().toISOString(),
      status: 'pending',
    };

    pendingScrollRef.current = true;
    setMessages((current) => sortMessages([...current, optimistic]));
    pendingRef.current.set(clientMsgId, optimistic);
    scheduleRetry(optimistic);
    setMessageText('');

    if (socketRef.current?.readyState === WebSocket.OPEN) {
      socketRef.current.send(JSON.stringify({ body: optimistic.body, client_msg_id: optimistic.client_msg_id }));
    }
  };

  const handleSearch = async () => {
    if (!chatId || !searchQuery.trim()) {
      return;
    }
    try {
      const response = await api.searchMessages(chatId, searchQuery.trim());
      setSearchResults(response.items ?? []);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Ошибка поиска');
    }
  };

  const handleAddMember = async (event: React.FormEvent) => {
    event.preventDefault();
    if (!chatId || !memberUserId.trim()) {
      return;
    }
    try {
      const targetId = memberUserId.trim();
      await api.addMember(chatId, targetId, memberRole);
      updateChatMeta(chatId, (prev) => ({
        ...prev,
        members: Array.from(new Set([
          currentChat?.created_by,
          ...(prev.members ?? []),
          targetId,
        ].filter(Boolean) as string[]))
      }));
      setMemberUserId('');
      setError(null);
      await refreshChatMembers(chatId);
      await refreshChats();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Не удалось добавить участника');
    }
  };

  const handleRemoveMember = async (targetId: string) => {
    if (!chatId) {
      return;
    }
    try {
      await api.removeMember(chatId, targetId);
      updateChatMeta(chatId, (prev) => ({
        ...prev,
        members: (prev.members ?? []).filter((member) => member !== targetId),
      }));
      setError(null);
      await refreshChatMembers(chatId);
      await refreshChats();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Не удалось удалить участника');
    }
  };

  const handleLeave = async () => {
    if (!chatId || !user?.id) {
      return;
    }
    try {
      await api.removeMember(chatId, user.id);
      navigate('/');
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Не удалось покинуть чат');
    }
  };

  const handleDeleteChat = async () => {
    if (!chatId) {
      return;
    }
    try {
      await api.deleteChat(chatId);
      setError(null);
      await refreshChats();
      navigate('/');
    } catch (err) {
      if (currentChat?.type === 'direct' && user?.id && err instanceof Error && err.message === 'method_not_allowed') {
        try {
          await api.removeMember(chatId, user.id);
          setError(null);
          await refreshChats();
          navigate('/');
          return;
        } catch {
          // Fall through to the original delete error below.
        }
      }
      setError(err instanceof Error ? err.message : 'Не удалось удалить чат');
    }
  };

  if (!chatId) {
    return <Navigate to="/" replace />;
  }

  const headerTitle = currentChat
    ? chatTitle(currentChat, currentMembers, user?.id, currentMeta.label)
    : 'Чат';

  return (
    <div className="shell">
      <header className="topbar">
        <div>
          <button className="secondary" onClick={() => navigate('/')}>Назад</button>
          <h1>{headerTitle}</h1>
          <p>{status}</p>
        </div>
      </header>

      <div className="layout chat-layout">
        <section className="card chat-card">
          <div className="messages" ref={messagesContainerRef}>
            {loading ? <p className="empty">Загрузка истории...</p> : null}
            {!loading && hasMoreHistory ? (
              <button type="button" className="secondary" onClick={() => void loadMoreHistory()} disabled={loadingMore}>
                {loadingMore ? 'Загрузка...' : 'Загрузить более ранние сообщения'}
              </button>
            ) : null}
            {messages.map((message) => (
              <div key={message.id} className={`message-row ${message.sender_id === user?.id ? 'mine' : ''}`}>
                <div className="message-bubble">
                  <div>{message.body}</div>
                  <small>
                    {message.status === 'pending' ? 'Отправка...' : message.status === 'failed' ? 'Ошибка' : 'Доставлено'}
                  </small>
                </div>
              </div>
            ))}
          </div>

          <form className="composer" onSubmit={handleSend}>
            <input value={messageText} onChange={(event) => setMessageText(event.target.value)} placeholder="Введите сообщение" />
            <button type="submit">Отправить</button>
          </form>
          {error ? <p className="error">{error}</p> : null}
        </section>

        <aside className="card side-card">
          <h2>Поиск</h2>
          <div className="stack">
            <input value={searchQuery} onChange={(event) => setSearchQuery(event.target.value)} placeholder="Поиск сообщений" />
            <button onClick={() => void handleSearch()}>Найти</button>
            {searchResults.map((item) => (
              <div key={item.id} className="search-result">
                <strong>{item.body}</strong>
                <small>{item.created_at}</small>
              </div>
            ))}
          </div>

          {currentChat?.type === 'group' ? (
            <>
              <h2>Участники</h2>
              <form className="stack" onSubmit={handleAddMember}>
                <input value={memberUserId} onChange={(event) => setMemberUserId(event.target.value)} placeholder="ID пользователя" />
                <select value={memberRole} onChange={(event) => setMemberRole(event.target.value as 'member' | 'admin')}>
                  <option value="member">Участник</option>
                  <option value="admin">Администратор</option>
                </select>
                <button type="submit">Добавить участника</button>
              </form>
              <div className="participants-block">
                <strong>Участники</strong>
                <ul className="member-list">
                  {currentMembers.length > 0 ? currentMembers.map((member) => (
                    <li key={member.user_id} className="member-row">
                      <span>
                        {memberLabel(member, user?.id)}
                        <small>{member.role}</small>
                      </span>
                      {member.user_id !== user?.id ? (
                        <button className="secondary danger compact" onClick={() => void handleRemoveMember(member.user_id)}>Удалить</button>
                      ) : null}
                    </li>
                  )) : <li>Пока нет участников</li>}
                </ul>
              </div>
            </>
          ) : null}

          {currentChat?.type === 'direct' ? (
            <button className="secondary danger" onClick={() => void handleDeleteChat()}>Удалить чат</button>
          ) : (
            <button className="secondary danger" onClick={() => void handleLeave()}>Покинуть группу</button>
          )}
        </aside>
      </div>
    </div>
  );
}

export default App;
