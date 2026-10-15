# My Cleaner Actions

> Converted from TextSoap 9's own help book (`my_cleaner_actions.html`).

---

Overview

- Welcome
- Clipboard Workspace
- A Quick Tour
- Cleaner Groups Summary
- What’s New in TextSoap 9

MyLibrary

- Cleaner Editor
- My Cleaner Actions
- Custom Group Editor

Tutorials

- Your First Cleaner
- Removing parameters from a URL
- Using Lists in Your Cleaners
- Matching Typefaces, Attributes

Cleaner Descriptions

- Date-related Cleaners
- HTML Cleaners
- Plist Cleaners
- Other Cleaner Descriptions

Extras

- Preferences
- Batch File Cleaning

Reference

- Glossary
- Regular Expression Syntax
- Files and Folders

My Cleaner Actions

Find and Replace Actions

Common options. These options are common to the collection of Find and Replace actions. Specific changes for each action will be noted below.

Regex - Indicates whether this is a regular expression

Match Case - Require case matching, otherwise ignores text case (Uppercase A = Lowercase a)

Options

Textual matches

- Contains : match text anywhere

- Starts with : Start of match must be start of word

- Whole Words : Do not match partial words

- Ends with : End of match must be at end of word

Regex matches

- Matches Lines (m) : Control the behavior of “^” and “$” in a pattern. By default these will only match at the start and end, respectively, of the input text. If this flag is set, “^” and “$” will also match at the start and end of each line within the input text.

Sometimes called “Multiline”.

- Dot Matches Returns (s) : If set, a “.” in a pattern will match a line terminator in the input text. By default, it will not.

Note that a carriage-return / line-feed pair in text behave as a single line terminator, and will match a single “.” in a RE pattern. Line terminators are \u000a, \u000b, \u000c, \u000d, \u0085, \u2028, \u2029 and the sequence \u000d \u000a.

- Use Unicode Words (w) : Controls the behavior of \b in a pattern. If set, word boundaries are found according to the definitions of word found in Unicode UAX 29, Text Boundaries.

By default, word boundaries are identified by means of a simple classification of characters as either “word” or “non-word”, which approximates traditional regular expression behavior. The results obtained with the two options can be quite different in runs of spaces and other non-word characters.

