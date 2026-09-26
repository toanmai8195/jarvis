package main

import (
	"context"
	"io"
	"log/slog"
	"os"
)

// Giá trị của key shutdown_step trong log — cùng tên với core server (P0-T09)
// để một truy vấn log dùng chung cho cả hai binary.
const (
	stepSignal = "signal"
	stepDone   = "done"
)

// newLogger trả logger slog JSON (time, level, msg như core server) ghi ra w.
// Worker chưa có config nên mức log cố định Info.
func newLogger(w io.Writer) *slog.Logger {
	return slog.New(slog.NewJSONHandler(w, &slog.HandlerOptions{Level: slog.LevelInfo}))
}

// run log dòng khởi động rồi chờ tới khi nhận tín hiệu từ sigs hoặc ctx bị
// huỷ, sau đó log mốc signal và done. Chưa có job nên không có gì phải dừng:
// luôn trả exit code 0. Khi thêm job, run sẽ huỷ ctx của job và chờ chúng
// kết thúc giữa hai mốc này (có hạn, như core server).
func run(ctx context.Context, log *slog.Logger, sigs <-chan os.Signal) int {
	log.Info("core worker đã khởi động", "jobs", 0)

	var reason string
	select {
	case s := <-sigs:
		reason = s.String()
	case <-ctx.Done():
		reason = context.Cause(ctx).Error()
	}
	log.Info("bắt đầu dừng worker", "shutdown_step", stepSignal, "signal", reason)

	const code = 0
	log.Info("core worker đã dừng", "shutdown_step", stepDone, "exit_code", code)
	return code
}
