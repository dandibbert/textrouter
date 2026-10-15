# Custom Cleaner Tutorial 1

> Converted from TextSoap 9's own help book (`custom_cleaner_tutorial_1.html`).

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

Your First Cleaner

Tutorial

The best way to learn about custom cleaners is to jump in and create one. Let’s create a simple custom cleaner to fix any places where TextSoap is not properly capitalized. We have some sample text above to test against.

First we create a new cleaner and give it a useful name. Let’s call it “Fix TextSoap Spellings”

Next, we’ll add an action. For this tutorial, you will want to select the Find and Replace action.

In the Find field, enter “textsoap”
In the Replace field, enter “TextSoap”

We can test the cleaner with some provided sample text. Select, copy and then paste the following paragraph into the clipboard workspace (or a new text document).

Text Sample

Sample text: Textsoap offers a collection of tools to automate tedious manual text changes. textSOAP can save you time and increase your productivity. Once people use textsoap, they quickly find it indispensable.

Select the Preview button on the top of the window to test your cleaner.

How This Cleaner Works

By default, the Find and Replace Text action ignores case with text matching. If you need to match text specifically with case, remember to select the “Match Case” option.

In our case, “Textsoap”, “textSOAP” and “textsoap” will all match with the “textsoap” search. The action then replaces the found text with the correctly cased version “TextSoap”.

Congratulations!

You have created your first custom cleaner. You can now use this custom cleaner just like any of the pre-built cleaners.