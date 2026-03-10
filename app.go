/* app.go */
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/wailsapp/wails/v2/pkg/runtime"
	"github.com/winterssy/sreq"
)

// App struct
type App struct {
	ctx context.Context
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

// 获取热门 Steam 游戏列表
func (a *App) GetSteamFeatured() (string, error) {
	body, err := Client.Get("https://proxy.052222.xyz/https://store.steampowered.com/api/featured/?l=schinese&cc=CN").
		Text()
	if err != nil {
		return "", LogAndError("获取 Steam 热门游戏列表失败: %v", err)
	}
	return body, nil
}

// 入库
func (a *App) AddGameToLibrary(APPID string, name string) (string, error) {
	log.Printf("开始为游戏 %s 入库", APPID)

	var (
		depotkeys  map[string]string
		manifests  map[string]string
		err1, err2 error
		wg         sync.WaitGroup
	)

	runtime.EventsEmit(a.ctx, "progress", 10)

	wg.Add(2)
	// 并行获取depotkeys
	go func() {
		defer wg.Done()
		depotkeys, err1 = GetDepotkeys(false)
	}()
	// 并行获取manifests
	go func() {
		defer wg.Done()
		manifests, err2 = GetManifests(APPID)
	}()
	wg.Wait()

	if err1 != nil {
		return "", LogAndError("获取 DepotKeys 失败: %v", err1)
	}

	if err2 != nil {
		return "", LogAndError("获取 Manifests 失败: %v", err2)
	}

	runtime.EventsEmit(a.ctx, "progress", 20)

	// 收集当前游戏所需的所有密钥ID（主游戏APPID + 所有DepotID）
	requiredIDs := []string{APPID}
	for depotID := range manifests {
		requiredIDs = append(requiredIDs, depotID)
	}

	// 检查是否有缺失的密钥
	checkMissing := func(keys map[string]string) bool {
		for _, id := range requiredIDs {
			if _, exists := keys[id]; !exists {
				return true
			}
		}
		return false
	}

	runtime.EventsEmit(a.ctx, "progress", 40)

	// 若存在缺失，尝试重新下载 Depotkeys
	var isMissing = false
	if checkMissing(depotkeys) {
		log.Println("发现缺失的 DepotKey, 尝试重新下载...")
		newDepotkeys, err := GetDepotkeys(true) // 重新下载
		if err != nil {
			log.Printf("重新下载 DepotKey 失败: %v, 将使用现有密钥继续", err)
		} else {
			depotkeys = newDepotkeys
			// 再次检查是否仍有缺失
			if checkMissing(depotkeys) {
				log.Printf("警告: 重新下载后仍缺失部分 DepotKey, 将继续生成Lua文件")
				isMissing = true
			} else {
				log.Println("重新下载后, 缺失的 DepotKey 已获取")
			}
		}
	}

	runtime.EventsEmit(a.ctx, "progress", 50)

	// 生成 Lua 文件
	path, err := buildLuaPath(APPID)
	if err != nil {
		return "", LogAndError("构建 Lua 路径失败: %v", err)
	}

	runtime.EventsEmit(a.ctx, "progress", 70)

	err = GenerateLua(APPID, path, depotkeys, manifests)
	if err != nil {
		return "", LogAndError("生成 Lua 文件失败: %v", err)
	}

	outputPath, err := resolveOutputPath(path)
	if err != nil {
		log.Printf("解析 Lua 路径失败: %v", err)
	} else {
		if err := upsertLibraryRecord(APPID, name, outputPath); err != nil {
			log.Printf("写入入库记录失败: %v", err)
		}
	}

	runtime.EventsEmit(a.ctx, "progress", 100)

	if isMissing {
		return fmt.Sprintf("游戏 %s 已成功添加到库中, 但是缺少部分 DepotKey (可能导致空包)", APPID), nil
	}
	return fmt.Sprintf("游戏 %s 已成功添加到库中", APPID), nil
}

// 获取入库记录
func (a *App) GetAddedLibraries() (string, error) {
	libraryMu.Lock()
	records, err := readLibraryRecordMap()
	libraryMu.Unlock()
	if err != nil {
		return "", LogAndError("获取入库记录失败: %v", err)
	}

	list := make([]LibraryRecord, 0, len(records))
	for _, item := range records {
		list = append(list, item)
	}
	data, err := json.Marshal(list)
	if err != nil {
		return "", LogAndError("序列化入库记录失败: %v", err)
	}
	return string(data), nil
}

// 移除入库
func (a *App) RemoveGameFromLibrary(APPID string) (string, error) {
	var record LibraryRecord
	var exists bool

	libraryMu.Lock()
	records, err := readLibraryRecordMap()
	if err != nil {
		libraryMu.Unlock()
		return "", LogAndError("读取入库记录失败: %v", err)
	}
	record, exists = records[APPID]
	libraryMu.Unlock()

	var luaPath string
	if exists && record.LuaPath != "" {
		luaPath = record.LuaPath
	} else {
		path, err := buildLuaPath(APPID)
		if err != nil {
			return "", LogAndError("构建 Lua 路径失败: %v", err)
		}
		luaPath, err = resolveOutputPath(path)
		if err != nil {
			return "", LogAndError("解析 Lua 路径失败: %v", err)
		}
	}

	var fileRemoved bool
	if _, err := os.Stat(luaPath); err == nil {
		if err := os.Remove(luaPath); err != nil {
			return "", LogAndError("删除 Lua 文件失败: %v", err)
		}
		fileRemoved = true
	}

	if err := deleteLibraryRecord(APPID); err != nil {
		return "", LogAndError("更新入库记录失败: %v", err)
	}

	if fileRemoved {
		return fmt.Sprintf("已移除入库: %s", APPID), nil
	}
	return fmt.Sprintf("入库记录已移除，但未找到 Lua 文件: %s", APPID), nil
}

// 游戏搜索
func (a *App) SearchSteamGames(searchTerm string) (string, error) {
	// 编码
	params := sreq.Params{
		"term": searchTerm,
		"l":    "schinese",
		"cc":   "CN",
	}
	headers := sreq.Headers{
		"User-Agent": []string{"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36"},
	}

	body, err := Client.Get("https://proxy.052222.xyz/store.steampowered.com/api/storesearch/",
		sreq.WithQuery(params),
		sreq.WithHeaders(headers),
	).Text()

	if err != nil {
		return "", LogAndError("搜索Steam游戏失败: %v", err)
	}
	return body, nil
}

// 配置文件
func (a *App) GetConfig() (Config, error) {
	// 返回当前所有配置
	return Config{
		ReadSteamPath: CONFIG_READ_STEAM_PATH,
		DownloadPath:  CONFIG_DOWNLOAD_PATH,
		AddDLC:        CONFIG_ADD_DLC,
		SetManifestid: CONFIG_SET_MANIFESTID,
		GithubToken:   CONFIG_GITHUB_TOKEN,
		LibraryChoice: CONFIG_LIBRARY_CHOICE,
	}, nil
}

// 修改配置文件
func (a *App) ModifyConfig(item string, value interface{}) error {
	// 调用配置修改函数
	err := ModifyConfig(item, value)
	if err != nil {
		return err
	}
	// 同步更新全局配置变量
	initGlobalConfig()
	return nil
}
