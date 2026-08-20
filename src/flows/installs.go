package flows

import (
	"fmt"
	"strings"

	"github.com/powerdesigninc/action-oracle-db-deploy/src/history"
	"github.com/powerdesigninc/action-oracle-db-deploy/src/utils"
)

// execute the sql files under the install path
func runInstalls() error {
	fmt.Println("Install Folder: " + utils.Cfg.InstallPath)

	files, err := utils.ListSQLFiles(utils.Cfg.InstallPath)
	if err != nil {
		return err
	}

	if len(files) == 0 {
		fmt.Println(utils.Yellow("No file under " + utils.Cfg.InstallPath))
		return nil
	}

	if err := utils.TestConnection(); err != nil {
		return err
	}

	fmt.Println("Database Connection: " + utils.Cfg.Host)
	fmt.Println("User/Schema: " + utils.Cfg.User)

	his, err := history.LoadInstallHistory(files)
	if err != nil {
		return err
	}

	summary := make(map[string]bool)

	var isFailed = false
	var hasIncludedFiles = false

	for _, file := range files {
		fileHash := file.RecurredSQLHash()
		if his.CheckHash(file.Filename, fileHash) {
			// file has been executed
			continue
		}

		fmt.Println("\n" + utils.Green(fmt.Sprintf("## Executing %v", file.Filename)))

		output, err := utils.ExecuteSQLOutput(file.Bytes())

		fmt.Println(output)

		if err != nil {
			// fail
			summary[file.Filename] = false
			utils.PrintError(err)

			if !utils.Cfg.ContinueOnError {
				isFailed = true
				break
			}

			continue
		}

		// succeed, check if the output includes special errors
		if strings.Contains(output, "compilation errors") ||
			strings.Contains(output, "unable to open file") ||
			strings.Contains(output, "insufficient privileges fail") {
			utils.PrintWarning("Non-ignorable errors in " + file.Filename)
		}

		summary[file.Filename] = true
		his.SetHash(file.Filename, fileHash)

		if utils.ScanIncludedFiles(file.Bytes()) {
			hasIncludedFiles = true
		}
	}

	if err := his.Save(); err != nil {
		return err
	}

	// summary
	fmt.Println("\n" + utils.Green("## SUMMARY"))
	for _, file := range files {
		status, ok := summary[file.Filename]
		switch {
		case !ok:
			fmt.Printf("- %v : Skipped\n", file.Filename)
		case status:
			fmt.Printf("- %v : %v\n", file.Filename, utils.Green("Success"))
		default:
			fmt.Printf("- %v : %v\n", file.Filename, utils.Red("Fail"))
		}
	}

	if isFailed {
		return utils.ErrNoPrint
	}

	if hasIncludedFiles {
		fmt.Println("\n" + utils.Green("## Uploading included files"))

		return uploadIncludedFiles()
	}

	return nil
}
