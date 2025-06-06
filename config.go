package main

import (
	"log"
	"os"

	"gopkg.in/ini.v1"
)

type Config struct {
	ApiKey  string
	BaseUrl string
}

var cfg = new(Config)

func init() {
	err := ini.MapTo(cfg, "settings.ini")
	if os.IsNotExist(err) {
		log.Println(err)
		log.Println("使用空配置初始化")
	}
}

func saveConfig() {
	iniFile := ini.Empty()
	ini.ReflectFrom(iniFile, cfg)
	iniFile.SaveTo("settings.ini")
}
