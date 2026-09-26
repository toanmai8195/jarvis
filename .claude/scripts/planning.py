#!/usr/bin/env python3
"""Đọc trạng thái planning của snaptix từ markdown + git.

Dùng bởi các skill task-detail, phase-detail, sumup.

    planning.py task [TASK_ID]     # task trước / hiện tại / tiếp theo
    planning.py phase [N]          # chi tiết một phase (mặc định: phase hiện tại)
    planning.py sumup              # tổng kết toàn dự án
    planning.py validate [TASK_ID] # kiểm tra bước 0, exit 1 nếu chưa đạt
    planning.py step [TASK_ID]     # task cần làm + bước tiếp theo (cho skill execute-task)
    planning.py guard              # PreToolUse hook: đọc JSON từ stdin, chặn sửa code sai quy trình
"""
import re
import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
PLANNING = ROOT / "com/tm/docs/technical/planning"
HANDBOOK = ROOT / "com/tm/docs/technical/handbook"

TASK_RE = re.compile(r"^- \[( |x)\] \*\*(P\d+-T\d+[a-z]?)\*\* (.*)$")
STEP_RE = re.compile(r"^\s+- \[( |x)\] (\d)\. (.*)$")
CHK_RE = re.compile(r"^- \[( |x)\] (.*)$")
CH_TAG_RE = re.compile(r"\[([GPANMRDS]\d+)(?: [^\]]*)?\]")
TC_ID_RE = re.compile(r"P\d+-T\d+[a-z]?-TC\d+")
DONE = {"✅"}


def sections(text):
    """Tách markdown theo heading '## '."""
    out, cur = {}, None
    for line in text.splitlines():
        if line.startswith("## "):
            cur = line[3:].strip()
            out[cur] = []
        elif cur:
            out[cur].append(line)
    return out


def table_rows(lines, id_pattern):
    rows = []
    for line in lines:
        if re.match(rf"^\| {id_pattern} \|", line):
            rows.append([c.strip() for c in line.strip().strip("|").split(" | ")])
    return rows


def git(*args):
    try:
        return subprocess.run(["git", "-C", str(ROOT), *args], capture_output=True, text=True, timeout=10).stdout.strip()
    except Exception:
        return ""


def load_phase_status():
    text = (PLANNING / "README.md").read_text()
    status = {}
    for m in re.finditer(r"^\| \[(\d+)\]\([^)]*\) \| (.+?) \| (.+?) \| (\S+) \|$", text, re.M):
        status[int(m.group(1))] = {"name": m.group(2), "milestone": m.group(3), "status": m.group(4)}
    return status


def load_phase(d: Path, status):
    n = int(re.match(r"phase-(\d+)-", d.name).group(1))
    text = (d / "README.md").read_text()
    sec = sections(text)
    title = text.splitlines()[0].lstrip("# ").strip()

    goal = "\n".join(l for l in sec.get("Mục tiêu", []) if l.strip()).strip()

    tasks, ws, cur = [], None, None
    for line in sec.get("Task", []):
        if line.startswith("### "):
            ws = line[4:].strip()
        elif m := TASK_RE.match(line):
            cur = {"id": m.group(2), "done": m.group(1) == "x", "desc": m.group(3), "ws": ws,
                   "phase": n, "steps": [], "challenges": CH_TAG_RE.findall(m.group(3))}
            tasks.append(cur)
        elif (m := STEP_RE.match(line)) and cur:
            cur["steps"].append({"n": int(m.group(2)), "done": m.group(1) == "x", "text": m.group(3)})

    challenges = {}
    for r in table_rows(sec.get("Challenge", []), r"[GPANMRDS]\d+"):
        challenges[r[0]] = {"id": r[0], "tech": r[1], "name": r[2], "done_when": r[5], "status": r[-1]}

    def checklist(name):
        return [{"done": m.group(1) == "x", "text": m.group(2)}
                for l in sec.get(name, []) if (m := CHK_RE.match(l))]

    reqs = [{"id": r[0], "type": r[1], "desc": r[2]} for r in table_rows(sec.get("Requirement", []), r"P\d+-N?FR\d+")]

    ats, task_tcs = [], []
    at_file = d / "acceptance-tests.md"
    if at_file.exists():
        for r in table_rows(at_file.read_text().splitlines(), r"P\d+-AT\d+"):
            ats.append({"id": r[0], "type": r[1], "scenario": r[2], "expect": r[3], "refs": r[4], "status": r[-1]})
    for tc_file in sorted(d.glob("tasks/*/test-cases.md")):
        for r in table_rows(tc_file.read_text().splitlines(), r"P\d+-T\d+[a-z]?-TC\d+"):
            task_tcs.append({"id": r[0], "task": r[0].rsplit("-TC", 1)[0], "type": r[1], "scenario": r[2],
                             "expect": r[3], "status": r[-1]})

    ll = (d / "lessons-learned.md").read_text() if (d / "lessons-learned.md").exists() else ""
    ll_written = bool(re.search(r"^\| Bắt đầu \| *[^| ]", ll, re.M))

    return {"n": n, "dir": d.name, "title": title, "goal": goal, "status": status.get(n, {}).get("status", "?"),
            "milestone": status.get(n, {}).get("milestone", ""), "tasks": tasks, "challenges": challenges,
            "reqs": reqs, "dod": checklist("Definition of Done"), "closing": checklist("Checklist đóng phase"),
            "ats": ats, "task_tcs": task_tcs, "lessons_written": ll_written}


