package main

import (
	"flag"
	"log"

	"monitor/internal/config"
	"monitor/internal/server"
)

func main() {
	configPath := flag.String("config", "", "配置文件路径（默认使用内置默认值）")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("[main] %v", err)
	}

	srv, err := server.New(cfg)
	if err != nil {
		log.Fatalf("[main] 初始化失败: %v", err)
	}

	if err := srv.Run(); err != nil {
		log.Fatalf("[main] 服务异常退出: %v", err)
	}
}
