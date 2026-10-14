# Built-in cleaner specs (extracted, not reverse-engineered)

Source: TextSoap's own shipped documentation inside the app bundle:
- `Contents/Resources/TextSoap9Help/topics/*.html` (and `TextSoap 9 Help.md`)
- cleaner `desc` fields in the built-in library plist (`textsoap9-groups-builtin.plist`)

Each entry: token, app-supplied description, and the help-book paragraph if any.

| # | Cleaner | token | app description | help-book text |
|---|---|---|---|---|
| 1 | Scrub | `#SCRUB` | Addresses typical text cleaning needs. Combines stripping spaces, forwarding characters, MIME encoded characters, and paragraphs. SCRUB calls the cleaners in a specific order to insure best results. |  |
| 2 | MyScrub | `#MYSCRUB` | A shortcut to a user selected custom cleaner. Specify in Preferences > General. | A customizable version of the SCRUB version. Defined using the customize window, it allows augmenting or replacing cleaners used in SCRUB. Any available cleaner may be added to this list. |
| 3 | Remove Extra Spaces | `#REMOVE_EXTRA_SPACES` | Replaces multiple spaces with a single space. Note: Leading spaces at beginning of a line are also removed. | Converts non-breaking spaces to standard spaces, then removes two or more spaces with a single space. Leading spaces at the beginning of a line are also removed, as are trailing spaces. |
| 4 | Remove Forwarding (>) Characters | `#REMOVE_FORWARDING_CHARS` | Removes leading '>' (forwarding) marks from a line that are used in some emails. | Removes one or more sets of forwarding characters (>) on a line, commonly found in forwarded email. |
| 5 | Convert MIME-encoded Characters | `#CONVERT_MIME` |  | Converts %Hex or =Hex characters to their appropriate ASCII character. For example, a %20 or a =20 is converted to a space (hex 0x20 is the ASCII value of a space character). |
| 6 | Make Paragraphs | `#MAKE_PARAGRAPHS` | Converts multi-line paragraph with hard returns <CR> into a single paragraph. New paragraphs are marked by two successive returns. | Converts multi-line paragraphs (with hard carriage returns CR) into a single paragraph. Also adds a space to the text when removing CRs to avoid smashing words together. Two or more CRs in a row indicate a new paragraph. |
| 7 | Remove Extra Returns | `#REMOVE_EXTRA_RETURNS` | Converts multiple successive returns into a single return. Same as 'Multiple Returns to 2 Returns' | Converts multiple carriage returns into a single carriage return. Behaves the same as Multiple Return to 1 Return |
| 8 | Remove All Tabs | `#REMOVE_ALL_TABS` | Delete all tab characters. | Deletes all tabs from the selected text. |
| 9 | Remove Control Characters | `#REMOVE_CONTROL_CHARS` | Delete control characters (ASCII value less than 32) with exception of tab or return. | Deletes any control characters (characters with an ASCII value less than 32) from the selected text (except for line feeds or carriage returns). |
| 10 | Uppercase | `#UPPERCASE` | Convert text to “UPPERCASE”. | Converts text to “UPPERCASE”. |
| 11 | Lowercase | `#LOWERCASE` | Convert text to “lowercase”. | Converts text to “lowercase”. |
| 12 | Capitalize Sentences | `#CAPITALIZE_SENTENCES` | Capitalize first character of each sentence, with changing existing case of text. | Converts text to lowercase, capitalizing the first character of each sentence. |
| 13 | Capitalize Words | `#CAPITALIZE_WORDS` | Capitalize each word. | Converts text to lowercase, then capitalizes each individual word. |
| 14 | Capitalize with Title Case | `#TITLE_CASE1` | Capitalize words using basic Title Case rules. |  |
| 15 | Straighten Quotes | `#STRAIGHTEN_QUOTES` | Convert single and double smart (curly) quotes to straight quotes. | Converts single and double smart (or curly) quotes to straight quotes. |
| 16 | Smarten Quotes | `#SMARTEN_QUOTES` | Convert single and double straight quotes to smart (or curly) quotes. | Convert single and double straight quotes to smart (or curly) quotes. |
| 17 | Expand Tabs | `#EXPAND_TABS` |  | Converts tabs (hex 0x09) to four (4) spaces. |
| 18 | Rewrap Text | `#REWRAP_TEXT` | Fix any paragraphs (see Make Paragraphs), then rewrap text at 70 characters per line. (see Wrap At 70). | Fixes any paragraphs (using Make Paragraphs), then rewraps the text at 70 characters per line. Behaves the same as the Wrap At 70 cleaner. |
| 19 | Wrap At 50 | `#WRAP_AT_50` |  | Fixes any paragraphs (using Make Paragraphs), then rewraps the text at 50 characters per line. |
| 20 | Wrap At 60 | `#WRAP_AT_60` | Fix any paragraphs (see Make Paragraphs), then rewrap text at 60 characters per line. | Fixes any paragraphs (using Make Paragraphs), then rewraps the text at 60 characters per line. |
| 21 | Wrap At 70 | `#WRAP_AT_70` | Fix any paragraphs (see Make Paragraphs), then rewrap text at 70 characters per line. | Fixes any paragraphs (using Make Paragraphs), then rewraps the text at 70 characters per line. |
| 22 | Quote Text | `#QUOTE_TEXT` | Rewrap text and insert forwarding marks (‘>’) at beginning of each line. Put a hard return at the end of each line to ensure proper wrapping. | Rewraps the text and re-inserts forwarding marks (i.e.. ‘>’) at the beginning of each line. It also places hard returns at the end of each line to ensure proper wrapping in email applications. |
| 23 | Internet Friendly Text | `#NETTEXT` | Convert to Internet Friendly Text |  |
| 24 | Remove All High Ascii Characters | `#REMOVE_HIASCII` |  | Removes All High Ascii Characters. Strips off any ASCII values 128–256 from the text. |
| 25 | Trim Lines | `#TRIM_LINES` |  | Removes any extra spaces from both the beginning and the end of the selected text lines. |
| 26 | Trim Beginning of Lines | `#TRIM_BEGINNING_OF_LINES` |  | Removes any extra spaces from the beginning of the selected text lines. |
| 27 | Trim End of Lines | `#TRIM_END_OF_LINES` |  | Removes any extra spaces from the end of the selected text lines. |
| 28 | Strip 2 or more Spaces | `#STRIP_GT_1SPACE` | Strip two (2) or more consecutive spaces to a single (1) space. | Strips any 2 or more consecutive spaces to a single space. |
| 29 | Remove All Spaces | `#REMOVE_ALL_SPACES` | Delete all space characters. |  |
| 30 | Remove All Whitespace | `#REMOVE_ALL_WHITESPACE` | Delete all whitespace (spaces,tabs) chcaraters. |  |
| 31 | Remove Duplicate Lines | `#REMOVE_DUPLICATE_LINES` | Remove duplicate lines. |  |
| 32 | Multiple Returns to 1 Return | `#MULTIPLE_CR_TO_1CR` | Convert multiple returns to a single (1) return. |  |
| 33 | Convert Non-Breaking Spaces | `#CONVERT_NON_BREAKING_SPACES` |  | Converts Non-Breaking Spaces to standard spaces. |
| 34 | Convert Bullets to ASCII | `#CONVERT_BULLETS` |  | Converts bullets to standard ASCII dashes. |
| 35 | Extract Text from HTML Source | `#EXTRACT_TEXT_FROM_HTML` | Clean up HTML raw text. Will strip out anything between ‘<’ and ‘>’. Useful to extract text contents from HTML source, handling HTML entities. Finally, remove tab characters and remove multiple returns. | Cleans up HTML raw text. It strips out anything between ‘<’ and ‘>’. Extract the text contents from HTML source with this cleaner. It also handles Ampersand escape codes ( or Œ) and removes tab characters and multiple carriage returns. |
| 36 | HTML Tag: Bold | `#HTML_TAG_BOLD` | Wrap text in given tag: <b>{text}</b> | Wraps the selected text in a given tag. |
| 37 | BBCode: Bold | `#BBCODE_TAG_BOLD` | [b]{text}[/b] - Bold {text} |  |
| 38 | PLIST Tag: String | `#PLIST_TAG_STRING` | Wrap text in given tag: <string>{text}</string> | Wraps the selected text in a given tag. |
