# 简历编辑器 · resume-editor

一个基于 [mygo](https://github.com/egoist/mygo) 的桌面简历编辑器，内置固定双页 A4 简历模板：左侧模块导航、中间字段表单、右侧实时预览，编辑后自动保存，可一键导出 PDF。

- **技术栈**：Go 1.27 + mygo v0.9.0（WebView2） + 原生 HTML / CSS / JavaScript。
- **零前端依赖**：不需要 `npm install`、Vite、Bun 或打包器，构建脚本只用 Node 标准库。
- **一套前端两种运行方式**：mygo 桌面模式调用 Go 服务读写文件；浏览器直接打开 `dist/index.html` 时自动降级为纯前端模式。
- 界面与目录划分参考 [MyGit](https://github.com/Binaryify/MyGit)：根级负责装配与窗口，`internal/` 负责领域逻辑，`frontend/` 按界面区域拆分。

---

## 1. 环境要求

| 依赖 | 版本 / 说明 |
| --- | --- |
| Go | 1.27.4+（`go.mod` 中 `go 1.27.4`，并用 `tool` 指令固定 mygo v0.9.0） |
| Node.js | 22+，仅用于执行两个 `.mjs` 脚本与让 mygo 评估配置 |
| mygo CLI | `go tool mygo ...` 会直接使用 `go.mod` 固定版本；全局 CLI 可选：`go install github.com/egoist/mygo/cmd/mygo@latest`（用于 `mygo doctor`） |
| 操作系统 | Windows（编译产物输出到 `build/windows-amd64/`，需 WebView2，Win10/11 一般已自带） |
| Edge / Chrome | 导出 PDF 时通过无头打印调用，`internal/resume/render.go` 的 `FindBrowser()` 按固定路径 + PATH 查找 |
| Bun | **不需要**，项目刻意不使用 |

> 若 Go 模块走官方代理失败，可设 `GOPROXY=https://goproxy.cn,direct` 与 `GOSUMDB=sum.golang.google.cn`。

---

## 2. 快速开始

在项目根目录执行（PowerShell / Git Bash 均可）：

```bash
# 1) 生成静态前端（同时把 docs/ 原稿同步到 assets/）
node scripts/build-web.mjs

# 2) 校验（字段双向一致性、写回格式、PDF 页数）
go test ./...

# 3) 编译桌面应用
go tool mygo build -skip-installer
```

等价的 npm 脚本（**无需 `npm install`**）：

```bash
npm run web      # = node scripts/build-web.mjs
npm run test     # = go test ./...
npm run build    # = go tool mygo build -skip-installer
npm run dev      # = npm run web && go tool mygo dev（启动桌面开发窗口）
npm run schema   # = node scripts/gen-schema.mjs（仅改动模板字段后需要）
```

`mygo.json` 的 `buildCommand` 已指向 `node scripts/build-web.mjs`，所以 `go tool mygo build` 会**自动先执行前端构建**，直接跑第 3 步也可以。

### 产物

| 路径 | 说明 |
| --- | --- |
| `dist/index.html` | 静态前端成品，可直接用浏览器打开（降级模式） |
| `build/windows-amd64/resume-editor.exe` | 桌面应用，文件名由 `mygo.json` 的 `name` 决定 |
| `%APPDATA%\resume-editor\简历内容.md` | 桌面模式下用户实际编辑的内容文件 |
| `%APPDATA%\resume-editor\output\` | 导出结果：`成品简历.pdf`，以及 Edge 打印所需的中间文件 `成品简历.html` |

> **命名说明**：`mygo.json` 的 `name` 决定 exe 文件名，`windows.displayName` 写入 exe 版本资源、决定进程名与任务管理器显示名；`slug` / `identifier` 用于安装目录与应用标识。当前三者均为 `resume-editor`，因此编译产出 `resume-editor.exe`。应用内界面文字保持中文，与这些名称无关。
>
> **版本控制**：`dist/`、`build/`、`mygo.ts` 都是生成物，已由 `.gitignore` 排除、**不入库**；干净 clone 后按上面三步执行即可得到它们。仓库必须保留 `assets/`（`//go:embed` 的编译输入）与 `docs/`（模板与内容原稿），否则 `go build` / `go test` 会因缺少内嵌资源而失败。

可用环境变量 `RESUME_EDITOR_HOME` 覆盖数据目录（便携模式、测试隔离）。

---

## 3. 项目结构

```
resume_polishing/
├── README.md                    本文件：结构 / 编译 / 接续开发
├── mygo.json                    mygo 应用配置（名称、版本、构建命令、输出目录）
├── package.json                 npm 脚本，无任何依赖
├── go.mod / go.sum              Go 模块，tool 指令固定 mygo v0.9.0
├── mygo.ts                      [自动生成] 前端 TypeScript 绑定，勿手改
│
│  ── 装配层（根级 .go，对标 MyGit 的 titlebar.go / main.go）──
├── main.go                      入口：绑定服务、创建主窗口
├── titlebar.go                  隐藏原生标题栏的 WindowOptions
├── assets.go                    go:embed 资源、数据目录、服务装配
├── app_test.go                  根级集成测试
├── service_test.go              端到端服务测试（含 PDF 页数）
│
├── internal/resume/             领域层：纯逻辑，不依赖窗口 / 资源路径
│   ├── document.go              Markdown 解析、53 字段校验、写回
│   ├── render.go                模板填充、Edge 无头打印 PDF
│   ├── service.go               Resume 服务（mygo.Bind 暴露给前端）
│   └── resume_test.go           纯逻辑单测
│
├── frontend/                    编辑器源码
│   ├── index.html               结构骨架：顶栏 / 侧栏 / 编辑区 / 预览
│   ├── app.js                   数据、表单、预览、保存的编排
│   ├── components/titlebar.js   顶栏模块切换 + 自绘窗口控制按钮
│   ├── components/panes.js      两条中线分隔条 + 侧栏折叠
│   └── styles/                  base / titlebar / sidebar / workspace
│                               按此顺序拼接成 dist/style.css
│
├── scripts/
│   ├── gen-schema.mjs           模板 data-f → assets/schema.json（字段元数据）
│   └── build-web.mjs            docs → assets 同步 + 生成 dist/
│
├── docs/                        ★ 人编辑的唯一真相源
│   ├── 简历内容.md              纯数据：6 个模块 + 53 个 @@字段块
│   ├── 简历模板.html            唯一版式原稿（A4、CSS、栏目文案、data-f 占位）
│   ├── 内容文档规范.md          字段语法、版式刻度、填写与构建规则
│   └── 求职档案.md 等           参考资料
│
├── assets/                      [同步/生成] go:embed 副本：模板、内容 md、schema.json
├── resources/icon.png           应用图标
├── dist/                        [生成] 静态前端
├── build/                       [生成] 可执行文件
└── .agents/                     项目内 mygo 维护技能说明
```

**标注 `[生成]` 的目录不要手工编辑**：`assets/` 由 `build-web.mjs` 从 `docs/` 同步，`dist/` 由构建脚本整体重建，`mygo.ts` 由 mygo 编译时生成。`.workbuddy/` 是项目数据目录，不要删除。

---

## 4. 架构与数据流

```
docs/简历模板.html ──┐
docs/简历内容.md   ──┼─→ build-web.mjs ─→ assets/ ──→ go:embed (assets.go)
                    │                          └─→ internal/resume ─→ mygo.Bind
                    │                                                   ↓
                    └─→ dist/data.js ──→ 浏览器降级模式            window.mygo.call
                                                                       ↓
                                                              frontend/app.js
```

- **桌面模式**：`app.js` 的 `mygoBackend` 通过 `window.mygo.call("Resume.*")` 调用 Go；Go 服务用 `go:embed` 中的模板与默认内容，用户数据落在数据目录的 `简历内容.md`。
- **浏览器降级模式**：`browserBackend` 用 `dist/data.js` 内嵌的模板 / schema / 初始内容，内容存 `localStorage`（key `resume-editor-content`），导出 PDF 用 `window.open` + `print()`。分栏偏好另存为 `resume-editor-panes`。
- **两种模式共用同一套渲染规则**：Go 的 `FillTemplate`（`internal/resume/render.go`）与 JS 的 `fillTemplate`（`frontend/app.js`）都是扫描 `data-f="..."` 并替换占位元素，**改动其一必须同步其二**，否则预览与导出结果会不一致。
- **前端模块解耦**：`components/*.js` 与 `app.js` 只通过 `document` 自定义事件通信：`resume:groups`、`resume:group`、`resume:select-group`、`resume:pane-resize`。
- **Go 服务契约**：类型名必须是 `Resume`（`mygo.Bind` 据此生成 `Resume.*`）。当前 5 个导出方法：`Load` / `Fill` / `Save` / `Export` / `Info`。内部辅助方法一律小写不导出，否则会被误暴露成前端 API。

---

## 5. 如何接续开发

### 5.1 按改动场景找文件

| 想做什么 | 改哪里 | 之后必跑 |
| --- | --- | --- |
| 只换简历文字 | `docs/简历内容.md` | `npm run web` |
| 改版式 / 配色 / 分页 / 栏目标题 | `docs/简历模板.html` | `npm run web && npm run test` |
| 增删字段 | `docs/简历模板.html` 的 `data-f` + `docs/简历内容.md` 的 `@@` 块 | `npm run schema && npm run web && npm run test` |
| 改字段中文标签 / 分组 / 控件类型 | `scripts/gen-schema.mjs` 的 `LABELS` / `GROUPS` / `typeOf` | `npm run schema && npm run web` |
| 改界面样式 | `frontend/styles/*.css`（拼接顺序：base → titlebar → sidebar → workspace，后者覆盖前者） | `npm run web`，浏览器打开 `dist/index.html` 看效果 |
| 改顶栏交互 / 窗口按钮 | `frontend/components/titlebar.js` + `frontend/index.html` | `npm run web` |
| 改分栏宽度 / 折叠 | `frontend/components/panes.js` + `frontend/styles/workspace.css`、`base.css` 的宽度变量 | `npm run web` |
| 改表单、预览、保存逻辑 | `frontend/app.js` | `npm run web && npm run test` |
| 加 Go 服务方法 | `internal/resume/service.go`（及 `document.go` / `render.go`） | `go test ./... && go tool mygo build -skip-installer` |
| 改窗口尺寸 / 标题 / 顶栏高度 | `main.go`、`titlebar.go`，并同步 `base.css` 的 `--titlebar-h`（当前 40px） | `npm run build` |
| 改应用名 / 版本 / 输出目录 | `mygo.json` | `npm run build` |
| 改字段规则、填写规范、版式说明 | `docs/内容文档规范.md`（**不要**写回 `简历内容.md`，它是纯数据文件） | 无 |

### 5.2 几条必须遵守的约定

1. **`docs/` 是唯一真相源**。改完必须跑 `npm run web`，脚本会把 `docs/简历模板.html`、`docs/简历内容.md` 同步到 `assets/` 并重建 `dist/`；直接改 `assets/` 会在下次构建时被覆盖。
2. **53 字段双向校验**。`build-web.mjs` 与 Go 的 `Validate()` 都会比对模板 `data-f` 集合与内容文档 `@@` 字段集合，缺一个或多一个都会直接报错。`app_test.go` 也写死了 `53`，增删字段时要同步这个断言。
3. **内容文档保持纯数据**：只有 `## 模块…` 标题、`@@字段ID` 和值行，不要加说明、注释或字段索引。编辑器保存时会按 `schema.json` 顺序重写同格式 Markdown，测试里有「写回结果与原稿逐字一致」的断言。
4. **模板占位元素的写法**：带 `data-f` 的标签必须有对应的闭合标签（扫描器按标签名查找 `</tag>`）；`data-multiline` 字段每行渲染成独立一行；`photo` 字段支持 `![证件照](./photo.jpg)` 语法。
5. **分页是固定的**：模板内部两页各自独立容器 + `overflow:hidden` + 显式分页，第 1 页放个人信息 / 个人优势 / 工作经历 / 项目 1~4，第 2 页放项目 5~6 / 教育背景 / 资格证书。描述过长时优先精简文字，不要改间距破坏分页。
6. **不要给 `.sidebar` / `.preview` 加 `transition:width`**：无头浏览器 / 虚拟时间环境下过渡会冻结，导致折叠后宽度不生效（已踩过坑，现在改为即时生效）。
7. **两种运行方式都要自测**：桌面模式改 Go 后要编译；浏览器模式改前端后直接开 `dist/index.html` 即可，不必编译 exe。

### 5.3 加一个 Go 服务方法的完整流程

1. 在 `internal/resume/service.go` 添加导出方法（参数 / 返回值用可 JSON 序列化的类型）。
2. 在 `frontend/app.js` 的 `mygoBackend` 加 `window.mygo.call('Resume.Xxx', ...)`；同时在 `browserBackend` 提供等价实现，否则降级模式会报错。
3. 在 `app_test.go` / `service_test.go` 补断言。
4. 跑 `gofmt -w . && go test ./...`；`go tool mygo build -skip-installer` 会重新生成 `mygo.ts`，核对里面的方法名与前端调用一致。

---

## 6. 验证清单

```bash
gofmt -l .                  # 应无输出
go vet ./...
go test ./...               # 或 go test -count=1 -v ./... 强制重跑
node scripts/build-web.mjs  # 会打印字段数，应为 53
go tool mygo build -skip-installer
```

测试覆盖的关键点（`app_test.go` / `service_test.go` / `internal/resume/resume_test.go`）：

- 内嵌资源可读取、schema 与内容文档均为 53 个字段且与模板 `data-f` 完全一致；
- `WriteMarkdown` 输出与 `docs/简历内容.md` 逐字一致（不含说明、`::` 注释）；
- `FillTemplate` 的 HTML 转义、`class="filled"` 成品标记；
- 自定义顶栏配置（`TitleBarHidden` / `HideWindowButtons` / 高度 40）；
- 「`Load → Save → Fill → Export`」端到端，用 `RESUME_EDITOR_HOME` 隔离到临时目录，校验 PDF 恰为 **2 页**；本机无 Edge/Chrome 时该段自动跳过。

需要目视验证界面时，可用 Edge 无头截图（不需要装额外工具）：

```bash
msedge --headless=new --disable-gpu --no-sandbox \
  --user-data-dir="$TEMP/resume-shot" --window-size=1440,920 \
  --force-device-scale-factor=1 --virtual-time-budget=6000 \
  --screenshot=editor.png "file:///E:/Projects/WorkBuddy/resume_polishing/dist/index.html"
```

---

## 7. 常见问题

| 现象 | 原因 / 处理 |
| --- | --- |
| 编译时提示目录被占用 | 应用正在运行，关闭窗口后重试；或用 `go tool mygo build -skip-installer -o build-fresh` 编译到独立目录再替换 |
| `go tool mygo` 拉取依赖失败 | 配置 `GOPROXY=https://goproxy.cn,direct`、`GOSUMDB=sum.golang.google.cn` |
| 导出 PDF 报「未找到 Edge / Chrome」 | 在 `internal/resume/render.go` 的 `FindBrowser()` 候选路径里补上本机浏览器路径 |
| 浏览器模式点「导出 PDF」无反应 | 打印窗口被拦截，允许本站弹出窗口后重试；打印对话框选 A4 / 缩放 100% / 勾选「背景图形」，否则深蓝横条不会打印 |
| 改了 `docs/` 但界面没变化 | 忘了跑 `npm run web`，或页面缓存了旧的 `data.js`（强制刷新） |
| 预览与导出结果不一致 | Go 的 `FillTemplate` 与 JS 的 `fillTemplate` 扫描规则被改得不同步 |
| 新增字段后应用报错 | 三处没对齐：模板 `data-f`、内容文档 `@@`、`schema.json`（需 `npm run schema`）；`app_test.go` 的 53 也要改 |
| 内容文档越看越像说明文档 | 只应改 `docs/简历内容.md` 的值行；规则写在 `docs/内容文档规范.md` |

---

## 8. 相关文档

- `docs/内容文档规范.md` — 字段语法、53 项校验、特殊字段、A4 双页版式刻度、填写规范、编辑器交互。
- `docs/简历模板.html` — 版式与样式的唯一实现，`#244664` 主色、楷体页眉、A4 打印规则都在这里。
- `.agents/skills/` — 项目内附带的 mygo 维护规范（升级 mygo、改 UI、依赖变更时的注意事项）。
