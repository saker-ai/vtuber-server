/* eslint-disable no-sparse-arrays */
/* eslint-disable react-hooks/exhaustive-deps */
// eslint-disable-next-line object-curly-newline
import { useEffect, useState, useCallback, useMemo, useRef } from 'react';
import { useTranslation } from 'react-i18next';
import { wsService, MessageEvent } from '@/services/websocket-service';
import {
  WebSocketContext, HistoryInfo, defaultWsUrl, defaultBaseUrl,
} from '@/context/websocket-context';
import { ModelInfo, useLive2DConfig } from '@/context/live2d-config-context';
import { useSubtitle } from '@/context/subtitle-context';
import { audioTaskQueue } from '@/utils/task-queue';
import { useAudioTask } from '@/components/canvas/live2d';
import { useBgUrl } from '@/context/bgurl-context';
import { useConfig } from '@/context/character-config-context';
import { useChatHistory } from '@/context/chat-history-context';
import { toaster } from '@/components/ui/toaster';
import { useVAD } from '@/context/vad-context';
import { AiState, useAiState } from "@/context/ai-state-context";
import { useLocalStorage } from '@/hooks/utils/use-local-storage';
import { useGroup } from '@/context/group-context';
import { useInterrupt } from '@/hooks/utils/use-interrupt';
import { useBrowser } from '@/context/browser-context';
import { useMediaCapture } from '@/hooks/utils/use-media-capture';
import { useAppStore } from '@/store/app-store';

const isAudioDebugEnabled = () => {
  if (import.meta.env.VITE_DEBUG_AUDIO === 'true') return true;
  try {
    return localStorage.getItem('debugAudio') === '1';
  } catch {
    return false;
  }
};

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

const shouldRewriteWsScheme = (baseUrl: string, wsUrl: string): boolean => {
  try {
    const base = new URL(baseUrl);
    const ws = new URL(wsUrl);
    return base.host === ws.host;
  } catch {
    return true;
  }
};

