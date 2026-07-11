import { useState, useCallback, useRef, useEffect } from 'react';
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
    simplifyColors: 32,
    mode: 'color',
  });

  const pollRef = useRef<ReturnType<typeof setInterval> | null>(null);
  const mountedRef = useRef(true);

  // 组件卸载时取消轮询
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
    form.append('color_count', String(params.colorCount));
    form.append('mode', params.mode);
    form.append('simplify_colors', String(params.simplifyColors));

    try {
      const r = await fetch('/api/convert', { method: 'POST', body: form });
      const d = await r.json().catch(() => ({}));
      if (!r.ok) {
        setTask({ id: '', status: 'failed', error: d.error || `服务器错误 (${r.status})` });
        setUploading(false);
        return;
      }
      const taskId: string = d.id;
      if (!taskId) {
        setTask({ id: '', status: 'failed', error: '服务器返回异常' });
        setUploading(false);
        return;
      }
      setTask({ id: taskId, status: 'running' });
      setUploading(false);

      // 轮询状态
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
        } catch {
          // 轮询失败不中断，等下次重试
        }
      }, 500);
    } catch {
      if (mountedRef.current) {
        setTask({ id: '', status: 'failed', error: '网络错误，请检查连接后重试' });
        setUploading(false);
      }
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
