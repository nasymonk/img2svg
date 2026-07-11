import { useState } from 'react';

interface Task {
  id: string;
  status: string;
  error?: string;
}

interface Props {
  task: Task;
  onReset: () => void;
}

export default function PreviewPanel({ task, onReset }: Props) {
  const [sliderPos, setSliderPos] = useState(50);
  const [exporting, setExporting] = useState('');

  const isRunning = task.status === 'running' || task.status === 'pending';
  const isFailed = task.status === 'failed';
  const isDone = task.status === 'succeeded';

  const handleExport = async (format: string) => {
    setExporting(format);
    window.open(`/api/export/${task.id}/${format}`, '_blank');
    setTimeout(() => setExporting(''), 1000);
  };

  return (
    <div className="bg-surface border border-border rounded-xl p-6 space-y-4">
      <div className="flex items-center justify-between">
        <div>
          <span className={`inline-block w-2 h-2 rounded-full mr-2 ${
            isRunning ? 'bg-yellow-400 animate-pulse' :
            isFailed ? 'bg-red-400' :
            'bg-primary'
          }`} />
          <span className="text-sm text-text-secondary">
            {isRunning ? '转换中...' : isFailed ? '转换失败' : '转换完成'}
          </span>
        </div>
        <div className="flex gap-2">
          {isDone && (
            <>
              {['svg', 'eps', 'pdf'].map((fmt) => (
                <button
                  key={fmt}
                  onClick={() => handleExport(fmt)}
                  disabled={exporting === fmt}
                  className="px-3 py-1.5 bg-primary text-white rounded-lg text-sm font-medium hover:bg-primary-dark disabled:opacity-50 transition-colors"
                >
                  {exporting === fmt ? '下载中...' : `.${fmt}`}
                </button>
              ))}
            </>
          )}
          <button
            onClick={onReset}
            className="px-3 py-1.5 border border-border text-text-secondary rounded-lg text-sm hover:bg-bg transition-colors"
          >
            重新上传
          </button>
        </div>
      </div>

      {isFailed && (
        <div className="bg-red-400/10 border border-red-400/30 rounded-lg p-4 text-red-400 text-sm">
          {task.error || '转换失败，请重试'}
        </div>
      )}

      {isRunning && (
        <div className="flex items-center justify-center py-12">
          <div className="w-16 h-16 border-4 border-primary border-t-transparent rounded-full animate-spin" />
        </div>
      )}

      {isDone && (
        <ComparisonSlider
          sourceUrl={`/api/convert/${task.id}/source`}
          svgUrl={`/api/convert/${task.id}/preview`}
          sliderPos={sliderPos}
          onSliderChange={setSliderPos}
        />
      )}
    </div>
  );
}

function ComparisonSlider({
  sourceUrl, svgUrl, sliderPos, onSliderChange,
}: {
  sourceUrl: string; svgUrl: string; sliderPos: number; onSliderChange: (v: number) => void;
}) {
  return (
    <div className="space-y-2">
      <div className="relative overflow-hidden rounded-lg border border-border bg-white" style={{ aspectRatio: '16/10' }}>
        {/* 原图（底层） */}
        <img src={sourceUrl} alt="原图" className="absolute inset-0 w-full h-full object-contain p-2" />
        {/* SVG（上层，裁剪显示） */}
        <div
          className="absolute inset-0 overflow-hidden"
          style={{ width: `${sliderPos}%` }}
        >
          <div className="absolute inset-0 bg-white flex items-center justify-center p-2" style={{ width: `${100 / (sliderPos / 100)}%` }}>
            <img src={svgUrl} alt="SVG" className="max-w-full max-h-full w-full h-full object-contain" />
          </div>
        </div>
        {/* 分割线 */}
        <div className="absolute top-0 bottom-0 w-0.5 bg-white shadow-lg" style={{ left: `${sliderPos}%` }} />
        <div
          className="absolute top-1/2 -translate-y-1/2 w-8 h-8 bg-white rounded-full shadow-lg flex items-center justify-center cursor-ew-resize"
          style={{ left: `calc(${sliderPos}% - 16px)` }}
        >
          <svg className="w-4 h-4 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 9l4-4 4 4M16 15l-4 4-4-4" />
          </svg>
        </div>
        {/* 滑块拖动区域 */}
        <input
          type="range" min={0} max={100} value={sliderPos}
          onChange={(e) => onSliderChange(+e.target.value)}
          className="absolute inset-0 w-full h-full opacity-0 cursor-ew-resize"
        />
      </div>
      <div className="flex justify-between text-xs text-text-secondary">
        <span>原图</span>
        <span>SVG</span>
      </div>
    </div>
  );
}
