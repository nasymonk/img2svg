import { useState, useCallback, useRef, useEffect } from 'react';
import FileUpload from '../components/FileUpload';
import PreviewPanel from '../components/PreviewPanel';

interface Task {
  id: string;
  status: string;
  error?: string;
}

export default function ConvertPage() {
  const [task, setTask] = useState<Task | null>(null);
  const [uploading, setUploading] = useState(false);
  const pollRef = useRef<ReturnType<typeof setInterval> | null>(null);
  const mountedRef = useRef(true);

  useEffect(() => {
    return () => {
      mountedRef.current = false;
      if (pollRef.current) clearInterval(pollRef.current);
    };
  }, []);

  const handleUpload = useCallback(async (file: File) => {
    setUploading(true);
    setTask(null);

    const form = new FormData();
    form.append('file', file);

    try {
      const r = await fetch('/api/convert', { method: 'POST', body: form });
      const d = await r.json().catch(() => ({}));
      if (!r.ok) {
        setTask({ id: '', status: 'failed', error: d.error || `服务器错误 (${r.status})` });
        setUploading(false);
        return;
      }
      const taskId: string = d.id;
      setTask({ id: taskId, status: 'running' });
      setUploading(false);

      pollRef.current = setInterval(async () => {
        try {
          const sr = await fetch(`/api/convert/${taskId}/status`);
          const sd = await sr.json().catch(() => ({}));
          if (!mountedRef.current) return;
          const status: string = sd.status || 'running';
          setTask((prev) => {
            if (prev?.id !== taskId) return prev;
            return { id: taskId, status, error: sd.error };
          });
          if (status === 'succeeded' || status === 'failed') {
            if (pollRef.current) clearInterval(pollRef.current);
          }
        } catch { /* 轮询失败不中断 */ }
      }, 500);
    } catch {
      if (mountedRef.current) {
        setTask({ id: '', status: 'failed', error: '网络错误，请重试' });
        setUploading(false);
      }
    }
  }, []);

  const handleReset = useCallback(() => {
    setTask(null);
    setUploading(false);
    if (pollRef.current) clearInterval(pollRef.current);
  }, []);

  return (
    <div className="space-y-6">
      <p className="text-text-secondary text-sm text-center">智能矢量化 — 自动检测颜色并优化，无需手动调参</p>

      {!task ? (
        <FileUpload onUpload={handleUpload} uploading={uploading} />
      ) : (
        <PreviewPanel task={task} onReset={handleReset} />
      )}
    </div>
  );
}
