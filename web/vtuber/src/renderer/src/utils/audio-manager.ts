/**
 * Global audio manager for handling audio playback and interruption
 * This ensures all components share the same audio reference
 */
type AudioHandle = HTMLAudioElement | { stop?: () => void };

class AudioManager {
  private currentAudio: AudioHandle | null = null;
  private currentModel: any | null = null;

  /**
   * Set the current playing audio
   */
  setCurrentAudio(audio: AudioHandle, model: any) {
    this.currentAudio = audio;
    this.currentModel = model;
  }

  /**
   * Stop current audio playback and lip sync
   */
  stopCurrentAudioAndLipSync() {
    if (this.currentAudio) {
      console.log('[AudioManager] Stopping current audio and lip sync');
      const audio = this.currentAudio;
      
      // Stop audio playback
      if ('pause' in audio && typeof audio.pause === 'function') {
        audio.pause();
        // @ts-expect-error - HTMLAudioElement fields
        audio.src = '';
        // @ts-expect-error - HTMLAudioElement fields
        audio.load?.();
      } else if ('stop' in audio && typeof audio.stop === 'function') {
        audio.stop();
      }

      // Stop Live2D lip sync
      const model = this.currentModel;
      if (model && model._wavFileHandler) {
        try {
          // Release PCM data to stop lip sync calculation in update()
          model._wavFileHandler.releasePcmData();
          console.log('[AudioManager] Called _wavFileHandler.releasePcmData()');

          // Additional reset of state variables as fallback
          model._wavFileHandler._lastRms = 0.0;
          model._wavFileHandler._sampleOffset = 0;
          model._wavFileHandler._userTimeSeconds = 0.0;
          console.log('[AudioManager] Also reset _lastRms, _sampleOffset, _userTimeSeconds as fallback');
        } catch (e) {
          console.error('[AudioManager] Error stopping/resetting wavFileHandler:', e);
        }
      } else if (model) {
        console.warn('[AudioManager] Current model does not have _wavFileHandler to stop/reset.');
      } else {
        console.log('[AudioManager] No associated model found to stop lip sync.');
      }

      // Clear references
      this.currentAudio = null;
      this.currentModel = null;
    } else {
      console.log('[AudioManager] No current audio playing to stop.');
    }
  }

  /**
   * Clear the current audio reference (called when audio ends naturally)
   */
  clearCurrentAudio(audio: AudioHandle) {
    if (this.currentAudio === audio) {
      this.currentAudio = null;
      this.currentModel = null;
    }
  }

  /**
   * Check if there's currently playing audio
   */
  hasCurrentAudio(): boolean {
    return this.currentAudio !== null;
  }
}

// Export singleton instance
export const audioManager = new AudioManager();
