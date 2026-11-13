# 简历编辑器 · resume-editor

基于 Go 和 mygo 的桌面简历编辑器，面向 Windows、Ubuntu 和 macOS。左侧按模块导航，中间编辑字段，右侧实时预览；内容自动保存，可导出固定双页 A4 PDF。

前端使用原生 HTML、CSS、JavaScript ES 模块，只通过 mygo 调用 Go 服务。模板解析、字段校验与渲染全部由 Go 负责。

顶栏通过「内容模块 / 文件管理」分段控件切换工作区。内容模块侧栏展示各模块的填写进度；文件管理可以新建 JD 简历、切换文件、修改对应的职位描述。新建时可复制当前简历，或使用默认模板。文件规则、内容复制、完成度计算和上次打开的文件记忆均由 Go 服务负责。

通用版继续保存为 `简历内容.md`；JD 版本使用 `简历_公司_岗位.md`，未填写公司时使用「未指定公司」，重名时追加 `_02`、`_03` 等序号。文件名中的非法字符由后端替换，已有文件不会被新建操作覆盖。切换文件前会先保存待写入内容，自动保存始终绑定明确的文件名。

JD 版本在 Markdown 首行保存 `:: resume-profile {"company":"公司","role":"岗位","jd":"职位描述"}`，其后仍为原有 `@@字段ID` 正文。JD 的换行通过 JSON 转义，元数据不会参与简历预览和 PDF 渲染。修改通用版的 JD 时也会写入同样的文件头；不含 JD 的原有通用文件保持原格式。

## 开发与构建

开发需要 Go 1.27.4+、Node.js 22+。运行环境与构建产物如下：

| 平台 | 运行依赖 | 构建产物 |
| --- | --- | --- |
| Windows 10/11 x64 | WebView2 Runtime | `build/windows-amd64/resume-editor.exe` |
| Ubuntu 22.04+ x64 | GTK 3、WebKitGTK 4.1、中文字体 | `build/linux-amd64/` 下的可执行文件、`.deb` 和安装归档 |
| macOS 15+，Apple Silicon | 系统自带 WKWebView | `build/darwin-arm64/resume-editor.app`，在 macOS 上另生成 `.dmg` |

Ubuntu 的 WebView 和中文字体依赖可用以下命令安装，`.deb` 包也声明了这些依赖：

~~~bash
sudo apt update
sudo apt install libwebkit2gtk-4.1-0 fonts-noto-cjk
~~~

~~~powershell
npm run web       # Go 生成资源和字段定义，Node 复制前端模块、合并样式
npm run dev       # 构建前端并启动 mygo 开发窗口
npm run check     # JS 语法检查、Go 静态检查，不执行测试
npm run build         # 构建当前系统与架构
npm run build:ubuntu  # 构建 Linux x64 和 Ubuntu .deb
npm run build:macos   # 构建 macOS 15+ Apple Silicon app
npm run build:all     # 构建上述所有目标及 Windows x64
~~~

构建命令仅使用 Go 和 Node 标准库，不需要先安装 npm 依赖。mygo 的版本通过 go.mod 固定；桌面构建会自动执行前端构建。

可以从 Windows 交叉构建 Ubuntu 可执行文件、`.deb` 和 macOS `.app`。macOS 签名与 `.dmg` 制作需要在 Mac 上运行；正式分发时可配置 `macos.signingIdentity` 和公证信息，默认使用 mygo 的临时签名方式。

`.github/workflows/build.yml` 分别在 Windows、Ubuntu 和 macOS 执行格式检查、静态检查及桌面构建，并保存构建产物；不执行测试，也不发布版本。

统一格式使用可选开发工具 Prettier，版本由 package-lock.json 锁定：

~~~powershell
npm ci
npm run format
npm run format:check
~~~

Go 文件使用 gofmt。编辑器缩进与换行约定见 .editorconfig，JS/CSS/HTML 格式见 .prettierrc.json。

### Windows → WSL Ubuntu 安装与启动

Windows 已安装 WSL2、WSLg 和 Ubuntu x64，并为 Ubuntu 配置普通默认用户后，可以一条命令完成构建、安装和打开界面：

~~~powershell
npm run wsl
npm run wsl -- --skip-build             # 打包已完成，直接安装最新的 Ubuntu x64 .deb
npm run wsl -- --distro Ubuntu-24.04    # 指定 WSL 发行版，默认 Ubuntu
npm run wsl -- --package "E:/产物/resume-editor_0.1.0_amd64.deb"
~~~

