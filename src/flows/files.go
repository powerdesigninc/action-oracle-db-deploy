package flows

import (
	"errors"
	"fmt"
	"io/fs"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/powerdesigninc/action-oracle-db-deploy/src/history"
	"github.com/powerdesigninc/action-oracle-db-deploy/src/utils"
)

var rootFolder = "files"
var scriptFolder = filepath.Join(rootFolder, "scripts")
var scriptDistFolder = filepath.Join(scriptFolder, "dist")
var styleFolder = filepath.Join(rootFolder, "styles")
var staticFolder = filepath.Join(rootFolder, "statics")

const cssFile = "application.css"
const blockSize = 200

var fileHistory *history.History
var includedScripts = false
var includedStyles = false

// hash-based upload, build and upload the files collected by @includeFile
func uploadIncludedFiles() error {
	if utils.Cfg.AppID == "" {
		return errors.New("--app-id is required to upload files")
	}

	var err error

	fileHistory, err = history.LoadFileHistory()
	if err != nil {
		return err
	}

	for _, file := range utils.IncludedFiles() {
		if strings.HasPrefix(file, "scripts") {
			includedScripts = true
		} else if strings.HasPrefix(file, "styles") {
			includedStyles = true
		}
	}

	steps := []struct {
		message string
		do      func() error
	}{
		{"Upload scripts files failed", uploadScripts},
		{"Upload styles file failed", uploadStyles},
		{"Upload static files failed", uploadStatics},
	}

	for _, step := range steps {
		if err := step.do(); err != nil {
			utils.PrintWarning(step.message)
			utils.PrintError(err)
			return utils.ErrIgnorable
		}
	}

	return fileHistory.Save()
}

// build/upload javascript files
func uploadScripts() error {
	hash, err := utils.FolderHash(scriptFolder)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			// no scripts folder, skip
			return nil
		}

		return err
	}

	// check files are changed
	if !includedScripts && fileHistory.CheckHash("scripts", hash) {
		return nil
	}

	if err := runIn(scriptFolder, "npm", "install"); err != nil {
		return err
	}

	if err := runIn(scriptFolder, "npm", "run", "build"); err != nil {
		return err
	}

	files, err := utils.ListAllFiles(scriptDistFolder)
	if err != nil {
		return err
	}

	err = uploadFiles(files, func(name string) string {
		return strings.ReplaceAll(name, scriptDistFolder, "scripts")
	})

	fileHistory.SetHash("scripts", hash)

	return err
}

// build/upload css files
func uploadStyles() error {
	hash, err := utils.FolderHash(styleFolder)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			// no styles folder, skip
			return nil
		}

		return err
	}

	// check files are changed
	if !includedStyles && fileHistory.CheckHash("styles", hash) {
		return nil
	}

	files, err := utils.ListAllFiles(styleFolder)
	if err != nil {
		return err
	}

	var indexFile string

	needFiles := []*utils.FileInfo{
		{Filename: cssFile, Fullname: filepath.Join(styleFolder, cssFile), Extension: ".css"},
	}

	for _, file := range files {
		if file.Extension == ".map" || file.Extension == ".css" || file.Extension == ".scss" || file.Extension == ".sass" {
			if strings.HasPrefix(file.Filename, "index") {
				indexFile = file.Filename
			}

			continue
		}

		needFiles = append(needFiles, file)
	}

	if indexFile == "" {
		return errors.New("no index.s[ac]ss file")
	}

	if err := runIn(styleFolder, "npx", "sass", "--style", "compressed", indexFile, cssFile); err != nil {
		return err
	}

	err = uploadFiles(needFiles, func(name string) string {
		return strings.TrimPrefix(name, rootFolder+string(filepath.Separator))
	})

	fileHistory.SetHash("styles", hash)

	return err
}

func uploadStatics() error {
	files, err := utils.ListAllFiles(staticFolder)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			// no statics folder, skip
			return nil
		}

		return err
	}

	return uploadFiles(files, func(name string) string {
		return strings.TrimPrefix(name, staticFolder+string(filepath.Separator))
	})
}

func uploadFiles(files []*utils.FileInfo, rename func(string) string) error {
	for _, file := range files {
		var apexFilePath = rename(file.Fullname)

		if fileHistory.CheckHash(file.Fullname, file.Hash()) {
			// file has been uploaded
			continue
		}

		included := utils.IncludedFiles()
		if len(included) > 0 && !utils.ContainsString(included, strings.ToLower(apexFilePath)) {
			// when there are included files, only upload the ones they list
			continue
		}

		var filebytes = file.Bytes()
		var blobBuilder = strings.Builder{}

		var tableIndex = 1
		var length = len(filebytes)
		var eof = false

		fmt.Println(utils.Green("- uploading: " + file.Fullname + ", to: " + apexFilePath))

		for start := 0; !eof; {
			end := start + blockSize
			if end >= length {
				end = length
				eof = true
			}

			fmt.Fprintf(&blobBuilder, "wwv_flow_api.g_varchar2_table(%v) := '%X';\n", tableIndex, filebytes[start:end])

			tableIndex++
			start = end
		}

		var sql = fmt.Sprintf(`
			DECLARE
				l_workspace_id number;
				l_appId number := %s;
			BEGIN
				SELECT workspace_id
				INTO l_workspace_id
				FROM apex_applications
				WHERE application_id = l_appId;

				apex_util.set_security_group_id (p_security_group_id => l_workspace_id);

				wwv_flow_api.g_varchar2_table := wwv_flow_api.empty_varchar2_table;
				%s

				wwv_flow_api.create_app_static_file (
					p_flow_id      => l_appId,
					p_file_name    => '%s',
					p_mime_type    => '%s',
					p_file_content => wwv_flow_api.varchar2_to_blob(wwv_flow_api.g_varchar2_table)
				);
			END;
			/
			Commit;`, utils.Cfg.AppID, blobBuilder.String(), apexFilePath, file.Mime())

		if err := utils.ExecuteSQLQuiet([]byte(sql)); err != nil {
			return err
		}

		// save only uploaded
		fileHistory.SetHash(file.Fullname, file.Hash())
	}

	return nil
}

func runIn(dir string, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir

	result, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println(string(result))
	}

	return err
}
