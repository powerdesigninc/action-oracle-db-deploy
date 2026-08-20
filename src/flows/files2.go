package flows

import (
	"fmt"

	"github.com/powerdesigninc/action-oracle-db-deploy/src/utils"
)

// include-based upload, scan @includeFile in the install files and upload them
func runFiles2() error {
	files, err := utils.ListSQLFiles(utils.Cfg.InstallPath)
	if err != nil {
		return err
	}

	if len(files) == 0 {
		fmt.Println(utils.Yellow("No file under " + utils.Cfg.InstallPath))
		return nil
	}

	var hasIncludedFiles = false

	for _, file := range files {
		if utils.ScanIncludedFiles(file.Bytes()) {
			hasIncludedFiles = true
		}
	}

	if hasIncludedFiles {
		fmt.Println("\n" + utils.Green("## Uploading included files"))

		return uploadIncludedFiles()
	}

	return nil
}
