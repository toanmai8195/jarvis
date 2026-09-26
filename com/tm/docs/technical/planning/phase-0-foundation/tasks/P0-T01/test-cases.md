# Test cases — P0-T01: Khởi tạo cấu trúc `com/tm/{server,app,docs}`, `deploy/`, `loadtest/` theo project-structure

> Phạm vi: dựng **khung thư mục** monorepo theo [project-structure](../../../../project-structure.md) (mục "Tổng quan monorepo" và tầng thư mục thứ nhất của `com/tm/server`, `com/tm/app`) cùng script kiểm tra cấu trúc trong `scripts/`.
> **Không thuộc task này**: `MODULE.bazel`/`go.mod`/`.bazelrc`/`tools/rules` (P0-T01a), `package.json`/`pnpm-workspace.yaml` (P0-T01b), `deploy/docker-compose.yml` (P0-T02), `db/*/migrations` (P0-T03), `.github/workflows` (P0-T04), `Makefile` (P0-T05), code service (P0-T07+).
> Mọi lệnh chạy từ gốc repo. Thư mục rỗng được giữ trong git bằng `.gitkeep` (hoặc `README.md` mô tả mục đích thư mục).

| ID | Loại | Kịch bản | Kết quả mong đợi | Trạng thái |
|---|---|---|---|---|
| P0-T01-TC01 | Structure | Kiểm tra thư mục cấp gốc: `for d in deploy loadtest scripts com/tm/server com/tm/app com/tm/docs; do test -d "$d" \|\| echo "MISSING $d"; done` | Không in dòng `MISSING` nào | ✅ |
| P0-T01-TC02 | Structure | Kiểm tra khung `com/tm/server`: `for d in api pkg services; do test -d "com/tm/server/$d" \|\| echo "MISSING $d"; done` | Không in dòng `MISSING` nào | ✅ |
| P0-T01-TC03 | Structure | Kiểm tra khung `com/tm/app`: `for d in api apps packages; do test -d "com/tm/app/$d" \|\| echo "MISSING $d"; done` | Không in dòng `MISSING` nào | ✅ |
| P0-T01-TC04 | Structure | Kiểm tra `loadtest/results/` (nơi lưu kết quả k6 chính thức theo phase): `test -d loadtest/results` | Exit code 0 | ✅ |
| P0-T01-TC05 | Git | Mọi thư mục mới được git track (kể cả thư mục rỗng): với mỗi thư mục ở TC01–TC04, `git ls-files <dir> \| head -1` | Mỗi thư mục trả về ít nhất 1 file (`.gitkeep` hoặc `README.md`); không thư mục nào rỗng trong git | ✅ |
| P0-T01-TC06 | Negative | Không có thư mục kiểu Java / tên chung chung: `find com/tm/server com/tm/app deploy loadtest scripts -type d \( -name service -o -name repository -o -name controller -o -name model -o -name utils -o -name util -o -name common -o -name helpers -o -name shared \)` | Không in kết quả nào (lưu ý: `com/tm/server/services` hợp lệ, không bị bắt vì khác tên `service`) | ✅ |
| P0-T01-TC07 | Negative | Không commit file rác / build output: `git ls-files \| grep -E '(^\|/)(\.DS_Store\|node_modules\|bazel-[^/]+\|dist\|\.idea)(/\|$)'` | Không in kết quả nào | ✅ |
| P0-T01-TC08 | Negative | Không lấn phạm vi task sau: `ls com/tm/server/MODULE.bazel com/tm/server/go.mod com/tm/app/package.json com/tm/app/pnpm-workspace.yaml deploy/docker-compose.yml Makefile .github/workflows 2>/dev/null` và `test ! -d com/tm/server/db` | Lệnh `ls` không in gì; `test ! -d` exit 0 | ✅ |
| P0-T01-TC09 | Git | `.gitignore` đúng với cấu trúc mới: `git check-ignore -q loadtest/run.tmp.json` (phải bị ignore); `git check-ignore -q loadtest/results/phase-7.json`, `git check-ignore -q deploy/.env.example`, `git check-ignore -q com/tm/server/pkg/.gitkeep`, `git check-ignore -q com/tm/app/packages/.gitkeep` (không được ignore); `git check-ignore -q deploy/.env` (phải bị ignore) | Lệnh 1 và lệnh cuối exit 0; các lệnh còn lại exit 1 | ✅ |
| P0-T01-TC10 | Script | Chạy script kiểm tra cấu trúc trên repo hiện tại: `bash scripts/check-structure.sh; echo $?` | In `0`; script kiểm tra tối thiểu các thư mục ở TC01–TC04 và luật cấm ở TC06 | ✅ |
| P0-T01-TC11 | Script (lỗi) | Thiếu thư mục bắt buộc: chạy script trên bản sao repo (`git worktree`/thư mục tạm, không sửa repo thật) đã xoá `loadtest/` | Exit code ≠ 0, stderr nêu tên thư mục thiếu `loadtest` | ✅ |
| P0-T01-TC12 | Script (lỗi) | Có thư mục cấm: trên bản sao tạm, tạo `com/tm/server/pkg/utils/.gitkeep` rồi chạy script | Exit code ≠ 0, stderr nêu đường dẫn `com/tm/server/pkg/utils` | ✅ |
| P0-T01-TC13 | Unit | Chạy test của script: `bash scripts/check-structure_test.sh` (bao phủ ca hợp lệ, thiếu thư mục, thư mục cấm) | Exit code 0, mọi ca PASS | ✅ |
| P0-T01-TC14 | Docs | Cây thư mục trong [project-structure](../../../../project-structure.md) khớp thực tế: mọi thư mục cấp gốc liệt kê ở mục "Tổng quan monorepo" (trừ `.github/workflows/` thuộc P0-T04) tồn tại; không có thư mục cấp gốc mới (ngoài `.claude`, `.git`) chưa được ghi trong docs: `ls -d */ .github 2>/dev/null` | Danh sách thư mục cấp gốc ⊆ {`com`, `deploy`, `loadtest`, `scripts`} (và `.github` nếu đã có); docs được cập nhật nếu có khác biệt | ✅ |