脚本通过 WSL 的 root 用户安装 `.deb` 及包声明的依赖，不需要输入 sudo 密码；应用以 Ubuntu 的普通默认用户运行。更新前会关闭该用户运行的已安装版，安装位置为 `/opt/resume-editor/`，用户简历和导出文件仍使用 Ubuntu 中的原数据目录。安装阶段会补齐程序执行权限，兼容 mygo 0.9 在 Windows 上交叉打包时记录的文件权限。脚本等待编辑器加载完成后退出，WSLg 窗口继续运行。`--skip-build` 会按修改时间选择最新的安装包；`--package` 指定文件并跳过构建。

如果窗口只出现在任务栏、标题带 `[WARN:COPY MODE]`，脚本会检测 WSLg 的共享内存图形通道并停止。关闭其他 Linux 图形应用后，可用以下命令自动重建故障会话，再安装和启动：

~~~powershell
npm run wsl -- --skip-build --repair-wslg
~~~

`--repair-wslg` 只在图形通道故障时重建 WSLg，会关闭所有 Linux 图形窗口，不会终止 Ubuntu 后台服务。安装使用 Ubuntu 已配置的 APT 软件源；首次安装缺少包索引时自动刷新索引，软件源不可访问时需先修复软件源再重试。

应用使用独立进程启动，脚本退出后继续运行，Windows 下可能额外显示一个 WSL 终端窗口。启动输出保存在 `~/.local/state/resume-editor/wsl-launch.log`（设置 `XDG_STATE_HOME` 时使用其对应目录），每次由脚本启动都会覆盖；应用自身的按日日志仍保存在数据目录的 `logs/`。`npm run wsl -- --help` 可查看全部参数。

## 目录与职责

~~~text
main.go                         启动、日志生命周期、服务绑定、窗口创建
assets.go                       内嵌资源读取、数据路径、服务装配
titlebar.go / windowsize.go      原生窗口配置
pdf.go                          隐藏 WebView 的加载、原生 PDF 打印与释放
internal/
  apperr/errors.go               全项目错误码、安全提示和错误封装
  logging/logging.go             JSON 日志初始化与统一失败记录
  resume/
    config.go                   资源、存储路径和应用元数据类型
    document.go                 Markdown 解析与序列化
    schema.go                   字段和分组定义、schema 生成
    template.go                 唯一字段扫描器、模板编译、校验与渲染
    export.go                   PDF 渲染函数类型、中间 HTML 与成品 PDF 存储
    service.go                  Load/Fill/Save/Export/Info/Log 服务编排
cmd/resume-assets/main.go       使用运行时规则生成构建资源
frontend/
  app.js                        应用装配和启动
  backend.js                    唯一 mygo 桥接适配器
  platform.js                   macOS Command / 其他平台 Ctrl 快捷键约定
  content.js                    文档快照，兼容系统 WebView
  actions.js                    自动保存和导出流程
  errors.js / logger.js         前端错误归一化与结构化日志
  dom.js                        DOM 查询和 CSS 数值读取
  contracts.js / data.d.ts       模块契约与生成数据声明
  components/                   editor / preview / titlebar / panes / notices
  styles/                       全局设计变量及各区域样式
scripts/                        前端构建、资源生成、语法检查与 Windows → WSL 部署
docs/                           人编辑的内容、模板、字段配置与填写规范
assets/                         提交到仓库的生成资源，供 go:embed 使用
dist/                           生成的前端模块及 data.js
build/                          桌面构建产物
~~~

Go 服务在创建时显式接收 Resources、StoragePaths、PDF 渲染函数和日志实例。模板只编译一次，预览和 PDF 导出复用同一个 Template。简历模块负责输出文件存储，桌面适配器接收中间 HTML 路径并返回 PDF 字节，集中管理原生窗口生命周期。

前端组件通过参数和回调协作。界面模块不直接调用 mygo，不使用全局模块注册表或自定义事件总线。各组件保管自己的界面状态和定时器，app.js 负责组装协作关系。

## 数据与构建流程

人编辑的源文件：

- docs/简历内容.md：默认内容，只有模块标题、@@字段 ID 和值行。
- docs/简历模板.html：版式及 data-f 字段，字段顺序和多行属性以模板为准。
- docs/editor.json：分组名称、英文名、说明，以及每个字段的标签、控件类型和可选提示。
- mygo.json：应用名称和版本的唯一配置来源。

每次 npm run web 都会调用 cmd/resume-assets，使用 Go 中的同一套解析、扫描和校验规则：

~~~text
docs/内容 + 模板 + editor.json + mygo.json
                  ↓
          cmd/resume-assets
                  ↓
     assets/模板、内容、schema.json、app.json
                  ↓
        dist/data.js + 前端 ES 模块
