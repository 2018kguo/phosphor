# Rerender docs/demo.gif from the repo root: vhs docs/demo.tape
Output docs/demo.gif
Set Shell bash
Set FontSize 16
Set Width 1100
Set Height 660
Set Padding 16
Set TypingSpeed 40ms
Set Theme "Catppuccin Mocha"

Hide
Type "(cd examples && go build -o /tmp/phosphor-mirror ./mirror); export PS1='\[\e[1;35m\]$ \[\e[0m\]'; clear" Enter
Sleep 3s
Sleep 300ms
Type "/tmp/phosphor-mirror bash" Enter
Sleep 800ms
Type "clear" Enter
Sleep 400ms
Show

Type "# a bash running inside phosphor; this window is drawn from its cells" Enter
Sleep 600ms
Type `for i in $(seq 0 63); do printf '\e[48;2;%d;%d;200m ' $((i*4)) $((255-i*4)); done; printf '\e[0m\n'` Enter
Sleep 700ms
Type `printf '\e[1mbold\e[0m \e[3mitalic\e[0m \e[4munderline\e[0m \e[9mstrike\e[0m \e[7mreverse\e[0m \e[38;2;255;140;60mtruecolor\e[0m\n'` Enter
Sleep 1200ms
Type "vim -c 'syntax on' examples/mirror/main.go" Enter
Sleep 1500ms
Type "/appendRow" Enter
Sleep 900ms
Type "zt"
Sleep 900ms
Type@60ms "jjjjjj"
Sleep 600ms
Type "o"
Type "// the gutter glows on the rows phosphor says changed"
Escape
Sleep 1200ms
Ctrl+D
Sleep 900ms
Ctrl+D
Sleep 900ms
Type ":q!" Enter
Sleep 1200ms
Type "exit" Enter
Sleep 600ms
