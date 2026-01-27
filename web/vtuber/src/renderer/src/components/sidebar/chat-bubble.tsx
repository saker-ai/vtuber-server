import { Box, Text, Flex } from '@chakra-ui/react';
import { Avatar, AvatarGroup } from '@/components/ui/avatar';
import { Message } from '@/services/websocket-service';

// Type definitions
interface ChatBubbleProps {
  message: Message;
  isSelected?: boolean;
  onClick?: () => void;
}

// Main component
export function ChatBubble({ message, isSelected, onClick }: ChatBubbleProps): JSX.Element {
  const isAI = message.role === 'ai';

  return (
    <Box
      onClick={onClick}
      cursor="pointer"
      bg={isSelected ? 'rgba(63, 224, 200, 0.14)' : 'transparent'}
      _hover={{ bg: 'rgba(255, 255, 255, 0.06)' }}
      p={2}
      borderRadius="md"
      transition="background-color 0.2s"
    >
      <Flex gap={3}>
        <AvatarGroup>
          <Avatar
            size="sm"
            name={message.name || (isAI ? 'AI' : 'Me')}
            bg={isAI ? 'var(--app-accent-3)' : 'var(--app-accent)'}
            color="#0b1116"
          />
        </AvatarGroup>
        <Box flex={1}>
          <Text fontSize="sm" fontWeight="bold" color="var(--app-text)">
            {message.name || (isAI ? 'AI' : 'Me')}
          </Text>
          <Text
            fontSize="sm"
            color="var(--app-text-muted)"
            truncate
          >
            {message.content}
          </Text>
          <Text fontSize="xs" color="var(--app-text-muted)" mt={1}>
            {new Date(message.timestamp).toLocaleTimeString()}
          </Text>
        </Box>
      </Flex>
    </Box>
  );
}
