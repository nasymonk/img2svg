import { useState, useCallback, useRef } from 'react';
import FileUpload from '../components/FileUpload';
import ParamsPanel from '../components/ParamsPanel';
import PreviewPanel from '../components/PreviewPanel';
import type { ConvertParams } from '../components/ParamsPanel';

interface Task {
  id: string;
  status: string;
  error?: string;
}

export default function ConvertPage() {
  const [task, setTask] = useState<Task | null>(null);
  const [uploading, setUploading] = useState(false);
  const [params, setParams] = useState<ConvertParams>({
    colorCount: 16,
    mode: 'color',
    layerMode: 'split',
    denoise: true,
    sharpen: true,
    transparentBg: false,
  });

  const pollRef = useRef<ReturnType<typeof setInterval> | null>(null);

  const handleUpload = useCallback(async (file: File) => {
    setUploading(true);
    setTask(null);

    const form = new FormData();
    form.append('file', file);
    form.append('color_count', String(params.colorCount));
    form.append('mode', params.mode);
    form.append('layer_mode', params.layerMode);
    form.append('denoise', String(params.denoise));
    form.append('sharpen', String(params.sharpen));
    form.append('transparent_bg', String(params.transparentBg));

    try {
      const r = await fetch('/api/convert', { method: 'POST', body: form });
      if (!r.ok) {
        const d = await r.json();
        setTask({ id: '', status: 'failed', error: d.error || '上传失败' });
        setUploading(false);
        return;
      }
      const data = await r.json();
      const taskId: string = data.id;
      setTask({ id: taskId, status: 'running' });
      setUploading(false);

      // 轮询状态
      pollRef.current = setInterval(async () => {
        const sr = await fetch(`/api/convert/${taskId}/status`);
        const sd = await sr.json();
        const status: string = sd.status;
        setTask((prev) => {
          if (prev?.id !== taskId) return prev;
          return { id: taskId, status, error: sd.error };
        });
        if (status === 'succeeded' || status === 'failed') {
          if (pollRef.current) clearInterval(pollRef.current);
        }
      }, 500);
    } catch {
      setTask({ id: '', status: 'failed', error: '网络错误' });
      setUploading(false);
    }
  }, [params]);

  const handleReset = useCallback(() => {
    setTask(null);
    setUploading(false);
    if (pollRef.current) clearInterval(pollRef.current);
  }, []);

  return (
    <div className="space-y-6">
      <ParamsPanel params={params} onChange={setParams} disabled={uploading || task?.status === 'running'} />

      {!task ? (
        <FileUpload onUpload={handleUpload} uploading={uploading} />
      ) : (
        <PreviewPanel
          task={task}
          onReset={handleReset}
        />
      )}
    </div>
  );
}
