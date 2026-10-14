package engine

import "sync"

// Support says how much of an action works on plain text.
type Support string

const (
	SupportFull        Support = "full"        // implemented
	SupportRich        Support = "rich"        // styles/links: nothing to do in plain text
	SupportUnavailable Support = "unavailable" // needs macOS services or a beta TextSoap feature
)

// ActionInfo describes an action for the toolbox and the editor.
type ActionInfo struct {
	ID        string  `json:"id"`
	Title     string  `json:"title"`
	En        string  `json:"en"`
	Category  string  `json:"category"`
	Desc      string  `json:"desc"`
	Container bool    `json:"container"`
	Support   Support `json:"support"`
	Defaults  Params  `json:"defaults"`
}

// Action categories, in toolbox order.
var ActionCategories = []string{"查找替换", "文本", "行", "条件", "运行", "样式"}

func findDefaults(regex bool) Params {
	t := 0
	opts := ""
	if regex {
		t, opts = 1, "m"
	}
	return Params{"find": "", "replace": "", "type": t, "match_case": false, "granularity": 0, "regex_options": opts, "list_name": ""}
}

// ActionCatalog lists every action TextSoap 9 offers in its cleaner editor
// (60 ids), plus Number Lines, which its help documents without an id.
func ActionCatalog() []ActionInfo {
	return []ActionInfo{
		{"find", "查找替换", "Find and Replace", "查找替换", "查找文字或正则并替换。可选列表：对列表的每一行执行一次，$v{A}–$v{D} 取该行的列。", false, SupportFull, findDefaults(false)},
		{"find.regex", "正则查找替换", "Regex Find and Replace", "查找替换", "用正则表达式查找并替换（旧版动作，等同于勾选了正则的查找替换）。", false, SupportFull, findDefaults(true)},
		{"find.delete", "查找并删除", "Find and Delete Text", "查找替换", "找到的内容全部删除。", false, SupportFull, findDefaults(false)},
		{"find.batch", "按列表查找替换", "Find and Replace with List", "查找替换", "对所选列表的每一行做一次查找替换，$v{A}–$v{D} 取该行的列。", false, SupportFull, withDefault(findDefaults(false), "find", "$v{A}").with("replace", "$v{B}")},
		{"find.batch.regex", "按列表正则查找替换", "Regex Find and Replace with List", "查找替换", "同上，查找内容按正则解释。", false, SupportFull, withDefault(findDefaults(true), "find", "$v{A}").with("replace", "$v{B}")},
		{"find.assignlinks", "创建超链接", "Create Hyperlink URLs", "查找替换", "给找到的文字加上超链接（富文本）。", false, SupportRich, Params{"find": "$v{A}", "replace": "$v{B}", "list_name": ""}},

		{"applycleaner", "应用清理器", "Apply Cleaner", "文本", "运行一个内置或自定义清理器。", false, SupportFull, Params{"cleaner": "Scrub"}},
		{"text.insert", "插入文字", "Insert Text", "文本", "在文本之前、之后插入，或替换整段文本。", false, SupportFull, Params{"text": "", "position": 0}},
		{"text.delete", "删除文字", "Delete Text", "文本", "删除当前文本。通常放在条件里，删除匹配到的内容。", false, SupportFull, Params{}},
		{"text.tag", "包裹文字", "Tag Text", "文本", "在前后加上前缀和后缀，例如 <b> 和 </b>。", false, SupportFull, Params{"prefix": "", "suffix": ""}},
		{"text.extract", "提取文字", "Extract Text", "文本", "只保留正则匹配到的内容（可选捕获组），每条一行；或追加在原文之后。", false, SupportFull, Params{"find": "", "group": 0, "match_case": false, "append": false, "regex_options": "m"}},
		{"text.substring", "提取字符", "Extract Characters", "文本", "从开头、结尾或指定位置截取若干个字符。", false, SupportFull, Params{"mode": 0, "count": 10, "start": 1, "length": 10}},
		{"text.wrap", "自定义折行", "Custom Text Wrap", "文本", "按指定的最大行宽重新折行。", false, SupportFull, Params{"width": 70}},
		{"text.quote", "引用文本", "Quote Text", "文本", "在每行前加引用符（如 >），可同时折行。", false, SupportFull, Params{"level": 1, "wrap": true, "width": 70, "prefix": "", "mark": ">", "suffix": " "}},
		{"text.titlecase", "标题式大小写（扩展）", "Title Case (Extended)", "文本", "按标题规则大写，可追加小词（保持小写）和特殊词（保持原样）。", false, SupportFull, Params{"small_words": "", "special_words": ""}},
		{"text.copytoclip", "复制到剪贴板", "Copy to Clipboard", "文本", "把当前文本放到剪贴板。", false, SupportFull, Params{}},
		{"text.appendtoclip", "追加到剪贴板", "Append Text to Clipboard", "文本", "把当前文本追加到剪贴板已有内容之后。", false, SupportFull, Params{"separator": ""}},
		{"text.pastefromclip", "从剪贴板粘贴", "Paste Text from Clipboard", "文本", "用剪贴板内容替换当前文本。", false, SupportFull, Params{}},
		{"text.clipaction", "剪贴板动作", "Clipboard Action", "文本", "TextSoap 的剪贴板联动动作。", false, SupportUnavailable, Params{}},
		{"text.hyperlinks2text", "超链接转文字", "Hyperlinks To Text", "文本", "把富文本中的链接转成文字（富文本）。", false, SupportRich, Params{"pattern": `<a href="{link}">{text}</a>`}},
		{"text.performservice", "执行系统服务", "Perform Named Service", "文本", "调用一个 macOS 服务处理文本。", false, SupportUnavailable, Params{"service": ""}},

		{"line.addto", "每行添加文字", "Add Text to Lines", "行", "在每行的开头或末尾加上文字。", false, SupportFull, Params{"text": "", "position": 0}},
		{"line.removefrom", "每行删除文字", "Remove Text from Lines", "行", "删掉每行开头或末尾的指定文字，没有就不动。", false, SupportFull, Params{"text": "", "position": 0}},
		{"line.sort", "行排序", "Sort Lines", "行", "按升序或降序排列各行，可忽略大小写、变音符、行首空白，数字按数值比较。", false, SupportFull, Params{"descending": false, "match_case": false, "numeric": false, "ignore_whitespace": false, "ignore_diacritics": false, "use_expression": false, "find": "", "replace": "$0"}},
		{"line.number", "行编号", "Number Lines", "行", "在每行前加编号。格式中连续的 0 代表编号及最少位数，\\t 是制表符，\\s 是空格。", false, SupportFull, Params{"lines": 0, "start": 1, "increment": 1, "format": `0.\s`}},
		{"line.loop", "遍历每一行", "Loop through Lines", "行", "把文本拆成行，对每一行（包括空行）分别执行里面的动作。", true, SupportFull, Params{}},

		{"if.match.text", "如果匹配文字", "If Matches Text", "条件", "找到匹配的文字，只对这些片段执行里面的动作。可反选、跳过前 N 个、最多 N 个。", true, SupportFull, Params{"find": "", "type": 0, "match_case": false, "granularity": 0, "regex_options": "", "group": 0, "invert": false, "skip": 0, "limit": 0}},
		{"if.match.text.regex", "如果匹配正则", "If Matches Regex", "条件", "同上，按正则查找，可只取某个捕获组。", true, SupportFull, Params{"find": "", "type": 1, "match_case": false, "regex_options": "m", "group": 0, "invert": false, "skip": 0, "limit": 0}},
		{"if.match.text.processlines", "对每一行", "For Each Line", "条件", "对每个非空行分别执行里面的动作。", true, SupportFull, Params{}},
		{"if.match.url", "如果是网址", "If URL", "条件", "找出文本中的网址，对它们执行里面的动作。", true, SupportFull, Params{"invert": false, "skip": 0, "limit": 0}},
		{"if.match.date", "如果是日期", "If Date", "条件", "找出文本中的日期，对它们执行里面的动作。", true, SupportFull, Params{"invert": false, "skip": 0, "limit": 0}},
		{"if.match.font", "如果匹配字体", "If Matches Font", "条件", "按字体、字号、粗斜体匹配（富文本）。", true, SupportRich, Params{}},
		{"if.match.attribute", "如果有属性", "If Has Attribute", "条件", "按下划线、颜色等属性匹配（富文本）。", true, SupportRich, Params{}},
		{"if.end", "结束条件", "Endif", "条件", "旧格式里条件块的结束标记。", false, SupportFull, Params{}},
		{"modify.results.invert", "调整结果：反选", "Modify Results: Invert Matches", "条件", "对条件的匹配结果取反。", false, SupportFull, Params{}},
		{"modify.results.limit", "调整结果：最多 N 个", "Modify Results: Limit Matches", "条件", "只使用前 N 个匹配。", false, SupportFull, Params{"count": 1}},
		{"modify.results.skip", "调整结果：跳过前 N 个", "Modify Results: Skip Matches", "条件", "忽略前 N 个匹配。", false, SupportFull, Params{"count": 1}},

		{"runtime.group", "分组", "Group", "运行", "把几个动作放在一起，整体开关、移动。", true, SupportFull, Params{"name": ""}},
		{"runtime.repeat", "重复", "Repeat", "运行", "把里面的动作重复执行指定次数。", true, SupportFull, Params{"count": 2, "until_unchanged": false}},
		{"runtime.definelist", "定义列表", "Define List", "运行", "定义一张最多 4 列（A–D）的表，供“按列表查找替换”和“处理列表”使用。", false, SupportFull, Params{"name": "列表 1", "columns": 2, "rows": []any{}}},
		{"runtime.processlist", "处理列表", "Process List", "运行", "对列表的每一行执行里面的动作；里面的 $v{A}–$v{D} 换成该行的列。", true, SupportFull, Params{"list_name": ""}},
		{"runtime.macro", "定义宏", "Define Macro", "运行", "定义一组动作，只在“应用宏”时执行。", true, SupportFull, Params{"name": "宏 1"}},
		{"runtime.applymacro", "应用宏", "Apply Macro", "运行", "执行之前定义的宏。", false, SupportFull, Params{"name": ""}},
		{"runtime.macrolist", "宏列表", "Macro List", "运行", "旧格式：顶层的命名分组都作为宏。", true, SupportFull, Params{}},
		{"runtime.usernote", "备注", "Note", "运行", "只是说明，不改变文本。", false, SupportFull, Params{"note": ""}},
		{"runtime.shellscript", "运行 Shell 脚本", "Run Shell Script (BETA)", "运行", "把文本作为标准输入交给脚本，用脚本输出替换文本（需在设置中开启）。", false, SupportFull, Params{"script": "cat"}},
		{"runtime.javascript", "运行 JavaScript", "Run Javascript (BETA)", "运行", "TextSoap 的测试功能，暂不支持。", false, SupportUnavailable, Params{"script": ""}},
		{"runtime.askuser", "询问用户", "Ask User (BETA)", "运行", "TextSoap 的测试功能，暂不支持。", false, SupportUnavailable, Params{}},
		{"runtime.definevar", "定义变量", "Define Variable (BETA)", "运行", "TextSoap 的测试功能，暂不支持。", false, SupportUnavailable, Params{}},
		{"runtime.workflow", "运行自动操作", "Run Automator Workflow", "运行", "运行 Automator 工作流，暂不支持。", false, SupportUnavailable, Params{"workflow": ""}},
		{"applycleaner.static", "应用已有清理器", "Apply Existing Cleaner", "运行", "旧版的“应用清理器”。", false, SupportFull, Params{"cleaner": "Scrub"}},

		{"style.text.font", "设置字体", "Set Font", "样式", "设置字体、字形、字号（富文本）。", false, SupportRich, Params{}},
		{"style.text.exactfont", "设置字体（高级）", "Set Font (Advanced)", "样式", "按精确字体名设置（富文本）。", false, SupportRich, Params{}},
		{"style.text.face", "设置字形", "Set Style", "样式", "设置粗体、斜体（富文本）。", false, SupportRich, Params{}},
		{"style.text.adjustsize", "调整字号", "Adjust Font Size", "样式", "相对调整字号（富文本）。", false, SupportRich, Params{}},
		{"style.text.underline", "设置下划线", "Set Underline", "样式", "富文本。", false, SupportRich, Params{}},
		{"style.text.strikethru", "设置删除线", "Set Strikethrough", "样式", "富文本。", false, SupportRich, Params{}},
		{"style.text.superscript", "设置上下标", "Set Script Attribute", "样式", "富文本。", false, SupportRich, Params{}},
		{"style.text.color", "设置文字颜色", "Set Text Color", "样式", "富文本。", false, SupportRich, Params{}},
		{"style.text.backcolor", "设置背景色", "Set Background Color", "样式", "富文本。", false, SupportRich, Params{}},
		{"style.attribute.remove", "移除字符属性", "Remove Character Attribute", "样式", "富文本。", false, SupportRich, Params{}},
	}
}

func (p Params) with(k string, v any) Params { return withDefault(p, k, v) }

var (
	actionIndexOnce sync.Once
	actionIdx       map[string]ActionInfo
)

func actionIndex() map[string]ActionInfo {
	actionIndexOnce.Do(func() {
		actionIdx = map[string]ActionInfo{}
		for _, a := range ActionCatalog() {
			actionIdx[a.ID] = a
		}
	})
	return actionIdx
}

// ActionByID looks up catalog information.
func ActionByID(id string) (ActionInfo, bool) {
	a, ok := actionIndex()[id]
	return a, ok
}
