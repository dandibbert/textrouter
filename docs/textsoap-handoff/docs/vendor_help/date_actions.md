# Date Actions

> Converted from TextSoap 9's own help book (`date_actions.html`).

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

Date-related Cleaners

Normalize Dates to Short Format

Normalizes Dates to Short format (format is based on System Text Preferences).

Normalize Dates to Medium Format

Normalizes Dates to Medium format (format is based on System Text Preferences).

Normalize Dates to Long Format

Normalizes Dates to Long format (format is based on System Text Preferences).

Normalize Dates to Full Format

Normalizes Dates to Full format (format is based on System Text Preferences).

Normalize Dates to MM-DD-YYYY

Normalizes Dates to MM-DD-YYYY format.

Normalize Dates to DD-MM-YYYY

Normalizes Dates to DD-MM-YYYY format.

Normalize Dates to YYYY-MM-DD

Normalizes Dates to YYYY-MM-DD format.

DateNow Expansion

DateNow Expansion is a more advanced cleaner and requires a deeper dive into how it all works, particularly beyond the basic use case.

The basic use of the DateNow Expansion cleaner is to replace ${DATENOW} with today’s date (using the localized medium date format).

${DATENOW} = ‘Apr 21, 2017’.

Simple Formatting

DateNow expansion has some additional formatting available. You can specify one of the system formats (Short, Medium, Long, Full) by using the these forms of the DATENOW token:

${DATENOW-SHORT} = 04/21/2017

${DATENOW-MEDIUM} = Apr 21, 2017

${DATENOW-LONG} = April 21, 2017

${DATENOW-FULL} = Friday, April 21, 2017

This formatting will use the localized formatting, so it will reflect your current Preference settings and locale.

Basic Custom Formatting

TextSoap includes three additional built-in cleaners to normalize dates, and you can specify one of these formats as well:

${DATENOW-MM-DD-YYYY} = 04–21–2017

${DATENOW-DD-MM-YYYY} = 21–04–2017

${DATENOW-YYYY-MM-DD} = 2017–04–21

These are all the same formats used by the various Normalize Dates cleaners.

Using this cleaner

There are a couple ideas for using this cleaner.

- You can have text defined as ${DATENOW} or one of the formatting variations and the cleaner will replace those with today’s date.

- You may have some custom designator (like $DATE or {TODAY} ). With a custom cleaner, you can find $DATE, or {TODAY}, replace it with ${DATENOW} and then apply the DateNow Expansion cleaner.

Tip: If you are using a regular expression, remember to escape the $ char with a backslash (), since it has special meaning in both the Find AND Replace text.

- You can also use find and replace to change ${DATENOW} to a specific formatted version. Find ${DATENOW} replace with ${DATENOW-LONG} or other formatted version and then apply the DateNow Expansion cleaner.

- You can insert a ${DATENOW} token via a custom cleaner and then apply the DateNow Expansion cleaner to convert it to the current date.