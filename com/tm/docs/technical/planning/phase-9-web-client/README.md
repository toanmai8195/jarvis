# Phase 9 — Web client (React)

> Chặng C — React (web) · Công nghệ: **React + TypeScript**

## Mục tiêu

Học React qua web-client: routing, server state, sơ đồ ghế realtime, checkout nhiều bước, accessibility.

**Mốc demo**: Người dùng đăng nhập, tìm chuyến, chọn ghế realtime, nạp tiền, đặt vé, huỷ vé trên web

## Kiến thức trọng tâm

React: component, state, TanStack Query, memo, state machine, code splitting, a11y, Testing Library

## Phạm vi

- **Trong**: Skeleton web, config TS dùng chung, Vitest, toàn bộ màn hình người dùng.
- **Ngoài**: Web admin.

## Workstream

`web` · `qa`

## Requirement

| ID | Loại | Mô tả |
|---|---|---|
| P9-FR1 | FR | Web: tìm chuyến, xem kết quả, xem chi tiết + sơ đồ ghế |
| P9-FR2 | FR | Web: trang cần đăng nhập tự chuyển sang đăng nhập rồi quay lại |
| P9-NFR1 | NFR | JS tải ban đầu web-client < 200KB gzip |
| P9-FR3 | FR | Vé của tôi: danh sách, chi tiết, mã QR |
| P9-NFR2 | NFR | Web client đạt Lighthouse accessibility ≥ 95 |

## Task

### web
- [ ] **P9-T01** Skeleton `web-client` bằng Vite + React + TS + Tailwind + shadcn/ui
- [ ] **P9-T02** `com/tm/app/packages/config`: eslint, prettier, tsconfig dùng chung _(trước đây P0-T15)_
- [ ] **P9-T03** Khung Vitest cho TS _(trước đây P0-T17)_
- [ ] **P9-T04** React Router, layout, TanStack Query, API client dùng chung (`packages/api-client`) _(trước đây P2-T12)_
- [ ] **P9-T05** Trang chủ + form tìm chuyến (autocomplete trạm) _(trước đây P2-T13)_
- [ ] **P9-T06** Trang kết quả: lọc, sắp xếp; trang chi tiết + sơ đồ ghế (chỉ xem) _(trước đây P2-T14)_
- [ ] **P9-T07** Auth context, protected route, redirect về trang cũ sau đăng nhập `[R10]` _(trước đây P2-T15)_
- [ ] **P9-T08** Code splitting theo route, phân tích bundle `[R11]` _(trước đây P2-T16)_
- [ ] **P9-T09** Trang ví: số dư, nạp tiền, lịch sử (infinite scroll); sinh `Idempotency-Key` một lần mỗi lần bấm _(trước đây P3-T13)_
- [ ] **P9-T10** Component sơ đồ ghế: mỗi ghế memo hoá, store theo ghế, cập nhật từ SSE `[R1]` _(trước đây P4-T13)_
- [ ] **P9-T11** Optimistic chọn ghế, rollback khi `SEAT_UNAVAILABLE` `[R2]` _(trước đây P4-T14)_
- [ ] **P9-T12** Checkout bằng state machine: chọn ghế → hành khách → thanh toán → kết quả; đồng hồ đếm ngược; khôi phục khi refresh `[R3]` _(trước đây P4-T15)_
- [ ] **P9-T13** Invalidate query ví, vé sau thanh toán `[R9]` _(trước đây P4-T16)_
- [ ] **P9-T14** Trang vé của tôi + mã QR _(trước đây P4-T17)_
- [ ] **P9-T15** UI huỷ vé kèm refund quote
- [ ] **P9-T16** Accessibility: sơ đồ ghế điều hướng bằng bàn phím, ARIA, focus management trong checkout và dialog `[R7]` _(trước đây P8-T10)_

### qa
- [ ] **P9-T17** Test component tìm chuyến bằng Testing Library _(trước đây P2-T18)_
- [ ] **P9-T18** Test Testing Library cho component quan trọng (sơ đồ ghế, checkout, ví) `[R8]` _(trước đây P8-T12)_

## Challenge

| # | Công nghệ | Challenge | Bối cảnh | Hướng giải | Hoàn thành khi | Trạng thái |
|---|---|---|---|---|---|---|
| R10 | React | Route guard & phân quyền UI | Trang cần đăng nhập, admin theo vai trò | Protected route, loader kiểm tra session, ẩn chức năng theo vai trò (server vẫn là nơi quyết định) | Truy cập trái phép bị chuyển hướng, không nháy nội dung | ⬜ |
| R11 | React | Bundle nhỏ, tải nhanh trên mobile | Người dùng đặt vé bằng điện thoại, mạng yếu | Code splitting theo route (`lazy`), phân tích bundle, prefetch dữ liệu khi hover | JS ban đầu < 200KB gzip, LCP < 2.5s trên 4G chậm | ⬜ |
| R1 | React | Sơ đồ ghế realtime hiệu năng cao | Toa tàu hàng trăm ghế, cập nhật liên tục | Memo hoá từng ghế, cập nhật state theo ghế, virtualize khi lớn | Không render lại toàn bộ sơ đồ khi 1 ghế đổi trạng thái | ⬜ |
| R2 | React | Optimistic UI & rollback | Chọn ghế, huỷ vé | TanStack Query mutation, rollback khi `SEAT_UNAVAILABLE` | UI luôn khớp server sau lỗi | ⬜ |
| R3 | React | Luồng checkout nhiều bước | Chọn ghế → hành khách → thanh toán | State machine (useReducer / XState), đồng hồ đếm ngược hold | Refresh/quay lại không mất trạng thái, không thanh toán 2 lần | ⬜ |
| R9 | React | Dữ liệu không cũ sau thanh toán | Số dư, vé của tôi | Chiến lược `staleTime`, invalidate query theo key sau mutation | Không bao giờ hiển thị số dư/vé cũ sau thanh toán | ⬜ |
| R7 | React | Accessibility | Chọn ghế bằng bàn phím, screen reader | ARIA, focus management | Lighthouse accessibility ≥ 95 | ⬜ |
| R8 | React | Test UI | Luồng đặt vé | Testing Library, Playwright E2E | E2E đặt vé → huỷ vé chạy trong CI | ⬜ |

## Definition of Done

- [ ] Luồng tìm chuyến → chọn ghế → thanh toán → vé → huỷ vé chạy trên web
- [ ] JS ban đầu < 200KB gzip
- [ ] Lighthouse accessibility ≥ 95
- [ ] Mọi test nghiệm thu trong [acceptance-tests.md](acceptance-tests.md) và test case của các task trong [tasks/](tasks/) pass

## Checklist đóng phase

- [ ] Cập nhật [client guide](../../../user-guide/client.md) theo UI thực tế
- [ ] Viết [lessons-learned.md](lessons-learned.md)
- [ ] Cập nhật trạng thái phase trong [planning](../README.md)