function WebSocketHandler({ children }: { children: React.ReactNode }) {
  const { t } = useTranslation();
  const [wsState, setWsState] = useState<string>('CLOSED');
  const [wsUrl, setWsUrl] = useLocalStorage<string>('wsUrl', defaultWsUrl);
  const [baseUrl, setBaseUrl] = useLocalStorage<string>('baseUrl', defaultBaseUrl);
  const { aiState, setAiState, setBackendSynthComplete } = useAiState();
  const { setModelInfo } = useLive2DConfig();
  const { setSubtitleText } = useSubtitle();
  const {
    clearResponse,
    setForceNewMessage,
    appendHumanMessage,
    appendOrUpdateToolCallMessage,
    upsertAIMessage,
  } = useChatHistory();
  const { addAudioTask } = useAudioTask();
  const bgUrlContext = useBgUrl();
  const { confUid, setConfName, setConfUid, setConfigFiles } = useConfig();
  const [pendingModelInfo, setPendingModelInfo] = useState<ModelInfo | undefined>(undefined);
  const { setSelfUid, setGroupMembers, setIsOwner } = useGroup();
  const {
    startMic,
    stopMic,
    autoStartMicOnConvEnd,
    voiceInterruptEnabled,
    continuousStreamingEnabled,
  } = useVAD();
  const autoStartMicOnConvEndRef = useRef(autoStartMicOnConvEnd);
  const { interrupt } = useInterrupt();
  const { setBrowserViewData } = useBrowser();
  const { captureCamera, captureScreen } = useMediaCapture();
  const normalizedBaseUrl = useMemo(() => {
    if (!baseUrl || baseUrl.startsWith('://')) {
      return defaultBaseUrl;
    }
    return baseUrl;
  }, [baseUrl]);
  const normalizedWsUrl = useMemo(() => {
    if (!wsUrl || wsUrl.startsWith('ws:///')) {
      const derived = deriveWsUrlFromBase(normalizedBaseUrl);
      return derived || defaultWsUrl;
    }
    if (
      normalizedBaseUrl.startsWith('https://') &&
      wsUrl.startsWith('ws://') &&
      shouldRewriteWsScheme(normalizedBaseUrl, wsUrl)
    ) {
      return wsUrl.replace(/^ws:\/\//, 'wss://');
    }
    if (
      normalizedBaseUrl.startsWith('http://') &&
      wsUrl.startsWith('wss://') &&
      shouldRewriteWsScheme(normalizedBaseUrl, wsUrl)
    ) {
      return wsUrl.replace(/^wss:\/\//, 'ws://');
    }
    return wsUrl;
  }, [wsUrl, normalizedBaseUrl]);

  useEffect(() => {
    if (baseUrl !== normalizedBaseUrl) {
      setBaseUrl(normalizedBaseUrl);
    }
  }, [baseUrl, normalizedBaseUrl, setBaseUrl]);

  useEffect(() => {
    if (wsUrl !== normalizedWsUrl) {
      setWsUrl(normalizedWsUrl);
    }
  }, [wsUrl, normalizedWsUrl, setWsUrl]);

  useEffect(() => {
    autoStartMicOnConvEndRef.current = autoStartMicOnConvEnd;
  }, [autoStartMicOnConvEnd]);

  useEffect(() => {
    if (pendingModelInfo && confUid) {
      setModelInfo(pendingModelInfo);
      setPendingModelInfo(undefined);
    }
  }, [pendingModelInfo, setModelInfo, confUid]);

  const {
    setCurrentHistoryUid, setMessages, setHistoryList,
  } = useChatHistory();
  const setStoreWSState = useAppStore((state) => state.setWSState);
  const setStoreWSEndpoints = useAppStore((state) => state.setWSEndpoints);
  const setStoreHistoryUid = useAppStore((state) => state.setCurrentHistoryUid);

  const handleControlMessage = useCallback((controlText: string) => {
    switch (controlText) {
      case 'start-mic':
        console.log('Starting microphone...');
        startMic();
        break;
      case 'stop-mic':
        console.log('Stopping microphone...');
        stopMic();
        break;
      case 'conversation-chain-start':
        setAiState('thinking-speaking');
        audioTaskQueue.clearQueue();
        clearResponse();
        break;
      case 'conversation-chain-end':
        audioTaskQueue.addTask(() => new Promise<void>((resolve) => {
          setAiState((currentState: AiState) => {
            if (currentState === 'thinking-speaking') {
              // Auto start mic if enabled
              if (autoStartMicOnConvEndRef.current) {
                startMic();
              }
              return 'idle';
            }
            return currentState;
          });
          resolve();
        }));
        break;
      default:
        console.warn('Unknown control command:', controlText);
    }
  }, [setAiState, clearResponse, setForceNewMessage, startMic, stopMic]);

  type HandlerMap = Record<string, (message: MessageEvent) => void>;
  const messageHandlers = useMemo<HandlerMap>(() => ({
    control: (message) => {
      if (message.text) {
        handleControlMessage(message.text);
      }
    },
    'set-model-and-conf': (message) => {
      setAiState('loading');
      if (message.conf_name) setConfName(message.conf_name);
      if (message.conf_uid) {
        setConfUid(message.conf_uid);
        console.log('confUid', message.conf_uid);
      }
      if (message.client_uid) setSelfUid(message.client_uid);
      setPendingModelInfo(message.model_info);
      if (message.model_info && !message.model_info.url.startsWith("http")) {
        message.model_info.url = baseUrl + message.model_info.url;
      }
      setAiState('idle');
    },
    'full-text': (message) => {
      if (!message.text) return;
      setSubtitleText(message.text);
      if (
        message.text !== 'Thinking...'
        && message.text !== 'Connection established'
        && message.text !== 'AI wants to speak something...'
      ) {
        upsertAIMessage(message.text);
      }
    },
    'config-files': (message) => {
      if (message.configs) setConfigFiles(message.configs);
    },
    'config-switched': () => {
      setAiState('idle');
      setSubtitleText(t('notification.characterLoaded'));
      toaster.create({ title: t('notification.characterSwitched'), type: 'success', duration: 2000 });
      wsService.sendMessage({ type: 'fetch-history-list' });
      wsService.sendMessage({ type: 'create-new-history' });
    },
    'background-files': (message) => {
      if (message.files) bgUrlContext?.setBackgroundFiles(message.files);
    },
    audio: (message) => {
      if (aiState === 'interrupted' || aiState === 'listening') {
        console.log('Audio playback intercepted. Sentence:', message.display_text?.text);
        return;
      }
      if (!voiceInterruptEnabled && !continuousStreamingEnabled) {
        if (isAudioDebugEnabled()) console.info('[mic] auto stop (tts playing)');
        stopMic('system');
      }
      if (isAudioDebugEnabled()) {
        console.info('[audio] tts chunk', {
          bytes: (message.audio_pcm || '').length,
          sampleRate: message.audio_sample_rate || 0,
          channels: message.audio_channels || 0,
          sliceLength: message.slice_length || 0,
        });
      }
      addAudioTask({
        audioBase64: message.audio || '',
        audioPcmBase64: message.audio_pcm || '',
        audioFormat: message.audio_format || '',
        audioSampleRate: message.audio_sample_rate || 0,
        audioChannels: message.audio_channels || 0,
        volumes: message.volumes || [],
        sliceLength: message.slice_length || 0,
        displayText: message.display_text || null,
        expressions: message.actions?.expressions || null,
        forwarded: message.forwarded || false,
      });
    },
    'history-data': (message) => {
      if (message.messages) setMessages(message.messages);
      toaster.create({ title: t('notification.historyLoaded'), type: 'success', duration: 2000 });
    },
    'new-history-created': (message) => {
      setAiState('idle');
      setSubtitleText(t('notification.newConversation'));
      if (!message.history_uid) return;
      setCurrentHistoryUid(message.history_uid);
      setStoreHistoryUid(message.history_uid);
      setMessages([]);
      const newHistory: HistoryInfo = {
        uid: message.history_uid,
        latest_message: null,
        timestamp: new Date().toISOString(),
      };
      setHistoryList((prev: HistoryInfo[]) => [newHistory, ...prev]);
      toaster.create({ title: t('notification.newChatHistory'), type: 'success', duration: 2000 });
    },
    'history-deleted': (message) => {
      toaster.create({
        title: message.success ? t('notification.historyDeleteSuccess') : t('notification.historyDeleteFail'),
        type: message.success ? 'success' : 'error',
        duration: 2000,
      });
    },
    'history-list': (message) => {
      if (!message.histories) return;
      setHistoryList(message.histories);
      if (message.histories.length > 0) {
        setCurrentHistoryUid(message.histories[0].uid);
        setStoreHistoryUid(message.histories[0].uid);
      }
    },
    'user-input-transcription': (message) => {
      if (message.text) appendHumanMessage(message.text);
    },
    error: (message) => {
      toaster.create({ title: message.message, type: 'error', duration: 2000 });
    },
    'group-update': (message) => {
      if (message.members) setGroupMembers(message.members);
      if (message.is_owner !== undefined) setIsOwner(message.is_owner);
    },
    'group-operation-result': (message) => {
      toaster.create({ title: message.message, type: message.success ? 'success' : 'error', duration: 2000 });
    },
    'backend-synth-complete': () => {
      setBackendSynthComplete(true);
    },
    'conversation-chain-end': () => {
      if (!audioTaskQueue.hasTask()) {
        setAiState((currentState: AiState) => (currentState === 'thinking-speaking' ? 'idle' : currentState));
      }
    },
    'force-new-message': () => {
      setForceNewMessage(true);
      if (!voiceInterruptEnabled && !continuousStreamingEnabled) {
        if (isAudioDebugEnabled()) console.info('[mic] auto start (tts finished)');
        startMic('auto');
      }
    },
    'interrupt-signal': () => {
      interrupt(false);
    },
    'tool_call_status': (message) => {
      if (!(message.tool_id && message.tool_name && message.status)) {
        console.warn('Received incomplete tool_call_status message:', message);
        return;
      }
      if (message.browser_view) setBrowserViewData(message.browser_view);
      appendOrUpdateToolCallMessage({
        id: message.tool_id,
        type: 'tool_call_status',
        role: 'ai',
        tool_id: message.tool_id,
        tool_name: message.tool_name,
        name: message.name,
        status: message.status as ('running' | 'completed' | 'error'),
        content: message.content || '',
        timestamp: message.timestamp || new Date().toISOString(),
      });
    },
    'mcp-capture-request': (message) => {
      void (async () => {
        const requestId = message.request_id || '';
        const source = message.source === 'screen' ? 'screen' : 'camera';
        const capture = source === 'screen' ? await captureScreen() : await captureCamera();
        if (!capture) {
          wsService.sendMessage({
            type: 'mcp-capture-response',
            request_id: requestId,
            success: false,
            message: `No ${source} stream available`,
          });
          return;
        }
        wsService.sendMessage({
          type: 'mcp-capture-response',
          request_id: requestId,
          success: true,
          image: capture.data,
          mime_type: capture.mime_type,
        });
      })();
    },
  }), [aiState, addAudioTask, appendHumanMessage, appendOrUpdateToolCallMessage, baseUrl, bgUrlContext, captureCamera, captureScreen, continuousStreamingEnabled, handleControlMessage, interrupt, setAiState, setBackendSynthComplete, setBrowserViewData, setConfName, setConfUid, setConfigFiles, setCurrentHistoryUid, setForceNewMessage, setGroupMembers, setHistoryList, setIsOwner, setMessages, setSelfUid, setStoreHistoryUid, setSubtitleText, startMic, stopMic, t, upsertAIMessage, voiceInterruptEnabled]);

  const handleWebSocketMessage = useCallback((message: MessageEvent) => {
    console.log('Received message from server:', message);
    const eventType = message.type || '';
    const handler = messageHandlers[eventType];
    if (handler) {
      handler(message);
      return;
    }
    console.warn('Unknown message type:', message.type);
  }, [messageHandlers]);

  useEffect(() => {
    wsService.connect(normalizedWsUrl);
  }, [normalizedWsUrl]);

  useEffect(() => {
    setStoreWSEndpoints(normalizedWsUrl, normalizedBaseUrl);
  }, [normalizedBaseUrl, normalizedWsUrl, setStoreWSEndpoints]);

  useEffect(() => {
    const handleBeforeUnload = () => {
      // Stop reconnect loop and close ws so backend can release XiaoZhi session promptly.
      wsService.setReconnectEnabled(false);
      wsService.disconnect();
    };
    window.addEventListener('beforeunload', handleBeforeUnload);
    window.addEventListener('pagehide', handleBeforeUnload);
    return () => {
      window.removeEventListener('beforeunload', handleBeforeUnload);
      window.removeEventListener('pagehide', handleBeforeUnload);
      wsService.setReconnectEnabled(true);
    };
  }, []);

  useEffect(() => {
    const stateSubscription = wsService.onStateChange(setWsState);
    const messageSubscription = wsService.onMessage(handleWebSocketMessage);
    return () => {
      stateSubscription.unsubscribe();
      messageSubscription.unsubscribe();
    };
  }, [normalizedWsUrl, handleWebSocketMessage]);

  useEffect(() => {
    const state = wsState as ('CONNECTING' | 'OPEN' | 'CLOSING' | 'CLOSED');
    setStoreWSState(state);
  }, [setStoreWSState, wsState]);

  useEffect(() => {
    if (wsState !== 'OPEN') {
      return;
    }
    const listenMode = voiceInterruptEnabled ? 'realtime' : 'auto';
    wsService.sendMessage({ type: 'set-listen-mode', listen_mode: listenMode });
  }, [wsState, voiceInterruptEnabled]);

  const webSocketContextValue = useMemo(() => ({
    sendMessage: wsService.sendMessage.bind(wsService),
    wsState,
    reconnect: () => wsService.connect(normalizedWsUrl),
    wsUrl: normalizedWsUrl,
    setWsUrl,
    baseUrl: normalizedBaseUrl,
    setBaseUrl,
  }), [wsState, normalizedWsUrl, normalizedBaseUrl]);

  return (
    <WebSocketContext.Provider value={webSocketContextValue}>
      {children}
    </WebSocketContext.Provider>
  );
}

export default WebSocketHandler;
