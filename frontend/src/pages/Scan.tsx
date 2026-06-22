import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { X } from 'lucide-react';
import { CameraInput } from '@/components/CameraInput';
import { Cropper } from '@/components/Cropper';
import { WarningBanner } from '@/components/WarningBanner';
import { Button } from '@/components/ui/button';
import { Spinner } from '@/components/ui/spinner';
import { useUploadImage } from '@/api/receipts';
import { ApiError } from '@/lib/api';

type Stage = 'capture' | 'crop' | 'uploading' | 'rejected';

const RETAKE_TIPS = [
  'Lay the receipt flat on a dark surface.',
  'Use good, even lighting — avoid shadows and glare.',
  'Fit the whole receipt in frame, edge to edge.',
];

export function Scan() {
  const navigate = useNavigate();
  const upload = useUploadImage();
  const [stage, setStage] = useState<Stage>('capture');
  const [src, setSrc] = useState<string | null>(null);
  const [rejectMessage, setRejectMessage] = useState<string>('');

  const reset = () => {
    if (src) URL.revokeObjectURL(src);
    setSrc(null);
    setStage('capture');
  };

  const onPick = (objectUrl: string) => {
    setSrc(objectUrl);
    setStage('crop');
  };

  const onConfirmCrop = async (blob: Blob) => {
    setStage('uploading');
    try {
      const result = await upload.mutateAsync(blob);
      // The backend already stored it. Go to review; flag best-effort parses.
      navigate(`/receipts/${result.receipt.id}`, {
        replace: true,
        state: {
          decision: result.decision,
          warnings: result.warnings,
          reconciled: result.detected.reconciled,
        },
      });
    } catch (err) {
      // decision === "reject" comes back as HTTP 422 — force a retake.
      if (err instanceof ApiError && err.status === 422) {
        setRejectMessage(err.message);
        setStage('rejected');
      } else {
        setRejectMessage(
          err instanceof Error ? err.message : 'Upload failed. Try again.',
        );
        setStage('rejected');
      }
    }
  };

  if (stage === 'crop' && src) {
    return <Cropper src={src} onConfirm={onConfirmCrop} onCancel={reset} />;
  }

  return (
    <div className="flex min-h-full flex-col p-4">
      <header className="mb-6 flex items-center justify-between">
        <h1 className="text-xl font-semibold">Scan receipt</h1>
        <Button variant="ghost" size="icon" onClick={() => navigate('/')}>
          <X className="h-5 w-5" />
        </Button>
      </header>

      {stage === 'uploading' && (
        <div className="flex flex-1 flex-col items-center justify-center gap-3 text-muted-foreground">
          <Spinner className="h-8 w-8" />
          <p>Reading your receipt…</p>
        </div>
      )}

      {stage === 'rejected' && (
        <div className="space-y-4">
          <WarningBanner
            tone="destructive"
            title="Couldn't read that receipt"
            messages={[rejectMessage, ...RETAKE_TIPS]}
          />
          <Button size="lg" className="w-full" onClick={reset}>
            Retake photo
          </Button>
        </div>
      )}

      {stage === 'capture' && (
        <div className="flex flex-1 flex-col justify-center gap-6">
          <p className="text-center text-sm text-muted-foreground">
            Take a clear photo of your receipt, then crop to just the white
            paper. We'll read the items and total automatically.
          </p>
          <CameraInput onPick={onPick} />
        </div>
      )}
    </div>
  );
}
