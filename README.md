# img2svg

AI 图片矢量化工具，将 AI 生成的流程图/架构图/示意图转换为可编辑的 SVG 矢量图，支持导出 EPS/PDF 用于科研论文和项目书。

## 技术栈

- **后端**: Go (net/http 标准库)
- **前端**: Vite + React + TypeScript + TailwindCSS
- **矢量化**: [vtracer](https://github.com/visioncortex/vtracer) (Rust CLI, Go subprocess 调用)
- **数据库**: SQLite (modernc.org/sqlite)
- **认证**: 共享 [pdf-translate-web](../pdf-translate-web/) 的用户表 (只读)

## 快速开始

```bash
# 1. 安装 vtracer
cargo install vtracer
# 或下载预编译二进制放到 bin/vtracer

# 2. 配置环境变量
cp .env.example .env
# 编辑 .env：COOKIE_SECRET（至少 32 字符）、TRANS_DB_PATH（trans 的 app.db 路径）

# 3. 构建前端
cd web && npm install && npm run build

# 4. 启动后端
cd .. && go run .
# 服务启动于 http://localhost:4003
```

### 前端开发

```bash
cd web
npm run dev       # http://localhost:5173，API 代理到 :4003
npm run build     # 构建到 web/dist
```

## 环境变量

| 变量 | 必需 | 说明 |
|------|------|------|
| `PORT` | 否 | 服务端口，默认 4003 |
| `DATA_DIR` | 否 | 数据目录，默认 ./data |
| `TRANS_DB_PATH` | 否 | trans 数据库路径，默认 ./data/app.db |
| `VTRACER_PATH` | 否 | vtracer 二进制路径，默认 ./bin/vtracer |
| `COOKIE_SECRET` | **是** | Session 签名密钥（≥32 字符） |

## 部署

GitHub Actions CI/CD：
1. push 到 `main` → 测试 → 构建前端 → 构建 Go → SCP 到服务器 → systemd 重启
2. vtracer 二进制在 CI 中从 GitHub Releases 下载

服务器路径：
- 二进制: `/usr/local/bin/img2svg`
- 发布目录: `/srv/img2svg/releases/<sha>/`
- 当前版本: `/srv/img2svg/current` (symlink)

## 项目结构

```
img2svg/
├── main.go                 # 入口
├── go.mod
├── internal/
│   ├── api/                # HTTP 路由、handler
│   ├── auth/               # 认证（读 trans users 表）
│   ├── config/             # 配置加载
│   ├── converter/          # vtracer 调用封装
│   ├── export/             # 格式导出（EPS/PDF）
│   ├── middleware/         # 日志/安全中间件
│   ├── models/             # 数据模型
│   ├── preprocess/         # 图像预处理
│   └── storage/            # SQLite 存储
├── web/                    # 前端 (React + Vite)
│   ├── src/
│   │   ├── components/     # Header, FileUpload, PreviewPanel, ParamsPanel
│   │   ├── pages/          # LoginPage, ConvertPage, HistoryPage
│   │   └── hooks/          # useAuth
│   └── vite.config.ts
├── deploy/                 # systemd + Nginx 配置
└── .github/workflows/      # CI/CD
```
