import { useRef } from 'react';
import { Camera, Image as ImageIcon } from 'lucide-react';
import { Button } from '@/components/ui/button';

interface CameraInputProps {
  onCameraPick: (src: string) => void;
  onLibraryPick: (file: File) => void;
}

export function CameraInput({ onCameraPick, onLibraryPick }: CameraInputProps) {
  const cameraRef = useRef<HTMLInputElement>(null);
  const libraryRef = useRef<HTMLInputElement>(null);

  const handleCameraChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (file) onCameraPick(URL.createObjectURL(file));
    e.target.value = '';
  };

  const handleLibraryChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (file) onLibraryPick(file);
    e.target.value = '';
  };

  return (
    <div className="space-y-3">
      <input
        ref={cameraRef}
        type="file"
        accept="image/*"
        capture="environment"
        className="hidden"
        onChange={handleCameraChange}
      />
      <input
        ref={libraryRef}
        type="file"
        accept="image/*"
        className="hidden"
        onChange={handleLibraryChange}
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
