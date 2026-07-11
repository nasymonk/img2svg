import { useCallback, useRef, useState, type DragEvent } from 'react';

interface Props {
  onUpload: (file: File) => void;
  uploading: boolean;
}

export default function FileUpload({ onUpload, uploading }: Props) {
  const [dragOver, setDragOver] = useState(false);
  const inputRef = useRef<HTMLInputElement>(null);

  const handleDrop = useCallback(
    (e: DragEvent) => {
      e.preventDefault();
      setDragOver(false);
      const file = e.dataTransfer.files?.[0];
      if (file) onUpload(file);
    },
    [onUpload],
  );

  const handleChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      const file = e.target.files?.[0];
      if (file) onUpload(file);
    },
    [onUpload],
  );

  return (
    <div
      onDragOver={(e) => { e.preventDefault(); setDragOver(true); }}
      onDragLeave={() => setDragOver(false)}
      onDrop={handleDrop}
      onClick={() => inputRef.current?.click()}
      className={`
        border-2 border-dashed rounded-2xl p-16 text-center cursor-pointer transition-all
        ${dragOver
          ? 'border-primary bg-primary/5 scale-[1.01]'
          : 'border-border hover:border-text-secondary'
        }
        ${uploading ? 'pointer-events-none opacity-50' : ''}
      `}
    >
      <input ref={inputRef} type="file" accept=".png,.jpg,.jpeg,.webp,.bmp,.gif" onChange={handleChange} className="hidden" />
      {uploading ? (
        <div className="space-y-3">
          <div className="w-10 h-10 border-2 border-primary border-t-transparent rounded-full animate-spin mx-auto" />
          <p className="text-text-secondary">转换中...</p>
        </div>
      ) : (
        <div className="space-y-3">
          <svg className="w-12 h-12 text-text-secondary mx-auto" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5}
              d="M4 16v2a2 2 0 002 2h12a2 2 0 002-2v-2M7 10l5-5 5 5M12 5v14" />
          </svg>
          <p className="text-text">拖放图片到此处，或点击上传</p>
          <p className="text-text-secondary text-sm">支持 PNG / JPG / WEBP / BMP / GIF</p>
        </div>
      )}
    </div>
  );
}
