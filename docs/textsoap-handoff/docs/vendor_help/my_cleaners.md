# My Cleaners

> Converted from TextSoap 9's own help book (`my_cleaners.html`).

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

Cleaner Editor

A custom cleaner is one or more user-defined actions that can be applied to text. For example, you can use a “Find and Replace” text action to set up an automated find and replace on given text. This action can include regular expressions, a powerful text matching syntax. Another example might be to apply another cleaner (either built-in or custom), and then add additional actions, allowing you to build on existing cleaners.

Text vs. Style

TextSoap supports working with both plain text and rich text. When working with rich text, the text formatting (style) of the text plays an additional role.

You can make text change actions on rich text and TextSoap will attempt to maintain the existing character styles from the original text. This means that bold words stay bold, italic words remain italic, and underlining, fonts and text color stays intact, even as the text is updated.

TextSoap also enables changing the underlying (character-level) format of your text. This is useful if you need to strip out existing styling like bold or italic, adjust the font, or size. TextSoap can perform the style changes selectively based on the underlying text.

Important Note: Microsoft Word text may also use Stylesheets for formatting. Unfortunately, macOS support for RTF does not extend to using document-based stylesheets. TextSoap is unable to support maintaining these styles.

Overview

The MyLibrary editor provides you with a single window interface for all your cleaners. A list of cleaners is displayed on the left along with some additional options: Import, Export, Remove.

- Import : Enables you to add cleaners from libraries or cleaner files. You select the files to import using the system dialog or drag the files into the window.

- You can add multiple files

- You can selectively import the individual cleaners from within a file.

- Export : Enables you to save one or more cleaners. Cleaners can be saved as individual files or combined into a single library file. Used these files as backups or to share your cleaners with others.

- Remove : Allows you to delete multiple items (both cleaners and groups) in one action. TextSoap will prompt you one more time before deletion.

Buttons:

- Add : Add a new cleaner or group.

- Duplicate : Duplicate the selected item.

- Export : Export the selected item.

- Delete : Delete the selected item.

Custom Cleaner Editor

Action Workspace

The action workspace defines the currently selected cleaner.

- Cleaner Name : Change name or other properties (author or description) of a cleaner. Click on the name at the top to display and edit its properties.

- Preview : Quickly test the current cleaner using the text in the clipboard workspace (or other specified open document).

- Action Toolbox : Categorized actions that you can add to a cleaner.

- All : combination of Actions & Cleaner actions.

- Actions : core actions within a cleaner you can add.

- Cleaners : Existing cleaners can be called as actions.

Actions have common attributes:

- Show Details : toggle details of an action

- Name : type of action

- Additional properties : Disable (or enable) and edit comment for specific action

Action Header

If action is disabled, the title is struck, details are hidden (and are no longer editable).

Action Header

Any comments are displayed as a tooltip when you hover your cursor over the title of the action.

Container Actions

This class of action can contain other actions, much like a folder holds files. They can be used as an organization tool (a simple Group action) or used to limit the text you wish to further work on (a Conditional). Containers have a visible notch to redirect your eye in the workflow, making the hierarchy easier to see.

When a container action is disabled, the contained actions will NOT be executed.

Additionally, containers have:

- Disclosure Button : Toggles whether items are displayed. Works similar to Finder’s folders.

- Contained Count : Indicates how many direct items are contained.

Action List

See My Cleaner Actions for descriptions of the various core actions.