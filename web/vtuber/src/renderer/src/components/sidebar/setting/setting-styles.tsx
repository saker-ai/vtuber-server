const isElectron = window.api !== undefined;
export const settingStyles = {
  settingUI: {
    container: {
      width: '100%',
      height: '100%',
      p: 4,
      gap: 4,
      position: 'relative',
      overflowY: 'auto',
      css: {
        '&::-webkit-scrollbar': {
          width: '4px',
        },
        '&::-webkit-scrollbar-track': {
          bg: 'var(--app-panel-weak)',
          borderRadius: 'full',
        },
        '&::-webkit-scrollbar-thumb': {
          bg: 'rgba(28, 29, 33, 0.28)',
          borderRadius: 'full',
        },
      },
    },
    header: {
      width: '100%',
      display: 'flex',
      alignItems: 'center',
      gap: 1,
    },
    title: {
      ml: 4,
      fontSize: 'lg',
      fontWeight: 'bold',
    },
    tabs: {
      root: {
        width: '100%',
        variant: 'plain' as const,
        colorPalette: 'gray',
      },
      content: {},
      trigger: {
        color: 'var(--app-text-muted)',
        px: 3,
        py: 2,
        borderRadius: 'md',
        _selected: {
          color: 'var(--app-text)',
          bg: 'var(--app-panel-strong)',
        },
        _hover: {
          color: 'var(--app-text)',
          bg: 'var(--app-panel-weak)',
        },
      },
      list: {
        display: 'flex',
        justifyContent: 'flex-start',
        width: '100%',
        borderBottom: 'none',
        bg: 'var(--app-panel-weak)',
        borderRadius: 'lg',
        p: 1,
        gap: 1,
        mb: 4,
        pl: 0,
      },
    },
    footer: {
      width: '100%',
      display: 'flex',
      justifyContent: 'flex-end',
      gap: 2,
      mt: 'auto',
      pt: 4,
      borderTop: '1px solid',
      borderColor: 'var(--app-border)',
    },
    drawerContent: {
      bg: 'var(--app-panel-strong)',
      backdropFilter: 'blur(12px) saturate(140%)',
      maxWidth: '440px',
      height: isElectron ? 'calc(100vh - 30px)' : '100vh',
      borderLeft: '1px solid',
      borderColor: 'var(--app-border)',
      boxShadow: 'var(--app-shadow)',
    },
    drawerHeader: {
      display: 'flex',
      alignItems: 'center',
      justifyContent: 'space-between',
      width: '100%',
      position: 'relative',
      px: 6,
      py: 4,
    },
    drawerTitle: {
      color: 'var(--app-text)',
      fontSize: 'lg',
      fontWeight: 'semibold',
    },
    closeButton: {
      position: 'absolute',
      right: 1,
      top: 1,
      color: 'var(--app-text)',

    },
  },
  general: {
    container: {
      align: 'stretch',
      gap: 6,
      p: 4,
    },
    field: {
      label: {
        color: 'var(--app-text)',
      },
    },
    select: {
      root: {
        colorPalette: 'gray',
        bg: 'var(--app-panel-weak)',
      },
      trigger: {
        bg: 'var(--app-panel-weak)',
        borderColor: 'var(--app-border)',
      },
      content: {
        bg: 'var(--app-panel-strong)',
        color: 'var(--app-text)',
        borderColor: 'var(--app-border)',
      },
      item: {
        color: 'var(--app-text)',
        _hover: {
          bg: 'var(--app-panel-weak)',
        },
        _highlighted: {
          bg: 'rgba(28, 29, 33, 0.08)',
        },
        _selected: {
          bg: 'rgba(28, 29, 33, 0.12)',
        },
      },
    },
    input: {
      bg: 'var(--app-panel-weak)',
      borderColor: 'var(--app-border)',
    },
    buttonGroup: {
      gap: 4,
      width: '100%',
    },
    button: {
      width: '50%',
      variant: 'outline' as const,
      bg: 'var(--app-accent-3)',
      color: '#0b1116',
      _hover: {
        bg: 'rgba(122, 162, 255, 0.75)',
      },
    },
    fieldLabel: {
      fontSize: '14px',
      color: 'var(--app-text-muted)',
    },
  },
  common: {
    field: {
      orientation: 'horizontal' as const,
    },
    fieldLabel: {
      fontSize: 'sm',
      color: 'var(--app-text)',
      whiteSpace: 'nowrap' as const,
    },
    switch: {
      size: 'md' as const,
      colorPalette: 'blue' as const,
      variant: 'solid' as const,
    },
    numberInput: {
      root: {
        pattern: '[0-9]*\\.?[0-9]*',
        inputMode: 'decimal' as const,
      },
      input: {
        bg: 'var(--app-panel-weak)',
        borderColor: 'var(--app-border)',
        _hover: {
          bg: 'rgba(28, 29, 33, 0.06)',
        },
      },
    },
    container: {
      gap: 8,
      maxW: 'sm',
      css: { '--field-label-width': '120px' },
    },
    input: {
      bg: 'var(--app-panel-weak)',
      borderColor: 'var(--app-border)',
      _hover: {
        bg: 'rgba(28, 29, 33, 0.06)',
      },
    },
  },
  live2d: {
    container: {
      gap: 8,
      maxW: 'sm',
      css: { '--field-label-width': '120px' },
    },
    emotionMap: {
      title: {
        fontWeight: 'bold',
        mb: 4,
      },
      entry: {
        mb: 2,
      },
      button: {
        colorPalette: 'blue',
        mt: 2,
      },
      deleteButton: {
        colorPalette: 'red',
      },
    },
  },
};
