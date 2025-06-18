package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"os"
	"regexp"
	"strings"
)

func main() {
	// 定义命令行参数
	inputFile := flag.String("i", "./cc/const_op.go", "输入文件路径")
	outputFile := flag.String("o", "op.csv", "输出CSV文件路径")
	flag.Parse()

	// 读取源文件
	content, err := os.ReadFile(*inputFile)
	if err != nil {
		fmt.Printf("无法读取输入文件: %v\n", err)
		os.Exit(1)
	}

	// 创建CSV文件
	csvFile, err := os.Create(*outputFile)
	if err != nil {
		fmt.Printf("无法创建输出文件: %v\n", err)
		os.Exit(1)
	}
	defer csvFile.Close()

	writer := csv.NewWriter(csvFile)
	defer writer.Flush()

	// 写入CSV头
	writer.Write([]string{"op", "name"})

	// 正则匹配常量定义行
	re := regexp.MustCompile(`OP_(\w+)\s*=\s*(\d+)\s*//\s*(.+)`)
	lines := strings.Split(string(content), "\n")

	for _, line := range lines {
		matches := re.FindStringSubmatch(line)
		if len(matches) == 4 {
			opCode := matches[2]
			opName := strings.TrimSpace(matches[3])
			writer.Write([]string{opCode, opName})
		}
	}

	fmt.Printf("CSV文件已生成: %s\n", *outputFile)
}