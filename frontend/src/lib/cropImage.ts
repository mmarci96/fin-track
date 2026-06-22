/** A crop rectangle in the image's natural (full-resolution) pixels. */
export interface PixelArea {
  x: number;
  y: number;
  width: number;
  height: number;
}

const MAX_EDGE = 1600; // cap the longest edge to keep uploads small

function loadImage(src: string): Promise<HTMLImageElement> {
  return new Promise((resolve, reject) => {
    const img = new Image();
    img.addEventListener('load', () => resolve(img));
    img.addEventListener('error', (e) => reject(e));
    img.src = src;
  });
}

/**
 * Render the selected crop area of `imageSrc` to a canvas and export it as a
 * JPEG Blob, downscaled so the longest edge is at most MAX_EDGE px. The result
 * is what gets uploaded — never the full original.
 */
export async function getCroppedBlob(
  imageSrc: string,
  area: PixelArea,
): Promise<Blob> {
  const image = await loadImage(imageSrc);

  const scale = Math.min(1, MAX_EDGE / Math.max(area.width, area.height));
  const canvas = document.createElement('canvas');
  canvas.width = Math.round(area.width * scale);
  canvas.height = Math.round(area.height * scale);

  const ctx = canvas.getContext('2d');
  if (!ctx) throw new Error('could not get canvas context');

  ctx.drawImage(
    image,
    area.x,
    area.y,
    area.width,
    area.height,
    0,
    0,
    canvas.width,
    canvas.height,
  );

  return new Promise((resolve, reject) => {
    canvas.toBlob(
      (blob) => (blob ? resolve(blob) : reject(new Error('crop failed'))),
      'image/jpeg',
      0.9,
    );
  });
}
