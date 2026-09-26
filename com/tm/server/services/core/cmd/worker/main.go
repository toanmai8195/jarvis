// Command worker là tiến trình nền của core (hết hạn hold, relay outbox, sinh
// slot, đối soát). Hiện là skeleton: chưa có job nào, chỉ log khởi động, chờ
// SIGTERM/SIGINT rồi dừng sạch theo quy ước shutdown_step của core server.
// Job thật (và config, pool PG, OTel) thêm ở P4-T08 / P6-T01.
//
// Không mở cổng, không nối DB/Redis, không bắt buộc biến môi trường nào.
package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	log := newLogger(os.Stdout)
	// Đăng ký tín hiệu trước khi log khởi động: SIGTERM đến sớm vẫn được xử lý
	// graceful thay vì giết process.
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGTERM, os.Interrupt)
	code := run(context.Background(), log, sigs)
	signal.Stop(sigs)
	os.Exit(code)
}