def load_all():
    status = load_phase_status()
    phases = [load_phase(d, status) for d in sorted(PLANNING.glob("phase-*"), key=lambda p: int(re.match(r"phase-(\d+)", p.name).group(1)))]
    return phases, [t for p in phases for t in p["tasks"]]


def committed(task):
    step6 = next((s for s in task["steps"] if s["n"] == 6), None)
    return bool(step6 and step6["done"])


def commits_for(task_id):
    return git("log", "--oneline", "-n", "5", "--fixed-strings", f"--grep=[{task_id}]")


def current_index(tasks):
    for i, t in enumerate(tasks):
        if not (t["done"] and committed(t)):
            return i
    return len(tasks)


def pct(a, b):
    return f"{a}/{b} ({round(100 * a / b) if b else 0}%)"


def ch_line(cid, phases):
    for p in phases:
        if cid in p["challenges"]:
            c = p["challenges"][cid]
            return f"{cid} {c['status']} {c['name']} — hoàn thành khi: {c['done_when']}"
    return cid


def print_task(t, label, phases, detail=True):
    print(f"\n### {label}: {t['id']} — Phase {t['phase']} · workstream `{t['ws']}`")
    print(f"- Mô tả: {t['desc']}")
    state = "đã xong" if t["done"] else "chưa xong"
    state += ", đã commit" if committed(t) else ", chưa commit"
    print(f"- Trạng thái: {state}")
    if t["challenges"]:
        print("- Challenge:")
        for c in t["challenges"]:
            print(f"  - {ch_line(c, phases)}")
    if t["steps"]:
        print("- Checklist con:")
        for s in t["steps"]:
            print(f"  - [{'x' if s['done'] else ' '}] {s['n']}. {s['text']}")
    elif detail:
        print("- Checklist con: chưa tạo (task chưa bắt đầu)")
    if detail:
        phase = next(p for p in phases if p["n"] == t["phase"])
        step1 = next((s for s in t["steps"] if s["n"] == 1), None)
        own = [tc for tc in phase["task_tcs"] if tc["task"] == t["id"]]
        approved = bool(step1 and step1["done"])
        related = [at for at in phase["ats"] if any(c in re.split(r"[ ,]+", at["refs"]) for c in t["challenges"])]
        if own:
            print(f"- Test case của task ({'đã duyệt' if approved else 'CHƯA duyệt'}):")
            for tc in own:
                print(f"  - {tc['id']} {tc['status']} [{tc['type']}] {tc['scenario']} → {tc['expect']}")
        else:
            print("- Test case của task: chưa viết (bước 1)")
        if related:
            print("- Test nghiệm thu phase liên quan (theo challenge):")
            for at in related:
                print(f"  - {at['id']} {at['status']} [{at['type']}] {at['scenario']}")
    c = commits_for(t["id"])
    if c:
        print("- Commit:")
        for line in c.splitlines():
            print(f"  - {line}")


