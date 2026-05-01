package tail

import "regexp"

var (
	reFatal = regexp.MustCompile(`(?i)(fatal|panic)`)
	reError = regexp.MustCompile(`(?i)(error|err|critical)`)
	reWarn  = regexp.MustCompile(`(?i)(warn|warning)`)
	reDebug = regexp.MustCompile(`(?i)(debug|trace)`)
)

func detectLevel(line string) string {
	switch {
	case reFatal.MatchString(line):
		return "fatal"
	case reError.MatchString(line):
		return "error"
	case reWarn.MatchString(line):
		return "warn"
	case reDebug.MatchString(line):
		return "debug"
	default:
		return "info"
	}
}
