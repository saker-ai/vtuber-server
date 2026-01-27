import { SystemStyleObject } from '@chakra-ui/react';

export const inputSubtitleStyles = {
  container: {
    display: 'flex',
    alignItems: 'flex-end',
    justifyContent: 'center',
    maxW: 'fit-content',
    position: 'absolute' as const,
    bottom: '120px',
    left: '50%',
    transform: 'translateX(-50%)',
    zIndex: 1000,
    userSelect: 'none',
    willChange: 'transform',
    padding: 0,
  },

  box: {
    w: '400px',
    rounded: 'xl',
    overflow: 'hidden',
    boxShadow: 'lg',
    bg: 'var(--app-panel-strong)',
    backdropFilter: 'blur(12px)',
    css: { WebkitUserSelect: 'none' },
  },

  messageStack: {
    p: '3',
    gap: 1,
    alignItems: 'stretch',
    justify: 'flex-end',
  },

  messageText: {
    color: 'var(--app-text)',
    fontSize: 'sm',
    lineHeight: '1.5',
    transition: 'all 0.3s',
  },

  statusBox: {
    bg: 'var(--app-panel)',
    p: '3',
    borderTop: '1px',
    borderColor: 'var(--app-border)',
  },

  statusText: {
    fontSize: 'xs',
    color: 'var(--app-text-muted)',
    transition: 'all 0.3s',
  },

  iconButton: {
    size: 'xs',
    variant: 'ghost',
    color: 'var(--app-text-muted)',
    _hover: { bg: 'rgba(255, 255, 255, 0.12)' },
  },

  inputBox: {
    bg: 'var(--app-panel)',
    borderTop: '1px',
    borderColor: 'var(--app-border)',
  },

  input: {
    size: 'sm',
    bg: 'rgba(255, 255, 255, 0.06)',
    color: 'var(--app-text)',
    _placeholder: { color: 'var(--app-text-muted)' },
    borderColor: 'rgba(255, 255, 255, 0.22)',
    _focus: {
      borderColor: 'rgba(255, 255, 255, 0.35)',
      outline: 'none',
    },
    flex: '1',
  },

  sendButton: {
    p: '1.5',
    bg: 'rgba(255, 255, 255, 0.08)',
    rounded: 'lg',
    _hover: { bg: 'rgba(255, 255, 255, 0.14)' },
    transition: 'colors',
    color: 'var(--app-text)',
    size: 'sm',
  },

  draggableContainer: (isDragging: boolean): SystemStyleObject => ({
    cursor: isDragging ? 'grabbing' : 'grab',
    transition: isDragging ? 'none' : 'transform 0.1s ease',
    _active: { cursor: 'grabbing' },
  }),

  closeButton: {
    position: 'absolute' as const,
    top: 0,
    right: 0,
    size: '2xs',
    minW: '6',
    height: '6',
    padding: 0,
    variant: 'ghost',
    color: 'var(--app-text-muted)',
    bg: 'transparent',
    _hover: {
      bg: 'rgba(255, 255, 255, 0.12)',
      color: 'var(--app-text)',
    },
    zIndex: 10,
  },
} as const;
