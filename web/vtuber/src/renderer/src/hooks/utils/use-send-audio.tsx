import { useCallback } from 'react';
import { useWebSocket } from '@/context/websocket-context';
import { useMediaCapture } from '@/hooks/utils/use-media-capture';
import { float32ToPCM16, pcm16ToBase64 } from '@/utils/pcm-encoder';
import { wsService } from '@/services/websocket-service';

const isAudioDebugEnabled = () => {
  if (import.meta.env.VITE_DEBUG_AUDIO === 'true') return true;
  try {
    return localStorage.getItem('debugAudio') === '1';
  } catch {
    return false;
  }
};

export function useSendAudio() {
  const { sendMessage } = useWebSocket();
  const { captureAllMedia } = useMediaCapture();

  const sendMicAudioChunk = useCallback(
    (audio: Float32Array, audioSampleRate: number, audioChannels: number) => {
      if (!audio.length || wsService.getCurrentState() !== 'OPEN') {
        return;
      }
      const pcm = float32ToPCM16(audio);
      const base64 = pcm16ToBase64(pcm);
      sendMessage({
        type: 'mic-audio-data',
        audio_pcm: base64,
        audio_format: 'pcm16',
        audio_sample_rate: audioSampleRate,
        audio_channels: audioChannels,
      });
    },
    [sendMessage],
  );

  const sendMicAudioEnd = useCallback(
    async (includeImages = false) => {
      if (wsService.getCurrentState() !== 'OPEN') {
        return;
      }
      const captureTimeoutMs = 400;
      let images: unknown[] = [];
      if (includeImages) {
        try {
          images = await Promise.race([
            captureAllMedia(),
            new Promise<[]>(resolve => setTimeout(() => resolve([]), captureTimeoutMs)),
          ]);
        } catch (error) {
          console.warn('[audio] capture all media failed:', error);
        }
      }
      sendMessage({ type: 'mic-audio-end', images });
      if (isAudioDebugEnabled()) {
        console.info('[audio] send mic end', { images: images.length });
      }
    },
    [captureAllMedia, sendMessage],
  );

  const sendAudioPartition = useCallback(
    async (audio: Float32Array, audioSampleRate: number, audioChannels: number) => {
      const chunkSize = 4096;
      const endDelayMs = 80;
      const debug = isAudioDebugEnabled();
      if (debug) {
        console.info('[audio] send mic start', {
          frames: audio.length,
          sampleRate: audioSampleRate,
          channels: audioChannels,
        });
      }

      // Send the audio data in chunks
      for (let index = 0; index < audio.length; index += chunkSize) {
        const endIndex = Math.min(index + chunkSize, audio.length);
        const chunk = audio.slice(index, endIndex);
        sendMicAudioChunk(chunk, audioSampleRate, audioChannels);
      }

      // Allow in-flight frames to flush before signaling end
      await new Promise((resolve) => setTimeout(resolve, endDelayMs));

      await sendMicAudioEnd(true);
    },
    [sendMicAudioChunk, sendMicAudioEnd],
  );

  return {
    sendAudioPartition,
    sendMicAudioChunk,
    sendMicAudioEnd,
  };
}