- Allow comments (x) : If set, allow use of white space and (#comments) within patterns

Find - Text to find

Replace - Text to replace

Changing text case when replacing using regular expressions

TextSoap supports additional commands to change the case of the replacement text.

Cmd - Action
$U - All future text is uppercased
$u - Next character is uppercased
$L - All future text is lowercased
$l - Next character is lowercased
$E - All future text case is preserved

You can use $u and $l with the capture groups. For example, on the following text:

The big brown fox jumped over the blue cow

Find: (b\w+)

Replace: $u$1

Will capitalize the first letter of first capture group, the result is:

The Big Brown fox jumped over the Blue cow

Where all the matchees (words that start with b) are capitalized.

Find and Replace with List

List - When you specify a list, the action will repeat for each row in the list, replacing references to $v{A}, $v{B}, $v{C}, $v{D} with the value in the specified column for the row.

If you need to specify $v{..} w/o the preprocessing, use . For example: $v{A} would find “$v{A}” and not be replaced in the list processing.

Note: For more advanced functionality, you can use the Process List action. However, do not use Find and Replace with List within a Process List group.

Find and Replace Text

Regular text fields may use the following special characters

\\ = backslash (\)
\t = tab
\n = end of line (return)
\s = space
\x{hhhh} = hex character value, ie. tab = \x{09}, space = \x{20}

Regex Find and Replace Text

When Regex option is selected, find text is a regular expression and is highlighted as such. See Reference on Regular Expressions for more details.

Replace text supports both the characters supported in regular text fields (\t, \n, etc) as well as replacement template ($0 for entire match, $1 for first captured group, etc).

Create Hyperlink URLs

This is a special variant of Batch Find and Replace Text action.

Use the find field to match the text (most common is simple text match, but you can use an expression as needed). Use $v{col:A} to specify the column A value. See above on “Using Lists” for more details on specifying the columns within the list.

The “replace” value is somewhat unique. The replace field specifies the text used in the hyperlink. If column B specifies a complete URL, you can simply use $v{col:B}.

General Actions

Title Case (Extended)

TextSoap has a built-in cleaner called “Capitalize with Title Case”. However, if you need a more customized behavior, you can use this action, which provides for extending or replacing both the small words list, and the special words list used.

Small words specify words that are to not be capitalized.
Special Words specify words that are keep the case as specified in the list.

You can often use this in combination with the “Capitalize Common Tech Names” cleaner, which will fix up special-cased words like MacBook, iPhone, TextSoap, etc.

Custom Text Wrap

TextSoap has built-in cleaners to hard wrap text at 50, 60, and 70 character boundaries. You can use this action to specify a custom value. The custom wrap will word wrap the text, attempting to keep lines <= the specified character length.

Quote Text

Quoting references an old style quoting where a common mark such as ‘>’ is used before text to quote some previous text.

The options include:

Quote level (0–10) : specifies how many of the “Mark” characters to use.

Wrap text : allows for re-wrapping the text at a fix character length

Quote Chars:

Prefix: typically a space to provide padding

Mark: The character used to quote. Commonly ‘>’, but can be any character.

Suffix: padding after the mark.

Insert Text

Allows for inserting a piece of text either before, after or replacing the text.

Tag Text

Allows for simple tagging of text. Often used as part of a conditional (see below).
While HTML tags are most common, you can specify any tag and it will surround the given text with the open tag and close tag.

Extract Text

Expression : Regular expression to match the text to extract.

Capture Group : Result group to use. $0 indicates entire match, but you can specify a sub group.

Match Case : Indicates whether to require case matching. If not selected, text case is ignored.

Append Result: If selected, will return original text along with extracted text appended. Otherwise, will replace the text with the extracted text.

Extract Characters

This action will extract the specified character count from the given text.

- From Beginning : Specify number of characters from the beginning of text.

- From End : Specify number of characters from the end of text.
result, where n is number of characters.

- Starting At : Specify the starting position and length of the text.

Perform Named Service

This action will tap into the macOS Services. Enter in the name of the Service and it will perform that Service on the given text.

This can be useful if you have an app that provides a Service menu item that gives you a specific transformation that you cannot duplicate within TextSoap itself.

Services handling in macOS is tricky on the best of days. In general, it is not recommended, but is available if it is the only way to tap into specific functionality.

A big caveat: The name needs to match exactly, and with some apps, may exist in a subfolder even though it never appears as such. BBEdit is an example of this. All its Service items need to be prefixed with a “BBEdit/” in order to be correctly accessed.

Hyperlinks to Text

This action will find rich text with a link attribute and extract the link out as specified.

Result Pattern: text template used to extract the link. Default demonstrates how to convert to the HTML “<a>” tag.

<a href="{link}">{text}</a>

Note: Special tokens will substitute the original text “{text}” and the link, “{link}”.

Keep hyperlinks active : optionally keep the link specified as clickable.

Date Formatter

This action will format dates in the preferred order. It can replicate all of the built-in Normalize Dates cleaners. In addition, it supports DateNow expansion as well.

Date - Determines whether you are working with existing dates found in text or expanding ${DATENOW}

Date Format

Date Format Options

If choosing None, Short, Medium, Long, Full, can also specify the time format.

Time Format Options

Template

When specifying Template format, full customization is available. The following chart gives all the various template keys when formatting dates and times:

Template Format Options

Line Actions

Sort Lines

This action lets you sort the lines of the given text.

- Reverse sort : Sorts in descending order

- Match case : When comparing, match based on the case of text.

- Numbers match by value : Mimic Finder’s numeric sorting of numbers.

- Ignore whitespace : Any whitespace at the beginning of the line is ignored.

- Ignore diacritic marks : Any diacritics are ignored. Example: ä = a

- Use expression : Use the regular expression & result template to transform the line before comparing.

Add Text to Lines

Add text to the beginning or end of each line of text.

Remove Text from Lines

Remove given text from the beginning or end of each line of text. Ignored if the given text is not found.

For Each Line

Convenient approach to operating on each line individually. Text is divided up into lines (using return markers), and then actions contained in the For Each Line group are applied.

Number Lines

Lines : All, Non-Empty. If Non-Empty is selected, will skip over lines that are just returns.

Start : Starting number

Increment : Increment for each number.

Format : A simple formatting string that provides for padding, tabs and spaces

0 - minimum padding. 0 = No padding (1), 000 = 3 digits min (001)
\t - tab character, Useful for using tab to separate line number and text.
\s - space character. optional. More obvious than a just a " " character (which also works)

Style Actions

When working with rich text (vs. plain text), you can change various attributes of the text. These actions can be used globally (for example, to change a font) or selectively on text using conditionals.

Set Font

An option is applied only if selected. To change only one aspect of a font, leave the remaining options unselected. For example, to set the font, but keep existing sizes, along with bold and italics, only select the “Font family” option and specify the new font family to use.

Font family: The font name to use. For example “Helvetica Neue”.

Typeface: When selected, specify plain (to remove any bold & italic), bold and/or italic. Plain option overrides bold and italic.

Size: The size of the font to use.

Set Exact Font

*Note: Use more sparingly to specify the exact font name. This allows for additional choices outside of more standard “bold” and “italic”.

Exact font: The exact font to use. For example “HelveticalNeue-MediumItalic”

Size: The size of the font to use.

Set Style

A subset feature from Set Font, which changes the typeface style. Useful if wanting to simply bold or italicize text.

If using Set Font, specify the style there. Set Style is not necessary and will just cause additional processing of the text.

Style Options

Adjust Font Size

Relative changes to the font sizes. To make everything 5 points larger, set to +5. 12pt becomes 17pt, 20pt becomes 25pt. To reduce everything by 10 pts, set to –10 pts. 24pt becomes 14pt, 48pt becomes 38pt.

Set Underline Attributes

Style: Single, Thick, Double to specify line style. None = remove this attribute (if it exists).

Pattern: Line patters defined in Apple’s Rich Text.

Color: Color of the line.

Set Strikethrough Attributes

Style: Single, Thick, Double to specify line style. None = remove this attribute (if it exists).

Pattern: Line patters defined in Apple’s Rich Text.

Color: Color of the line.

Set Super/Subscript Attribute

Script attribute:

- Superscript : make text superscript

- None : remove any superscript/subscript attribute

- Subscript: make text subscript

Set Text Color

Set color of text.

Set Background Color

Set color of background.

Remove Character Attribute

Remove the specified attribute of text.

Conditionals

Conditional blocks are a way to apply specific actions only to text that matches given criteria. The actions inside the block can be virtually anything. Conditionals can even be nested to allow you to match more and more specific criteria. Using nest conditionals, it is possible to find words that are both bold and all uppercase. This example finds all capitalize bold words and changes the text color to red.

Example 1

Modify Results

This option enables you to adjust the results of a match.

- Invert Matches : This will flip the match results. For example, if you used a conditional to find bold words and then used invert matches, the match would be on all non-bolded words.
In some cases, it is easier to specify the text you don’t want and then invert the results.

- Limit Matches : If multiple matches are found, this will limit the matches used.

- Skip Matches : If multiple matches are found, this will skip the first N matches.

If Text Matches

Find the provided text and applies related actions to it.

When using a regular expression, you can also specify which capture group to use. $0 is the default for entire matched string. However, you could specify a find as (.*?) and use $1 to only specify the text within the given tag.

For Each Line

Process each non-blank line of the text. This action does not support modifier options.

If Font Matches

Finds text that matches the font family, size(s), or face attributes (italic or bold). All the attributes can be found independently of each other. For example, you can specify Typeface: Italic to only find italic text, no matter its font family or size.

If Text Has Attribute

Finds text that matches the specified attribute.

Misc

Group

Group containers encapsulate a hierarchy of actions. They represent the foundation for conditional actions and macros.

Additionally, they allow for encapsulated organization of actions. A group container controls whether the actions within are applied. If the container is disabled, all the contained actions are ignored. This is much easier than individually disabling actions.

If you move the group, all its contained actions move with it. And if you delete a group, all the actions within are deleted.

Define List

You can define a table of data which can be used with other actions.

The table consists of rows with up to 4 columns of text. You can copy or paste tab-delimited text and use drag-n-drop to re-arrange rows.

Click Edit to display the table editor.

- Bonus: Hold down the option key when you click Edit to edit it as tab-delimited text. Each line of text represents a row. Any columns defined past 4 are ignored.

Run Automator Workflow

This action runs an Automator Workflow. Workflows are located at:

~/Library/Application Support/TextSoap/Workflows/

Note: The result of this action will be plain text

Note

This is a basic action that allows you to store notes within your cleaner. This is useful if you need to add a note to remind yourself of some detail in your cleaner. It does nothing to the text (it is ignored) when the cleaner is applied.

Additionally, if you want to make notes for a particular action, you can use the comment button on the individual action to store a short note with it.

Delete Text

Simply deletes the provided text. This is most commonly used as part of a conditional.

Macros

Macros are way to specify a group of actions that you can use multiple times within a custom cleaner. Rather than duplicating the actions, you can create a macro with the actions and apply it instead.

Define Macro

This action defines a Macro. It is also a group container. You specify the name of the Macro (which will be used later with the “Apply Macro” action). Add your actions to this group container.

Note: The contained actions within will only be applied using Apply Macro action.

Apply Macro

To apply the actions defined within a Macro, use this action. It provides a popup list of names defined macros within the cleaner. The contained actions of the Define Macro group will be applied to the given text.

Here is a slightly altered version of our cleaner example:

Example 2

Explanation: In this example, you create a macro the changes the text to red.

Instead of nesting the two conditional actions, they are separated, which means any text that is bold, or any words that are all uppercase will have the macro applied, changing the text to red.

If you wanted to change the text color to green, it now requires only one change. You can add additional actions to the macro definition and they will use used everywhere the macro is applied.