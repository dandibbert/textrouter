# Glossary

> Converted from TextSoap 9's own help book (`glossary.html`).

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

Glossary

Action

Action is the core unit in TextSoap text processing. An action represents an individual step taken to transform the provided text. Some action examples:
- Find and Replace Text
- Rewrap Text
- Apply (an existing) Cleaner

Other apps sometimes refer to these as a “filter”.

Actions are stored in with a cleaner (see Cleaner definition).

Cleaner

A cleaner is one or more actions applied to text. It is the item you interact with when ‘cleaning’ text. There are two types of cleaners:

- Built-in : pre-defined cleaners provided with the application. There are current over 100 built-in cleaners.

- Custom : user-defined cleaners you either create or import.

Clipboard Workspace

The Clipboard Workspace represents an editable representations of the clipboard contents when the app launched. When you launch TextSoap, the contents of the clipboard will be pasted into the Workspace.

You can then apply any cleaners to that text, search and replace text, or manually edit it as you would any text document.

When you quit the TextSoap, the app will automatically copy the contents of the Clipboard Workspace back to the clipboard.

This behavior can be controlled in Preferences > General.

Group

A group is a collection of cleaners. A group can be used to organize specific cleaners based on a task or general need. When using the companion app, textsoapAgent, you can also use a custom group to define which cleaners will be used for custom services. Like cleaners, there are two types of groups:

- Built-in : collections organized based on common functionality such as typography, HTML-related or Email-related.

- Custom : user-defined groups you create for your own workflow, task needs.

Text Editing

TextSoap uses the standard macOS text editing environment to provide you with a full featured text editor that can handle both plain text and rich text. If you use TextEdit, Apple’s default text editor app, you know the fundamentals of TextSoap’s text editor.

There are a few additions added to help you in the process of cleaning up text:

- Show Invisibles : show lurking spaces, tabs, end-of-line markers and more

- Show Line Numbers : navigate the various lines of text

- Wrap Text : control whether to soft-wrap paragraphs or not. Unwrapped text will flow horizontally.

- Statistics : understand the basic structure of the text you are working with.