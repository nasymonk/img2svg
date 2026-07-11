import { useState, useEffect } from 'react';
import { Link } from 'react-router-dom';

interface HistoryItem {
  id: string;
  original_name: string;
  status: string;
  progress: number;
  created_at: string;
}

export default function HistoryPage() {
  const [items, setItems] = useState<HistoryItem[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    fetch('/api/history')
      .then((r) => (r.ok ? r.json() : []))
      .then((data) => { setItems(data); setLoading(false); })
      .catch(() => setLoading(false));
  }, []);

  if (loading) {
    return (
      <div className="flex items-center justify-center py-16">
        <div className="w-8 h-8 border-2 border-primary border-t-transparent rounded-full animate-spin" />
      </div>
    );
  }

  if (!items.length) {
    return (
      <div className="text-center py-16">
        <svg className="w-12 h-12 text-text-secondary mx-auto mb-3" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10" />
        </svg>
        <p className="text-text-secondary">暂无转换记录</p>
        <Link to="/" className="text-primary text-sm mt-2 inline-block hover:underline">
          前往转换
        </Link>
      </div>
    );
  }

  return (
    <div className="bg-surface border border-border rounded-xl overflow-hidden">
      <table className="w-full text-sm">
        <thead>
          <tr className="border-b border-border text-text-secondary text-left">
            <th className="px-4 py-3 font-medium">文件名</th>
            <th className="px-4 py-3 font-medium">状态</th>
            <th className="px-4 py-3 font-medium">时间</th>
            <th className="px-4 py-3 font-medium">操作</th>
          </tr>
        </thead>
        <tbody>
          {items.map((item) => (
            <tr key={item.id} className="border-b border-border/50 hover:bg-bg/50 transition-colors">
              <td className="px-4 py-3 text-text">{item.original_name}</td>
              <td className="px-4 py-3">
                <span className={`inline-block px-2 py-0.5 rounded text-xs ${
                  item.status === 'succeeded' ? 'bg-primary/20 text-primary' :
                  item.status === 'failed' ? 'bg-red-400/20 text-red-400' :
                  'bg-yellow-400/20 text-yellow-400'
                }`}>
                  {item.status === 'succeeded' ? '完成' : item.status === 'failed' ? '失败' : '处理中'}
                </span>
              </td>
              <td className="px-4 py-3 text-text-secondary">
                {new Date(item.created_at).toLocaleString('zh-CN')}
              </td>
              <td className="px-4 py-3">
                {item.status === 'succeeded' && (
                  <a
                    href={`/api/export/${item.id}/svg`}
                    className="text-primary hover:underline"
                  >
                    下载 SVG
                  </a>
                )}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
