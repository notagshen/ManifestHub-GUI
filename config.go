/* config.go */
package main

import (
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

// 配置文件结构
type Config struct {
	ReadSteamPath bool   `json:"read_steam_path"` // 下载后存入 SteamTools 读取路径文件夹
	DownloadPath  string `json:"download_path"`   // 下载路径
	AddDLC        bool   `json:"add_dlc"`         // 添加无 DepotKey DLC
	SetManifestid bool   `json:"set_manifestid"`  // 设置固定清单
	GithubToken   string `json:"github_token"`    // GitHub 令牌
	LibraryChoice string `json:"library_choice"`  // 库选择
	SteamRegion   string `json:"steam_region"`    // Steam 商店区域
}

var DefaultConfig = Config{
	ReadSteamPath: true,
	DownloadPath:  "./Download",
	AddDLC:        true,
	SetManifestid: false,
	GithubToken:   "",
	LibraryChoice: "Sudama",
	SteamRegion:   "CN",
}

// 创建配置文件
func CreateConfig() {
	// 设置配置文件绝对路径
	configDir := filepath.Join(_MainConfig_, "Config")
	configPath := filepath.Join(configDir, "config.json")

	// 创建目录
	if err := os.MkdirAll(configDir, 0755); err != nil {
		log.Printf("创建配置目录失败: %v", err)
	}

	// 配置 Viper
	viper.SetConfigFile(configPath)

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		log.Println("配置文件不存在, 将生成配置文件")

		viper.SetDefault("read_steam_path", DefaultConfig.ReadSteamPath)
		viper.SetDefault("download_path", DefaultConfig.DownloadPath)
		viper.SetDefault("add_dlc", DefaultConfig.AddDLC)
		viper.SetDefault("set_manifestid", DefaultConfig.SetManifestid)
		viper.SetDefault("github_token", DefaultConfig.GithubToken)
		viper.SetDefault("library_choice", DefaultConfig.LibraryChoice)
		viper.SetDefault("steam_region", DefaultConfig.SteamRegion)

		// 写入配置文件（生成 JSON）
		if err := viper.WriteConfig(); err != nil {
			log.Printf("生成默认配置失败: %v", err)
		}
		log.Printf("默认配置已生成: %s", configPath)
	} else {
		if err := viper.ReadInConfig(); err != nil {
			log.Printf("读取配置文件失败: %v", err)
		}
		log.Printf("配置文件已加载: %s\n", configPath)
	}
}

// 修改配置文件
func ModifyConfig(item string, value interface{}) error {
	if item == "steam_region" {
		region, ok := value.(string)
		if !ok {
			return LogAndError("steam_region 配置值类型错误: %T", value)
		}
		value = normalizeSteamRegion(region)
	}

	// Viper 设置项值
	viper.Set(item, value)

	// 保存
	if err := viper.WriteConfig(); err != nil {
		LogAndError("保存配置失败: %v", err)
	}

	// 输出日志
	log.Printf("%s 项已修改为 %v (类型为%t)", item, value, value)

	return nil
}

// 检查配置文件完整性
func CheckConfigIntegrity() bool {
	requiredKeys := []string{
		"read_steam_path",
		"download_path",
		"add_dlc",
		"set_manifestid",
		"github_token",
		"library_choice",
		"steam_region",
	}

	for _, key := range requiredKeys {
		if !viper.InConfig(key) {
			log.Printf("配置项缺失: %s", key)
			return false
		}
	}
	return true
}

func RepairConfig() {
	defaults := map[string]interface{}{
		"read_steam_path": DefaultConfig.ReadSteamPath,
		"download_path":   DefaultConfig.DownloadPath,
		"add_dlc":         DefaultConfig.AddDLC,
		"set_manifestid":  DefaultConfig.SetManifestid,
		"github_token":    DefaultConfig.GithubToken,
		"library_choice":  DefaultConfig.LibraryChoice,
		"steam_region":    DefaultConfig.SteamRegion,
	}

	missingKeys := make([]string, 0)
	for key, value := range defaults {
		if viper.InConfig(key) {
			continue
		}
		viper.Set(key, value)
		missingKeys = append(missingKeys, key)
	}

	if len(missingKeys) == 0 {
		return
	}

	if err := viper.WriteConfig(); err != nil {
		if err := viper.SafeWriteConfigAs(viper.ConfigFileUsed()); err != nil {
			log.Printf("补全配置文件失败: %v", err)
			return
		}
	}

	initGlobalConfig()
	log.Printf("已补全缺失配置项: %s", strings.Join(missingKeys, ", "))
}

// 重置配置文件
func ResetConfig() {
	viper.SetDefault("read_steam_path", DefaultConfig.ReadSteamPath)
	viper.SetDefault("download_path", DefaultConfig.DownloadPath)
	viper.SetDefault("add_dlc", DefaultConfig.AddDLC)
	viper.SetDefault("set_manifestid", DefaultConfig.SetManifestid)
	viper.SetDefault("github_token", DefaultConfig.GithubToken)
	viper.SetDefault("library_choice", DefaultConfig.LibraryChoice)
	viper.SetDefault("steam_region", DefaultConfig.SteamRegion)

	// 写入配置文件
	if err := viper.WriteConfig(); err != nil {
		// 如果文件不存在，使用SafeWriteConfigAs
		if err := viper.SafeWriteConfigAs(viper.ConfigFileUsed()); err != nil {
			log.Printf("保存配置文件失败: %v", err)
		}
	}
	initGlobalConfig()
	log.Printf("默认配置已生成/重置: %s", viper.ConfigFileUsed())
}
