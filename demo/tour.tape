Output tour.gif

Set TypingSpeed 200ms
Set Shell "bash"
Set FontSize 32
Set Width 1200
Set Height 600

# Create a demo directory
# TODO Maybe move this into another script
Hide
Type@0ms "rm -f -R /tmp/lsw-demo && mkdir -p /tmp/lsw-demo && cd /tmp/lsw-demo" Enter
Type@0ms "clear" Enter
Show

# Create a dummy file
Type "echo Hello world > test.txt" Enter
Type "ls" Sleep 500ms Enter

# Enter windows shell
Type "lsw shell win" Sleep 500ms Enter
Sleep 3

Type "cat test.txt" Sleep 500ms Enter
Sleep 1s
Type "rm test.txt" Enter


# Compile a project with xmake
Type "xmake create cxx-proj" Sleep 500ms Enter
Sleep 1s
Type "cd cxx-proj" Sleep 500ms Enter
Type "ls" Sleep 500ms Enter
Sleep 1s
Type "xmake" Sleep 500ms Enter
Sleep 1s
Type "xmake run" Sleep 500ms Enter
Sleep 1s


Type "cd .." Sleep 500ms Enter

# Compile a project with cargo
Type "cargo new rust-proj" Sleep 500ms Enter
Sleep 1s
Type "cd rust-proj" Sleep 500ms Enter
Type "cargo build" Sleep 500ms Enter
Sleep 1s
Type "cargo run" Sleep 500ms Enter
Sleep 1s

# Exit windows shell
Type "exit" Sleep 500ms Enter
Sleep 2s

Type "ls" Sleep 500ms Enter
