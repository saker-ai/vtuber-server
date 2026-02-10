import { useCallback } from "react";
import { useWebSocket } from "@/context/websocket-context";
import { useMediaCapture } from "@/hooks/utils/use-media-capture";
import { float32ToPCM16, pcm16ToBase64 } from "@/utils/pcm-encoder";

const isAudioDebugEnabled = () => {
  if (import.meta.env.VITE_DEBUG_AUDIO === "true") return true;
  try {
    return localStorage.getItem("debugAudio") === "1";
  } catch {
    return false;
  }
};

export function useSendAudio() {
  const { sendMessage } = useWebSocket();
  const { captureAllMedia } = useMediaCapture();

  const sendAudioPartition = useCallback(
    async (audio: Float32Array, audioSampleRate: number, audioChannels: number) => {
      const chunkSize = 4096;
      const endDelayMs = 80;
      const captureTimeoutMs = 400;
      const debug = isAudioDebugEnabled();
      if (debug) {
        console.info("[audio] send mic start", {
          frames: audio.length,
          sampleRate: audioSampleRate,
          channels: audioChannels,
        });
      }

      // Send the audio data in chunks
      for (let index = 0; index < audio.length; index += chunkSize) {
        const endIndex = Math.min(index + chunkSize, audio.length);
        const chunk = audio.slice(index, endIndex);
        const pcm = float32ToPCM16(chunk);
        const base64 = pcm16ToBase64(pcm);
        sendMessage({
          type: "mic-audio-data",
          audio_pcm: base64,
          audio_format: "pcm16",
          audio_sample_rate: audioSampleRate,
          audio_channels: audioChannels,
          // Only send images with first chunk
        });
      }

      // Allow in-flight frames to flush before signaling end
      await new Promise((resolve) => setTimeout(resolve, endDelayMs));

      // Send end signal after all chunks
      let images = [];
      try {
        images = await Promise.race([
          captureAllMedia(),
          new Promise<[]>(resolve => setTimeout(() => resolve([]), captureTimeoutMs)),
        ]);
      } catch (error) {
        console.warn("[audio] capture all media failed:", error);
      }
      sendMessage({ type: "mic-audio-end", images });
      if (debug) {
        console.info("[audio] send mic end", { images: images.length });
      }
    },
    [sendMessage, captureAllMedia],
  );

  return {
    sendAudioPartition,
  };
}
