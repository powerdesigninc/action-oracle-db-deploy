package utils

// Config holds the resolved cli inputs, populated once by main
type Config struct {
	Host            string
	User            string
	Password        string
	AppID           string
	InstallPath     string
	ContinueOnError bool

	// repo name, used as the history key
	Repo string
}

var Cfg Config
