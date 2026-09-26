// Command smoke là binary mẫu tối thiểu để kiểm chứng Bazel workspace:
// Gazelle map_kind go_binary → com_tm_go_image, build binary + OCI image
// distroless, và build được bằng cả `go` lẫn Bazel.
package main

import (
	"fmt"
	"io"
	"os"
	"strings"
)

// message trả về dòng trạng thái cố định dạng "snaptix <component> ok".
// component rỗng (sau khi bỏ khoảng trắng) được coi là "smoke".
func message(component string) string {
	component = strings.TrimSpace(component)
	if component == "" {
		component = "smoke"
	}
	return "snaptix " + component + " ok"
}

// run in dòng trạng thái ra w; tách khỏi main để test được.
func run(w io.Writer) error {
	_, err := fmt.Fprintln(w, message(""))
	return err
}

func main() {
	if err := run(os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "smoke:", err)
		os.Exit(1)
	}
}
