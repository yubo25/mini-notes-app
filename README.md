# Mini Notes with LLM Summary (Anna App)

这是一个基于 **Anna App 开发模型** 构建的本地笔记应用项目。

项目集成了：

- 前端工程化页面（React + Vite）
- Go 编写的本地 Executa Tool
- Anna Storage Host API
- Reverse Sampling（sampling/createMessage）
- JSON-RPC over stdio
- GitHub Actions 自动发布
- 三平台 Executa Binary 打包

---

# 1. 核心概念与技术关系

在 Anna App 中，各组件之间的关系如下：

## Manifest (`manifest.json`)

应用主配置文件（Schema 2）。

负责：

- 定义 View
- Bundle 入口
- Host API 权限
- Executa Tool 声明
- Permissions

---

## Bundle (`bundle/`)

由 React + Vite 构建得到。

运行时：

```
Anna Desktop
    ↓
Anna Runtime
    ↓
iframe
    ↓
bundle/index.html
```

---

## Executa Tool

独立运行的本地二进制插件。

通信方式：

```
stdin/stdout
        ↓
JSON-RPC 2.0
```

前端：

```
anna.tools.invoke(...)
```

宿主：

```
Executa Tool
```

---

## Storage / APS KV

Anna 提供的 Key-Value Storage。

本项目所有 Notes 均通过：

```
anna.storage.get(...)
anna.storage.set(...)
```

完成持久化。

---

## Reverse Sampling

Executa Tool **不会直接调用 OpenAI / Claude API**。

而是：

```
Executa

↓

sampling/createMessage

↓

Host LLM

↓

Summary
```

因此无需任何 LLM API Key。

---

## Binary Archive

Executa Tool 打包后的发布格式：

Windows：

```
.zip
```

macOS：

```
.tar.gz
```

压缩包根目录必须包含：

```
manifest.json
note-summarizer.exe
```

---

# 2. 项目目录

```text
mini-notes-app/
├── manifest.json
├── package.json
├── README.md
├── mock-sampling-fixture.jsonl
├── build_binary.sh
│
├── bundle/
│
├── ui/
│   ├── package.json
│   ├── vite.config.js
│   ├── index.html
│   └── src/
│       ├── main.jsx
│       ├── App.jsx
│       └── App.css
│
└── tool/
    ├── go.mod
    ├── main.go
    ├── executa.json
    └── note-summarizer.exe
```

---

# 3. 环境准备

安装：

- Node.js 22+
- Go 1.21+
- Git

安装 uv：

```powershell
winget install Astral-sh.uv
```

安装 Anna CLI：

```bash
npm install -g @anna-ai/cli
```

刷新 Windows PATH：

```powershell
$env:Path = [System.Environment]::GetEnvironmentVariable("Path","Machine") + ";" + [System.Environment]::GetEnvironmentVariable("Path","User")
```

检查：

```bash
anna-app --version
uv --version
go version
node --version
```

---

# 4. 构建前端 Bundle

进入前端目录：

```bash
cd ui
```

安装依赖：

```bash
npm install
```

构建：

```bash
npm run build
```

生成：

```
bundle/
    index.html
    assets/
```

---

# 5. Anna App 验证

## 5.1 Validate

项目根目录：

```bash
anna-app validate --strict
```

预期：

```
✓ validate passed
```

---

## 5.2 启动本地 Harness

```bash
anna-app dev --no-llm
```

浏览器打开：

```
http://localhost:5180
```

---

## 5.3 测试 Storage

创建 Note：

- 输入内容
- Save

删除：

- 点击 ×

观察 RPC Log：

```
storage.get
storage.set
```

说明：

Notes 已通过

```
anna.storage.*
```

保存。

---

### 关于刷新页面

由于：

```
anna-app dev
```

使用：

```
legacy runtime_state
```

刷新 iframe 后：

```
wid
```

重新生成。

因此：

Notes 会清空。

属于官方预期行为。

---

## 5.4 测试 Summary

点击：

```
✨ Summarize Notes
```

由于：

```
--no-llm
```

应得到：

```
[-32603]
harness started with --no-llm
```

RPC Log 应看到：

```
tools.invoke
```

说明：

```
UI

↓

Host API

↓

Executa Tool
```

路由正常。

---

# 6. Executa Tool 独立测试

进入：

```bash
cd tool
```

编译：

```bash
go build -o note-summarizer.exe main.go
```

启动：

```bash
anna-app executa dev \
--mock-sampling ../mock-sampling-fixture.jsonl
```

---

## describe

输入：

```
describe
```

返回：

- host_capabilities
- tools
- parameters

---

## invoke

输入：

```text
invoke summarize {"notes":["Fix_auth_bug","Write_persistence"]}
```

应看到：

```
sampling/createMessage
```

说明：

Reverse Sampling 已成功发起。

---

# 7. Executa Binary 打包

Linux / macOS：

```bash
sh build_binary.sh --all
```

Windows：

```powershell
go build -o .\dist\temp\note-summarizer.exe .\tool\main.go
```

生成：

```
note-summarizer-0.1.0-windows-x86_64.zip
```

压缩包结构：

```text
manifest.json
note-summarizer.exe
```

---

# 8. GitHub Actions

工作流：

```
.github/workflows/release.yml
```

支持：

- workflow_dispatch
- Git Tag

自动：

- Build
- Smoke Test
- Upload Release Assets

生成：

```
darwin-arm64.tar.gz

darwin-x86_64.tar.gz

windows-x86_64.zip
```


