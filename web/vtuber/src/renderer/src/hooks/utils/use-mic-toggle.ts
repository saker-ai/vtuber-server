import { useVAD } from '@/context/vad-context';
import { useAiState } from '@/context/ai-state-context';

export function useMicToggle() {
  const { startMic, stopMic, micOn } = useVAD();
  const { aiState, setAiState } = useAiState();

  const handleMicToggle = async (): Promise<void> => {
    if (micOn) {
      stopMic('user');
      if (aiState === 'listening') {
        setAiState('idle');
      }
    } else {
      await startMic('user');
    }
  };

  return {
    handleMicToggle,
    micOn,
  };
}
