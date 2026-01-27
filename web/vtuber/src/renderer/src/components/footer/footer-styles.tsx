import { SystemStyleObject } from '@chakra-ui/react';

interface FooterStyles {
  container: (isCollapsed: boolean) => SystemStyleObject
  toggleButton: SystemStyleObject
  actionButton: SystemStyleObject
  input: SystemStyleObject
  attachButton: SystemStyleObject
}

interface AIIndicatorStyles {
  container: SystemStyleObject
  text: SystemStyleObject
}

export const footerStyles: {
  footer: FooterStyles
  aiIndicator: AIIndicatorStyles
} = {
  footer: {
    container: (isCollapsed) => ({
      bg: isCollapsed ? 'transparent' : 'var(--app-panel-strong)',
      borderTopRadius: isCollapsed ? 'none' : 'lg',
      borderTop: isCollapsed ? 'none' : '1px solid',
      borderColor: isCollapsed ? 'transparent' : 'var(--app-border)',
      transform: isCollapsed ? 'translateY(calc(100% - 24px))' : 'translateY(0)',
      transition: 'all 0.3s cubic-bezier(0.4, 0, 0.2, 1)',
      height: '100%',
      position: 'relative',
      overflow: isCollapsed ? 'visible' : 'hidden',
      pb: '4',
      backdropFilter: 'blur(12px) saturate(140%)',
      boxShadow: 'var(--app-shadow)',
      animation: 'appFadeUp 640ms ease-out',
    }),
    toggleButton: {
      height: '24px',
      display: 'flex',
      alignItems: 'center',
      justifyContent: 'center',
      cursor: 'pointer',
      color: 'var(--app-text-muted)',
      _hover: { color: 'var(--app-text)' },
      bg: 'transparent',
      transition: 'all 0.3s cubic-bezier(0.4, 0, 0.2, 1)',
    },
    actionButton: {
      borderRadius: '14px',
      width: '46px',
      height: '46px',
      minW: '46px',
      border: '1px solid',
      borderColor: 'var(--app-border)',
      boxShadow: '0 10px 20px rgba(19, 19, 19, 0.12)',
      transition: 'transform 0.2s ease, box-shadow 0.2s ease',
      _hover: {
        transform: 'translateY(-1px)',
        boxShadow: '0 14px 26px rgba(19, 19, 19, 0.16)',
      },
    },
    input: {
      bg: 'var(--app-panel-weak)',
      border: '1px solid',
      borderColor: 'var(--app-border)',
      height: '72px',
      borderRadius: '14px',
      fontSize: '16px',
      pl: '12',
      pr: '4',
      color: 'var(--app-text)',
      _placeholder: {
        color: 'var(--app-text-muted)',
      },
      _focusVisible: {
        borderColor: 'var(--app-accent)',
        boxShadow: '0 0 0 3px rgba(243, 106, 90, 0.16)',
        bg: 'rgba(28, 29, 33, 0.04)',
      },
      resize: 'none',
      minHeight: '72px',
      maxHeight: '72px',
      py: '3',
      display: 'flex',
      alignItems: 'center',
      lineHeight: '1.4',
    },
    attachButton: {
      position: 'absolute',
      left: '2',
      top: '50%',
      transform: 'translateY(-50%)',
      color: 'var(--app-text-muted)',
      zIndex: 2,
      _hover: {
        bg: 'transparent',
        color: 'var(--app-text)',
      },
    },
  },
  aiIndicator: {
    container: {
      background: 'linear-gradient(135deg, var(--app-accent), #f2c389)',
      color: '#1a1b1e',
      width: '112px',
      height: '30px',
      borderRadius: '12px',
      display: 'flex',
      alignItems: 'center',
      justifyContent: 'center',
      boxShadow: '0 10px 18px rgba(19, 19, 19, 0.12)',
      overflow: 'hidden',
    },
    text: {
      fontSize: '12px',
      whiteSpace: 'nowrap',
      overflow: 'hidden',
      textOverflow: 'ellipsis',
    },
  },
};