def cmd_task(arg):
    phases, tasks = load_all()
    if not tasks:
        print("Không tìm thấy task nào.")
        return
    if arg:
        idx = next((i for i, t in enumerate(tasks) if t["id"].lower() == arg.lower()), None)
        if idx is None:
            print(f"Không tìm thấy task {arg}.")
            return
    else:
        idx = current_index(tasks)
    if idx >= len(tasks):
        print("Tất cả task đã xong và đã commit.")
        print_task(tasks[-1], "Task cuối cùng", phases, detail=False)
        return

    cur = tasks[idx]
    print("## Task" + ("" if arg else " (tự xác định: task đầu tiên chưa xong hoặc chưa commit)"))
    if idx > 0:
        print_task(tasks[idx - 1], "Task trước", phases, detail=False)
    else:
        print("\n### Task trước: không có (đây là task đầu tiên)")
    print_task(cur, "Task hiện tại", phases, detail=True)
    if idx + 1 < len(tasks):
        print_task(tasks[idx + 1], "Task tiếp theo", phases, detail=False)
    else:
        print("\n### Task tiếp theo: không có (task cuối cùng)")

    print("\n### Validate bước 0 cho task hiện tại")
    for ok, text in validate(phases, tasks, idx):
        print(f"- {'✅' if ok else '❌'} {text}")


def validate(phases, tasks, idx):
    """Điều kiện bước 0 trong CLAUDE.md cho task ở vị trí idx."""
    cur = tasks[idx]
    checks = []
    if idx > 0:
        prev = tasks[idx - 1]
        checks.append((prev["done"] and all(s["done"] for s in prev["steps"]) and len(prev["steps"]) == 6,
                       f"Task trước {prev['id']} đủ checklist 6 bước"))
        checks.append((committed(prev) and bool(commits_for(prev["id"])), f"Task trước {prev['id']} đã commit (có trong git log)"))
        if prev["phase"] != cur["phase"]:
            pp = next(p for p in phases if p["n"] == prev["phase"])
            checks.append((pp["status"] in DONE, f"Phase {pp['n']} đã đóng (✅)"))
    dirty = git("status", "--porcelain")
    checks.append((not dirty, "Working tree sạch" + (f" — đang có {len(dirty.splitlines())} file thay đổi" if dirty else "")))
    return checks


def cmd_validate(arg):
    """Exit 0 nếu được phép làm task (mặc định: task hiện tại), 1 nếu không. Dùng cho hook."""
    phases, tasks = load_all()
    idx = next((i for i, t in enumerate(tasks) if t["id"].lower() == arg.lower()), None) if arg else current_index(tasks)
    if idx is None or idx >= len(tasks):
        print("Không có task để validate.")
        sys.exit(0)
    fails = [text for ok, text in validate(phases, tasks, idx) if not ok]
    if fails:
        print(f"Bước 0 chưa đạt cho {tasks[idx]['id']}:\n" + "\n".join(f"- {f}" for f in fails))
        sys.exit(1)
    print(f"Bước 0 đạt cho {tasks[idx]['id']}.")


def cmd_phase(arg):
    phases, tasks = load_all()
    if arg and arg.strip().isdigit():
        p = next((p for p in phases if p["n"] == int(arg)), None)
        if not p:
            print(f"Không có phase {arg}.")
            return
    else:
        idx = current_index(tasks)
        n = tasks[idx]["phase"] if idx < len(tasks) else phases[-1]["n"]
        p = next(p for p in phases if p["n"] == n)

    i = phases.index(p)
    print(f"## {p['title']} — trạng thái {p['status']}")
    print(f"\n### Mục tiêu\n{p['goal']}")

    done = [t for t in p["tasks"] if t["done"]]
    print(f"\n### Tiến độ\n- Task: {pct(len(done), len(p['tasks']))}, đã commit {sum(committed(t) for t in p['tasks'])}")
    print(f"- Test case theo task: {pct(sum(tc['status'] in DONE for tc in p['task_tcs']), len(p['task_tcs']))}")
    print(f"- Test nghiệm thu: {pct(sum(at['status'] in DONE for at in p['ats']), len(p['ats']))}")
    print(f"- Challenge: {pct(sum(c['status'] in DONE for c in p['challenges'].values()), len(p['challenges']))}")
    print(f"- DoD: {pct(sum(d['done'] for d in p['dod']), len(p['dod']))} · Checklist đóng phase: {pct(sum(d['done'] for d in p['closing']), len(p['closing']))}")
    print(f"- Lessons learned: {'đã viết' if p['lessons_written'] else 'chưa viết'}")

    print("\n### Requirement")
    for r in p["reqs"]:
        print(f"- {r['id']} ({r['type']}): {r['desc']}")

    print("\n### Task theo workstream")
    ws_order = []
    for t in p["tasks"]:
        if t["ws"] not in ws_order:
            ws_order.append(t["ws"])
    for ws in ws_order:
        ts = [t for t in p["tasks"] if t["ws"] == ws]
        print(f"- `{ws}` {pct(sum(t['done'] for t in ts), len(ts))}")
        for t in ts:
            mark = "x" if t["done"] else ("~" if t["steps"] else " ")
            print(f"  - [{mark}] {t['id']} {t['desc']}")

    print("\n### Challenge")
    for c in p["challenges"].values():
        print(f"- {c['id']} {c['status']} [{c['tech']}] {c['name']} — hoàn thành khi: {c['done_when']}")

    print("\n### Definition of Done")
    for d in p["dod"]:
        print(f"- [{'x' if d['done'] else ' '}] {d['text']}")

    print("\n### Phase trước / sau")
    if i > 0:
        q = phases[i - 1]
        print(f"- Trước: {q['title']} — {q['status']}")
    if i + 1 < len(phases):
        q = phases[i + 1]
        print(f"- Sau: {q['title']} — {q['status']} — mốc: {q['milestone']}")


