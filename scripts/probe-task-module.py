#!/usr/bin/env python3
"""Sesi probe HTTP nyata untuk modul Task (T-062).

Menjalankan bukti terhadap server yang **benar-benar hidup** di lokal: login,
membuat project + anggota + dokumen + task, lalu memeriksa setiap perilaku yang
diklaim halaman Tasks di frontend:

  * penyaring tri-state `?overdue=` (tidak dikirim / true / false)
  * interval tertutup `[due_from, due_to]`, termasuk kedua batas sama dan
    rentang terbalik (`422` pada `due_to`)
  * `meta.total` pada halaman di luar rentang (tambalan C-048/T-043)
  * tiga transisi status: Start, Complete, Reopen, plus `409` di luar tabel
  * cakupan baca dan cakupan tulis (viewer non-anggota `404`/`total 0`)

Kredensial dibaca dari `.env` dan **tidak pernah** dicetak.
"""

from __future__ import annotations

import json
import os
import subprocess
import sys
import urllib.error
import urllib.request

BASE = "http://localhost:8081/api/v1"
ROOT = "/Users/posindonesia/Documents/Learn/Business Dev"

# Hash bcrypt dari "Rahasia123!" yang dihasilkan sesi probe ini lewat
# `bcrypt.GenerateFromPassword` (cost Minimum, khusus baris uji). Password uji
# bukan rahasia, dan hash-nya tidak dipakai di luar sesi ini.
PROBE_HASH = "$2a$04$wG.DkuHGUdfh4CvErDNDA.YuPE2Y03gtUNxutr7Q0tU/ehg6DXI1m"
PROBE_PASSWORD = "Rahasia123!"


def env() -> dict[str, str]:
    values: dict[str, str] = {}
    with open(os.path.join(ROOT, ".env"), encoding="utf-8") as handle:
        for line in handle:
            line = line.strip()
            if not line or line.startswith("#") or "=" not in line:
                continue
            key, _, value = line.partition("=")
            values[key.strip()] = value.strip()
    return values


ENV = env()


def call(method: str, path: str, token: str | None = None, body=None):
    data = None
    headers = {"Accept": "application/json"}
    if body is not None:
        data = json.dumps(body).encode()
        headers["Content-Type"] = "application/json"
    if token:
        headers["Authorization"] = f"Bearer {token}"
    request = urllib.request.Request(
        BASE + path, data=data, headers=headers, method=method
    )
    try:
        with urllib.request.urlopen(request) as response:
            payload = json.loads(response.read().decode() or "{}")
            return response.status, payload
    except urllib.error.HTTPError as error:
        raw = error.read().decode()
        try:
            return error.code, json.loads(raw or "{}")
        except json.JSONDecodeError:
            return error.code, {"raw": raw}


def psql(sql: str) -> str:
    command = [
        "psql",
        "-h",
        ENV["DB_HOST"],
        "-p",
        ENV["DB_PORT"],
        "-U",
        ENV["DB_USER"],
        "-d",
        ENV["DB_NAME"],
        # `-q` membuang tag perintah: tanpa itu hasil `INSERT … RETURNING`
        # datang sebagai nilai + baris "INSERT 0 1" dan jadi UUID tidak sah pada
        # kueri berikutnya.
        "-qAt",
        "-c",
        sql,
    ]
    completed = subprocess.run(
        command,
        capture_output=True,
        text=True,
        env={**os.environ, "PGPASSWORD": ENV["DB_PASSWORD"]},
        check=False,
    )
    if completed.returncode != 0:
        raise RuntimeError(completed.stderr.strip())
    return completed.stdout.strip()


RESULTS: list[str] = []


def field_of(payload: dict) -> object:
    """Nama field dari `error.details`, yang dapat berupa objek maupun daftar."""
    details = (payload.get("error") or {}).get("details")
    if isinstance(details, list):
        return [item.get("field") for item in details if isinstance(item, dict)]
    if isinstance(details, dict):
        return details.get("field")
    return None


def check(label: str, ok: bool, detail: str) -> None:
    RESULTS.append(f"{'PASS' if ok else 'FAIL'} | {label} | {detail}")
    print(f"[{'PASS' if ok else 'FAIL'}] {label}: {detail}", flush=True)


# Baris `audit_logs` yang ada sebelum sesi probe: keadaan dev yang sudah
# tercatat di ledger (`audit_logs 43`). Pemangkasan memakai urutan penyisipan,
# jadi sesi probe yang terpotong di tengah tidak membuat baseline bergeser.
BASELINE_AUDIT_ROWS = 43


