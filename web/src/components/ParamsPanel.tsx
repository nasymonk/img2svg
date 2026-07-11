export interface ConvertParams {
  colorCount: number;
  simplifyColors: number;
  mode: 'color' | 'binary';
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
          <label className="text-text-secondary block mb-1">色彩精度: {params.colorCount}</label>
          <input
            type="range" min={2} max={64} value={params.colorCount}
            onChange={(e) => update('colorCount', +e.target.value)}
            disabled={disabled}
            className="w-full accent-primary"
          />
          <p className="text-text-secondary/60 text-xs mt-0.5">越高色彩越丰富</p>
        </div>
        <div>
          <label className="text-text-secondary block mb-1">
            抗锯齿: {params.simplifyColors === 0 ? '关闭' : params.simplifyColors}
          </label>
          <input
            type="range" min={0} max={64} step={8} value={params.simplifyColors}
            onChange={(e) => update('simplifyColors', +e.target.value)}
            disabled={disabled}
            className="w-full accent-primary"
          />
          <p className="text-text-secondary/60 text-xs mt-0.5">
            消除 AI 图的杂色和锯齿，越小越干净
          </p>
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
      </div>
    </div>
  );
}
