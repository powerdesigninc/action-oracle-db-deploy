package history

import (
	"bytes"
	"encoding/csv"
	"io"
	"strings"

	"github.com/powerdesigninc/action-oracle-db-deploy/src/utils"
)

const installFlow = "install"
const fileFlow = "file"

type History struct {
	Flow  string
	items map[string]*historyItem
}

type historyItem struct {
	File    string
	Hash    string
	Updated bool
}

func LoadInstallHistory(files []*utils.FileInfo) (*History, error) {
	if len(files) == 0 {
		return newHistory(installFlow), nil
	}

	names := make([]string, 0, len(files))
	for _, f := range files {
		names = append(names, "'"+f.Filename+"'")
	}

	return loadHistory(installFlow, strings.Join(names, ","))
}

func LoadFileHistory() (*History, error) {
	return loadHistory(fileFlow, "")
}

func (h *History) CheckHash(file string, hash string) bool {
	item, ok := h.items[file]

	return ok && item.Hash == hash
}

func (h *History) SetHash(file string, hash string) {
	if item, ok := h.items[file]; ok {
		item.Hash = hash
		item.Updated = true
	} else {
		h.items[file] = &historyItem{File: file, Hash: hash, Updated: true}
	}
}

func (h *History) Save() error {
	var buffer = &bytes.Buffer{}

	for _, item := range h.items {
		if !item.Updated {
			continue
		}

		err := mergeHistoryTemplate.Execute(buffer, templateData{
			Repo: utils.Cfg.Repo,
			Flow: h.Flow,
			File: item.File,
			Hash: item.Hash,
		})

		if err != nil {
			return err
		}
	}

	if buffer.Len() == 0 {
		return nil
	}

	buffer.WriteString("\nCOMMIT;")

	if err := utils.ExecuteSQLQuiet(buffer.Bytes()); err != nil {
		utils.PrintWarning("Save History Failed, Detail: ", err.Error())
		return utils.ErrNoPrint
	}

	return nil
}

func newHistory(flow string) *History {
	return &History{Flow: flow, items: map[string]*historyItem{}}
}

func loadHistory(flow string, files string) (*History, error) {
	h := newHistory(flow)

	var buffer = &bytes.Buffer{}
	if err := fetchHistoryTemplate.Execute(buffer, templateData{Repo: utils.Cfg.Repo, Flow: flow, Files: files}); err != nil {
		return nil, err
	}

	output, err := utils.ExecuteSQLOutput(buffer.Bytes())
	if err != nil {
		utils.PrintWarning("Load History Failed, Detail: ", output, "\n", err.Error())
		return nil, utils.ErrNoPrint
	}

	reader := csv.NewReader(strings.NewReader(output))
	// sqlcl can emit non-row noise, rows are filtered by length below
	reader.FieldsPerRecord = -1

	for {
		line, err := reader.Read()
		if err == io.EOF {
			break
		} else if err != nil {
			return nil, err
		}

		if len(line) < 2 || line[0] == "FILENAME" {
			// noise or the csv header
			continue
		}

		h.items[line[0]] = &historyItem{File: line[0], Hash: line[1]}
	}

	return h, nil
}
