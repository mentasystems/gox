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
Type "cd demo/example && clear" Enter
Sleep 300ms
Show

Type "cat main.go"
Enter
Sleep 3s

Type "# Both args are strings. Wrong order compiles fine."
Sleep 1500ms
Enter
Sleep 800ms

Type "gox check ./..."
Enter
Sleep 4s

Type "# gox catches the swap-prone call site."
Sleep 1500ms
Enter
Sleep 600ms

# Fix by adding inline /* paramName */ comments.
Type `sed -i '' 's|transfer("o-42", "u-7")|transfer(/* userID */ "u-7" /* orderID */, "o-42")|' main.go`
Enter
Sleep 800ms

Type "cat main.go"
Enter
Sleep 3s

Type "gox check ./..."
Enter
Sleep 2500ms

Type "# clean. Bug prevented, call site self-documenting."
Sleep 2s
Enter
Sleep 1500ms