def cleanup() -> None:
    """Membuang sisa sesi probe sebelumnya (probe ini harus dapat dijalankan ulang)."""
    psql(
        "BEGIN; SET LOCAL bwdcs.audit_maintenance = 'on'; "
        # `audit_logs` lebih dulu: FK-nya menunjuk `users`, dan urutan ini juga
        # wajib bagi `documents` yang punya versi (trigger append-only).
        "DELETE FROM audit_logs WHERE id NOT IN ("
        f"SELECT id FROM audit_logs ORDER BY created_at, id LIMIT {BASELINE_AUDIT_ROWS}); "
        "DELETE FROM tasks WHERE project_id IN (SELECT id FROM projects WHERE code = 'PROBE062'); "
        "DELETE FROM documents WHERE project_id IN (SELECT id FROM projects WHERE code = 'PROBE062'); "
        "DELETE FROM project_members WHERE project_id IN (SELECT id FROM projects WHERE code = 'PROBE062'); "
        "DELETE FROM user_roles WHERE user_id IN (SELECT id FROM users WHERE username LIKE 'uji-task-%'); "
        # Telemetri login sesi probe (ADR-0022 menyimpannya di luar `audit_logs`):
        # baris milik user uji **dan** milik admin yang dipakai probe ini.
        f"DELETE FROM login_attempts WHERE username_attempted = '{ENV['ADMIN_USERNAME']}' "
        "OR username_attempted LIKE 'uji-task-%'; "
        "DELETE FROM users WHERE username LIKE 'uji-task-%'; "
        "DELETE FROM projects WHERE code = 'PROBE062'; "
        "COMMIT;"
    )


