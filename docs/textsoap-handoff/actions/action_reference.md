# Action reference

60 user-composable actions. These are the blocks a user assembles into a cleaner. Titles/descriptions are the app's own strings.

| id | title | description |
|---|---|---|
| `applycleaner` | Apply Cleaner | Applies the selected cleaner to the provided text. |
| `applycleaner.static` | Apply Existing Cleaner | Applies the selected cleaner to the provided text. |
| `find` | Find and Replace | Finds and replaces specified text. |
| `find.assignlinks` | Create Hyperlink URLs | Finds provided text and associates a hyperlinked URL to that text. |
| `find.batch` | Find and Replace with List | Find and Replace using specified list values. |
| `find.batch.regex` | Regex Find and Replace with List |  |
| `find.delete` | Find and Delete Text | Finds specified text and deletes it. |
| `find.regex` | Regex Find and Replace | Finds and replaces specified text using regular expressions. |
| `if.end` | Endif | Marks the end of a conditional block. |
| `if.match.attribute` | If Has Attribute | Finds text matching specified attribute, then applies contained actions to it. |
| `if.match.date` | If Date | Finds any dates using Apple's Data Detector. |
| `if.match.font` | If Matches Font | Finds text matching specific font, size, color attributes, then applies contained actions to it. |
| `if.match.text` | If Matches Text | Finds provided text then applies contained actions to it. |
| `if.match.text.processlines` | For Each Line | Perform specified actions on each line specified. |
| `if.match.text.regex` | If Matches Regex |  |
| `if.match.url` | If URL | Finds any URLS using Apple's Data Detector |
| `line.addto` | Add Text to Lines | Add specified text to each line. |
| `line.loop` | Loop through Lines | Loop through each line, allowing each specified line to be processed separately |
| `line.removefrom` | Remove Text from Lines | Remove specified text from the beginning of each line. |
| `line.sort` | Sort Lines | Sorts the selected lines in either ascending or descending order. |
| `modify.results.invert` | Modify Results: Invert Matches | Inverts the selected text of the this condition. |
| `modify.results.limit` | Modify Results: Limit Matches | Set the maximum matches for this condition. |
| `modify.results.skip` | Modify Results: Skip Matches | Ignore the specified matches for this condition. |
| `runtime.applymacro` | Apply Macro | Applies the defined Macro to the current text. |
| `runtime.askuser` | Ask User (BETA) | Define a Variable by Asking User |
| `runtime.definelist` | Define List | Defines a list that can be used with Find & Replace |
| `runtime.definevar` | Define Variable (BETA) | Define Variable from Expression |
| `runtime.group` | Group | Container to organize actions within a cleaner. |
| `runtime.javascript` | Run Javascript (BETA) | Applies the provided Javascript on the provided text. |
| `runtime.macro` | Define Macro | Defines a macro (collection of actions) that you apply using 'Apply Macro' action. |
| `runtime.macrolist` | Macro List | Placed at the top level, specifies named groups which should be considered macros (containing collections of actions). Can be applied with 'Apply Macro' action. |
| `runtime.processlist` | Process List | Loop through each row and apply contained cleaners. |
| `runtime.repeat` | Repeat | Repeat contained items using the rules provided. |
| `runtime.shellscript` | Run Shell Script (BETA) | Run the given shell script. |
| `runtime.usernote` | Note | Allows user to specify notes within a cleaner. |
| `runtime.workflow` | Run Automator Workflow | This action runs an Automator Workflow (located in the ~/Library/Application Support/TextSoap/Workflows folder). |
| `style.attribute.remove` | Remove Character Attribute | Finds and removes the specified character attribute from text. |
| `style.text.adjustsize` | Adjust Font Size | Provides relative adjustments to the font size of the text. |
| `style.text.backcolor` | Set Background Color | Set background color of the text. |
| `style.text.color` | Set Text Color | Set text color of the text. |
| `style.text.exactfont` | Set Font (Advanced) | Applies specific font name, size to the text. |
| `style.text.face` | Set Style | Change Typeface of text. |
| `style.text.font` | Set Font | Applies font family, face, size attributes to the text. |
| `style.text.strikethru` | Set Strikethrough | Set strikethrough options of the text. |
| `style.text.superscript` | Set Script Attribute | Set text as superscript, subscript, or no script. |
| `style.text.underline` | Set Underline | Set underline options of the text. |
| `text.appendtoclip` | Append Text to Clipboard | Append the provided text on the clipboard. |
| `text.clipaction` | Clipboard Action | Perform clipboard-related action on text. |
| `text.copytoclip` | Copy to Clipboard | Copy current 'text' to clipboard. |
| `text.delete` | Delete Text | Deletes the provided text. Use with a conditional statement to remove matched text. |
| `text.extract` | Extract Text | Extracts text matching the provided regular expression. |
| `text.hyperlinks2text` | Hyperlinks To Text | Converts rich text hyperlinks to text. |
| `text.insert` | Insert Text | Inserts specified text. |
| `text.pastefromclip` | Paste Text from Clipboard | Set the provided text to the contents of the clipboard. |
| `text.performservice` | Perform Named Service | Perform named OS X Service on provided text. |
| `text.quote` | Quote Text | Rewraps text using the selected maximum line length, placing quote characters as appropriate. |
| `text.substring` | Extract Characters | Extracts specified number of characters from text. |
| `text.tag` | Tag Text | Wraps the selected text in a prefix and suffix of text. Useful for tagging HTML or XML text. |
| `text.titlecase` | Title Case (Extended) | Capitalizes words for title case but allows specifying additional small words (which are always lowercase) as well as exception words (which are ignored). |
| `text.wrap` | Custom Text Wrap | Rewraps specified text using the indicated maximum line length. |

