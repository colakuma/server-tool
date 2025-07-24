package main

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
)

// DefaultTemplateProcessor 默认模板处理器实现
type DefaultTemplateProcessor struct {
	fileOp      FileOperator
	templateDir string
}

// NewTemplateProcessor 创建新的模板处理器
func NewTemplateProcessor(baseDir string, fileOp FileOperator) TemplateProcessor {
	templateDir := filepath.Join(baseDir, "_init")
	return &DefaultTemplateProcessor{
		fileOp:      fileOp,
		templateDir: templateDir,
	}
}

// LoadTemplates 加载所有模板文件
func (t *DefaultTemplateProcessor) LoadTemplates() ([]TemplateFile, error) {
	// 检查模板目录是否存在
	if !t.fileOp.FileExists(t.templateDir) {
		return nil, fmt.Errorf("模板目录不存在: %s", t.templateDir)
	}

	// 扫描所有 .init 文件
	templateFiles, err := t.fileOp.ScanDirectory(t.templateDir, ".init")
	if err != nil {
		return nil, fmt.Errorf("扫描模板目录失败: %w", err)
	}

	if len(templateFiles) == 0 {
		return nil, fmt.Errorf("在目录 %s 中未找到 .init 模板文件", t.templateDir)
	}

	// 按文件名排序，确保生成顺序一致
	sort.Strings(templateFiles)

	var templates []TemplateFile
	for _, sourcePath := range templateFiles {
		// 验证文件是否可读
		if !t.fileOp.FileExists(sourcePath) {
			return nil, fmt.Errorf("模板文件不存在: %s", sourcePath)
		}

		content, err := t.fileOp.ReadFile(sourcePath)
		if err != nil {
			return nil, fmt.Errorf("读取模板文件失败 %s: %w", sourcePath, err)
		}

		// 验证模板文件不为空
		if strings.TrimSpace(content) == "" {
			return nil, fmt.Errorf("模板文件为空: %s", sourcePath)
		}

		// 验证模板文件包含必要的变量占位符
		if err := t.validateTemplateContent(content, sourcePath); err != nil {
			return nil, err
		}

		templates = append(templates, TemplateFile{
			SourcePath: sourcePath,
			Content:    content,
		})
	}

	return templates, nil
}

// validateTemplateContent 验证模板文件内容
func (t *DefaultTemplateProcessor) validateTemplateContent(content, sourcePath string) error {
	// 检查是否包含模板变量
	hasTemplateVars := strings.Contains(content, "%pkg%") ||
		strings.Contains(content, "%pkg_l%") ||
		strings.Contains(content, "%pkg_u%")

	if !hasTemplateVars {
		return fmt.Errorf("模板文件 %s 不包含任何模板变量 (%%pkg%%, %%pkg_l%%, %%pkg_u%%)", sourcePath)
	}

	// 检查是否有未知的模板变量
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		// 查找所有 %...% 格式的变量
		start := 0
		for {
			startIdx := strings.Index(line[start:], "%")
			if startIdx == -1 {
				break
			}
			startIdx += start

			endIdx := strings.Index(line[startIdx+1:], "%")
			if endIdx == -1 {
				break
			}
			endIdx += startIdx + 1

			variable := line[startIdx : endIdx+1]
			if variable != "%pkg%" && variable != "%pkg_l%" && variable != "%pkg_u%" {
				return fmt.Errorf("模板文件 %s 第 %d 行包含未知变量: %s", sourcePath, i+1, variable)
			}

			start = endIdx + 1
		}
	}

	return nil
}

// ProcessTemplate 处理单个模板文件
func (t *DefaultTemplateProcessor) ProcessTemplate(template TemplateFile, config *ModuleConfig) (TemplateFile, error) {
	// 处理文件名中的变量
	targetFilename := filepath.Base(template.SourcePath)
	targetFilename = strings.ReplaceAll(targetFilename, "%pkg%", config.Name)
	targetFilename = strings.TrimSuffix(targetFilename, ".init")

	// 生成目标路径
	targetPath := filepath.Join(config.TargetDir, targetFilename)

	// 处理文件内容中的变量
	processedContent := t.replaceVariables(template.Content, config)

	return TemplateFile{
		SourcePath: template.SourcePath,
		TargetPath: targetPath,
		Content:    processedContent,
	}, nil
}

// GenerateFiles 生成所有文件
func (t *DefaultTemplateProcessor) GenerateFiles(templates []TemplateFile, config *ModuleConfig) error {
	// 检查目标目录是否已存在
	if t.fileOp.FileExists(config.TargetDir) {
		return fmt.Errorf("目标目录已存在: %s", config.TargetDir)
	}

	// 创建目标目录
	if err := t.fileOp.CreateDirectory(config.TargetDir); err != nil {
		return err
	}

	// 生成所有文件
	for _, template := range templates {
		if config.Verbose {
			fmt.Printf("生成文件: %s\n", template.TargetPath)
		}

		if err := t.fileOp.WriteFile(template.TargetPath, template.Content); err != nil {
			return err
		}
	}

	return nil
}

// replaceVariables 替换模板变量
func (t *DefaultTemplateProcessor) replaceVariables(content string, config *ModuleConfig) string {
	content = strings.ReplaceAll(content, "%pkg%", config.Name)
	content = strings.ReplaceAll(content, "%pkg_l%", config.NameLower)
	content = strings.ReplaceAll(content, "%pkg_u%", config.NameUpper)
	return content
}
