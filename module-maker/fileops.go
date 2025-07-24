package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// DefaultFileOperator 默认文件操作实现
type DefaultFileOperator struct{}

// NewFileOperator 创建新的文件操作器
func NewFileOperator() FileOperator {
	return &DefaultFileOperator{}
}

// CreateDirectory 创建目录
func (f *DefaultFileOperator) CreateDirectory(path string) error {
	if err := os.MkdirAll(path, 0755); err != nil {
		return fmt.Errorf("创建目录失败 %s: %w", path, err)
	}
	return nil
}

// WriteFile 写入文件
func (f *DefaultFileOperator) WriteFile(path, content string) error {
	// 确保目录存在
	dir := filepath.Dir(path)
	if err := f.CreateDirectory(dir); err != nil {
		return err
	}

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return fmt.Errorf("写入文件失败 %s: %w", path, err)
	}
	return nil
}

// FileExists 检查文件是否存在
func (f *DefaultFileOperator) FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// ReadFile 读取文件内容
func (f *DefaultFileOperator) ReadFile(path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("读取文件失败 %s: %w", path, err)
	}
	return string(content), nil
}

// ScanDirectory 扫描目录中指定扩展名的文件
func (f *DefaultFileOperator) ScanDirectory(dir, extension string) ([]string, error) {
	if !f.FileExists(dir) {
		return nil, fmt.Errorf("目录不存在: %s", dir)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("读取目录失败 %s: %w", dir, err)
	}

	var matchedFiles []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		filename := entry.Name()
		// 匹配以指定后缀结尾的文件
		if extension == "" || strings.HasSuffix(filename, extension) {
			fullPath := filepath.Join(dir, filename)

			// 验证文件可读性
			if info, err := entry.Info(); err == nil {
				if info.Mode().IsRegular() && info.Size() > 0 {
					matchedFiles = append(matchedFiles, fullPath)
				}
			}
		}
	}

	return matchedFiles, nil
}

// GetDirectoryInfo 获取目录信息
func (f *DefaultFileOperator) GetDirectoryInfo(dir string) (int, int64, error) {
	if !f.FileExists(dir) {
		return 0, 0, fmt.Errorf("目录不存在: %s", dir)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0, 0, fmt.Errorf("读取目录失败 %s: %w", dir, err)
	}

	var fileCount int
	var totalSize int64

	for _, entry := range entries {
		if !entry.IsDir() {
			fileCount++
			if info, err := entry.Info(); err == nil {
				totalSize += info.Size()
			}
		}
	}

	return fileCount, totalSize, nil
}
