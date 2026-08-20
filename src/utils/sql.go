package utils

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
)

var bom = []byte{0xef, 0xbb, 0xbf}

// sqlcl reads the script from stdin, -S drops the banner and
// -L makes a bad credential fail instead of re-prompting
func sqlclCmd(content []byte) *exec.Cmd {
	var buffer = &bytes.Buffer{}

	buffer.WriteString("WHENEVER OSERROR EXIT FAILURE;\n")
	buffer.Write(bytes.TrimPrefix(content, bom))
	buffer.WriteString("\nEXIT;\n")

	cmd := exec.Command("sql", "-S", "-L", fmt.Sprintf("%v/%v@%v", Cfg.User, Cfg.Password, Cfg.Host))
	cmd.Stdin = buffer

	return cmd
}

// execute sql and only print the output on error
func ExecuteSQLQuiet(content []byte) error {
	out, err := sqlclCmd(content).CombinedOutput()

	if err != nil {
		os.Stdout.Write(out)
	}

	return err
}

// execute sql and return the output
func ExecuteSQLOutput(content []byte) (string, error) {
	cmd := sqlclCmd(content)
	cmd.Stderr = os.Stderr

	out, err := cmd.Output()

	return string(out), err
}

func TestConnection() error {
	out, err := sqlclCmd([]byte("WHENEVER SQLERROR EXIT FAILURE;\nSET FEEDBACK OFF;\nSELECT 1 FROM dual;")).CombinedOutput()
	if err != nil {
		os.Stdout.Write(out)
		return fmt.Errorf("connection error for %s", Cfg.Host)
	}

	return nil
}
