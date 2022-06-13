package cmd

type Logger interface {
	Printf(format string, args ...interface{})
}
