package main

import (
	"flag"
	"fmt"
)

func main() {
	// 定义命令行参数
	baseDir := flag.String("base", "internal/app/game/player/module", "基础目录")
	moduleName := flag.String("module", "mytest", "模块名")
	flag.Parse()

	// 创建模块配置
	config, err := NewModuleConfig(*baseDir, *moduleName)
	if err != nil {
		panic(fmt.Sprintf("配置错误: %v", err))
	}

	fmt.Printf("开始生成模块: %s\n", *moduleName)
	fmt.Printf("目标目录: %s\n", config.TargetDir)

	// 创建文件操作器和模板处理器
	fileOp := NewFileOperator()
	processor := NewTemplateProcessor(*baseDir, fileOp)

	// 加载模板文件
	templates, err := processor.LoadTemplates()
	if err != nil {
		panic(fmt.Sprintf("加载模板失败: %v", err))
	}

	fmt.Printf("找到 %d 个模板文件\n", len(templates))

	// 处理模板文件
	var processedTemplates []TemplateFile
	for _, template := range templates {
		processed, err := processor.ProcessTemplate(template, config)
		if err != nil {
			panic(fmt.Sprintf("处理模板失败: %v", err))
		}
		processedTemplates = append(processedTemplates, processed)
	}

	// 生成文件
	if err := processor.GenerateFiles(processedTemplates, config); err != nil {
		panic(fmt.Sprintf("生成文件失败: %v", err))
	}

	fmt.Printf("✅ 模块 '%s' 生成成功！\n", *moduleName)

	fmt.Printf("生成了 %d 个文件到目录: %s\n", len(processedTemplates), config.TargetDir)
}
