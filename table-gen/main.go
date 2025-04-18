package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

var basePkg = flag.String("base", "mmo_server/pkg", "base import path")

func main() {
	flag.Parse()

	// 步骤3: 生成table.go和table_after_load.go
	if err := step3(); err != nil {
		fmt.Printf("Step3 failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Table generation completed successfully")
}

func step3() error {
	os.Rename("./table.go", "./table2.go")
	os.Rename("./table_after_load.go", "./table_after_load2.go")
	files, _ := enumFile(".", "c_")
	for _, v := range files {
		walkFile(v)
	}

	makeTableGo()
	// os.Remove("./var.go")
	// os.Remove("./v_custom.go")
	os.Remove("./table2.go")
	os.Remove("./table_after_load2.go")
	goFmt("table_after_load.go")
	goFmt("table.go")
	return nil
}

func goFmt(file string) error {
	cmd := exec.Command("gofmt", "-l", "-w", "-s", file)
	cmd.Dir = "./"
	out, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("gofmt failed: %v, output: %s", err, out)
	}
	fmt.Println(string(out))
	return os.Chmod(filepath.Join("./", file), 0o444)
}
