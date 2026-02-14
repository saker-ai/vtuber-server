import { create } from 'zustand';

type WSConnState = 'CONNECTING' | 'OPEN' | 'CLOSING' | 'CLOSED';

interface AppStoreState {
  wsState: WSConnState;
  wsUrl: string;
  baseUrl: string;
  currentHistoryUid: string;
  setWSState: (state: WSConnState) => void;
  setWSEndpoints: (wsUrl: string, baseUrl: string) => void;
  setCurrentHistoryUid: (uid: string) => void;
}

export const useAppStore = create<AppStoreState>((set) => ({
  wsState: 'CLOSED',
  wsUrl: '',
  baseUrl: '',
  currentHistoryUid: '',
  setWSState: (state) => set({ wsState: state }),
  setWSEndpoints: (wsUrl, baseUrl) => set({ wsUrl, baseUrl }),
  setCurrentHistoryUid: (uid) => set({ currentHistoryUid: uid }),
}));

