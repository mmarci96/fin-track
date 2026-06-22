import { useRef, useState } from 'react';
import ReactCrop, { type Crop, type PixelCrop } from 'react-image-crop';
import 'react-image-crop/dist/ReactCrop.css';
import { Button } from '@/components/ui/button';
import { Spinner } from '@/components/ui/spinner';
import { getCroppedBlob } from '@/lib/cropImage';

/**
 * Full-screen, touch-first cropper with a free-form selection rectangle. The
 * user drags the box to enclose only the white receipt; on confirm we map the
 * on-screen selection to natural pixels and export a downscaled JPEG Blob.
 */
export function Cropper({
  src,
  onConfirm,
  onCancel,
}: {
  src: string;
  onConfirm: (blob: Blob) => void;
  onCancel: () => void;
}) {
  const imgRef = useRef<HTMLImageElement>(null);
  const [crop, setCrop] = useState<Crop>();
  const [completed, setCompleted] = useState<PixelCrop>();
  const [working, setWorking] = useState(false);

  // Start with a generous centered selection so there's always something to drag.
  const onImageLoad = (e: React.SyntheticEvent<HTMLImageElement>) => {
    const { width, height } = e.currentTarget;
    setCrop({
      unit: 'px',
      x: width * 0.1,
      y: height * 0.05,
      width: width * 0.8,
      height: height * 0.9,
    });
  };

  const confirm = async () => {
    const img = imgRef.current;
    if (!img || !completed || completed.width === 0) return;
    setWorking(true);
    try {
      // ReactCrop reports display pixels; scale up to the image's natural size.
      const scaleX = img.naturalWidth / img.width;
      const scaleY = img.naturalHeight / img.height;
      const blob = await getCroppedBlob(src, {
        x: completed.x * scaleX,
        y: completed.y * scaleY,
        width: completed.width * scaleX,
        height: completed.height * scaleY,
      });
      onConfirm(blob);
    } finally {
      setWorking(false);
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex flex-col bg-black">
      <div className="flex flex-1 items-center justify-center overflow-hidden p-2">
        <ReactCrop
          crop={crop}
          onChange={(c) => setCrop(c)}
          onComplete={(c) => setCompleted(c)}
          className="max-h-full"
        >
          <img
            ref={imgRef}
            src={src}
            alt="Receipt to crop"
            onLoad={onImageLoad}
            className="max-h-[80vh] w-auto object-contain"
          />
        </ReactCrop>
      </div>

      <div className="space-y-3 bg-black/90 p-4 pb-[calc(env(safe-area-inset-bottom)+1rem)]">
        <p className="text-center text-sm text-white/80">
          Drag the box around only the white receipt — leave out the table and
          background.
        </p>
        <div className="flex gap-3">
          <Button
            variant="outline"
            size="lg"
            className="flex-1 border-white/30 bg-transparent text-white hover:bg-white/10"
            onClick={onCancel}
            disabled={working}
          >
            Retake
          </Button>
          <Button
            size="lg"
            className="flex-1"
            onClick={confirm}
            disabled={!completed || completed.width === 0 || working}
          >
            {working ? <Spinner /> : 'Use photo'}
          </Button>
        </div>
      </div>
    </div>
  );
}
