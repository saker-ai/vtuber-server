class PcmPlayer {
  private context: AudioContext;
  private scheduledTime: number;
  private activeSources: Set<AudioBufferSourceNode>;
  private unlockListenersAttached: boolean;
  private readonly tryUnlockHandler: () => void;
  private readonly latencySeconds = 0.02;
  private readonly fadeMs = 5;

  constructor() {
    const AudioContextCtor = window.AudioContext || (window as any).webkitAudioContext;
    this.context = new AudioContextCtor();
    this.scheduledTime = this.context.currentTime;
    this.activeSources = new Set();
    this.unlockListenersAttached = false;
    this.tryUnlockHandler = () => {
      void this.resumeContext(true);
    };
    this.attachUnlockListeners();
  }

  async enqueue(
    base64: string,
    sampleRate: number,
    channels: number,
    onEnded?: () => void,
  ): Promise<{ pcm: Int16Array; started: boolean }> {
    const pcm = this.decodeBase64ToInt16(base64);
    const ready = await this.ensureRunning();
    if (!ready) {
      return { pcm, started: false };
    }

    const buffer = this.toAudioBuffer(pcm, sampleRate, channels);
    const source = this.context.createBufferSource();
    source.buffer = buffer;
    source.connect(this.context.destination);

    const startAt = Math.max(this.scheduledTime, this.context.currentTime + this.latencySeconds);
    source.start(startAt);
    this.scheduledTime = startAt + buffer.duration;

    this.activeSources.add(source);
    source.onended = () => {
      this.activeSources.delete(source);
      onEnded?.();
    };

    return { pcm, started: true };
  }

  stopAll() {
    this.activeSources.forEach((source) => {
      try {
        source.stop();
      } catch (error) {
        console.warn('[PcmPlayer] Failed to stop source:', error);
      }
    });
    this.activeSources.clear();
    this.scheduledTime = this.context.currentTime;
  }

  makeWavBlob(pcm: Int16Array, sampleRate: number, channels: number): Blob {
    const bytesPerSample = 2;
    const blockAlign = channels * bytesPerSample;
    const byteRate = sampleRate * blockAlign;
    const dataSize = pcm.length * bytesPerSample;
    const buffer = new ArrayBuffer(44 + dataSize);
    const view = new DataView(buffer);

    const writeString = (offset: number, value: string) => {
      for (let i = 0; i < value.length; i += 1) {
        view.setUint8(offset + i, value.charCodeAt(i));
      }
    };

    writeString(0, 'RIFF');
    view.setUint32(4, 36 + dataSize, true);
    writeString(8, 'WAVE');
    writeString(12, 'fmt ');
    view.setUint32(16, 16, true);
    view.setUint16(20, 1, true);
    view.setUint16(22, channels, true);
    view.setUint32(24, sampleRate, true);
    view.setUint32(28, byteRate, true);
    view.setUint16(32, blockAlign, true);
    view.setUint16(34, 16, true);
    writeString(36, 'data');
    view.setUint32(40, dataSize, true);

    let offset = 44;
    for (let i = 0; i < pcm.length; i += 1) {
      view.setInt16(offset, pcm[i], true);
      offset += 2;
    }

    return new Blob([view], { type: 'audio/wav' });
  }

  private async ensureRunning(): Promise<boolean> {
    return this.resumeContext(false);
  }

  private async resumeContext(fromUserGesture: boolean): Promise<boolean> {
    if ((this.context.state as AudioContextState) === 'running') {
      this.detachUnlockListeners();
      return true;
    }
    try {
      await this.context.resume();
    } catch (error) {
      if (fromUserGesture) {
        console.warn('[PcmPlayer] AudioContext resume failed after user gesture:', error);
      } else {
        console.warn('[PcmPlayer] AudioContext resume failed:', error);
      }
    }

    if (this.context.state === 'running') {
      this.detachUnlockListeners();
      return true;
    }

    if (!fromUserGesture) {
      this.attachUnlockListeners();
    }
    return false;
  }

  private attachUnlockListeners() {
    if (this.unlockListenersAttached) {
      return;
    }
    this.unlockListenersAttached = true;
    window.addEventListener('pointerdown', this.tryUnlockHandler, { passive: true });
    window.addEventListener('keydown', this.tryUnlockHandler);
    window.addEventListener('touchstart', this.tryUnlockHandler, { passive: true });
  }

  private detachUnlockListeners() {
    if (!this.unlockListenersAttached) {
      return;
    }
    this.unlockListenersAttached = false;
    window.removeEventListener('pointerdown', this.tryUnlockHandler);
    window.removeEventListener('keydown', this.tryUnlockHandler);
    window.removeEventListener('touchstart', this.tryUnlockHandler);
  }

  private decodeBase64ToInt16(base64: string): Int16Array {
    const binary = atob(base64);
    const length = binary.length;
    const buffer = new ArrayBuffer(length);
    const view = new Uint8Array(buffer);
    for (let i = 0; i < length; i += 1) {
      view[i] = binary.charCodeAt(i);
    }
    return new Int16Array(buffer);
  }

  private toAudioBuffer(pcm: Int16Array, sampleRate: number, channels: number): AudioBuffer {
    const frames = Math.floor(pcm.length / channels);
    const buffer = this.context.createBuffer(channels, frames, sampleRate);
    const fadeSamples = Math.min(
      Math.floor(frames / 2),
      Math.max(1, Math.floor((sampleRate * this.fadeMs) / 1000)),
    );
    for (let channel = 0; channel < channels; channel += 1) {
      const data = buffer.getChannelData(channel);
      let idx = channel;
      for (let i = 0; i < frames; i += 1) {
        data[i] = pcm[idx] / 32768;
        idx += channels;
      }
      if (fadeSamples > 0) {
        for (let i = 0; i < fadeSamples; i += 1) {
          data[i] *= i / fadeSamples;
        }
        for (let i = 0; i < fadeSamples; i += 1) {
          const at = frames - 1 - i;
          data[at] *= i / fadeSamples;
        }
      }
    }
    return buffer;
  }
}

export const pcmPlayer = new PcmPlayer();
