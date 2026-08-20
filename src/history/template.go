package history

import "text/template"

// values available to the sql templates
type templateData struct {
	Repo  string
	Flow  string
	File  string
	Hash  string
	Files string
}

var fetchHistoryTemplate = template.Must(template.New("FetchHistory").Parse(`
SET SQLFORMAT CSV;
SET FEEDBACK OFF;

SELECT
  filename,
  hash
FROM
  apps.xxcicd_histories
WHERE
    repo = '{{.Repo}}'
  AND flow = '{{.Flow}}'{{if .Files}} AND filename in ({{.Files}}){{end}};`))

var mergeHistoryTemplate = template.Must(template.New("MergeHistory").Parse(`
MERGE INTO apps.xxcicd_histories
USING dual ON ( repo = '{{.Repo}}'
                AND flow = '{{.Flow}}'
                AND filename = '{{.File}}' )
WHEN MATCHED THEN UPDATE
SET hash = '{{.Hash}}',
    update_date = sysdate
WHEN NOT MATCHED THEN
INSERT (
  repo,
  flow,
  filename,
  hash,
  update_date )
VALUES
  ( '{{.Repo}}',
  '{{.Flow}}',
  '{{.File}}',
  '{{.Hash}}',
  sysdate );`))
