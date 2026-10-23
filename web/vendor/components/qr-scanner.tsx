'use client';

// Live camera QR scanner — requests the device camera, decodes frames on a
// canvas via jsQR (chosen over qr-scanner/@zxing/browser specifically
// because it needs no Web Worker file to bundle/copy, which is fiddly
// under Next.js/Turbopack; a few decodes/sec on the main thread is plenty
// for a code someone is holding still at a counter). Self-contained: owns
// its own permission/status UI, only reports decoded text upward. Must
// only run client-side (touches navigator/getUserMedia), same reason
// location-picker.tsx and vendor-map.tsx are client components.

import jsQR from 'jsqr';
import { AlertTriangle, CameraOff, RotateCw } from 'lucide-react';
import { useEffect, useRef, useState } from 'react';

function ViewfinderGuide() {
  return (
    <div className="pointer-events-none absolute inset-0 flex items-center justify-center">
      <div className="relative w-40 h-40">
        <span className="absolute top-0 left-0 w-8 h-8 border-t-2 border-l-2 border-accent rounded-tl-xl" />
        <span className="absolute top-0 right-0 w-8 h-8 border-t-2 border-r-2 border-accent rounded-tr-xl" />
        <span className="absolute bottom-0 left-0 w-8 h-8 border-b-2 border-l-2 border-accent rounded-bl-xl" />
        <span className="absolute bottom-0 right-0 w-8 h-8 border-b-2 border-r-2 border-accent rounded-br-xl" />
      </div>
      <span className="absolute bottom-4 rounded-full bg-black/40 text-white text-[12px] font-bold px-3 py-1.5 flex items-center gap-1.5">
        <span className="w-1.5 h-1.5 rounded-full bg-[#3ec97a]" /> Camera ready
      </span>
    </div>
  );
}

type Status = 'requesting' | 'active' | 'denied' | 'unavailable';

export default function QrScanner({ onDecode }: { onDecode: (text: string) => void }) {
  const videoRef = useRef<HTMLVideoElement>(null);
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const [status, setStatus] = useState<Status>('requesting');
  const [attempt, setAttempt] = useState(0);
  // onDecode is re-created every render on the caller's side — capturing
  // it in a ref lets the camera-setup effect below run once per attempt
  // without needing onDecode in its dependency array.
  const onDecodeRef = useRef(onDecode);
  onDecodeRef.current = onDecode;

  useEffect(() => {
    if (typeof navigator === 'undefined' || !navigator.mediaDevices?.getUserMedia) {
      setStatus('unavailable');
      return;
    }
    let cancelled = false;
    let stream: MediaStream | null = null;
    let raf = 0;

    const tick = () => {
      if (cancelled) return;
      const video = videoRef.current;
      const canvas = canvasRef.current;
      if (video && canvas && video.readyState === video.HAVE_ENOUGH_DATA) {
        canvas.width = video.videoWidth;
        canvas.height = video.videoHeight;
        const ctx = canvas.getContext('2d');
        if (ctx) {
          ctx.drawImage(video, 0, 0, canvas.width, canvas.height);
          const imageData = ctx.getImageData(0, 0, canvas.width, canvas.height);
          const code = jsQR(imageData.data, imageData.width, imageData.height);
          if (code?.data) onDecodeRef.current(code.data);
        }
      }
      raf = requestAnimationFrame(tick);
    };

    navigator.mediaDevices
      .getUserMedia({ video: { facingMode: 'environment' } })
      .catch(() => navigator.mediaDevices.getUserMedia({ video: true }))
      .then((s) => {
        if (cancelled) {
          s.getTracks().forEach((t) => t.stop());
          return;
        }
        stream = s;
        if (videoRef.current) {
          videoRef.current.srcObject = s;
          videoRef.current.play().catch(() => {});
        }
        setStatus('active');
        raf = requestAnimationFrame(tick);
      })
      .catch(() => {
        if (!cancelled) setStatus('denied');
      });

    return () => {
      cancelled = true;
      cancelAnimationFrame(raf);
      stream?.getTracks().forEach((t) => t.stop());
    };
  }, [attempt]);

  return (
    <div className="relative w-full h-full">
      <video ref={videoRef} muted autoPlay playsInline className="w-full h-full object-cover" />
      <canvas ref={canvasRef} className="hidden" />

      {status === 'active' && <ViewfinderGuide />}

      {status !== 'active' && (
        <div className="absolute inset-0 flex items-center justify-center bg-[#0b1f22]">
          {status === 'requesting' && (
            <p className="text-[13px] font-semibold text-white/70">Requesting camera access…</p>
          )}
          {(status === 'denied' || status === 'unavailable') && (
            <div className="text-center px-6 max-w-xs">
              {status === 'denied' ? (
                <AlertTriangle size={28} className="mx-auto mb-2 text-white/70" />
              ) : (
                <CameraOff size={28} className="mx-auto mb-2 text-white/70" />
              )}
              <p className="text-[13px] font-bold text-white">
                {status === 'denied' ? 'Camera access denied' : 'No camera available'}
              </p>
              <p className="text-[12px] font-semibold text-white/60 mt-1">
                {status === 'denied'
                  ? 'Allow camera access in your browser, or enter the code manually below.'
                  : 'Enter the code manually below.'}
              </p>
              {status === 'denied' && (
                <button
                  type="button"
                  onClick={() => setAttempt((a) => a + 1)}
                  className="mt-3 rounded-lg bg-white/10 hover:bg-white/15 transition-colors text-white text-[12.5px] font-bold px-3 py-1.5 flex items-center gap-1.5 mx-auto"
                >
                  <RotateCw size={13} /> Try again
                </button>
              )}
            </div>
          )}
        </div>
      )}
    </div>
  );
}
