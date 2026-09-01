package server

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"monitor/internal/alert"
	"monitor/internal/api"
	"monitor/internal/collector"
	"monitor/internal/config"
	"monitor/internal/manager"
	"monitor/internal/notifier"
	"monitor/internal/store"
	"monitor/internal/ws"
)

// Server 应用根对象：组装 store、collector、alert、notifier、api、ws 并协调生命周期。
type Server struct {
	cfg    *config.Config
	st     *store.Store
	col    *collector.Collector
	hub    *ws.Hub
	engine *alert.Engine
	mgr    *notifier.Manager
}

// New 初始化依赖并确保默认设置写入。
func New(cfg *config.Config) (*Server, error) {
	st, err := store.Open(cfg.Server.DBPath)
	if err != nil {
		return nil, err
	}

	// 用配置默认值补齐缺失的 settings（不覆盖已有值）
	defs := store.DefaultSettings()
	defs[store.SettingCollectIntervalSec] = cfg.Collector.IntervalSec
	defs[store.SettingProcessIntervalSec] = cfg.Collector.ProcessIntervalSec
	defs[store.SettingServiceIntervalSec] = cfg.Collector.ServiceIntervalSec
	defs[store.SettingProcessTopN] = cfg.Collector.ProcessTopN
	if err := st.EnsureDefaultSettings(defs); err != nil {
		return nil, err
	}

	hub := ws.NewHub()
	engine := alert.New(st)
	engine.SetHub(hub)
	mgr := notifier.NewManager(st)
	engine.SetNotifier(mgr)
	col := collector.New(st, cfg)
	col.SetHub(hub)
	col.SetAlertEngine(engine)
	col.SetSampleUpdater(engine)
	return &Server{cfg: cfg, st: st, col: col, hub: hub, engine: engine, mgr: mgr}, nil
}

// Run 启动 HTTP 服务并等待优雅退出信号。
func (s *Server) Run() error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	router := api.New(s.st, s.cfg, s.hub, s.engine, s.mgr, manager.New())

	// 启动采集器与规则自动刷新（后台运行，随进程退出）
	collectCtx, stopCollect := context.WithCancel(context.Background())
	s.col.Run(collectCtx)
	go s.engine.AutoReload(collectCtx.Done())
	go s.runCleanup(collectCtx)
	defer stopCollect()

	srv := &http.Server{
		Addr:              s.cfg.Server.Addr,
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		log.Printf("[server] HTTP 服务启动，监听 %s", s.cfg.Server.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		log.Println("[server] 收到退出信号，正在优雅关闭…")
		stopCollect()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			log.Printf("[server] 关闭 HTTP 服务异常: %v", err)
		}
		if err := s.st.Close(); err != nil {
			log.Printf("[server] 关闭数据库异常: %v", err)
		}
		log.Println("[server] 已退出")
		return nil
	}
}

// runCleanup 启动时补跑一次清理，之后每 6 小时执行。
func (s *Server) runCleanup(ctx context.Context) {
	s.st.Cleanup()
	t := time.NewTicker(6 * time.Hour)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			s.st.Cleanup()
		}
	}
}
