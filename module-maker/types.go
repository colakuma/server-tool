package main

import (
	"fmt"
	"path/filepath"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// ModuleConfig 模块配置信息
type ModuleConfig struct {
	Name      string // 原始模块名
	NameLower string // 小写模块名
	NameUpper string // 大写模块名
	BaseDir   string // 模板目录
	TargetDir string // 目标目录
	Verbose   bool   // 是否显示详细信息
}

// NewModuleConfig 创建新的模块配置
func NewModuleConfig(baseDir string, name string) (*ModuleConfig, error) {
	if err := validateModuleName(name); err != nil {
		return nil, err
	}

	return &ModuleConfig{
		Name:      name,
		NameLower: strings.ToLower(name),
		NameUpper: cases.Title(language.English).String(name),
		BaseDir:   baseDir,
		TargetDir: filepath.Join(baseDir, name),
		Verbose:   true,
	}, nil
}

// TemplateFile 模板文件信息
type TemplateFile struct {
	SourcePath string // 源模板文件路径
	TargetPath string // 目标文件路径
	Content    string // 文件内容
}

// TemplateProcessor 模板处理器接口
type TemplateProcessor interface {
	// LoadTemplates 加载所有模板文件
	LoadTemplates() ([]TemplateFile, error)

	// ProcessTemplate 处理单个模板文件
	ProcessTemplate(template TemplateFile, config *ModuleConfig) (TemplateFile, error)

	// GenerateFiles 生成所有文件
	GenerateFiles(templates []TemplateFile, config *ModuleConfig) error
}

// FileOperator 文件操作接口
type FileOperator interface {
	// CreateDirectory 创建目录
	CreateDirectory(path string) error

	// WriteFile 写入文件
	WriteFile(path, content string) error

	// FileExists 检查文件是否存在
	FileExists(path string) bool

	// ReadFile 读取文件内容
	ReadFile(path string) (string, error)

	// ScanDirectory 扫描目录中的文件
	ScanDirectory(dir string, pattern string) ([]string, error)
}

// validateModuleName 验证模块名称
func validateModuleName(name string) error {
	if name == "" {
		return fmt.Errorf("模块名称不能为空")
	}

	if strings.ContainsAny(name, " \t\n\r") {
		return fmt.Errorf("模块名称不能包含空白字符")
	}

	if strings.ContainsAny(name, "\\/:*?\"<>|") {
		return fmt.Errorf("模块名称包含非法字符")
	}

	return nil
}
