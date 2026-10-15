# Cleaner Descriptions

> Converted from TextSoap 9's own help book (`cleaner_descriptions.html`).

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

Other Cleaner Descriptions

SCRUB

Multi-step cleaner addresses most common cases of extra spaces, forwarding characters, MIME encoding issues and broken paragraphs. SCRUB calls the cleaners in order to insure the best results.

MyScrub

A customizable version of the SCRUB version. Defined using the customize window, it allows augmenting or replacing cleaners used in SCRUB. Any available cleaner may be added to this list.

If no cleaners are specified, MyScrub defaults to SCRUB.

Remove Extra Spaces

Converts non-breaking spaces to standard spaces, then removes two or more spaces with a single space. Leading spaces at the beginning of a line are also removed, as are trailing spaces.

Remove Forwarding (>) Characters

Removes one or more sets of forwarding characters (>) on a line, commonly found in forwarded email.

Note:Does not yet support HTML archives, like those used in Apple Mail.

Convert MIME-encoded Characters

Converts %Hex or =Hex characters to their appropriate ASCII character. For example, a %20 or a =20 is converted to a space (hex 0x20 is the ASCII value of a space character).

Make Paragraphs

Converts multi-line paragraphs (with hard carriage returns CR) into a single paragraph. Also adds a space to the text when removing CRs to avoid smashing words together. Two or more CRs in a row indicate a new paragraph.

Remove All Tabs

Deletes all tabs from the selected text.

Remove Control Characters

Deletes any control characters (characters with an ASCII value less than 32) from the selected text (except for line feeds or carriage returns).

Remove Extra Returns

Converts multiple carriage returns into a single carriage return. Behaves the same as Multiple Return to 1 Return

Uppercase

Converts text to “UPPERCASE”.

Lowercase

Converts text to “lowercase”.

Capitalize Sentences

Converts text to lowercase, capitalizing the first character of each sentence.

Capitalize Words

Converts text to lowercase, then capitalizes each individual word.

Title Case

Converts text to “Title Case”. Words with non-standard case (iTunes, iPhone, TextSoap) remain unchanged, small words (a, an, and, as, at, but, by, en, for, if, in, of, on, or, the, to, via,) are converted to lowercase unless the word appears at the beginning or end of the line. All other words are capitalized. Create a custom cleaner with the “Title Case with Options” action to customize these options.

Convert Bullets to ASCII

Converts bullets to standard ASCII dashes.

Straighten Quotes

Converts single and double smart (or curly) quotes to straight quotes.

Smarten Quotes

Convert single and double straight quotes to smart (or curly) quotes.

Expand Tabs

Converts tabs (hex 0x09) to four (4) spaces.

Rewrap Text

Fixes any paragraphs (using Make Paragraphs), then rewraps the text at 70 characters per line. Behaves the same as the Wrap At 70 cleaner.

Quote Text

Rewraps the text and re-inserts forwarding marks (i.e.. ‘>’) at the beginning of each line. It also places hard returns at the end of each line to ensure proper wrapping in email applications.

Wrap At 50

Fixes any paragraphs (using Make Paragraphs), then rewraps the text at 50 characters per line.

Wrap At 60

Fixes any paragraphs (using Make Paragraphs), then rewraps the text at 60 characters per line.

Wrap At 70

Fixes any paragraphs (using Make Paragraphs), then rewraps the text at 70 characters per line.

Quote Text L1

Rewraps the text and places a single forwarding mark in front of the text.

“Sample Text” becomes “> Sample Text”.

Quote Text L2

Rewraps the text and places two (2) forwarding marks in front of the text.

“Sample Text” becomes “>> Sample Text”.

Quote Text L3

Rewraps the text and places three (3) forwarding marks in front of the text.

“Sample Text” becomes “>>> Sample Text”.

Quote Text L4

Rewraps the text and places four (4) forwarding marks in front of the text.

“Sample Text” becomes “>>>> Sample Text”.

Convert to Internet Friendly Text

Converts text to Internet Friendly Text. Diacritical values are stripped, quotes straightened, and additional characters (copyright, trademark, registered, ellipse, em-dash, en-dash) are converted to ASCII equivalents.

Remove All High Ascii Characters

