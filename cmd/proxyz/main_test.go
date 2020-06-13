package main

import (
	"os"
	"os/exec"
	"strconv"
	"testing"
)

func TestMain(t *testing.T) {
	for i, tt := range []struct {
		CmdLines [][]string
		GoRunPkg string
	}{
		{
			CmdLines: [][]string{
				{"proxyz", "github.com/roy2220/proxyz/testdata/foo", "Test1A", "github.com/roy2220/proxyz/testdata/test1", "Test1AProxy", "-w", "test1_generated.go"},
			},
			GoRunPkg: "github.com/roy2220/proxyz/testdata/test1",
		},
		{
			CmdLines: [][]string{
				{"proxyz", "github.com/roy2220/proxyz/testdata/test2", "File", "github.com/roy2220/proxyz/testdata/test2", "FileWrap", "-w", "test2_generated1.go"},
				{"proxyz", "github.com/roy2220/proxyz/testdata/test2", "Server", "github.com/roy2220/proxyz/testdata/test2", "ServerWrap", "-w", "test2_generated2.go"},
				{"proxyz", "testing", "TB", "github.com/roy2220/proxyz/testdata/test2", "TPWrap", "-w", "test2_generated3.go"},
			},
			GoRunPkg: "github.com/roy2220/proxyz/testdata/test2",
		},
		{
			CmdLines: [][]string{
				{"proxyz", "github.com/roy2220/proxyz/testdata/test3", "Calc", "github.com/roy2220/proxyz/testdata/test3", "CalcProxy", "-w", "test3_generated.go"},
			},
			GoRunPkg: "github.com/roy2220/proxyz/testdata/test3",
		},
	} {
		t.Run("test"+strconv.Itoa(i), func(t *testing.T) {
			for _, cmdLine := range tt.CmdLines {
				os.Args = cmdLine
				main()
			}
			cmd := exec.Command("go", "run", tt.GoRunPkg)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stdout
			if err := cmd.Run(); err != nil {
				t.Fatal(err)
			}
		})
	}
}
