# Clipboard Workspace

> Converted from TextSoap 9's own help book (`clipboard_workspace.html`).

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

Clipboard Workspace

What is it?

The clipboard workspace is a special editing window that makes it easy to edit and clean up the contents of the clipboard in virtually automatic fashion. Skip the manual pasting in text after a launch and copying it to the clipboard before you quit the app. Clipboard workspace does this automatically.

You can disable the clipboard workspace via Preferences > General. When disabled, TextSoap will open an untitled document on launch, like most document-based apps.

How does it work?

When TextSoap launches, the app creates a window and pastes in the current contents of the clipboard into that window for you.

You can then edit the text, clean it with any of the existing cleaners, manually find and replace text, and so forth until the text is satisfactory for your needs. While working with your text, you may use the clipboard as normal.

When you quit TextSoap, the app (automatically) copies the contents of the workspace window on the clipboard for you.

Clipboard Workspace Workflow Example

Here is one workflow available to clean text with TextSoap:

Starting text, copy to clipboard

- Open TextSoap app (contents of clipboard is pasted into Clipboard Workspace).

If you have no selection, cleaners apply to the entire text. If you optionally select text, only the selected text is cleaned.

After TextSoap Launch

- Find and click the “Uppercase” cleaner in cleaner list to apply it.

Cleaned Text

- Quit TextSoap (text from Clipboard Workspace is copied back to clipboard).

- Paste clipboard contents back into app.

Pasted Back into App

Other integration options

The clipboard workspace is great option when you may need to do more than just apply a cleaner and need to possibly change the text manually.

If you just need to apply a cleaner to text, there are some additional integration options available to simplify the workflow further using the companion textsoapAgent app:

- TextSoap Menu

- AppleScript commands

- Global Hot Key