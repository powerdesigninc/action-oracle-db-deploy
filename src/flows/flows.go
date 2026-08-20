package flows

// Flows maps the --flows names to their entry point
var Flows = map[string]func() error{
	"installs": runInstalls,
	"files2":   runFiles2,
}
