/* eslint-disable no-use-before-define */
import {
  createContext, useContext, useRef, useCallback, useEffect, useReducer, useMemo,
} from 'react';
import { useTranslation } from 'react-i18next';
import { MicVAD } from '@ricky0123/vad-web';
import { useInterrupt } from '@/components/canvas/live2d';
import { audioTaskQueue } from '@/utils/task-queue';
import { useSendAudio } from '@/hooks/utils/use-send-audio';
import { SubtitleContext } from './subtitle-context';
import { AiStateContext, AiState } from './ai-state-context';
import { useLocalStorage } from '@/hooks/utils/use-local-storage';
import { toaster } from '@/components/ui/toaster';
import { audioManager } from '@/utils/audio-manager';

/**
 * VAD settings configuration interface
 * @interface VADSettings
 */
export interface VADSettings {
  /** Threshold for positive speech detection (0-100) */
  positiveSpeechThreshold: number;

  /** Threshold for negative speech detection (0-100) */
  negativeSpeechThreshold: number;

  /** Number of frames for speech redemption */
  redemptionFrames: number;
}

/**
 * VAD context state interface
 * @interface VADState
 */
interface VADState {
  /** Auto stop mic feature state */
  autoStopMic: boolean;

  /** Microphone active state */
  micOn: boolean;

  /** Set microphone state */
  setMicOn: (value: boolean) => void;

  /** Set Auto stop mic state */
  setAutoStopMic: (value: boolean) => void;

  /** Start microphone and VAD */
  startMic: (source?: 'auto' | 'user') => Promise<void>;

  /** Stop microphone and VAD */
  stopMic: (source?: 'system' | 'user') => void;

  /** Previous speech probability value */
  previousTriggeredProbability: number;

  /** Set previous speech probability */
  setPreviousTriggeredProbability: (value: number) => void;

  /** VAD settings configuration */
  settings: VADSettings;

  /** Update VAD settings */
  updateSettings: (newSettings: VADSettings) => void;

  /** Auto start microphone when AI starts speaking */
  autoStartMicOn: boolean;

  /** Set auto start microphone state */
  setAutoStartMicOn: (value: boolean) => void;

  /** Auto start microphone when conversation ends */
  autoStartMicOnConvEnd: boolean;

  /** Set auto start microphone when conversation ends state */
  setAutoStartMicOnConvEnd: (value: boolean) => void;

  /** Enable voice interruption when AI is speaking */
  voiceInterruptEnabled: boolean;

  /** Set voice interruption state */
  setVoiceInterruptEnabled: (value: boolean) => void;

  /** Auto start microphone on page load */
  autoStartOnLoad: boolean;

  /** Set auto start microphone on page load */
  setAutoStartOnLoad: (value: boolean) => void;
}

/**
 * Default values and constants
 */
const DEFAULT_VAD_SETTINGS: VADSettings = {
  positiveSpeechThreshold: 50,
  negativeSpeechThreshold: 35,
  redemptionFrames: 35,
};

const getDefaultBoolean = (value: string | undefined, fallback: boolean): boolean => {
  if (value === 'true' || value === '1') {
    return true;
  }
  if (value === 'false' || value === '0') {
    return false;
  }
  return fallback;
};

const getDefaultVADState = () => {
  const isElectron = typeof window !== 'undefined' && (window as any).api !== undefined;
  const isWeb = !isElectron;
  const defaultModeEnv = import.meta.env.VITE_DEFAULT_MODE;
  const isPetDefault = isElectron && defaultModeEnv === 'pet';
  const defaultMicOn = getDefaultBoolean(
    import.meta.env.VITE_DEFAULT_MIC_ON,
    isPetDefault || isWeb,
  );
  const defaultAutoStartMicOnConvEnd = getDefaultBoolean(
    import.meta.env.VITE_DEFAULT_AUTO_START_MIC_ON_CONV_END,
    true,
  );
  return {
    micOn: defaultMicOn,
    autoStopMic: false,
    autoStartMicOn: false,
    autoStartMicOnConvEnd: defaultAutoStartMicOnConvEnd,
    voiceInterruptEnabled: false,
  };
};

const DEFAULT_VAD_STATE = getDefaultVADState();

/**
 * Create the VAD context
 */
export const VADContext = createContext<VADState | null>(null);

/**
 * VAD Provider Component
 * Manages voice activity detection and microphone state
 *
 * @param {Object} props - Provider props
 * @param {React.ReactNode} props.children - Child components
 */
