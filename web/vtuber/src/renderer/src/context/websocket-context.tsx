/* eslint-disable react/jsx-no-constructed-context-values */
import React, { useContext, useCallback } from 'react';
import { wsService } from '@/services/websocket-service';
import { useLocalStorage } from '@/hooks/utils/use-local-storage';

const fallbackBaseUrl = 'https://127.0.0.1:12393';
const fallbackWsUrl = 'wss://127.0.0.1:12393/client-ws';

const getDefaultBaseUrl = () => {
  if (typeof window === 'undefined') {
    return fallbackBaseUrl;
  }
  const protocol = window.location.protocol;
  if (protocol === 'http:' || protocol === 'https:') {
    return window.location.origin;
  }
  return fallbackBaseUrl;
};

const getDefaultWsUrl = () => {
  if (typeof window === 'undefined') {
    return fallbackWsUrl;
  }
  const protocol = window.location.protocol;
  if (protocol === 'http:' || protocol === 'https:') {
    const wsProtocol = protocol === 'https:' ? 'wss' : 'ws';
    return `${wsProtocol}://${window.location.host}/client-ws`;
  }
  return fallbackWsUrl;
};

const DEFAULT_BASE_URL = getDefaultBaseUrl();
const DEFAULT_WS_URL = getDefaultWsUrl();
const deriveWsUrlFromBase = (baseUrl: string): string | null => {
  try {
    const url = new URL(baseUrl);
    if (url.protocol !== 'http:' && url.protocol !== 'https:') {
      return null;
    }
    const wsProtocol = url.protocol === 'https:' ? 'wss:' : 'ws:';
    return `${wsProtocol}//${url.host}/client-ws`;
  } catch {
    return null;
  }
};

export interface HistoryInfo {
  uid: string;
  latest_message: {
    role: 'human' | 'ai';
    timestamp: string;
    content: string;
  } | null;
  timestamp: string | null;
}

interface WebSocketContextProps {
  sendMessage: (message: object) => void;
  wsState: string;
  reconnect: () => void;
  wsUrl: string;
  setWsUrl: (url: string) => void;
  baseUrl: string;
  setBaseUrl: (url: string) => void;
}

export const WebSocketContext = React.createContext<WebSocketContextProps>({
  sendMessage: wsService.sendMessage.bind(wsService),
  wsState: 'CLOSED',
  reconnect: () => wsService.connect(DEFAULT_WS_URL),
  wsUrl: DEFAULT_WS_URL,
  setWsUrl: () => {},
  baseUrl: DEFAULT_BASE_URL,
  setBaseUrl: () => {},
});

export function useWebSocket() {
  const context = useContext(WebSocketContext);
  if (!context) {
    throw new Error('useWebSocket must be used within a WebSocketProvider');
  }
  return context;
}

export const defaultWsUrl = DEFAULT_WS_URL;
export const defaultBaseUrl = DEFAULT_BASE_URL;

export function WebSocketProvider({ children }: { children: React.ReactNode }) {
  const [wsUrl, setWsUrl] = useLocalStorage('wsUrl', DEFAULT_WS_URL);
  const [baseUrl, setBaseUrl] = useLocalStorage('baseUrl', DEFAULT_BASE_URL);
  const normalizedBaseUrl = useCallback(() => {
    if (!baseUrl || baseUrl.startsWith('://')) {
      return DEFAULT_BASE_URL;
    }
    return baseUrl;
  }, [baseUrl]);
  const normalizedWsUrl = useCallback(() => {
    if (!wsUrl || wsUrl.startsWith('ws:///')) {
      const derived = deriveWsUrlFromBase(normalizedBaseUrl());
      return derived || DEFAULT_WS_URL;
    }
    if (normalizedBaseUrl().startsWith('https://') && wsUrl.startsWith('ws://')) {
      return wsUrl.replace(/^ws:\/\//, 'wss://');
    }
    if (normalizedBaseUrl().startsWith('http://') && wsUrl.startsWith('wss://')) {
      return wsUrl.replace(/^wss:\/\//, 'ws://');
    }
    return wsUrl;
  }, [wsUrl, normalizedBaseUrl]);
  const handleSetWsUrl = useCallback((url: string) => {
    setWsUrl(url);
    wsService.connect(url);
  }, [setWsUrl]);

  const value = {
    sendMessage: wsService.sendMessage.bind(wsService),
    wsState: 'CLOSED',
    reconnect: () => wsService.connect(normalizedWsUrl()),
    wsUrl: normalizedWsUrl(),
    setWsUrl: handleSetWsUrl,
    baseUrl: normalizedBaseUrl(),
    setBaseUrl,
  };

  return (
    <WebSocketContext.Provider value={value}>
      {children}
    </WebSocketContext.Provider>
  );
}
