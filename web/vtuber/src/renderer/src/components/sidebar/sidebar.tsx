/* eslint-disable react/require-default-props */
import { Box, Button, Menu } from '@chakra-ui/react';
import {
  FiSettings,
  FiClock,
  FiPlus,
  FiChevronLeft,
  FiUsers,
  FiLayers,
  FiMaximize,
} from 'react-icons/fi';
import { memo } from 'react';
import { sidebarStyles } from './sidebar-styles';
import SettingUI from './setting/setting-ui';
import ChatHistoryPanel from './chat-history-panel';
import BottomTab from './bottom-tab';
import HistoryDrawer from './history-drawer';
import { useSidebar } from '@/hooks/sidebar/use-sidebar';
import GroupDrawer from './group-drawer';
import { ModeType } from '@/context/mode-context';
import { ColorModeButton } from '@/components/ui/color-mode';

// Type definitions
interface SidebarProps {
  isCollapsed?: boolean
  onToggle: () => void
  onToggleFullscreen: () => void
}

interface HeaderButtonsProps {
  onSettingsOpen: () => void
  onNewHistory: () => void
  setMode: (mode: ModeType) => void
  currentMode: 'window' | 'pet'
  isElectron: boolean
  onToggleFullscreen: () => void
}

// Reusable components
const ToggleButton = memo(({ isCollapsed, onToggle }: {
  isCollapsed: boolean
  onToggle: () => void
}) => (
  <Box
    {...sidebarStyles.sidebar.toggleButton}
    style={{
      transform: isCollapsed ? 'rotate(180deg)' : 'rotate(0deg)',
    }}
    onClick={onToggle}
  >
    <FiChevronLeft />
  </Box>
));

ToggleButton.displayName = 'ToggleButton';

const ModeMenu = memo(({ setMode, currentMode, isElectron }: {
  setMode: (mode: ModeType) => void
  currentMode: ModeType
  isElectron: boolean
}) => (
  <Menu.Root>
    <Menu.Trigger
      as={Button}
      aria-label="Mode menu"
      title="Mode"
      {...sidebarStyles.sidebar.headerButton}
    >
      <FiLayers />
    </Menu.Trigger>
    <Menu.Positioner>
      <Menu.Content>
        <Menu.RadioItemGroup value={currentMode}>
          <Menu.RadioItem value="window" onClick={() => setMode('window')}>
            <Menu.ItemIndicator />
            Live Mode
          </Menu.RadioItem>
          <Menu.RadioItem 
            value="pet" 
            onClick={() => {
              if (isElectron) {
                setMode('pet');
              }
            }}
            disabled={!isElectron}
            title={!isElectron ? "Pet mode is only available in desktop app" : undefined}
          >
            <Menu.ItemIndicator />
            Pet Mode
          </Menu.RadioItem>
        </Menu.RadioItemGroup>
      </Menu.Content>
    </Menu.Positioner>
  </Menu.Root>
));

ModeMenu.displayName = 'ModeMenu';

const HeaderButtons = memo(({
  onSettingsOpen,
  onNewHistory,
  setMode,
  currentMode,
  isElectron,
  onToggleFullscreen,
}: HeaderButtonsProps) => (
  <Box display="flex" gap={1}>
    <Button
      onClick={onSettingsOpen}
      aria-label="Settings"
      title="Settings"
      {...sidebarStyles.sidebar.headerButton}
    >
      <FiSettings />
    </Button>

    <GroupDrawer>
      <Button
        aria-label="Group"
        title="Group"
        {...sidebarStyles.sidebar.headerButton}
      >
        <FiUsers />
      </Button>
    </GroupDrawer>

    <HistoryDrawer>
      <Button
        aria-label="History"
        title="History"
        {...sidebarStyles.sidebar.headerButton}
      >
        <FiClock />
      </Button>
    </HistoryDrawer>

    <Button
      onClick={onNewHistory}
      aria-label="New chat"
      title="New chat"
      {...sidebarStyles.sidebar.headerButton}
    >
      <FiPlus />
    </Button>

    <ModeMenu setMode={setMode} currentMode={currentMode} isElectron={isElectron} />

    <ColorModeButton
      title="Day/Night"
      {...sidebarStyles.sidebar.headerButton}
    />

    <Button
      onClick={onToggleFullscreen}
      aria-label="Fullscreen canvas"
      title="Fullscreen"
      {...sidebarStyles.sidebar.headerButton}
    >
      <FiMaximize />
    </Button>
  </Box>
));

HeaderButtons.displayName = 'HeaderButtons';

const SidebarContent = memo(({ 
  onSettingsOpen, 
  onNewHistory, 
  setMode, 
  currentMode,
  isElectron,
  onToggleFullscreen,
}: HeaderButtonsProps) => (
  <Box {...sidebarStyles.sidebar.content}>
    <Box {...sidebarStyles.sidebar.header}>
      <HeaderButtons
        onSettingsOpen={onSettingsOpen}
        onNewHistory={onNewHistory}
        setMode={setMode}
        currentMode={currentMode}
        isElectron={isElectron}
        onToggleFullscreen={onToggleFullscreen}
      />
    </Box>
    <ChatHistoryPanel />
    <BottomTab />
  </Box>
));

SidebarContent.displayName = 'SidebarContent';

// Main component
function Sidebar({
  isCollapsed = false,
  onToggle,
  onToggleFullscreen,
}: SidebarProps): JSX.Element {
  const {
    settingsOpen,
    onSettingsOpen,
    onSettingsClose,
    createNewHistory,
    setMode,
    currentMode,
    isElectron,
  } = useSidebar();

  return (
    <Box {...sidebarStyles.sidebar.container(isCollapsed)}>
      <ToggleButton isCollapsed={isCollapsed} onToggle={onToggle} />

      {!isCollapsed && !settingsOpen && (
        <SidebarContent
          onSettingsOpen={onSettingsOpen}
          onNewHistory={createNewHistory}
          setMode={setMode}
          currentMode={currentMode}
          isElectron={isElectron}
          onToggleFullscreen={onToggleFullscreen}
        />
      )}

      {!isCollapsed && settingsOpen && (
        <SettingUI
          open={settingsOpen}
          onClose={onSettingsClose}
          onToggle={onToggle}
        />
      )}
    </Box>
  );
}

export default Sidebar;
