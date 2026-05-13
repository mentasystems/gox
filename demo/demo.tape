# vhs tape for gox demo. Render with: vhs demo.tape
#
# Shows the killer use case: two same-type args swapped at the call site
# compile fine, would ship, and gox catches them.

Output demo.gif

Set FontSize 18
Set Width 1200
Set Height 660
Set Padding 28
Set TypingSpeed 35ms
Set PlaybackSpeed 1.0

# Menta Systems brand palette: white bg, teal #0d5c84 accents.
Set Theme { "name": "Menta", "background": "#ffffff", "foreground": "#1a1a1a", "cursor": "#0d5c84", "selection": "#e2e8f0", "black": "#1a1a1a", "red": "#c53030", "green": "#2d7a3c", "yellow": "#94793e", "blue": "#0d5c84", "magenta": "#553c9a", "cyan": "#1a7ab0", "white": "#4a5568", "brightBlack": "#718096", "brightRed": "#e53e3e", "brightGreen": "#38a169", "brightYellow": "#b7791f", "brightBlue": "#1a7ab0", "brightMagenta": "#805ad5", "brightCyan": "#3182ce", "brightWhite": "#1a1a1a" }

# Subtle brand chrome: thin teal frame around the terminal.
Set Margin 18
Set MarginFill "#0d5c84"
Set BorderRadius 8

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
