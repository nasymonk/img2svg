export interface ConvertParams {
  colorCount: number;
  mode: 'color' | 'binary';
  denoise: boolean;
  sharpen: boolean;
  transparentBg: boolean;
}

interface Props {
  params: ConvertParams;
  onChange: (p: ConvertParams) => void;
  disabled: boolean;
}

export default function ParamsPanel({ params, onChange, disabled }: Props) {
  const update = <K extends keyof ConvertParams>(k: K, v: ConvertParams[K]) =>
    onChange({ ...params, [k]: v });

  return (
    <div className="bg-surface border border-border rounded-xl p-4">
      <h3 className="text-sm font-medium text-text-secondary mb-3">转换参数</h3>
      <div className="grid grid-cols-2 md:grid-cols-3 gap-4 text-sm">
        <div>
          <label className="text-text-secondary block mb-1">颜色数: {params.colorCount}</label>
          <input
            type="range" min={2} max={64} value={params.colorCount}
            onChange={(e) => update('colorCount', +e.target.value)}
            disabled={params.mode === 'binary' || disabled}
            className="w-full accent-primary"
          />
        </div>
        <div>
          <label className="text-text-secondary block mb-1">模式</label>
          <select
            value={params.mode}
            onChange={(e) => update('mode', e.target.value as ConvertParams['mode'])}
            disabled={disabled}
            className="w-full bg-bg border border-border rounded-lg px-2 py-1.5 text-text text-sm"
          >
            <option value="color">彩色</option>
            <option value="binary">黑白</option>
          </select>
        </div>
        <div className="flex flex-col gap-1.5">
          <label className="flex items-center gap-2 text-text cursor-pointer">
            <input type="checkbox" checked={params.denoise} onChange={(e) => update('denoise', e.target.checked)} className="accent-primary" disabled={disabled} />
            去噪
          </label>
          <label className="flex items-center gap-2 text-text cursor-pointer">
            <input type="checkbox" checked={params.sharpen} onChange={(e) => update('sharpen', e.target.checked)} className="accent-primary" disabled={disabled} />
            锐化
          </label>
          <label className="flex items-center gap-2 text-text cursor-pointer">
            <input type="checkbox" checked={params.transparentBg} onChange={(e) => update('transparentBg', e.target.checked)} className="accent-primary" disabled={disabled} />
            透明背景
          </label>
        </div>
      </div>
    </div>
  );
}
