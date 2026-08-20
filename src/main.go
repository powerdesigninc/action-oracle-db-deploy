package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/powerdesigninc/action-oracle-db-deploy/src/flows"
	"github.com/powerdesigninc/action-oracle-db-deploy/src/utils"
)

func main() {
	defer func() {
		if err := recover(); err != nil {
			utils.PrintError(err)
			os.Exit(1)
		}
	}()

	var flowsInput string

	flag.StringVar(&utils.Cfg.Host, "host", "", "database connection string, e.g. host:1521/service")
	flag.StringVar(&utils.Cfg.User, "user", "", "database user/schema")
	flag.StringVar(&utils.Cfg.Password, "password", "", "database password")
	flag.StringVar(&utils.Cfg.AppID, "app-id", "", "APEX application id, required when files are uploaded")
	flag.StringVar(&utils.Cfg.InstallPath, "install-path", "", "folder holding the install sql files")
	flag.StringVar(&flowsInput, "flows", "installs", "comma separated flows to run: installs, files2")
	flag.BoolVar(&utils.Cfg.ContinueOnError, "continue-on-error", false, "keep executing the remaining install files after a failure")
	flag.Parse()

	if err := resolveConfig(); err != nil {
		utils.PrintError(err)
		os.Exit(1)
	}

	for _, name := range strings.Split(flowsInput, ",") {
		name = strings.TrimSpace(strings.ToLower(name))
		if name == "" {
			continue
		}

		flow, ok := flows.Flows[name]
		if !ok {
			utils.PrintError("unknown flow: " + name)
			os.Exit(1)
		}

		fmt.Println(utils.Green("# Executing " + name))

		err := flow()
		if err == nil {
			fmt.Println()
			continue
		}

		if errors.Is(err, utils.ErrIgnorable) {
			break
		}

		if !errors.Is(err, utils.ErrNoPrint) {
			utils.PrintError(err)
		}

		os.Exit(1)
	}
}

func resolveConfig() error {
	required := map[string]string{
		"host":         utils.Cfg.Host,
		"user":         utils.Cfg.User,
		"password":     utils.Cfg.Password,
		"install-path": utils.Cfg.InstallPath,
	}

	for name, value := range required {
		if value == "" {
			return fmt.Errorf("--%s is required", name)
		}
	}

	repo := os.Getenv("GITHUB_REPOSITORY")
	if repo == "" {
		return errors.New("$GITHUB_REPOSITORY is required")
	}
	utils.Cfg.Repo = repo[strings.LastIndex(repo, "/")+1:]

	return nil
}