Test nghiệm thu liên quan: không có trực tiếp (P0-T01 là tiền đề cho P0-AT01, P0-AT11, P0-AT12, P0-AT13 nhưng không tự làm pass test nghiệm thu nào).

## Review

### Review lần 1 — APPROVED

Reviewer: review agent độc lập (execute-all). Đối chiếu với `CLAUDE.md`, `phase-0-foundation/README.md`, `acceptance-tests.md`, `project-structure.md`, ADR-0001/0002.

- Phạm vi: phủ đủ dòng task (cấp gốc `deploy/`, `loadtest/`, `scripts/`, `com/tm/{server,app,docs}` + tầng 1 của server/app). P0-T01 không gắn challenge. Ranh giới với P0-T01a/T01b/T02/T03/T04/T05 được ghi rõ và kiểm bằng TC08.
- `scripts/check-structure.sh` + `_test.sh` (TC10–TC13): hợp lệ — `project-structure.md` mục "Tổng quan monorepo" ghi rõ `scripts/  # script kiểm tra repo (cấu trúc, .gitignore) + test của chúng`; không trùng CI (P0-T04) hay Makefile (P0-T05), task sau chỉ việc gọi lại script.
- TC09: đã kiểm với `.gitignore` hiện tại — `loadtest/run.tmp.json` → 0 (rule `loadtest/*.tmp.json`), `loadtest/results/phase-7.json` → 1, `deploy/.env.example` → 1 (`!.env.example`), `deploy/.env` → 0, `com/tm/server/pkg/.gitkeep` → 1. Khớp kỳ vọng.
- TC06 khớp danh sách cấm trong "Go cho người từ Java" và "Không đưa vào tầng 3" (`common`/`utils`/`helpers`/`shared`).
- Định dạng đúng: tiêu đề, 5 cột, ID `P0-T01-TC01..TC14` liên tục, trạng thái ⬜, có dòng "Test nghiệm thu liên quan". Không liên quan tiền.

Ghi chú khi thực thi (không chặn duyệt):
- TC05, TC07 dùng `git ls-files` chỉ thấy file trong index: ở bước 5 (trước commit) cần `git add` các file của task trước khi chạy, nếu không TC05 fail giả và TC07 pass giả.
- TC08 (và phần `.github` của TC14) là kiểm tra tại thời điểm task P0-T01; khi đóng phase, các file này sẽ tồn tại hợp lệ do task sau tạo — hiểu TC08 là "tại commit của P0-T01", không chạy lại ở DoD.
- TC11/TC12: script nên nhận thư mục gốc qua tham số hoặc chạy theo cwd để chạy được trên bản sao tạm.
