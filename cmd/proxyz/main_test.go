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
				{"proxyz", "../../testdata/foo", "Test1A", "../../testdata/test1", "Test1AProxy", "-w", "test1_generated.go"},
			},
			GoRunPkg: "../../testdata/test1",
		},
		{
			CmdLines: [][]string{
				{"proxyz", "../../testdata/test2", "File", "../../testdata/test2", "FileWrap", "-w", "test2_1_generated.go"},
				{"proxyz", "github.com/roy2220/proxyz/testdata/test2", "Server", "../../testdata/test2", "ServerWrap", "-w", "test2_2_generated.go"},
				{"proxyz", "../../testdata/test2/", "Err2", "github.com/roy2220/proxyz/testdata/test2", "Err2Wrap", "-w", "test2_3_generated.go"},
				{"proxyz", "github.com/roy2220/proxyz/testdata/test2", "Err3", "github.com/roy2220/proxyz/testdata/test2/", "Err3Wrap", "-w", "test2_4_generated.go"},
				{"proxyz", "../../testdata/test2/", "Err4", "../../testdata/test2/", "Err4Wrap", "-w", "test2_5_generated.go"},
				{"proxyz", "testing", "TB", "../../testdata/test2", "TPWrap", "-w", "test2_6_generated.go"},
			},
			GoRunPkg: "../../testdata/test2",
		},
		{
			CmdLines: [][]string{
				{"proxyz", "../../testdata/test3", "Calc", "../../testdata/test3", "CalcProxy", "-w", "test3_generated.go"},
			},
			GoRunPkg: "../../testdata/test3",
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
