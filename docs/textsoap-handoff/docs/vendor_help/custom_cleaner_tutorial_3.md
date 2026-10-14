# Custom Cleaner Tutorial 3

> Converted from TextSoap 9's own help book (`custom_cleaner_tutorial_3.html`).

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

Using Lists in Your Cleaners

Tutorial

Lists are a powerful feature in TextSoap. Rather than creating individual find and replace actions for each piece of data, you can create a template find and replace action and connect it to a list of data. The action is then applied with each row of the data.

The following actions currently be used within lists:

- Find and Replace Text

- Create Hyperlink URLs

- If Matches Text

Let’s create an example using Create Hyperlink URLs. This action will take matches found in column A and convert them to hyperlinks in Rich Text using the link address in column B.

For URLs like https://www.unmarked.com or https://www.apple.com, we could just use the cleaner ‘Convert URLs to Hyperlinks’. In this example, we will take it one step further. We will take references to ‘Unmarked Software’ and ’Apple and link them to the appropriate URL.

First, we will create a new cleaner and call it “Lists Example”. Then we will add a Define List action.

Click on Edit to add some table data. We set the column count to 2. Column A will be company names, and column B will be the associated URLs.

We will fill it with following data:

Unmarked Software https://www.unmarked.com
Apple https://www.apple.com
TextSoap https://textsoap.com/

For this example, we have any entry for Unmarked Software, Apple, and TextSoap.

To use a list, you add a Process List container action. Then add the action(s) you want to be processed using the list data. The cleaner will process each row in the list, passing the data to each of the actions contained.

Now add a “Create Hyperlink URLs” action and specify we want match column A ( $v{A} ) and add the hyperlink indicated by column B ( $v{B} ).

Sample text

Unmarked Software prepares for the release of macOS Mojave from Apple. TextSoap 9 has been tested against macOS Sierra and looks to be compatible.

Because we want to create hyperlinks, we need to to make sure the clipboard workspace we test against is defined as rich text.

And here’s what the preview looks like:

With this cleaner, we can now add a name and URL to the list data to handle more hyperlinks.

You can also copy and paste the Define List action to another cleaner if you want to use the data somewhere else.