export function VADProvider({ children }: { children: React.ReactNode }) {
  const { t } = useTranslation();
  // Refs for VAD instance and state
  const vadRef = useRef<MicVAD | null>(null);
  const previousTriggeredProbabilityRef = useRef(0);
  const previousAiStateRef = useRef<AiState>('idle');

  // Persistent state management
  const [micOn, setMicOn] = useLocalStorage('micOn', DEFAULT_VAD_STATE.micOn);
  const autoStopMicRef = useRef(true);
  const [autoStopMic, setAutoStopMicState] = useLocalStorage(
    'autoStopMic',
    DEFAULT_VAD_STATE.autoStopMic,
  );
  const [settings, setSettings] = useLocalStorage<VADSettings>(
    'vadSettings',
    DEFAULT_VAD_SETTINGS,
  );
  const [autoStartMicOn, setAutoStartMicOnState] = useLocalStorage(
    'autoStartMicOn',
    DEFAULT_VAD_STATE.autoStartMicOn,
  );
  const autoStartMicRef = useRef(false);
  const [autoStartMicOnConvEnd, setAutoStartMicOnConvEndState] = useLocalStorage(
    'autoStartMicOnConvEnd',
    DEFAULT_VAD_STATE.autoStartMicOnConvEnd,
  );
  const autoStartMicOnConvEndRef = useRef(false);
  const [voiceInterruptEnabled, setVoiceInterruptEnabledState] = useLocalStorage(
    'voiceInterruptEnabled',
    DEFAULT_VAD_STATE.voiceInterruptEnabled,
  );
  const voiceInterruptEnabledRef = useRef(false);
  const [autoStartOnLoad, setAutoStartOnLoadState] = useLocalStorage(
    'autoStartOnLoad',
    true,
  );
  const autoStartOnLoadRef = useRef(true);

  // Force update mechanism for ref updates
  const [, forceUpdate] = useReducer((x) => x + 1, 0);

  // External hooks and contexts
  const { interrupt } = useInterrupt();
  const { sendAudioPartition } = useSendAudio();
  const { setSubtitleText } = useContext(SubtitleContext)!;
  const { aiState, setAiState } = useContext(AiStateContext)!;

  // Refs for callback stability
  const interruptRef = useRef(interrupt);
  const sendAudioPartitionRef = useRef(sendAudioPartition);
  const aiStateRef = useRef<AiState>(aiState);
  const setSubtitleTextRef = useRef(setSubtitleText);
  const setAiStateRef = useRef(setAiState);

  const isProcessingRef = useRef(false);
  const autoStartAttemptedRef = useRef(false);
  const startInFlightRef = useRef(false);
  const gestureStartHandlerRef = useRef<(() => void) | null>(null);

  // Update refs when dependencies change
  useEffect(() => {
    aiStateRef.current = aiState;
  }, [aiState]);

  useEffect(() => {
    interruptRef.current = interrupt;
  }, [interrupt]);

  useEffect(() => {
    sendAudioPartitionRef.current = sendAudioPartition;
  }, [sendAudioPartition]);

  useEffect(() => {
    setSubtitleTextRef.current = setSubtitleText;
  }, [setSubtitleText]);

  useEffect(() => {
    setAiStateRef.current = setAiState;
  }, [setAiState]);

  useEffect(() => {
    autoStopMicRef.current = autoStopMic;
  }, [autoStopMic]);

  useEffect(() => {
    autoStartMicRef.current = autoStartMicOn;
  }, [autoStartMicOn]);

  useEffect(() => {
    autoStartMicOnConvEndRef.current = autoStartMicOnConvEnd;
  }, [autoStartMicOnConvEnd]);

  useEffect(() => {
    voiceInterruptEnabledRef.current = voiceInterruptEnabled;
  }, [voiceInterruptEnabled]);

  useEffect(() => {
    autoStartOnLoadRef.current = autoStartOnLoad;
  }, [autoStartOnLoad]);

  /**
   * Update previous triggered probability and force re-render
   */
  const setPreviousTriggeredProbability = useCallback((value: number) => {
    previousTriggeredProbabilityRef.current = value;
    forceUpdate();
  }, []);

  /**
   * Handle speech start event (initial detection)
   */
  const handleSpeechStart = useCallback(() => {
    console.log('Speech started - saving current state');
    if (
      aiStateRef.current === 'thinking-speaking'
      && !voiceInterruptEnabledRef.current
      && audioManager.hasCurrentAudio()
    ) {
      console.log('Voice interruption disabled; ignore speech during AI response');
      isProcessingRef.current = false;
      return;
    }
    // Save current AI state but DON'T change to listening yet
    previousAiStateRef.current = aiStateRef.current;
    isProcessingRef.current = true;
    // Don't change state here - wait for onSpeechRealStart
  }, []);

  /**
   * Handle real speech start event (confirmed speech)
   */
  const handleSpeechRealStart = useCallback(() => {
    if (!isProcessingRef.current) {
      return;
    }
    console.log('Real speech confirmed - checking if need to interrupt');
    // Check if we need to interrupt based on the PREVIOUS state (before speech started)
    if (
      previousAiStateRef.current === 'thinking-speaking'
      && voiceInterruptEnabledRef.current
      && audioManager.hasCurrentAudio()
    ) {
      console.log('Interrupting AI speech due to user speaking');
      interruptRef.current();
    }
    // Now change to listening state
    setAiStateRef.current('listening');
  }, []);

  /**
   * Handle frame processing event
   */
  const handleFrameProcessed = useCallback((probs: { isSpeech: number }) => {
    if (probs.isSpeech > previousTriggeredProbabilityRef.current) {
      setPreviousTriggeredProbability(probs.isSpeech);
    }
  }, []);

  /**
   * Handle speech end event
   */
  const handleSpeechEnd = useCallback((audio: Float32Array) => {
    if (!isProcessingRef.current) return;
    console.log('Speech ended');
    audioTaskQueue.clearQueue();

    if (autoStopMicRef.current) {
      stopMic();
    } else {
      console.log('Auto stop mic is OFF, keeping mic active');
    }

    setPreviousTriggeredProbability(0);
    sendAudioPartitionRef.current(audio);
    isProcessingRef.current = false;
    setAiStateRef.current("thinking-speaking");
  }, []);

  /**
   * Handle VAD misfire event
   */
  const handleVADMisfire = useCallback(() => {
    if (!isProcessingRef.current) return;
    console.log('VAD misfire detected');
    setPreviousTriggeredProbability(0);
    isProcessingRef.current = false;

    // Restore previous AI state and show helpful misfire message
    setAiStateRef.current(previousAiStateRef.current);
    setSubtitleTextRef.current(t('error.vadMisfire'));
  }, [t]);

  /**
   * Update VAD settings and restart if active
   */
  const updateSettings = useCallback((newSettings: VADSettings) => {
    setSettings(newSettings);
    if (vadRef.current) {
      stopMic();
      setTimeout(() => {
        startMic();
      }, 100);
    }
  }, []);

  /**
   * Initialize new VAD instance
   */
  const initVAD = async () => {
    const newVAD = await MicVAD.new({
      model: "v5",
      preSpeechPadFrames: 20,
      positiveSpeechThreshold: settings.positiveSpeechThreshold / 100,
      negativeSpeechThreshold: settings.negativeSpeechThreshold / 100,
      redemptionFrames: settings.redemptionFrames,
      baseAssetPath: './libs/',
      onnxWASMBasePath: './libs/',
      onSpeechStart: handleSpeechStart,
      onSpeechRealStart: handleSpeechRealStart,
      onFrameProcessed: handleFrameProcessed,
      onSpeechEnd: handleSpeechEnd,
      onVADMisfire: handleVADMisfire,
    });

    vadRef.current = newVAD;
    newVAD.start();
  };

  /**
   * Start microphone and VAD processing
   */
  const startMic = useCallback(async (source: 'auto' | 'user' = 'user') => {
    if (startInFlightRef.current) return;
    startInFlightRef.current = true;
    try {
      if (!vadRef.current) {
        console.log('Initializing VAD');
        await initVAD();
      } else {
        console.log('Starting VAD');
        vadRef.current.start();
      }
      setMicOn(true);
      if (source === 'user') {
        setAutoStartOnLoadState(true);
      }
      if (gestureStartHandlerRef.current) {
        window.removeEventListener('pointerdown', gestureStartHandlerRef.current);
        window.removeEventListener('keydown', gestureStartHandlerRef.current);
        gestureStartHandlerRef.current = null;
      }
    } catch (error) {
      console.error('Failed to start VAD:', error);
      const errorName = (error as { name?: string })?.name ?? '';
      const errorMessage = String(error);
      const isPermissionDenied = errorName === 'NotAllowedError'
        || errorName === 'SecurityError'
        || /NotAllowedError|Permission denied/i.test(errorMessage);
      if (source === 'auto' && isPermissionDenied) {
        toaster.create({
          title: t('error.micAutoStartBlocked'),
          type: 'warning',
          duration: 3000,
        });
      } else {
        setMicOn(false);
        toaster.create({
          title: `${t('error.failedStartVAD')}: ${error}`,
          type: 'error',
          duration: 2000,
        });
      }
      return;
    } finally {
      startInFlightRef.current = false;
    }
  }, [t]);

  // Attempt auto-start on initial load if mic is persisted as on.
  useEffect(() => {
    if (!autoStartAttemptedRef.current && (micOn || autoStartOnLoadRef.current)) {
      autoStartAttemptedRef.current = true;
      startMic('auto');
    }
  }, [micOn, startMic]);

  // Retry on first user interaction if mic is intended to be on but not started.
  useEffect(() => {
    if (!micOn || vadRef.current) return;
    const handleUserGesture = () => {
      startMic('user');
    };
    gestureStartHandlerRef.current = handleUserGesture;
    window.addEventListener('pointerdown', handleUserGesture, { once: true });
    window.addEventListener('keydown', handleUserGesture, { once: true });
    return () => {
      window.removeEventListener('pointerdown', handleUserGesture);
      window.removeEventListener('keydown', handleUserGesture);
      if (gestureStartHandlerRef.current === handleUserGesture) {
        gestureStartHandlerRef.current = null;
      }
    };
  }, [micOn, startMic]);

  /**
   * Stop microphone and VAD processing
   */
  const stopMic = useCallback((source: 'system' | 'user' = 'system') => {
    console.log('Stopping VAD');
    if (vadRef.current) {
      vadRef.current.pause();
      vadRef.current.destroy();
      vadRef.current = null;
      console.log('VAD stopped and destroyed successfully');
      setPreviousTriggeredProbability(0);
    } else {
      console.log('VAD instance not found');
    }
    setMicOn(false);
    if (source === 'user') {
      setAutoStartOnLoadState(false);
    }
    isProcessingRef.current = false;
  }, []);

  /**
   * Set Auto stop mic state
   */
  const setAutoStopMic = useCallback((value: boolean) => {
    autoStopMicRef.current = value;
    setAutoStopMicState(value);
    forceUpdate();
  }, []);

  const setAutoStartMicOn = useCallback((value: boolean) => {
    autoStartMicRef.current = value;
    setAutoStartMicOnState(value);
    forceUpdate();
  }, []);

  const setAutoStartMicOnConvEnd = useCallback((value: boolean) => {
    autoStartMicOnConvEndRef.current = value;
    setAutoStartMicOnConvEndState(value);
    forceUpdate();
  }, []);

  const setVoiceInterruptEnabled = useCallback((value: boolean) => {
    voiceInterruptEnabledRef.current = value;
    setVoiceInterruptEnabledState(value);
    forceUpdate();
  }, []);

  const setAutoStartOnLoad = useCallback((value: boolean) => {
    autoStartOnLoadRef.current = value;
    setAutoStartOnLoadState(value);
    forceUpdate();
  }, []);

  // Memoized context value
  const contextValue = useMemo(
    () => ({
      autoStopMic: autoStopMicRef.current,
      micOn,
      setMicOn,
      setAutoStopMic,
      startMic,
      stopMic,
      previousTriggeredProbability: previousTriggeredProbabilityRef.current,
      setPreviousTriggeredProbability,
      settings,
      updateSettings,
      autoStartMicOn: autoStartMicRef.current,
      setAutoStartMicOn,
      autoStartMicOnConvEnd: autoStartMicOnConvEndRef.current,
      setAutoStartMicOnConvEnd,
      voiceInterruptEnabled: voiceInterruptEnabledRef.current,
      setVoiceInterruptEnabled,
      autoStartOnLoad: autoStartOnLoadRef.current,
      setAutoStartOnLoad,
    }),
    [
      micOn,
      startMic,
      stopMic,
      settings,
      updateSettings,
      voiceInterruptEnabled,
      setVoiceInterruptEnabled,
      setAutoStartOnLoad,
    ],
  );

  return (
    <VADContext.Provider value={contextValue}>
      {children}
    </VADContext.Provider>
  );
}

/**
 * Custom hook to use the VAD context
 * @throws {Error} If used outside of VADProvider
 */
export function useVAD() {
  const context = useContext(VADContext);

  if (!context) {
    throw new Error('useVAD must be used within a VADProvider');
  }

  return context;
}