def cmd_sumup():
    phases, tasks = load_all()
    idx = current_index(tasks)

    print("## Tổng quan")
    print("\n| Phase | Trạng thái | Task | Test nghiệm thu | Challenge | DoD |")
    print("|---|---|---|---|---|---|")
    for p in phases:
        print(f"| {p['title']} | {p['status']} | {pct(sum(t['done'] for t in p['tasks']), len(p['tasks']))} "
              f"| {pct(sum(at['status'] in DONE for at in p['ats']), len(p['ats']))} "
              f"| {pct(sum(c['status'] in DONE for c in p['challenges'].values()), len(p['challenges']))} "
              f"| {pct(sum(d['done'] for d in p['dod']), len(p['dod']))} |")
    all_tc = [at for p in phases for at in p["ats"]]
    all_ttc = [tc for p in phases for tc in p["task_tcs"]]
    all_ch = [c for p in phases for c in p["challenges"].values()]
    print(f"\n- Toàn dự án: task {pct(sum(t['done'] for t in tasks), len(tasks))} · test nghiệm thu {pct(sum(tc['status'] in DONE for tc in all_tc), len(all_tc))} · test case theo task {pct(sum(tc['status'] in DONE for tc in all_ttc), len(all_ttc))} · challenge {pct(sum(c['status'] in DONE for c in all_ch), len(all_ch))}")
    if idx < len(tasks):
        t = tasks[idx]
        print(f"- Đang ở: {t['id']} (Phase {t['phase']}) — {t['desc']}")
    else:
        print("- Đã hoàn thành toàn bộ task.")

    print("\n## Challenge theo công nghệ")
    techs = {}
    for c in all_ch:
        techs.setdefault(c["tech"], []).append(c)
    for tech, cs in techs.items():
        done = [c["id"] for c in cs if c["status"] in DONE]
        doing = [c["id"] for c in cs if c["status"] == "🟨"]
        todo = [c["id"] for c in cs if c["status"] not in DONE and c["status"] != "🟨"]
        print(f"- {tech}: xong {pct(len(done), len(cs))}" + (f" — {', '.join(done)}" if done else "")
              + (f" · đang làm: {', '.join(doing)}" if doing else "") + (f" · còn: {', '.join(todo)}" if todo else ""))

    print("\n## Đã làm được")
    done_tasks = [t for t in tasks if t["done"]]
    if not done_tasks:
        print("- Chưa có task nào hoàn thành.")
    for t in done_tasks[-15:]:
        print(f"- {t['id']} {'(đã commit)' if committed(t) else '(CHƯA commit)'} — {t['desc']}")
    if len(done_tasks) > 15:
        print(f"- ... và {len(done_tasks) - 15} task trước đó")

    hb = HANDBOOK / "README.md"
    if hb.exists():
        idx_lines = sections(hb.read_text()).get("Chỉ mục theo chủ đề", [])
        rows = [r for r in table_rows(idx_lines, r"[^|\s-][^|]*") if r[0] != "Chủ đề" and len(r) >= 3]
        if rows:
            print("\n## Kiến thức đã ghi trong handbook")
            for r in rows:
                print(f"- {r[0]} — {r[1]}: {r[2]}")

    print("\n## Còn lại")
    if idx < len(tasks):
        cur_phase = tasks[idx]["phase"]
        remaining = [t for t in tasks[idx:] if t["phase"] == cur_phase]
        print(f"- Phase {cur_phase} còn {len(remaining)} task: {', '.join(t['id'] for t in remaining)}")
        for p in phases:
            if p["n"] > cur_phase:
                print(f"- {p['title']}: {len(p['tasks'])} task — mốc: {p['milestone']}")
    not_written = [str(p["n"]) for p in phases if p["status"] in DONE and not p["lessons_written"]]
    if not_written:
        print(f"- Lessons learned chưa viết cho phase đã đóng: {', '.join(not_written)}")

    log = git("log", "--oneline", "-n", "10")
    if log:
        print("\n## Commit gần đây")
        for line in log.splitlines():
            print(f"- {line}")


