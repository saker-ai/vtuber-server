import { Box, Text } from '@chakra-ui/react';
import { memo, useEffect, useState } from 'react';
import { canvasStyles } from './canvas-styles';
import { useSubtitleDisplay } from '@/hooks/canvas/use-subtitle-display';
import { useSubtitle } from '@/context/subtitle-context';

// Type definitions
interface SubtitleTextProps {
  text: string
}

interface SubtitleProps {
  bottomOffset: string
}

// Reusable components
const SubtitleText = memo(({ text }: SubtitleTextProps) => (
  <Text {...canvasStyles.subtitle.text}>
    {text}
  </Text>
));

SubtitleText.displayName = 'SubtitleText';

// Main component
const markdownImageRegex = /!\[([^\]]*)\]\(([^)]+)\)/g;

const parseSubtitleContent = (content: string) => {
  const images: { alt: string; url: string }[] = [];
  markdownImageRegex.lastIndex = 0;
  let match: RegExpExecArray | null;

  while ((match = markdownImageRegex.exec(content)) !== null) {
    const alt = match[1] || 'image';
    const url = match[2].trim();
    if (url) {
      images.push({ alt, url });
    }
  }

  const cleanedText = content.replace(markdownImageRegex, '').trim();
  return { cleanedText, images };
};

const Subtitle = memo(({ bottomOffset }: SubtitleProps): JSX.Element | null => {
  const { subtitleText, isLoaded } = useSubtitleDisplay();
  const { showSubtitle } = useSubtitle();

  const [persistentImages, setPersistentImages] = useState<
    { alt: string; url: string }[]
  >([]);

  const { cleanedText, images } = parseSubtitleContent(subtitleText || '');

  useEffect(() => {
    if (images.length > 0) {
      setPersistentImages(images);
    }
  }, [images]);

  if (!isLoaded || !showSubtitle) return null;

  if (!cleanedText && persistentImages.length === 0) return null;

  return (
    <>
      {cleanedText && (
        <Box
          position="absolute"
          bottom={bottomOffset}
          left="50%"
          transform="translateX(-50%)"
          width="60%"
        >
          <Box {...canvasStyles.subtitle.container}>
            <SubtitleText text={cleanedText} />
          </Box>
        </Box>
      )}
      {persistentImages.length > 0 && (
        <Box {...canvasStyles.subtitle.imageContainer}>
          {persistentImages.map((image, index) => (
            <Box key={`${image.url}-${index}`} maxW="100%">
              <img
                src={image.url}
                alt={image.alt}
                loading="lazy"
                style={canvasStyles.subtitle.image}
              />
            </Box>
          ))}
        </Box>
      )}
    </>
  );
});

Subtitle.displayName = 'Subtitle';

export default Subtitle;
