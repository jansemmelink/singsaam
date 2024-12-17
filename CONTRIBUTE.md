# Contribute

## Next
- export auto formatted text as new source in another directory, e.g. `import` -> `files` when formatted, then allow user to edit in files, but futile when editing in imports.
- then stop auto-formatting in files to allow manual updates
- detect and list things to do on songs
- maintain word list and detect and verify new words, identify which are names and which are not.

## Progress Notes
* 2024-12-17
  * Added transformations in main.go and songs.go when loading and generating songs, to change text from all uppercase to Title case and sentence case.
    * This works well, but it is not 100%
    * Names are often now lowercase, and some places puntuation is missing, then the whole rest of the song is lowercase.
    * The bulk change is good, but then one must be able to fine tune it.
    * So these must be written to a text file that can then be edited and indicate that the transform should no longer be done after the song was fixed.
  * Before doing that, I want to make a list of all words to identify the names and automatically fix them as well, as many names repeat in many songs.
  * Once that is done, a manual review of songs must begin, and need to track who and when it was reviewed, and then no more automatic formatting.
  * Also need to get clever about all kinds of repeats, to know when parts of song repeat and display it short hand, e.g. `Herhaal Koor` or in long form by just repeating all lines from the chorus as text, depending on how the user prefers to see it.
  * Then need to start adding key changes and breaking words into sillables, only where necessary, and display the word as one when no key change occurs on the word.