STEP_NAMES = {
    0: "Validate + tạo checklist con",
    1: "Gen test case → chờ người dùng duyệt",
    2: "Code",
    3: "Agent tự viết unit test",
    4: "Build + chạy lại unit test",
    5: "Test theo test case + ghi handbook",
    6: "Hỏi commit / push",
}


def cmd_step(arg):
    """Task cần làm và bước tiếp theo — dùng cho skill execute-task."""
    phases, tasks = load_all()
    idx = next((i for i, t in enumerate(tasks) if t["id"].lower() == arg.lower()), None) if arg else current_index(tasks)
    if idx is None:
        print(f"Không tìm thấy task {arg}.")
        return
    if idx >= len(tasks):
        print("Tất cả task đã xong và đã commit. Không còn task để thực hiện.")
        return
    t = tasks[idx]
    phase = next(p for p in phases if p["n"] == t["phase"])
    pdir = PLANNING / phase["dir"]
    tc_file = pdir / "tasks" / t["id"] / "test-cases.md"
    if not t["steps"]:
        step = 0
    else:
        step = next((s["n"] for s in sorted(t["steps"], key=lambda s: s["n"]) if not s["done"]), 6)
    print(f"TASK: {t['id']}")
    print(f"MÔ TẢ: {t['desc']}")
    print(f"PHASE: {phase['title']} ({phase['status']})")
    print(f"README PHASE: {(pdir / 'README.md').relative_to(ROOT)}")
    print(f"TEST CASE TASK: {tc_file.relative_to(ROOT)} ({'đã có' if tc_file.exists() else 'chưa có'})")
    print(f"ACCEPTANCE TESTS: {(pdir / 'acceptance-tests.md').relative_to(ROOT)}")
    hb = HANDBOOK / f"phase-{phase['n']}" / f"{t['id']}.md"
    print(f"HANDBOOK TASK: {hb.relative_to(ROOT)} ({'đã có' if hb.exists() else 'chưa có'})")
    if t["challenges"]:
        print("CHALLENGE: " + " | ".join(ch_line(c, phases) for c in t["challenges"]))
    print(f"BƯỚC TIẾP THEO: {step} — {STEP_NAMES[step]}")
    if step == 1 and tc_file.exists():
        print("GHI CHÚ: test case đã viết, đang chờ người dùng duyệt.")
    if step == 0:
        print("VALIDATE BƯỚC 0:")
        for ok, text in validate(phases, tasks, idx):
            print(f"- {'✅' if ok else '❌'} {text}")
    if t["steps"]:
        print("CHECKLIST CON:")
        for st in t["steps"]:
            print(f"- [{'x' if st['done'] else ' '}] {st['n']}. {st['text']}")


def _task_file(task_id):
    phases, tasks = load_all()
    t = next((t for t in tasks if t["id"] == task_id), None)
    if not t:
        sys.exit(f"Không tìm thấy task {task_id}")
    phase = next(p for p in phases if p["n"] == t["phase"])
    return t, PLANNING / phase["dir"]


