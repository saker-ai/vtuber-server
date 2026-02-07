export function float32ToPCM16(audio: Float32Array): Int16Array {
  const pcm = new Int16Array(audio.length);
  for (let i = 0; i < audio.length; i += 1) {
    let sample = audio[i];
    if (sample > 1) sample = 1;
    if (sample < -1) sample = -1;
    pcm[i] = sample < 0 ? sample * 0x8000 : sample * 0x7fff;
  }
  return pcm;
}

export function pcm16ToBase64(pcm: Int16Array): string {
  const bytes = new Uint8Array(pcm.buffer, pcm.byteOffset, pcm.byteLength);
  let binary = '';
  const chunkSize = 0x8000;
  for (let i = 0; i < bytes.length; i += chunkSize) {
    const chunk = bytes.subarray(i, i + chunkSize);
    binary += String.fromCharCode(...chunk);
  }
  return btoa(binary);
}
