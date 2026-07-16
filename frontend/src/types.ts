export interface User {
  id: string;
  login: string;
  created_at: string;
}

export interface AuthResponse {
  user: User;
  access_token: string;
  refresh_token: string;
}

export interface Chat {
  id: string;
  type: 'direct' | 'group';
  title?: string | null;
  created_by?: string | null;
  last_message_at?: string | null;
  created_at: string;
}

export interface Message {
  id: string;
  chat_id: string;
  sender_id?: string | null;
  body: string;
  client_msg_id?: string | null;
  seq: number;
  created_at: string;
}

export interface MessageListResponse {
  items: Message[];
  limit: number;
  offset: number;
}

export interface ChatMember {
  chat_id: string;
  user_id: string;
  login?: string;
  role: 'member' | 'admin';
}

export interface SocketMessageFrame {
  type: 'message' | 'ack' | 'error';
  message?: Message;
  error?: string;
}
