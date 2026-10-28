module github.com/2018kguo/phosphor/examples

go 1.26.0

require (
	github.com/2018kguo/phosphor v0.1.0
	github.com/creack/pty v1.1.24
	golang.org/x/term v0.47.0
)

require golang.org/x/sys v0.49.0 // indirect

replace github.com/2018kguo/phosphor => ../
