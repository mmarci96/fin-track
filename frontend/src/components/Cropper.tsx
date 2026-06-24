import { useEffect, useRef, useState } from 'react';
import ReactCrop, { type Crop } from 'react-image-crop';
import 'react-image-crop/dist/ReactCrop.css';
import { Minus, Plus, Maximize } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Spinner } from '@/components/ui/spinner';
import { getCroppedBlob } from '@/lib/cropImage';

const MIN_ZOOM = 0.5;
const MAX_ZOOM = 8;
const clampZoom = (z: number) => Math.min(MAX_ZOOM, Math.max(MIN_ZOOM, z));

/**
 * Full-screen, touch-first cropper with a free-form selection rectangle. The
 * image renders at "fit width" by default; tall receipts overflow the viewport
 * and can be panned by scrolling. A zoom control (−/Fit/+ and pinch) enlarges
 * the image so any portion of a long receipt is reachable.
 *
 * The selection is kept as a PERCENT crop so it stays valid across zoom and
 * resize, and on confirm we map those percentages straight onto the image's
 * natural pixels — independent of whatever on-screen size the image has.
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
  const containerRef = useRef<HTMLDivElement>(null);
  const imgRef = useRef<HTMLImageElement>(null);
  const [crop, setCrop] = useState<Crop>();
  const [working, setWorking] = useState(false);
  const [baseWidth, setBaseWidth] = useState<number>();
  const [zoom, setZoom] = useState(1);

  // Fit the image to the container width on load (tall images then scroll), and
  // seed a generous percent selection so there's always something to drag.
  const onImageLoad = () => {
    const pad = 16; // matches the container's p-2
    const fit = (containerRef.current?.clientWidth ?? 0) - pad;
    if (fit > 0) setBaseWidth(fit);
    setCrop({ unit: '%', x: 5, y: 3, width: 90, height: 94 });
  };

  // Pinch-to-zoom (best effort). ReactCrop owns single-pointer drags for the
  // selection box; two-finger gestures are handled here on the container. The
  // −/Fit/+ buttons are the guaranteed path.
  useEffect(() => {
    const el = containerRef.current;
    if (!el) return;
    let startDist = 0;
    let startZoom = 1;
    const dist = (t: TouchList) =>
      Math.hypot(t[0].clientX - t[1].clientX, t[0].clientY - t[1].clientY);

    const onStart = (e: TouchEvent) => {
      if (e.touches.length === 2) {
        startDist = dist(e.touches);
        startZoom = zoom;
      }
    };
    const onMove = (e: TouchEvent) => {
      if (e.touches.length === 2 && startDist > 0) {
        e.preventDefault();
        setZoom(clampZoom((startZoom * dist(e.touches)) / startDist));
      }
    };
    const onEnd = (e: TouchEvent) => {
      if (e.touches.length < 2) startDist = 0;
    };

    el.addEventListener('touchstart', onStart, { passive: false });
    el.addEventListener('touchmove', onMove, { passive: false });
    el.addEventListener('touchend', onEnd);
    return () => {
      el.removeEventListener('touchstart', onStart);
      el.removeEventListener('touchmove', onMove);
      el.removeEventListener('touchend', onEnd);
    };
  }, [zoom]);

  const confirm = async () => {
    const img = imgRef.current;
    if (!img || !crop || !crop.width || !crop.height) return;
    setWorking(true);
    try {
      // crop is a percentage of the image — map directly onto natural pixels,
      // so the result is independent of the current zoom/display size.
      const blob = await getCroppedBlob(src, {
        x: (crop.x / 100) * img.naturalWidth,
        y: (crop.y / 100) * img.naturalHeight,
        width: (crop.width / 100) * img.naturalWidth,
        height: (crop.height / 100) * img.naturalHeight,
      });
      onConfirm(blob);
    } finally {
      setWorking(false);
    }
  };

  const width = baseWidth ? baseWidth * zoom : undefined;

  return (
    <div className="fixed inset-0 z-50 flex flex-col bg-black">
      <div
        ref={containerRef}
        className="flex flex-1 justify-center overflow-auto p-2"
      >
        <ReactCrop
          crop={crop}
          onChange={(_, percent) => setCrop(percent)}
          style={{ maxWidth: 'none' }}
        >
          <img
            ref={imgRef}
            src={src}
            alt="Receipt to crop"
            onLoad={onImageLoad}
            className="block h-auto"
            style={{ width, maxWidth: 'none' }}
          />
        </ReactCrop>
      </div>

      <div className="space-y-3 bg-black/90 p-4 pb-[calc(env(safe-area-inset-bottom)+1rem)]">
        <div className="flex items-center justify-center gap-3">
          <Button
            variant="outline"
            size="icon"
            className="border-white/30 bg-transparent text-white hover:bg-white/10"
            onClick={() => setZoom((z) => clampZoom(z / 1.25))}
            disabled={working}
            aria-label="Zoom out"
          >
            <Minus className="h-5 w-5" />
          </Button>
          <Button
            variant="outline"
            size="icon"
            className="border-white/30 bg-transparent text-white hover:bg-white/10"
            onClick={() => setZoom(1)}
            disabled={working}
            aria-label="Fit to width"
          >
            <Maximize className="h-5 w-5" />
          </Button>
          <Button
            variant="outline"
            size="icon"
            className="border-white/30 bg-transparent text-white hover:bg-white/10"
            onClick={() => setZoom((z) => clampZoom(z * 1.25))}
            disabled={working}
            aria-label="Zoom in"
          >
            <Plus className="h-5 w-5" />
          </Button>
        </div>
        <p className="text-center text-sm text-white/80">
          Drag the box around only the white receipt. Scroll or pinch to reach a
          long receipt.
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
            disabled={!crop || !crop.width || working}
          >
            {working ? <Spinner /> : 'Use photo'}
          </Button>
        </div>
      </div>
    </div>
  );
}