Removes All High Ascii Characters. Strips off any ASCII values 128–256 from the text.

1 Return to 2 Returns

Converts each carriage return to 2 carriage returns.

1 Return to Space

Converts each carriage return to a space.

Strip 2 or more Returns

Strips any 2 or more consecutive returns to a single return.

Strip 2 or more Spaces

Strips any 2 or more consecutive spaces to a single space.

Strip 2 or more Tabs

Strips any 2 or more consecutive tabs to a single tab.

Strip 3 or more Returns

Strips any 3 or more consecutive returns to a single return.

Strip 3 or more Spaces

Strips any 3 or more consecutive spaces to a single space.

Strip 3 or more Tabs

Strips any 3 or more consecutive tabs to a single tab.

Strip 4 or more Returns

Strips any 4 or more consecutive returns to a single return.

Strip 4 or more Spaces

Strips any 4 or more consecutive spaces to a single space.

Strip 4 or more Tabs

Strips any 4 or more consecutive tabs to a single tab.

Strip Quoting (>) Characters

Removes the forwarding characters on a line. Same as Remove Forwarding (>) Characters.

Multiple Return to 1 Return

Converts multiple carriage returns to a single return.

Multiple Return to 2 Returns

Converts multiple carriage returns to 2 returns.

Strip NULL

Strips any NULL characters in the selected text.

Note: This is a remnant cleaner from 1998. Most null values are stripped out before the text is ever processed

Ellipsis to Three periods

Converts ellipsis characters to three periods.

Em Dash to 2 Hyphens

Converts Em Dash characters to two hyphens.

En Dash to Space-Space

Converts En Dash character to a space dash space sequence “ - ”.

Three periods to Ellipsis

Converts three periods to an ellipsis character.

2 Hyphens to Em Dash

Converts two hyphens to an Em Dash character.

Space-Space to En Dash

Converts a space-dash-space sequence to an En Dash character.

Remove Forwarded Text

Delete any quoted text. Any lines of text preceded by a forwarding mark (>) are removed.

ROT 13 Encryption

Applies a very rudimentary encryption. Apply the encryption again to restore the text.

String to Hexadecimal

Converts a string of characters to a hexadecimal sequence. Use Hexadecimal to String to restore.

Hexadecimal to String

Converts a hexadecimal sequence to a string of characters. Use with String to Hexadecimal.

Replace @ with (AT)

Replaces “@” with “(AT)”.

Increase Quote Level

Increases the quote level of a selection by one, adding a forwarding mark (>) to the beginning of each line. Rewraps the paragraph as necessary.

Get Version String

Returns the current version of TextSoap.

Unix (LF) Line Endings

Converts the selected text to use Unix (LF) Line Endings.

Obsolete: use the new line-ending popup for best results.

Mac (CR) Line Endings

Converts the selected text to use old-style Mac (CR) Line Endings.

Obsolete: use the new line-ending popup for best results.

DOS (CR+LF) Line Endings

Converts the selected text to use DOS (CR+LF) Line Endings.

Obsolete: use the new line-ending popup for best results.

Trim Beginning of Lines

Removes any extra spaces from the beginning of the selected text lines.

Trim End of Lines

Removes any extra spaces from the end of the selected text lines.

Trim Lines

Removes any extra spaces from both the beginning and the end of the selected text lines.

Convert Non-Breaking Spaces

Converts Non-Breaking Spaces to standard spaces.

Remove Style from Text

Removes formatting style from the selected text, leaving text in a single font and style.

Quote Based on First Line

Quotes the selected text based on first line. The text is rewrapped. If forward marks are found at the very beginning of the text, the number found determines the level of quoting applied to that paragraph.

Remove Hyperlinks from Text

Removes clickable hyperlinks from the text, leaving only the text.

Make URLs Clickable

Finds any recognizable URLs and turns them into clickable hyperlinks.

Extract URLs by Replacing

Finds any recognizable URLs and extracts them, replacing the original text.

Extract URLs by Appending

Finds any recognizable URLs and extracts them, appending the result to the original text.

Sort Lines in Ascending Order

Sorts lines in increasing order. Lines are marked by paragraph markers.

Sort Lines in Descending Order

Sorts lines in decreasing order. Lines are marked by paragraph markers.