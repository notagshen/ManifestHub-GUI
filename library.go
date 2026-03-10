package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

type LibraryRecord struct {
	AppID   string `json:"appid"`
	Name    string `json:"name"`
	AddedAt string `json:"added_at"`
	LuaPath string `json:"lua_path"`
}

var libraryMu sync.Mutex

func libraryFilePath() string {
	return filepath.Join(_MainConfig_, "Library", "added.json")
}

func readLibraryRecordMap() (map[string]LibraryRecord, error) {
	path := libraryFilePath()
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return map[string]LibraryRecord{}, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read library records failed: %v", err)
	}

	if len(data) == 0 {
		return map[string]LibraryRecord{}, nil
	}

	var list []LibraryRecord
	if err := json.Unmarshal(data, &list); err != nil {
		return nil, fmt.Errorf("parse library records failed: %v", err)
	}

	records := make(map[string]LibraryRecord, len(list))
	for _, item := range list {
		if item.AppID != "" {
			records[item.AppID] = item
		}
	}

	return records, nil
}

func writeLibraryRecordMap(records map[string]LibraryRecord) error {
	path := libraryFilePath()
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create library dir failed: %v", err)
	}

	list := make([]LibraryRecord, 0, len(records))
	for _, item := range records {
		list = append(list, item)
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].AddedAt > list[j].AddedAt
	})

	data, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal library records failed: %v", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write library records failed: %v", err)
	}

	return nil
}

func upsertLibraryRecord(appid, name, luaPath string) error {
	libraryMu.Lock()
	defer libraryMu.Unlock()

	records, err := readLibraryRecordMap()
	if err != nil {
		return err
	}

	record := LibraryRecord{
		AppID:   appid,
		Name:    name,
		AddedAt: time.Now().Format(time.RFC3339),
		LuaPath: luaPath,
	}
	records[appid] = record

	return writeLibraryRecordMap(records)
}

func deleteLibraryRecord(appid string) error {
	libraryMu.Lock()
	defer libraryMu.Unlock()

	records, err := readLibraryRecordMap()
	if err != nil {
		return err
	}

	delete(records, appid)

	return writeLibraryRecordMap(records)
}

func resolveOutputPath(path string) (string, error) {
	if filepath.IsAbs(path) {
		return path, nil
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("get cwd failed: %v", err)
	}
	return filepath.Join(cwd, path), nil
}

func buildLuaPath(appid string) (string, error) {
	if CONFIG_READ_STEAM_PATH {
		steamPath, err := GetSteamGamePath()
		if err == nil {
			return filepath.Join(steamPath, appid+".lua"), nil
		}
		log.Printf("获取 Steam 游戏路径失败: %v, 将使用配置的下载路径", err)
	}
	return filepath.Join(CONFIG_DOWNLOAD_PATH, appid+".lua"), nil
}

