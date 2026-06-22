import { useRef } from 'react';
import { Camera, Image as ImageIcon } from 'lucide-react';
import { Button } from '@/components/ui/button';

/**
 * Two entry points to get a photo into the crop flow:
 *  - "Take photo" opens the rear camera on phones (capture="environment"),
 *  - "Choose from library" picks an existing image.
 * Both hand back an object URL the cropper can load.
 */
export function CameraInput({ onPick }: { onPick: (src: string) => void }) {
  const cameraRef = useRef<HTMLInputElement>(null);
  const libraryRef = useRef<HTMLInputElement>(null);

  const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (file) onPick(URL.createObjectURL(file));
    e.target.value = ''; // allow re-picking the same file
  };

  return (
    <div className="space-y-3">
      <input
        ref={cameraRef}
        type="file"
        accept="image/*"
        capture="environment"
        className="hidden"
        onChange={handleChange}
      />
      <input
        ref={libraryRef}
        type="file"
        accept="image/*"
        className="hidden"
        onChange={handleChange}
      />

      <Button
        size="lg"
        className="w-full"
        onClick={() => cameraRef.current?.click()}
      >
        <Camera className="h-5 w-5" />
        Take photo
      </Button>
      <Button
        size="lg"
        variant="outline"
        className="w-full"
        onClick={() => libraryRef.current?.click()}
      >
        <ImageIcon className="h-5 w-5" />
        Choose from library
      </Button>
    </div>
  );
}