def cmd_mark(argv):
    """Thao tác checklist của task (dùng khi làm task, tránh sửa tay):

    mark start <ID> <mô tả test case>      thêm checklist con 6 bước (bước 0)
    mark step <ID> <n> [ghi chú bước 6]    tick bước n; n=5 tick cả dòng task
    mark tc <ID>                           đánh ✅ mọi test case của task
    """
    if len(argv) < 2:
        sys.exit(cmd_mark.__doc__)
    action, tid = argv[0], argv[1]
    t, pdir = _task_file(tid)
    readme = pdir / "README.md"
    text = readme.read_text()
    line_re = re.compile(rf"^- \[( |x)\] \*\*{re.escape(tid)}\*\* .*$", re.M)
    m = line_re.search(text)
    if action == "start":
        if t["steps"]:
            sys.exit(f"{tid} đã có checklist con")
        tc = " ".join(argv[2:]) or f"{tid}-TC01..TCnn"
        block = "\n".join([
            f"  - [ ] 1. Test case: {tc} — đã được duyệt",
            "  - [ ] 2. Code", "  - [ ] 3. Unit test", "  - [ ] 4. Build + unit test pass",
            "  - [ ] 5. Test case pass + handbook",
            f"  - [ ] 6. Commit: `<type(scope): mô tả [{tid}]>` · Push: có/không"])
        text = text[:m.end()] + "\n" + block + text[m.end():]
    elif action == "step":
        n = int(argv[2])
        note = " ".join(argv[3:])
        seg_end = text.find("\n- [", m.end())
        seg_end = len(text) if seg_end == -1 else seg_end
        seg = text[m.end():seg_end]
        step_re = re.compile(rf"^(\s+- )\[ \]( {n}\. )(.*)$", re.M)
        if not step_re.search(seg):
            sys.exit(f"Bước {n} của {tid} không có hoặc đã tick")
        if n == 6 and note:
            seg = step_re.sub(lambda mm: f"{mm.group(1)}[x]{mm.group(2)}{note}", seg, count=1)
        else:
            seg = step_re.sub(r"\1[x]\2\3", seg, count=1)
        text = text[:m.end()] + seg + text[seg_end:]
        if n == 5:
            text = line_re.sub(lambda mm: mm.group(0).replace("- [ ]", "- [x]", 1), text, count=1)
    elif action == "tc":
        f = pdir / "tasks" / tid / "test-cases.md"
        s2 = re.sub(rf"^(\| {re.escape(tid)}-TC\d+ \|.*)\| ⬜ \|$", r"\1| ✅ |", f.read_text(), flags=re.M)
        f.write_text(s2)
        print(f"{tid}: {s2.count('| ✅ |')} test case ✅")
        return
    else:
        sys.exit(cmd_mark.__doc__)
    readme.write_text(text)
    print(f"{tid}: {action} {' '.join(argv[2:3])} ok")


GUARDED = ("com/tm/server/", "com/tm/app/")


def cmd_guard():
    """PreToolUse hook: chặn sửa code khi bước 0 chưa đạt hoặc test case chưa được duyệt.

    Đọc JSON hook từ stdin. Chỉ áp dụng cho file trong com/tm/server và com/tm/app.
    Không kiểm tra working tree sạch (khi đang làm task, tree luôn có thay đổi của chính task).
    """
    import json
    try:
        payload = json.load(sys.stdin)
    except Exception:
        sys.exit(0)
    ti = payload.get("tool_input") or {}
    path = ti.get("file_path") or ti.get("notebook_path") or ""
    if not path:
        sys.exit(0)
    try:
        rel = Path(path).resolve().relative_to(ROOT).as_posix()
    except ValueError:
        sys.exit(0)
    if not rel.startswith(GUARDED):
        sys.exit(0)

    phases, tasks = load_all()
    idx = current_index(tasks)
    if idx >= len(tasks):
        sys.exit(0)
    cur = tasks[idx]
    fails = [text for ok, text in validate(phases, tasks, idx) if not ok and not text.startswith("Working tree")]
    step1 = next((s for s in cur["steps"] if s["n"] == 1), None)
    if not cur["steps"]:
        fails.append(f"Task {cur['id']} chưa có checklist con (chưa qua bước 0)")
    elif not (step1 and step1["done"]):
        fails.append(f"Test case của {cur['id']} chưa được người dùng duyệt (bước 1 chưa [x])")
    if not fails:
        sys.exit(0)
    reason = (f"[snaptix guard] Không được sửa code ({rel}) cho task {cur['id']}:\n"
              + "\n".join(f"- {f}" for f in fails)
              + "\nLàm theo quy trình trong CLAUDE.md. Người dùng có thể tắt hook qua /hooks nếu thật sự cần.")
    print(json.dumps({"hookSpecificOutput": {"hookEventName": "PreToolUse",
                                             "permissionDecision": "deny",
                                             "permissionDecisionReason": reason}}, ensure_ascii=False))
    sys.exit(0)


if __name__ == "__main__":
    cmd = sys.argv[1] if len(sys.argv) > 1 else "sumup"
    arg = " ".join(sys.argv[2:]).strip()
    {"task": lambda: cmd_task(arg), "phase": lambda: cmd_phase(arg), "sumup": cmd_sumup,
     "validate": lambda: cmd_validate(arg), "guard": cmd_guard, "step": lambda: cmd_step(arg),
     "mark": lambda: cmd_mark(sys.argv[2:])}.get(cmd, lambda: print(__doc__))()
