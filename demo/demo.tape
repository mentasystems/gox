# vhs tape for gox demo. Render with: vhs demo.tape
#
# Shows the killer use case: two same-type args swapped at the call site
# compile fine, would ship, and gox catches them.

Output demo.gif

Set FontSize 18
Set Width 1200
Set Height 700
Set Padding 24
Set Theme "Catppuccin Latte"
Set TypingSpeed 35ms
Set PlaybackSpeed 1.0

# Land in the demo project.
Hide
Type "cd example && clear" Enter
Sleep 400ms
Show

Type "cat main.go"
Enter
Sleep 3500ms

Type "# Both args are strings. Wrong order compiles fine. Tests pass."
Sleep 1800ms
Enter
Sleep 700ms

Type "gox check ./..."
Enter
Sleep 4500ms

Type "# Bug caught at edit time, before it ever runs."
Sleep 2500ms
Enter
Sleep 1800ms
