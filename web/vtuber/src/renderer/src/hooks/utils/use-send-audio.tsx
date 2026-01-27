import { useCallback } from "react";
import { useWebSocket } from "@/context/websocket-context";
import { useMediaCapture } from "@/hooks/utils/use-media-capture";

export function useSendAudio() {
  const { sendMessage } = useWebSocket();
  const { captureAllMedia } = useMediaCapture();

  const sendAudioPartition = useCallback(
    async (audio: Float32Array) => {
      const chunkSize = 4096;
      const endDelayMs = 150;

      // Send the audio data in chunks
      for (let index = 0; index < audio.length; index += chunkSize) {
        const endIndex = Math.min(index + chunkSize, audio.length);
        const chunk = audio.slice(index, endIndex);
        sendMessage({
          type: "mic-audio-data",
          audio: Array.from(chunk),
          // Only send images with first chunk
        });
      }

      // Allow in-flight frames to flush before signaling end
      await new Promise((resolve) => setTimeout(resolve, endDelayMs));

      // Send end signal after all chunks
      const images = await captureAllMedia();
      sendMessage({ type: "mic-audio-end", images });
    },
    [sendMessage, captureAllMedia],
  );

  return {
    sendAudioPartition,
  };
}