def main() -> int:
    cleanup()

    # 1. Login
    status, payload = call(
        "POST",
        "/auth/login",
        body={
            "username": ENV["ADMIN_USERNAME"],
            "password": ENV["ADMIN_PASSWORD"],
        },
    )
    admin = payload.get("data", {}).get("token")
    check("1. login admin", status == 200 and bool(admin), f"status={status}")

    # 2. Project (owner_id wajib, dan owner langsung menjadi anggota)
    status, payload = call("GET", "/auth/me", admin)
    admin_id = payload.get("data", {}).get("id")
    org_id = payload.get("data", {}).get("organization_id")
    check("2a. GET /auth/me", status == 200 and bool(admin_id), f"status={status}")

    status, payload = call(
        "POST",
        "/projects",
        admin,
        {
            "code": "PROBE062",
            "name": "Uji Modul Task",
            "description": "sesi probe T-062",
            "owner_id": admin_id,
        },
    )
    project_id = payload.get("data", {}).get("project", {}).get("id")
    check("2b. POST /projects", status == 201 and bool(project_id), f"status={status}")

    # 3. Dua user uji. `POST /admin/users` belum terdaftar (hanya
    #    `/admin/users/:id/unlock`), jadi barisnya ditulis langsung dengan hash
    #    yang benar-benar dihasilkan bcrypt — bukan hash karangan.
    users = {}
    for username, role in (
        ("uji-task-contrib", "contributor"),
        ("uji-task-viewer", "viewer"),
    ):
        users[username] = psql(
            "INSERT INTO users (organization_id, username, email, password_hash) VALUES "
            f"('{org_id}', '{username}', '{username}@example.test', '{PROBE_HASH}') RETURNING id;"
        )
        psql(
            "INSERT INTO user_roles (user_id, role_id) SELECT "
            f"'{users[username]}', id FROM roles WHERE name = '{role}';"
        )
        check(
            f"3. user uji {role} dibuat dengan role-nya",
            bool(users[username]),
            f"id='{users[username][:8]}…' role={role}",
        )

    contributor = users["uji-task-contrib"]
    viewer = users["uji-task-viewer"]

    status, payload = call(
        "POST",
        f"/projects/{project_id}/members",
        admin,
        {"user_id": contributor, "role": "contributor"},
    )
    check("4. anggota project ditambahkan", status in (200, 201), f"status={status}")

    # 5. Dokumen (tanpa berkas) untuk document_id
    status, payload = call(
        "POST",
        "/documents",
        admin,
        {
            "project_id": project_id,
            "title": "BRD",
            "description": "dokumen terkait task",
        },
    )
    document = payload.get("data", {}).get("document", {})
    document_id = document.get("id")
    check(
        "5. POST /documents",
        status == 201 and bool(document_id),
        f"status={status} nomor={document.get('document_number')}",
    )

    # 6. Task uji: A overdue (tenggat lewat), B belum lewat, C akan selesai
    past = "2026-09-01T10:00:00+07:00"
    future = "2026-12-31T10:00:00+07:00"
    tasks = {}
    for key, due, priority in (
        ("A", past, "high"),
        ("B", future, "urgent"),
        ("C", past, "low"),
    ):
        status, payload = call(
            "POST",
            "/tasks",
            admin,
            {
                "project_id": project_id,
                "title": f"Task {key}",
                "description": f"task uji {key}",
                "assignee_id": contributor,
                "priority": priority,
                "due_date": due,
                "document_id": document_id,
            },
        )
        data = payload.get("data", {})
        tasks[key] = data.get("id")
        check(
            f"6{key}. POST /tasks {key}",
            status == 201 and data.get("status") == "open" and data.get("is_overdue") is (key != "B"),
            f"status={status} kanonik={data.get('status')} overdue={data.get('is_overdue')}",
        )

    # 7. Task D tanpa tenggat, lewat jalur pemeliharaan (due_date NOT NULL)
    status, payload = call(
        "POST",
        "/tasks",
        admin,
        {
            "project_id": project_id,
            "title": "Task D tanpa tenggat",
            "assignee_id": contributor,
            "priority": "medium",
            "due_date": future,
        },
    )
    tasks["D"] = payload.get("data", {}).get("id")
    psql(f"UPDATE tasks SET due_date = NULL WHERE id = '{tasks['D']}'")
    status, payload = call("GET", f"/tasks/{tasks['D']}", admin)
    check(
        "7. task tanpa tenggat tidak pernah overdue",
        status == 200
        and payload["data"]["due_date"] is None
        and payload["data"]["is_overdue"] is False,
        f"due_date={payload['data']['due_date']} is_overdue={payload['data']['is_overdue']}",
    )

    def titles(path: str):
        status, payload = call("GET", path, admin)
        data = payload.get("data") or []
        return status, [row["title"] for row in data], payload.get("meta", {})

    # 8. Tri-state overdue
    status, names, meta = titles(f"/tasks?project_id={project_id}")
    check(
        "8a. tanpa penyaring overdue",
        status == 200 and len(names) == 4,
        f"total={meta.get('total')} judul={sorted(names)}",
    )

    status, names, meta = titles(f"/tasks?project_id={project_id}&overdue=true")
    check(
        "8b. ?overdue=true hanya yang lewat tenggat dan belum selesai",
        status == 200 and sorted(names) == ["Task A", "Task C"],
        f"total={meta.get('total')} judul={sorted(names)}",
    )

    status, names, meta = titles(f"/tasks?project_id={project_id}&overdue=false")
    check(
        "8c. ?overdue=false memuat task tanpa tenggat (berbeda dari tidak dikirim)",
        status == 200 and sorted(names) == ["Task B", "Task D tanpa tenggat"],
        f"total={meta.get('total')} judul={sorted(names)}",
    )

    status, payload = call("GET", f"/tasks?project_id={project_id}&overdue=benar", admin)
    check(
        "8d. nilai overdue di luar true/false ditolak 422",
        status == 422 and field_of(payload) in ("overdue", ["overdue"]),
        f"status={status} field={field_of(payload)}",
    )

    # 9. Interval tertutup [due_from, due_to]
    status, names, meta = titles(
        f"/tasks?project_id={project_id}&due_from=2026-09-01T09:00:00%2B07:00&due_to=2026-09-01T11:00:00%2B07:00"
    )
    check(
        "9a. rentang memuat kedua batas secara inklusif",
        status == 200 and sorted(names) == ["Task A", "Task C"],
        f"total={meta.get('total')} judul={sorted(names)}",
    )

    status, names, meta = titles(
        f"/tasks?project_id={project_id}&due_from=2026-09-01T10:00:00%2B07:00&due_to=2026-09-01T10:00:00%2B07:00"
    )
    check(
        "9b. due_to == due_from sah dan berarti satu instan",
        status == 200 and sorted(names) == ["Task A", "Task C"],
        f"total={meta.get('total')} judul={sorted(names)}",
    )

    status, names, meta = titles(f"/tasks?project_id={project_id}&due_to=2026-09-01T10:00:00%2B07:00")
    check(
        "9c. hanya due_to (task tanpa tenggat tidak ikut)",
        status == 200 and sorted(names) == ["Task A", "Task C"],
        f"total={meta.get('total')} judul={sorted(names)}",
    )

    status, names, meta = titles(f"/tasks?project_id={project_id}&due_from=2026-09-01T10:00:00%2B07:00")
    check(
        "9d. hanya due_from",
        status == 200 and sorted(names) == ["Task A", "Task B", "Task C"],
        f"total={meta.get('total')} judul={sorted(names)}",
    )

    status, payload = call(
        "GET",
        f"/tasks?project_id={project_id}&due_from=2026-12-31T10:00:00%2B07:00&due_to=2026-09-01T10:00:00%2B07:00",
        admin,
    )
    check(
        "9e. rentang terbalik ditolak 422 pada due_to",
        status == 422 and field_of(payload) in ("due_to", ["due_to"]),
        f"status={status} field={field_of(payload)}",
    )

    # 10. meta.total pada halaman di luar rentang (T-043/C-048)
    status, payload = call("GET", f"/tasks?project_id={project_id}&page=9&limit=20", admin)
    check(
        "10. halaman di luar rentang tetap melaporkan total",
        status == 200
        and payload.get("data") == []
        and payload.get("meta", {}).get("total") == 4,
        f"baris={len(payload.get('data') or [])} total={payload.get('meta', {}).get('total')}",
    )

    # 11. Penyaring lain
    status, names, _ = titles(f"/tasks?project_id={project_id}&priority=urgent")
    check("11a. penyaring prioritas", status == 200 and names == ["Task B"], f"judul={names}")

    status, names, _ = titles(f"/tasks?project_id={project_id}&assignee_id={contributor}")
    check(
        "11b. penyaring penanggung jawab (My Tasks)",
        status == 200 and len(names) == 4,
        f"judul={sorted(names)}",
    )

    status, payload = call("GET", f"/tasks?project_id={project_id}&status=selesai", admin)
    check(
        "11c. status di luar kosakata kanonik ditolak 422",
        status == 422 and field_of(payload) in ("status", ["status"]),
        f"status={status} field={field_of(payload)}",
    )

    # 12. Transisi status
    status, payload = call("POST", f"/tasks/{tasks['A']}/complete", admin)
    check(
        "12a. Complete pada task Open ditolak 409",
        status == 409,
        f"status={status} code={payload.get('error', {}).get('code')}",
    )

    status, payload = call("PATCH", f"/tasks/{tasks['A']}", admin, {"status": "in_progress"})
    check(
        "12b. Start: open -> in_progress lewat PATCH",
        status == 200 and payload["data"]["status"] == "in_progress",
        f"status={status} kanonik={payload['data']['status']}",
    )

    status, payload = call("POST", f"/tasks/{tasks['A']}/complete", admin)
    check(
        "12c. Complete: in_progress -> completed lewat endpoint sendiri",
        status == 200 and payload["data"]["status"] == "completed",
        f"status={status} kanonik={payload['data']['status']}",
    )

    status, payload = call("POST", f"/tasks/{tasks['A']}/complete", admin)
    check(
        "12d. Complete kedua idempoten (200 tanpa perubahan)",
        status == 200 and payload["data"]["status"] == "completed",
        f"status={status}",
    )

    status, payload = call("PATCH", f"/tasks/{tasks['A']}", admin, {"status": "open"})
    check(
        "12e. Reopen: completed -> open lewat PATCH",
        status == 200 and payload["data"]["status"] == "open",
        f"status={status} kanonik={payload['data']['status']}",
    )

    status, payload = call("PATCH", f"/tasks/{tasks['B']}", admin, {"status": "completed"})
    check(
        "12f. PATCH status=completed (jalan pintas) ditolak 409",
        status == 409,
        f"status={status} code={payload.get('error', {}).get('code')}",
    )

    status, payload = call("PATCH", f"/tasks/{tasks['B']}", admin, {"project_id": project_id})
    check(
        "12g. memindahkan project ditolak 409",
        status == 409,
        f"status={status} code={payload.get('error', {}).get('code')}",
    )

    status, payload = call("PATCH", f"/tasks/{tasks['B']}", admin, {})
    check(
        "12h. PATCH tanpa field yang dapat diubah ditolak 422",
        status == 422 and field_of(payload) in ("body", ["body"]),
        f"status={status} field={field_of(payload)}",
    )

    status, payload = call("POST", "/tasks", admin, {"project_id": project_id, "title": "x", "assignee_id": contributor, "due_date": future, "status": "completed"})
    check(
        "12i. status pada POST ditolak 422 (task selalu lahir open)",
        status == 422 and field_of(payload) in ("status", ["status"]),
        f"status={status} field={field_of(payload)}",
    )

    # 13. Task selesai tidak pernah overdue meski tenggat lewat
    status, payload = call("GET", f"/tasks/{tasks['C']}", admin)
    before = payload["data"]["is_overdue"]
    call("PATCH", f"/tasks/{tasks['C']}", admin, {"status": "in_progress"})
    call("POST", f"/tasks/{tasks['C']}/complete", admin)
    status, payload = call("GET", f"/tasks/{tasks['C']}", admin)
    check(
        "13. penanda overdue hilang saat task selesai (dua nilai berbeda)",
        before is True and payload["data"]["is_overdue"] is False,
        f"sebelum={before} sesudah={payload['data']['is_overdue']}",
    )

    # 14. Cakupan data
    status, payload = call(
        "POST",
        "/auth/login",
        body={"username": "uji-task-viewer", "password": PROBE_PASSWORD},
    )
    viewer_token = payload.get("data", {}).get("token")
    check("14a. login viewer", status == 200 and bool(viewer_token), f"status={status}")

    status, payload = call("GET", f"/tasks?project_id={project_id}", viewer_token)
    check(
        "14b. viewer non-anggota melihat 0 baris",
        status == 200 and payload.get("meta", {}).get("total") == 0,
        f"status={status} total={payload.get('meta', {}).get('total')}",
    )

    status, payload = call("GET", f"/tasks/{tasks['B']}", viewer_token)
    check(
        "14c. task di luar cakupan dibalas 404, bukan 403",
        status == 404 and payload.get("error", {}).get("code") == "NOT_FOUND",
        f"status={status} code={payload.get('error', {}).get('code')}",
    )

    status, payload = call("PATCH", f"/tasks/{tasks['B']}", viewer_token, {"status": "in_progress"})
    check(
        "14d. viewer menulis task tanpa izin -> 403",
        status == 403,
        f"status={status} code={payload.get('error', {}).get('code')}",
    )

    call(
        "POST",
        f"/projects/{project_id}/members",
        admin,
        {"user_id": viewer, "role": "viewer"},
    )
    status, payload = call("GET", f"/tasks/{tasks['B']}", viewer_token)
    check(
        "14e. sesudah menjadi anggota, task yang sama terbaca",
        status == 200 and payload["data"]["title"] == "Task B",
        f"status={status} judul={payload.get('data', {}).get('title')}",
    )

    # 15. Jejak audit
    trace = psql(
        "SELECT action || '=' || count(*) FROM audit_logs "
        "WHERE created_at > NOW() - INTERVAL '1 hour' GROUP BY action ORDER BY action;"
    )
    check(
        "15. jejak audit transisi tertulis",
        "TASK_CREATED" in trace and "TASK_UPDATED" in trace and "TASK_COMPLETED" in trace,
        " ".join(trace.splitlines()),
    )

    # 16. Bersihkan database dev kembali ke baseline. `--keep` menyisakan data
    #     sesi ini supaya bukti antarmuka dapat diambil di peramban terhadap
    #     project dan task yang benar-benar ada; jalankan ulang tanpa `--keep`
    #     untuk mengembalikan baseline.
    if "--keep" in sys.argv:
        print("[INFO] --keep: data probe dibiarkan hidup untuk bukti antarmuka", flush=True)
        return 1 if [row for row in RESULTS if row.startswith("FAIL")] else 0

    cleanup()
    remaining = psql(
        "SELECT 'users ' || (SELECT count(*) FROM users) || ', projects ' || (SELECT count(*) FROM projects) "
        "|| ', documents ' || (SELECT count(*) FROM documents) || ', tasks ' || (SELECT count(*) FROM tasks) "
        "|| ', audit_logs ' || (SELECT count(*) FROM audit_logs) || ', login_attempts ' "
        "|| (SELECT count(*) FROM login_attempts) || ', skema ' || (SELECT max(version_id) FROM goose_db_version);"
    )
    baseline = (
        "users 1, projects 0, documents 0, tasks 0, audit_logs 43, login_attempts 0, skema 11"
    )
    check(
        "16. database dev kembali ke baseline",
        remaining == baseline,
        f"{remaining}" + ("" if remaining == baseline else f" (diharapkan: {baseline})"),
    )

    failures = [row for row in RESULTS if row.startswith("FAIL")]
    print(f"\n=== RINGKASAN: {len(RESULTS) - len(failures)}/{len(RESULTS)} PASS ===")
    return 1 if failures else 0


if __name__ == "__main__":
    sys.exit(main())
