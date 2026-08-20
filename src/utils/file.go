package utils

import (
	"crypto/sha256"
	"fmt"
	"mime"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

type FileInfo struct {
	// only filename
	Filename string
	// filename with path
	Fullname string

	Extension string
	mime      string
	bytes     []byte
	hash      string
}

var sqlFileRegex = regexp.MustCompile(`(?m)^@?@"(.+?)"`)
var includedFileRegex = regexp.MustCompile(`\@includeFile\((.+?)\)`)
var excludedFiles = []string{"00_template.sql"}
var includedExts = []string{".sql"}

var includedFiles []string

func (f *FileInfo) Bytes() []byte {
	if f.bytes == nil {
		var err error
		f.bytes, err = os.ReadFile(f.Fullname)
		if err != nil {
			panic(err)
		}
	}

	return f.bytes
}

func (f *FileInfo) Hash() string {
	if f.hash == "" {
		f.hash = hashBytes(f.Bytes())
	}

	return f.hash
}

// hash of the file plus every sql file it @"..." includes
func (f *FileInfo) RecurredSQLHash() string {
	walker := make(map[string]string)

	recurredSQLHash(walker, f)

	names := make([]string, 0, len(walker))
	for name := range walker {
		names = append(names, name)
	}
	sort.Strings(names)

	hash := sha256.New()
	for _, name := range names {
		fmt.Fprintf(hash, "%v:%v", name, walker[name])
	}

	return fmt.Sprintf("%X", hash.Sum(nil))
}

func (f *FileInfo) Mime() string {
	if f.mime == "" {
		f.mime = mime.TypeByExtension(f.Extension)
	}

	return f.mime
}

func recurredSQLHash(walker map[string]string, file *FileInfo) {
	if _, ok := walker[file.Fullname]; ok {
		return
	}

	walker[file.Fullname] = file.Hash()

	for _, matches := range sqlFileRegex.FindAllSubmatch(file.Bytes(), -1) {
		recurredSQLHash(walker, &FileInfo{
			Fullname: filepath.Clean(string(matches[1])),
		})
	}
}

// list one layer of sql files
func ListSQLFiles(path string) ([]*FileInfo, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}

	result := make([]*FileInfo, 0, len(entries))

	for _, e := range entries {
		if e.IsDir() {
			continue
		}

		ext := filepath.Ext(e.Name())

		if !ContainsString(includedExts, ext) || ContainsString(excludedFiles, e.Name()) {
			continue
		}

		result = append(result, &FileInfo{
			Filename:  e.Name(),
			Fullname:  filepath.Join(path, e.Name()),
			Extension: ext,
		})
	}

	return result, nil
}

// list all files including all children
func ListAllFiles(path string) ([]*FileInfo, error) {
	result := make([]*FileInfo, 0)

	err := filepath.Walk(path, func(fp string, f os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !f.IsDir() {
			result = append(result, &FileInfo{
				Filename:  f.Name(),
				Fullname:  fp,
				Extension: filepath.Ext(f.Name()),
			})
		}

		return nil
	})

	return result, err
}

func FolderHash(folder string) (string, error) {
	hash := sha256.New()

	err := filepath.Walk(folder, func(fp string, f os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !f.IsDir() {
			fmt.Fprintf(hash, "%v", (&FileInfo{Fullname: fp}).Hash())
		}

		return nil
	})

	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%X", hash.Sum(nil)), nil
}

func hashBytes(data []byte) string {
	hash := sha256.New()
	hash.Write(data)

	return fmt.Sprintf("%X", hash.Sum(nil))
}

// collect the @includeFile(...) targets declared in an install file
func ScanIncludedFiles(content []byte) bool {
	var found bool

	for _, matches := range includedFileRegex.FindAllSubmatch(content, -1) {
		found = true
		includedFiles = append(includedFiles, strings.ToLower(strings.TrimSpace(string(matches[1]))))
	}

	return found
}

// IncludedFiles returns the @includeFile targets collected from the install files
func IncludedFiles() []string {
	return includedFiles
}

func ContainsString(arr []string, e string) bool {
	for _, a := range arr {
		if a == e {
			return true
		}
	}

	return false
}
