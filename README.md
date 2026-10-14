# TextRouter

> 一个重新设计的 TextSoap：剪贴板一键清理、看得懂的查找替换、表格式批量替换、可视化动作链。
> A TextSoap-style text cleaner for macOS, written in pure Go with [MyGo](https://github.com/egoist/mygo) native UI.

![主窗口：查找替换（正则模式）](docs/screenshots/main-find-regex.png)

## 功能

**信息架构（照 TextSoap）**
- 主窗口就是一个文本文档。「剪贴板」是标签栏里固定的第一个文档，打开的文件（⌘O / 拖入）是它旁边的同级标签。
- 工具栏：一键清理、粘贴覆盖、全部复制、「工具」菜单、清理器面板（⌥⌘S）、查找替换面板（⌘F）；剪贴板文档另有「复制并完成 ⌘↩」。
- 右侧检查器：**清理器列表**（常用 / 全部 / 分类，搜索，单击即应用于选区或全文）和 **查找替换**（一级功能，见下）。
- 底部选项栏：缩放、行号、自动换行、不可见字符、等宽，以及行列 / 字数统计。
- 二级功能以面板（sheet）打开：批量替换、我的清理器（动作链）、清理器库、批量处理文件、设置——入口是「工具」菜单、菜单栏「工具」和 ⌘K。

**剪贴板文档**
- 点 Dock 图标 / 菜单栏图标 / 全局快捷键（默认 `⌘⇧V`）→ 窗口弹出并把当前剪贴板读入「剪贴板」文档。
- 每一步都能撤销（⌘⇧Z 重做）；`⌘K` 模糊搜索全部清理器 / 动作链 / 规则集 / 工具。
- `⌘↩` 复制回剪贴板并收起。

**查找替换**（检查器里的一级功能，⌘F）——为正则新手设计
- 默认「文字」模式（`.`、`*` 只代表自己），一键切到「正则表达式」。
- 可展开的「正则助手」逐段用中文解释规则：`(\d+)` → “第 1 组开始（替换里用 $1 引用）”、“任意一个数字”……替换串里的 `$1`、`$U` 等记号也逐个解释，写错的记号（如 `$$`、`${1}`）会标红并告诉你为什么。
- 文档里直接高亮匹配，上一个 / 下一个、替换、全部替换；可勾选「只在选区」；下方逐条列出匹配和捕获组。
- 18 个常用模式一键填入：邮箱、网址、数字、手机号、日期、汉字、中英文之间加空格、连续空格、空行、HTML 标签、重复单词、snake_case→camelCase……
- 可直接「加入批量规则」或「存为动作链步骤」。

**批量替换**——把 TextSoap 难懂的行/列换成规则表
- 每一行就是一条规则：`☑ 启用 | 查找 → 替换为 | .* 正则 | Aa 区分大小写 | 命中 N 处 | ✕`。
- 明确的执行顺序：从上到下，每条作用于上一条的结果；拖动行即可调整顺序。
- 每条规则单独显示命中次数，正则写错的行会标红；底部实时显示合并后的结果（可切换改动对照）。
- 导入 / 导出 CSV、TSV，也可以直接粘贴从 Excel / Numbers 复制的表格。

**动作链**（TextSoap 的「自定义清理器」）
- 竖排步骤卡片：应用清理器、查找替换、查找删除、批量替换、提取匹配、插入/包裹文字、每行加/删文字、行排序、折行、引用、标题式大小写、截取字符、逐行处理、对匹配内容处理、分组、重复、调用其他动作链、备注。
- 每一步显示改动量，并可展开「这一步之后的文本」；步骤可开关、上下移、复制、删除；容器步骤可以嵌套子步骤。
- 可固定到「常用」；导入 TextSoap 清理器库（`.textsoap9Lib` 二进制 plist），导出为 TextSoap 可读的格式（`fileVersion 6`）。

**清理器库**：189 个 TextSoap 9 内置清理器，163 个已实现；每个都能预览对当前文本的效果并固定到主界面。标签：`已实测`（38 个，与 TextSoap 实测输出逐条一致）/ `未验证`（按官方说明实现）/ `富文本`（只影响字体格式，纯文本下不改变内容）。

**编辑器**（基于 MyGo 的 ui/editor）：行号、当前行高亮、选中即「作用于选区」（动作 / 查找替换 / 批量替换 / 动作链都只改选区，多选区也行），显示不可见字符（空格、制表符、换行 ¬、NBSP ⍽、全角空格 □、零宽字符），自动换行、⌘+ / ⌘- 缩放、等宽 / 比例字体切换，多光标（⌘D、⇧⌘L、⌥⌘↑↓、⌥ 拖动列选），转到行（⌘L），⌥↑↓ 移动行、⇧⌥↓ 复制行、⇧⌘K 删除行、排序 / 反转 / 去重 / 合并行、大小写转换，括号和引号自动配对；打字与动作共用一条撤销历史；拖入文本文件打开，⌘O / ⌘S 打开存储。

**改动高亮**：每次动作 / 替换后，编辑器里只高亮真正改动的字符（字符级 diff：新增的字绿色描边，删除处一条红色竖线），约 4 秒后淡出，任何输入立即清除；查找替换面板的匹配也直接在文档里高亮。

**不可见字符**：¬ 换行（\r\n 文件显示 ¬↵）、⍽ NBSP、□ 全角空格、‸ 零宽字符、↵ 单独的 CR 都画在文字上方，不占任何宽度——开关时文字一个像素都不会移动。

**记住视图**：行号、换行、不可见字符、等宽、缩放、右侧面板与查找模式、窗口大小、字体都保存在资料库 JSON（`settings.view`），下次启动恢复。

**字体设置**：设置 → 编辑器字体。扫描系统字体目录（macOS：/System/Library/Fonts、/Library/Fonts、~/Library/Fonts）读出所有字体族，每一项用它自己的字体显示，可搜索、可只看等宽；分别设置正文字体、中文后备字体、等宽字体、界面字体（可选）、字号、行高，带实时预览。

**批量处理文件**：选文件或文件夹 → 选动作链/规则集/清理器 → 先预览哪些文件会变 → 再写入（可选 `.bak` 备份）。

**其他**：浅色 / 深色 / 跟随系统；键盘优先（⌘F、⌥⌘S、⌘, 设置、⌘K、⌘↩、⇧⌘R 批量替换）。

| 工具菜单 | 批量替换（sheet，深色） |
|---|---|
| ![](docs/screenshots/tools-menu.png) | ![](docs/screenshots/batch-sheet-dark.png) |
| **剪贴板文档 + 清理器列表** | **文件文档（同级标签）** |
| ![](docs/screenshots/main-cleaners-dark.png) | ![](docs/screenshots/doc-file.png) |
| **⌘K** | **我的清理器（sheet）** |
| ![](docs/screenshots/palette.png) | ![](docs/screenshots/chains-sheet.png) |

## 下载与安装（macOS）

每次推送到 `main`，GitHub Actions 会在 macOS 上构建：打开仓库的 **Actions → Build → 最新一次运行 → Artifacts → `TextRouter-macOS`**，里面有 Apple Silicon（`darwin-arm64`）和 Intel（`darwin-amd64`）的 `.dmg` 与 `.app.zip`。打 `v*` 标签时会附到 Release。

包是 ad-hoc 签名、未经 Apple 公证，第一次打开前执行：

```sh
xattr -cr /Applications/TextRouter.app
```

全局快捷键无需额外权限。

## 自己构建

需要 Go 1.27+（`go.mod` 固定了 MyGo CLI 为 tool）。

```sh
go test ./...                                   # 引擎测试 + 界面测试（无窗口）
go tool mygo dev                                # 开发运行，改代码自动重启
go tool mygo build -platform darwin/arm64       # 打包 .app（在 Mac 上还会生成 .dmg）
```

MyGo 不需要 cgo，在 Linux 上也能交叉编译出 macOS 的 `.app`（`GOOS=darwin` 或 `mygo build -platform darwin/arm64`），只有 `.dmg` 必须在 macOS 上生成。

## MyGo 编辑器的改动（third_party/mygo）

`ui/editor` 没有「按范围上色」和「不占位的不可见字符」，所以仓库里带了一份 MyGo v0.4.8 的副本（`go.mod` 里 `replace`），只加了：`State.Highlights` / `HighlightVersion`（字符范围背景 + 描边，零长度画竖线；版本变化即失效）、`State.Invisibles`（`ui/editor/invisibles.go`，标记画在字符自身的位置上，不参与排版），以及测试用的 `InvisibleMarks` / `RuneX`。

## 架构

```
engine/        纯 Go 核心，不依赖界面，可单独使用和测试
  regex.go       查找替换：按 TextSoap 实测语义实现（见下）
  cleaners.go    清理器注册表（读取内嵌的 TextSoap 清单 data/builtin_inventory.json）
  builtins.go    各清理器实现
  chain.go       动作链 Step/Chain 与执行（带逐步结果，用于预览）
  batch.go       批量规则 + CSV/TSV
  explain.go     正则 / 替换串的中文解释、常用模式
  textsoap.go    TextSoap 库导入、fileVersion 6 导出
  store.go       配置与资料库（JSON）
  diff.go files.go
gui/           MyGo 原生界面；只依赖 engine 和一个小的 Platform 接口
main.go        窗口、菜单栏图标、全局快捷键、Dock、对话框（实现 Platform）
```

用户数据保存在 `~/Library/Application Support/TextRouter/library.json`（动作链、规则集、设置），可以直接备份或手改。

### 实测语义（与 TextSoap 9 一致）

- 大小写只由「区分大小写」决定；`regex_options` 里的 `i` 无效，内联 `(?i)` 有效。
- 只有小写 `m`（^$ 按行）和 `s`（. 匹配换行）两个选项字母生效。
- 替换记号：`$0`–`$9`（只取一位数字，`$01` = `$0` + `1`）、`$U`/`$L` 之后全部大/小写、`$u`/`$l` 下一个字符大/小写、`$E`/`$e` 结束（本身为空）、`\$` 是美元符号；`$$` 为空，`${1}` 不支持，其他 `$x` 会被丢掉。
- 文字模式下替换串原样插入。
- 批量替换表里的正则规则默认开启多行（`^`/`$` 按行）——这是 TextRouter 有意的改动，更符合表格使用直觉。

### 添加一个清理器

在 `engine/builtins.go` 的 `init()` 里注册：

```go
reg("#MY_TOKEN", "我的清理器", func(s string) string {
    return strings.ReplaceAll(s, "，", ", ")
})
```

如果 token 不在 TextSoap 清单里，同时在 `engine/data/builtin_inventory.json` 的 `cleaners` 里加一条 `{"token": "#MY_TOKEN", "name": "My Cleaner", "description": "..."}`。它会自动出现在清理器库、⌘K 面板和动作链的「应用清理器」里。

### 添加一种动作（动作链步骤）

1. 在 `engine/chain.go` 加一个 `Kind…` 常量，在 `Step.Apply` 的 `switch` 里实现；
2. 在 `Actions()` 里登记标题、说明和要编辑的字段（`text`、`find`、`width`……）；
3. 字段编辑器在 `gui/chains.go` 的 `stepFields` 中，已有字段会自动显示，新字段在那里加一段即可。

## 测试与兼容性

| 测试 | 结果 |
|---|---|
| 清理器实测数据（38 个清理器 × 39 输入，`engine/testdata/behaviour_truth.json`） | 1482 / 1482 |
| 查找替换实测用例（`regex_cases.json`） | 51 / 51 |
| 在 TextSoap 中执行过的动作链样例（`chain_examples.json`） | 15 / 15 |
| 界面测试（`gui/gui_test.go`，MyGo Tester 无窗口运行） | 6 个场景通过 |

注意：实测数据只覆盖 39 个输入，100% 通过不代表所有输入都与 TextSoap 一致；其余 151 个未实测清理器按说明实现，界面中标为「未验证」。

## English (short)

TextRouter reimplements TextSoap 9's text cleaning with a clearer UX: a Clipboard Space opened by Dock/menu-bar/global shortcut, a find/replace screen that explains regexes and replacement tokens in plain language with per-match previews, batch replace as an ordered rule table (drag to reorder, per-rule hit counts, CSV/TSV), and a step-by-step chain builder with previews after each step. Pure Go (`engine/` has no UI deps), native UI via MyGo. `go test ./...`; `go tool mygo build -platform darwin/arm64`.

## 第三方资源 / Credits

- 图标：[Lucide](https://lucide.dev)（ISC，见 `gui/icons/LICENSE`）
- 界面字体：[Inter](https://rsms.me/inter/)（SIL OFL 1.1，见 `gui/fonts/OFL.txt`），中文回退到系统的苹方
