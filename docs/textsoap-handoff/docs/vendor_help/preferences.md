# Preferences

> Converted from TextSoap 9's own help book (`preferences.html`).

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

Preferences

Note: Preference options may differ based on edition.

General

Appearance

Specifies whether app’s appearance follows System, is always light mode or always dark mode.

- System

- Light

- Dark

Application

Quit application when last window is closed

Enable to automatically quit the application when the last text document is closed. This can be useful when the Clipboard Workspace (see below) is disabled.

Apply cleaner to entire contents when no selection

Enable to automatically apply selected cleaner to the entire document (or text contents) when no selection is found. If option is disabled, cleaners will only be applied if there is a selection.

MyScrub action

Specify cleaner to use when MyScrub is called. This may be Scrub (default) or any custom cleaner you create in MyLibrary.

Default group

Specify the default group to display when a new text window is created, whether it is the clipboard workspace, a new document or a document from file. Once the window is created, you can change the group displayed.

Groups: Specify which groups to display

- Built-in : Custom – List built-in groups, then custom groups.

- Custom : Built-in – List custom groups, then built-in groups.

- Custom – Show only custom groups.

Compact display: Enable to display cleaner group using a more compact layout.

Clipboard Workspace

Specifies whether to use Clipboard Workspace and what format the text should be:

- Plain Text Format

- Rich Text Format

- Disabled

New Documents

Specify the type of document to create by default (cmd-N).

- Plain Text Document

- Rich Text Document

Note: You can also directly create either a rich text document (ctrl-cmd-N) or a plain text document (opt-cmd-N), bypassing this preference.

Text Editing

Many of these options are default values for text editor. You can later change many of these options via the Edit or View menu.

Display

Editor

- Show ruler (rich text) : Display the paragraph ruler when editing rich text. Allows changing of tab stops, as well as left, right and indent margins on a per-paragraph basis.

- Show invisibles : Display spaces, tabs, end of line (returns) found in the text.

- Show line numbers : Show the line numbers. Paragraphs are considered lines.

- Show word count : Display count of paragraphs (lines), words and characters. Character count without spaces is displayed in parentheses.

- Highlight current line : Toggles whether to highlight where cursor is.

- Use Inspector Bar : Use newer inspector bar and ruler interface (vs. classic)

- Use Dark Background : Determines if text background is dark in dark mode. Otherwise, text is displayed as black text on a white background.

Search

- Regex : Enable initial use of Regex in interactive find.

- Match Case : Enable match case by default (if unselected: ignore case by default).

- Match Lines : When using Regex interactive find, indicates whether Match Lines option is initially set, causing ^ and $ to act as anchors for lines instead of entire text.

Zoom

Default zoom to use when displaying new or opened documents.

Invisibles

Color used to display invisible characters, such as spaces, tabs, returns, etc. Select “Default” to reset.

Current Line Color

Line color to use to highlight the current line. The color selected is blended together (to insure text remains readable), the result is shown next to the color selection. Select “Default” to reset.

Editing

Options

- Check spelling as you type : Enable the automatic spell checker to mark potentially misspelled words as they are entered into the document.

- Check grammar with spelling : Use the system’s grammar checker when spell checking.

- Correct spelling automatically : Allow for automatic correction of words with the spell checker.

- Text replacement : Automatic substitution of a variety of static text items based on user preferences. You can specify replacements in System Preferences > Keyboard > Text.

- Smart copy/paste : Controls whether the receiver inserts or deletes space around selected words so as to preserve proper spacing and punctuation.

- Smart quotes : Automatic quote substitution causes ASCII quotation marks and apostrophes to be automatically replaced, on a context-dependent basis, with more typographically accurate symbols.

- Smart dashes : Enables automatic conversion of a two ASCII hyphen (-) characters into an em-dash.

- Smart links : Causes text representing URLs typed in to be automatically made into hyperlinks.

- Data detectors : Automatic data detection enables detection of dates, addresses, and phone numbers.

Rich Text Font

Specify the default font when creating new Rich text editor. If the editor (for either Clipboard Workspace or Text documents) is a plain text editor, it will use Plain Text Font.

Plain Text Font

Specify the default font when creating or opening plain text documents.

TextSoap Agent

Provides access to learn about and download the textsoapAgent helper app.

The helper app works with all editions of the main TextSoap app, providing integration universal set of integration options, including:

- Global TextSoap Menu : Providing access to cleaners within other applications or to clean the clipboard

- Global Shortcuts : Providing hot key access to cleaners within other applications.

- AppleScripting : Providing a centralized scripting endpoint. By targeting application “textsoapAgent” – your scripts can work with all editions of TextSoap.

- Services : All Services are handled by textsoapAgent, including the standard Service item, which will present a dialog to choose which cleaner to apply.

- Custom macOS Services : Additionally, textsoapAgent supports providing a customized set of actions to apply specific cleaners to the selected text via Services. This new mechanism is more robust than previous efforts.

- PopClip : The latest extension for PopClip now supports the new textsoapAgent.

- Alfred : Another popular option is to use Alfred’s scripting to direct cleaning actions with the textsoapAgent.

Select the “Learn More…” button to direct you to the website with information and access to download the new agent. Whether you are working with the Direct edition, Setapp edition or some other edition, the TextSoap Agent provides a single access point to integrate TextSoap’s features with other apps.

Learn More…

Updates

Note: Only applies to Direct edition. Updates in other editions use their specific mechanism. Beta releases are not available in other editions.

Automatically check for updates

Recommended to remain on. When selected, this option will regularly check for any software updates.

Check Now

Immediately check for any updates available.

Show Beta Releases

When selected, this option will include beta updates as well as official updates. Recommended only for advanced users.

Advanced

Some more advanced functionality provided. Made available in the case they are needed. Some of the options may require a relaunching of the application to take full effect.

General

Disable Custom App Icons

Turns off special day icons and seasonal icons.

Flip Season Icons for Southern Hemisphere

Some of our customers live south of the equator where the seasons are reversed. So when we enjoy Spring, they are enjoying Fall. This flips the icons to match those seasonal dates better.

Text Editor

Don’t warn when switching from rich to plain text

Switching converting a document from rich text to plain text will throw away any style information with the text. Normally a warning is put up every time the conversion is selected. Enable this option to perform the conversion without a warning.

Append “.txt” extension to plain text files

Mimics TextEdit’s save option for plain text. Enable to automatically append the “.txt” extension when no extension is given. A plain text document saved as “MyDocument” becomes “MyDocument.txt”. If you specify an extension, TextSoap will use it instead. A plain text document named “MyDocument.html” stays unchanged.

Custom Cleaner Editor

New find action based on text search preferences

When adding a new Find action within a custom cleaner, this option determines whether to use the search preferences set in Text Editing panel. Useful to override the “Match Lines” and “Match Case” options for Regex (if you are so inclined).