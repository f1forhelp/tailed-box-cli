module github.com/f1forhelp/tailed-box-cli

go 1.25.1

require (
	github.com/f1forhelp/tailed-box-cli/packages/control v0.0.0
	github.com/f1forhelp/tailed-box-cli/packages/securemesh v0.0.0
)

replace github.com/f1forhelp/tailed-box-cli/packages/control => ./packages/control

replace github.com/f1forhelp/tailed-box-cli/packages/securemesh => ./packages/securemesh
