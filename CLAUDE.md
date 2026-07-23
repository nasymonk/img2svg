# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

AI 图片矢量化工具，将 AI 生成的流程图/架构图/示意图转换为可编辑的 SVG，支持导出 EPS/PDF。

## 技术栈

- **后端**: Go 1.24+(`net/http` 标准库)，单入口 `main.go`
- **前端**: Vite + React + TypeScript + TailwindCSS(`web/`)
- **矢量化**: ImageMagick(`convert`)预处理 + `potrace` 多色分层描边
- **数据库**: SQLite(`modernc.org/sqlite`)，仅用于任务元数据
- **认证**: 无

## 高频命令

```bash
# 本地运行（先构建前端）
cd img2svg/web && npm install && npm run build && cd ..
go run .

# 测试
go test ./...

# 前端开发
npm run dev       # http://localhost:5173，代理到 :4003
```

## 关键架构

### 矢量化管线

真实转换在 `internal/converter/converter.go`:

1. `convert`(ImageMagick)做 posterize + median 去噪。
2. 按颜色数自动选择 posterize 级别(`<64` 0 级 / `<256` 6 级 / `<1024` 5 级 / 否则 4 级)。
3. 每层调用 `potrace` 描边，合并为多色分层 SVG。
4. `internal/export/export.go` 负责 EPS/PDF 导出。

`potrace` 与 `convert`(ImageMagick)是仅有的外部依赖，启动时通过 `CheckDeps()` 做可用性检查。

### 目录要点

- `main.go` — 唯一入口，嵌入 `web/dist`，并启动 24 小时临时文件清理协程。
- `internal/api/` — HTTP 路由与 handler。
- `internal/config/` — 配置加载。
- `internal/converter/` — ImageMagick + potrace 转换核心。
- `internal/preprocess/` — 图像预处理。
- `internal/export/` — EPS/PDF 导出。
- `internal/storage/` — SQLite 任务存储与文件操作。
- `data/tmp/` — 临时上传/输出目录，自动清理 24 小时前文件。

## 环境变量

| 变量 | 必需 | 默认值 | 说明 |
|------|------|--------|------|
| `PORT` | 否 | `4003` | 服务端口 |
| `DATA_DIR` | 否 | `./data` | 数据/临时目录 |

## 部署

目标服务器：ECS `8.130.214.67`(`ssh ecs`)，systemd 服务 `img2svg.service`，域名 `svg.rootfly.xyz`。

GitHub Actions `deploy.yml` 流程：go test → 前端构建 → go build → scp 到 ECS → systemd 重启。每次部署保留最近 3 个 release，旧版本自动清理。

服务器路径：
- 二进制: `/usr/local/bin/img2svg`
- 发布目录: `/srv/img2svg/releases/<sha>/`
- 当前版本: `/srv/img2svg/current`(symlink)

## 修改指引

- 改矢量化效果 → `internal/converter/converter.go` 与 `internal/preprocess/preprocess.go`。
- 改前端 → `web/src/`。