~~~

schema 会自动更新，无需记住单独的生成步骤。npm run schema 保留为只生成资源的便捷入口。不要直接编辑 assets/、dist/、build/ 或生成的 mygo.ts。

mygo.ts 是 mygo 自动生成的客户端参考文件；原生 JS 前端的调用集中在 backend.js，模块数据结构由 contracts.js 的 JSDoc 描述。

## 扩展入口

| 改动 | 位置 |
| --- | --- |
| 修改默认内容 | docs/简历内容.md |
| 新增字段 | 模板 data-f、内容 @@ 块、docs/editor.json 的 fields |
| 新增分组 | docs/editor.json 的 groups，并为字段指定 group |
| 修改标签、控件或提示 | docs/editor.json，支持 text / textarea / lines |
| 修改版式、配色或分页 | docs/简历模板.html |
| 扩展字段渲染规则 | internal/resume/template.go |
| 增加简历操作 | internal/resume/service.go、frontend/backend.js、actions.js |
| 修改编辑控件或预览 | frontend/components/editor.js 或 preview.js |
| 修改分栏几何 | frontend/styles/base.css 的 CSS 变量 |
| 修改窗口尺寸策略 | windowsize.go |
| 增加错误码 | internal/apperr/errors.go；重新构建后自动同步到前端 |

布局数值由 CSS 变量提供，JS 读取这些变量。窄窗口规则统一放在 workspace.css，分栏组件读取实际布局状态，不再重复声明媒体断点。原生标题栏高度由 mygo 提供给前端。

标题栏遵循 MyGit 的做法：保留各平台原生窗口按钮，用 mygo 的 `--mygo-titlebar-*` 变量避让其空间。macOS 使用红绿灯按钮，Ubuntu 使用桌面主题指定的按钮布局，页面只绘制工具栏。系统隐藏按钮或进入全屏时，CSS 仍保留工具栏的最小内容高度。

快捷键在 macOS 上为 `⌘S` 保存、`⌘B` 收起侧栏，其他平台为 `Ctrl+S`、`Ctrl+B`。正文与界面提供系统中文字体及 Noto CJK 回退；不同系统字体和 WebView 的排版可能略有不同，分享时以导出的 PDF 为准。

## 数据目录、错误与日志

默认数据目录遵循系统的用户配置目录；可用环境变量 `RESUME_EDITOR_HOME` 覆盖，适用于便携运行和开发隔离：

| 平台 | 默认目录 |
| --- | --- |
| Windows | `%APPDATA%/resume-editor` |
| Ubuntu | `$XDG_CONFIG_HOME/resume-editor`，未设置时为 `~/.config/resume-editor` |
| macOS | `~/Library/Application Support/resume-editor` |

~~~text
数据目录/
  简历内容.md
  简历_公司_岗位.md
  active-document.json          # 上次打开的文件名；不保存简历正文或 JD
  output/成品简历.html
  output/成品简历.pdf
  logs/YYYY-MM-DD.jsonl
~~~

日志文件按启动日期命名并追加写入，同时输出到标准错误流。记录启动/关闭、内容加载/保存、导出起止与耗时、前端错误等事件；不记录字段值、简历正文或联系方式。前端日志写入开发者控制台，并通过 Resume.Log 转发到同一桌面日志文件。

PDF 导出通过 mygo 的 `Page.PrintToPDF` 完成，无需安装 Chrome、Edge 或 Chromium 浏览器，也无需配置浏览器路径。Windows 使用 WebView2 Runtime，Ubuntu 使用 WebKitGTK 的打印到文件能力，macOS 使用系统 WKWebView 打印能力；这些运行时同时用于应用界面。Windows 的 WebView2 内部仍使用 Chromium 内核。

每次导出创建隐藏的临时 WebView，加载 `output/成品简历.html`，等待字体与图片就绪后按 A4、零页边距、保留背景色打印，并在结束时释放窗口。相对图片路径以 `output/` 为基准。PDF 和预览的排版均由所在平台的 WebView 渲染。

错误码和安全中文提示只在 internal/apperr/errors.go 定义。Go 错误保留原始 cause 供本地日志排查；mygo v0.9 的错误桥接采用字符串，因此对外传输 JSON 格式的 code/message，前端统一解析。提示中的错误码可用于对应日志，原始异常不直接展示给用户。

新增对外服务方法时，参数和返回值应可 JSON 序列化；辅助方法保持不导出，避免被 mygo.Bind 暴露。新增前端调用集中在 backend.js，其他组件依赖其接口。
