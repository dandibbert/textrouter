# Whats New

> Converted from TextSoap 9's own help book (`whats_new.html`).

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

What’s New in TextSoap 9

TextSoap 9 built for macOS 13 (Ventura). Supports macOS 11 or later.

If you are coming from a previous version of TextSoap, here is a quick rundown of some of the changes from the previous major release. We’re always working to improve the experience.

- General

- Apple Silicon support

- Dark Mode support

- New interface

- New Custom Cleaner Actions

- Define List action

- Self contained lists items can be copied/pasted between cleaners

- Lists can be defined anywhere within a cleaner.

- Resizable List/Table editor

- Define each list with 1 to 4 columns

- Optional text-based editor allows editing list as tab-delimited text lines.

- To edit as text, hold down option key when clicking edit button.

- For Each Line action

- Simplifies processing all non-blank lines

- Perform Service action

- Perform a macOS Service by name.

- Several common services are provided in a popup menu.

- Repeat Action

- Repeat contained actions a specified number of times.

- Process List action

- Define variables for each row in selected list with your actions

- Use special variables with Find/Replace, If Text Matches to quickly process mulitple find/replace

- Updated Custom Cleaner Actions

- Find & Replace actions

- List support added (replaces Batch Find & Replace)

- More prominately displays option values (m,s,w,x) for Regex

- Defaults to most common regex option: m - match lines, enabled.

- More prominately displays granularity for plain text finds

- If Text Matches action

- Search history option added

- Insert special characters option added

- Now only displays valid capture groups, based on given regular expression.

- Sort Lines action

- New options added:

- Match case

- Ignore diacritic marks

- Skip leading whitespace for comparison

- Sort numbers as values

- Use a regular expression for comparison

- If-Match (aka Conditional) actions

- Integrated option to modify results (invert, skip, limit results of match)

- Set Underline/Strikethrough action

- New “By Word” option

- Extract Characters action

- Replaces Extract Beginning Characters, Extract End Characters, and Extract Middle Characters

- New Cleaners

- Ruler: Clear All Tab Stops - clear all tab stops from selected paragraph rulers.

- New BBCode-related tag cleaners

- Enable Ligatures, Disable Ligatures

- Remove Line Numbers - removes line numbers at the beginning of each line.

- Text Editor

- Support for text editing with dark background in Dark Mode. (#1360)

- Explicit option to use Inspector Bar or old-style Ruler.

- Cleaner groups now more prominent

- Sidebar is now resizable

- Improved handling of line numbers

- When using regex, each match selection also displays its

- capture group list

- replacement text result

- My Groups Editor

- New editor in a single window interface.

- Dark mode support

- You can use an existing group as a template for creating a new group.

- Item tags replace colors, dynamically adjusting for Light / Dark Mode.

- My Cleaners Editor

- New editor in a single window interface.

- Dark mode support

- Actions display options inline for a more compact display, where appropriate

- Container actions are more visually distinctive

- Container actions show number of contained actions

- New color scheme for action categories

- Disabled actions strikeout title

- Disabled actions no longer allow editing

- Action comments can contain returns & tabs

- More robust auto-resize handling for actions with text fields

- Preview window shows invisibiles in cleaned text preview

- Improved performance when cleaner contains a lot of actions

- New find actions are now based on text search preferences

- New Preferences

- Presentation

- Appearance : use Light or Dark Mode or System theme

- Text Editing Interactive Search Defaults:

- Search type

- Match Case option

- Match Lines option (for Regex)

- General Improvements

- Pick cleaner window (used by Services, AppleScript) moves to active space as well

- Showing horizontal ruler (when toggling between plain/rich text) no longer hides first few lines of text.

- Action colorization provides dynamic color support between Dark/Light mode.

- Regex syntax coloring supports Dark/Light mode.

- TextSoap Agent Companion App

- single app integrates with all editions of main TextSoap app

- provides more efficient interaction between agent/main app for better cleaning performance

- Global Cleaner Access (use within many of your favorite apps)

- Global Hot Keys (hot key to activate a cleaner within an app)

- AppleScripting

- Mac OS Services support

- Customized Services (bring specific cleaners into Service menu)