## Registration groups

- **defineFindActions**: `find.assignlinks`, `find.batch`, `find.batch.regex`, `find.delete`, `find.regex`
- **defineTextActions**: `text.appendtoclip`, `text.clipaction`, `text.copytoclip`, `text.delete`, `text.extract`, `text.hyperlinks2text`, `text.insert`, `text.pastefromclip`, `text.performservice`, `text.quote`, `text.substring`, `text.tag`, `text.titlecase`, `text.wrap`
- **defineLineActions**: `line.addto`, `line.loop`, `line.removefrom`, `line.sort`
- **defineStyleActions**: `style.attribute.remove`, `style.text.adjustsize`, `style.text.backcolor`, `style.text.color`, `style.text.exactfont`, `style.text.face`, `style.text.font`, `style.text.strikethru`, `style.text.superscript`, `style.text.underline`
- **defineConditionalActions**: `if.end`, `if.match.attribute`, `if.match.date`, `if.match.font`, `if.match.text`, `if.match.text.processlines`, `if.match.text.regex`, `if.match.url`
- **defineRuntimeActions**: `runtime.applymacro`, `runtime.askuser`, `runtime.definelist`, `runtime.definevar`, `runtime.group`, `runtime.javascript`, `runtime.macro`, `runtime.macrolist`, `runtime.processlist`, `runtime.repeat`, `runtime.shellscript`, `runtime.usernote`, `runtime.workflow`
- **defineAltTextActions**: `text.tag`
- **(no define group seen)**: `applycleaner`, `applycleaner.static`, `find`, `modify.results.invert`, `modify.results.limit`, `modify.results.skip`

## Serialized parameter keys observed in real saved data

- `find` — -[TSCleanerFindAction findString] (returns self->_findField stringValue)
- `replace` — written only when the replace field is present
- `type` — -[TSCleanerFindAction findType]
- `match_case` — -[TSCleanerFindAction matchCase]
- `granularity` — -[TSCleanerFindAction granularity]
- `regex_options` — -[TSCleanerFindAction regexOptions]
- `batch_mode` — registered directly in the definition table with the literal kCFBooleanTrue, i.e. it is a fixed flag of the 'find with list' action and not a per-cleaner setting; no accessor or writer seen.
- `list_name` — default value is the empty string and it is present in saved cleaners, so it is a real per-instance setting; the writer has not been decompiled.

## Known gaps

- definition-table parameter lists are lower bounds: keys written through setParam*:forName: are constructed inside the callee and do not appear in the harvested literal set.
- SingleAction factories are assembled from tables the tables' own literals do not repeat, so `define*Actions` names fewer action ids than it registers. Which body registers which id is not yet pinned down for more than a few ids.
- no writer has been decompiled for list_name, so the 'find with list' path is described by name only (batch_mode is known to be a fixed true flag, not a setting).
