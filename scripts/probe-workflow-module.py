#!/usr/bin/env python3
"""Sesi probe HTTP nyata untuk modul Workflow (T-064).

Menjalankan bukti terhadap server yang **benar-benar hidup** di lokal: login
lima aktor nyata (admin, manager, contributor, viewer, non-anggota), lalu
memeriksa setiap perilaku yang diklaim modul workflow:

  * definisi: pembuatan, penambahan step, `order` duplikat (`409`), validasi
    `422` beserta nama field-nya
  * submit: instance `running` di step pertama, dokumen `in_review`, deadline
    terisi, submit kedua `409`
  * aksi: izin dari **isi body** (`403`), penanggung jawab step (`403`),
    konflik `version` sebagai `409 WORKFLOW_CONFLICT` dengan `details` objek
  * `request_revision`: mundur **satu step** (bukan step 1), instance tetap
    `running`, jeda revisi menolak seluruh aksi
  * `resubmit`: instance yang **sama**, tanpa baris `workflow_actions` baru,
    siklus aksi terbuka kembali, deadline dihitung ulang
  * keterlambatan step sebagai **turunan** (`is_overdue`), bukan status
    (ADR-0012)
  * cakupan data: non-anggota `404`/`total 0`
  * jejak audit per aksi dan baris `notifications` per penerima

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

# Hash bcrypt dari "Rahasia123!" yang dihasilkan lewat
# `bcrypt.GenerateFromPassword` (cost Minimum, khusus baris uji). Password uji
# bukan rahasia, dan hash-nya tidak dipakai di luar sesi ini.
PROBE_HASH = "$2a$04$wG.DkuHGUdfh4CvErDNDA.YuPE2Y03gtUNxutr7Q0tU/ehg6DXI1m"
PROBE_PASSWORD = "Rahasia123!"
PROJECT_CODE = "PROBEWF"


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
        "-h", ENV["DB_HOST"],
        "-p", ENV["DB_PORT"],
        "-U", ENV["DB_USER"],
        "-d", ENV["DB_NAME"],
        # `-q` membuang tag perintah: tanpa itu hasil `INSERT … RETURNING`
        # datang sebagai nilai + baris "INSERT 0 1" dan jadi UUID tidak sah pada
        # kueri berikutnya.
        "-qAt",
        "-c", sql,
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


def psql_script(sql: str) -> None:
    command = [
        "psql",
        "-h", ENV["DB_HOST"],
        "-p", ENV["DB_PORT"],
        "-U", ENV["DB_USER"],
        "-d", ENV["DB_NAME"],
        # `ON_ERROR_STOP` bukan hiasan: tanpa itu psql melanjutkan ke pernyataan
        # berikutnya dan tetap keluar dengan status 0, sehingga pembersihan yang
        # gagal setengah terbaca sebagai pembersihan yang berhasil.
        "-v", "ON_ERROR_STOP=1",
        "-qAt",
        "-f", "-",
    ]
    completed = subprocess.run(
        command,
        input=sql,
        capture_output=True,
        text=True,
        env={**os.environ, "PGPASSWORD": ENV["DB_PASSWORD"]},
        check=False,
    )
    if completed.returncode != 0:
        raise RuntimeError(completed.stderr.strip())


RESULTS: list[str] = []


def field_of(payload: dict) -> object:
    """Nama field dari `error.details`, yang dapat berupa objek maupun daftar."""
    details = (payload.get("error") or {}).get("details")
    if isinstance(details, list):
        return [item.get("field") for item in details if isinstance(item, dict)]
    if isinstance(details, dict):
        return details
    return None


def check(label: str, ok: bool, detail: str) -> None:
    RESULTS.append(f"{'PASS' if ok else 'FAIL'} | {label} | {detail}")
    print(f"[{'PASS' if ok else 'FAIL'}] {label}: {detail}", flush=True)


# Baris `audit_logs` sebelum sesi ini. Dibaca saat mulai, bukan ditulis tangan:
# probe harus dapat dijalankan ulang tanpa menggeser baseline ledger.
BASELINE_AUDIT_ROWS = 0


def cleanup() -> None:
    """Membuang sisa sesi probe sebelumnya (probe harus dapat dijalankan ulang)."""
    # Urutannya mengikat, dan bukan pilihan gaya:
    #
    # 1. `audit_logs` lebih dulu dipangkas ke baseline; baris audit aktor uji
    #    tidak dapat dihapus setelah user-nya hilang (`actor_id` bersifat
    #    RESTRICT) dan tabelnya append-only tanpa `audit_maintenance`.
    # 2. `documents` **sebelum** `workflow_instances`: `documents.workflow_instance_id`
    #    adalah FK tanpa `ON DELETE`, sehingga menghapus instance yang masih
    #    ditunjuk dokumen ditolak. Menghapus dokumennya justru meng-kaskade
    #    `document_versions`, `workflow_instances`, dan `workflow_actions`.
    # 3. Definisi dihapus setelah instance-nya tidak ada lagi
    #    (`workflow_instances.workflow_def_id` juga tanpa `ON DELETE`).
    psql_script(
        f"""BEGIN;
SET LOCAL bwdcs.audit_maintenance = 'on';
DELETE FROM notifications WHERE user_id IN (SELECT id FROM users WHERE username LIKE 'uji-wf-%');
-- Baris audit aktor probe dibuang **berdasarkan aktornya**, bukan hanya
-- berdasarkan urutan: sesi yang terpotong di tengah meninggalkan barisnya, dan
-- baris itu akan memblokir penghapusan user-nya (`actor_id` RESTRICT).
DELETE FROM audit_logs WHERE actor_id IN (SELECT id FROM users WHERE username LIKE 'uji-wf-%');
DELETE FROM audit_logs WHERE id NOT IN (
    SELECT id FROM audit_logs ORDER BY created_at, id LIMIT {BASELINE_AUDIT_ROWS});
DELETE FROM documents WHERE project_id IN (SELECT id FROM projects WHERE code = '{PROJECT_CODE}');
DELETE FROM projects WHERE code = '{PROJECT_CODE}';
DELETE FROM workflow_definitions WHERE name LIKE 'Probe WF%';
COMMIT;
BEGIN;
DELETE FROM login_attempts WHERE username_attempted LIKE 'uji-wf-%';
COMMIT;
BEGIN;
DELETE FROM user_roles WHERE user_id IN (SELECT id FROM users WHERE username LIKE 'uji-wf-%');
DELETE FROM users WHERE username LIKE 'uji-wf-%';
COMMIT;
"""
    )


def bootstrap_users() -> dict[str, dict]:
    """Membuat user probe **di organisasi admin bootstrap** dan mengembalikan id-nya.

    Definisi workflow hidup per organisasi, jadi user probe yang berada di
    organisasi lain tidak dapat memakai definisi yang dibuat administrator probe,
    dan yang diuji bukan lagi cakupan datanya. Organisasi yang dipakai adalah
    organisasi admin bootstrap — milik yang sudah ada di database dev ini.
    """
    admin_username = ENV["ADMIN_USERNAME"]
    org_id = psql(
        f"SELECT organization_id FROM users WHERE username = '{admin_username}'"
    )
    if not org_id:
        raise RuntimeError(f"user admin {admin_username!r} tidak ditemukan di database")

    created: dict[str, dict] = {}
    for suffix, roles in (
        # Administrator **probe**, bukan admin bootstrap: sandi di `.env` bukan
        # urusan skrip ini, dan sesi bukti tidak boleh bergantung padanya.
        ("admin", ["administrator"]),
        ("manager", ["manager"]),
        ("manager2", ["manager"]),
        ("contributor", ["contributor"]),
        ("viewer", ["viewer"]),
        ("stranger", ["contributor"]),
    ):
        username = f"uji-wf-{suffix}"
        user_id = psql(
            "INSERT INTO users (organization_id, username, email, password_hash) "
            f"VALUES ('{org_id}', '{username}', '{username}@example.invalid', "
            f"'{PROBE_HASH}') RETURNING id"
        )
        for role in roles:
            psql(
                "INSERT INTO user_roles (user_id, role_id) "
                f"VALUES ('{user_id}', (SELECT id FROM roles WHERE name = '{role}'))"
            )
        created[suffix] = {"id": user_id, "username": username}
    return created


def login(username: str) -> str:
    status, payload = call(
        "POST", "/auth/login", body={"username": username, "password": PROBE_PASSWORD}
    )
    if status != 200:
        raise RuntimeError(f"login {username!r} gagal: {status} {payload}")
    return payload["data"]["token"]


def main() -> int:
    global BASELINE_AUDIT_ROWS

    # Baseline = baris audit yang **bukan** milik aktor probe. Menghitung
    # seluruh baris akan menghitung sisa sesi yang terpotong sebagai bagian dari
    # baseline, dan pembersihan berikutnya lalu dinilai gagal atas selisih yang
    # sebenarnya sudah dibereskan.
    BASELINE_AUDIT_ROWS = int(psql(
        "SELECT count(*) FROM audit_logs WHERE actor_id NOT IN "
        "(SELECT id FROM users WHERE username LIKE 'uji-wf-%')"
    ))
    cleanup()

    users = bootstrap_users()

    tokens = {name: login(user["username"]) for name, user in users.items()}
    admin_token = tokens["admin"]

    # --- Definisi ---

    status, invalid = call(
        "POST", "/workflows/definitions", admin_token,
        {"name": "Probe WF Kosong", "steps": []},
    )
    check(
        "definisi tanpa step ditolak 422 pada field steps",
        status == 422 and "steps" in (field_of(invalid) or []),
        f"HTTP {status}, details {field_of(invalid)}",
    )

    status, duplicate = call(
        "POST", "/workflows/definitions", admin_token,
        {"name": "Probe WF Duplikat", "steps": [
            {"name": "A", "order": 1, "responsible_role": "manager"},
            {"name": "B", "order": 1, "responsible_role": "manager"},
        ]},
    )
    check(
        "order duplikat di dalam satu permintaan ditolak 422 pada step keduanya",
        status == 422 and "steps[1].order" in (field_of(duplicate) or []),
        f"HTTP {status}, details {field_of(duplicate)}",
    )

    status, role_invalid = call(
        "POST", "/workflows/definitions", admin_token,
        {"name": "Probe WF Role", "steps": [
            {"name": "A", "order": 1, "responsible_role": "reviewer"}]},
    )
    check(
        "role di luar empat role sistem ditolak 422 (peran fungsional bukan role, C-006)",
        status == 422 and "steps[0].responsible_role" in (field_of(role_invalid) or []),
        f"HTTP {status}, details {field_of(role_invalid)}",
    )

    status, viewer_denied = call(
        "GET", "/workflows/definitions", tokens["viewer"]
    )
    check(
        "semua role boleh membaca daftar definisi (workflow_definition:read)",
        status == 200,
        f"HTTP {status}",
    )

    status, contributor_denied = call(
        "POST", "/workflows/definitions", tokens["contributor"],
        {"name": "Probe WF Ditolak", "steps": [
            {"name": "A", "order": 1, "responsible_role": "manager"}]},
    )
    check(
        "contributor tidak boleh mengubah definisi (workflow_definition:manage → 403)",
        status == 403,
        f"HTTP {status}, code {(contributor_denied.get('error') or {}).get('code')}",
    )

    status, definition = call(
        "POST", "/workflows/definitions", admin_token,
        {"name": "Probe WF Utama", "description": "dua step", "steps": [
            {"name": "Technical Review", "order": 1, "responsible_role": "manager", "deadline_days": 3},
            {"name": "Final Approval", "order": 2, "responsible_role": "administrator", "deadline_days": 2},
        ]},
    )
    check(
        "definisi dua step dibuat 201 beserta step-nya",
        status == 201 and len(definition["data"]["steps"]) == 2,
        f"HTTP {status}, {len((definition.get('data') or {}).get('steps', []))} step",
    )
    definition_id = definition["data"]["id"]

    # Penambahan step diuji pada definisi **terpisah**: menambah step pada
    # definisi yang dipakai siklus di bawah akan mengubah "step terakhir"-nya,
    # dan siklus itu berhenti membuktikan apa yang dimaksudnya.
    status, extra_definition = call(
        "POST", "/workflows/definitions", admin_token,
        {"name": "Probe WF Tambahan", "steps": [
            {"name": "Satu", "order": 1, "responsible_role": "manager"}]},
    )
    extra_definition_id = extra_definition["data"]["id"]

    status, step_added = call(
        "POST", f"/workflows/definitions/{extra_definition_id}/steps", admin_token,
        {"name": "QA Review", "order": 2, "responsible_role": "", "deadline_days": None},
    )
    check(
        "step baru boleh ditambahkan; role kosong tersimpan NULL (any authenticated)",
        status == 201 and step_added["data"]["responsible_role"] is None
        and step_added["data"]["deadline_days"] is None,
        f"HTTP {status}, role {step_added.get('data', {}).get('responsible_role')}",
    )

    status, order_taken = call(
        "POST", f"/workflows/definitions/{extra_definition_id}/steps", admin_token,
        {"name": "Bentrok", "order": 1, "responsible_role": "manager"},
    )
    check(
        "order yang sudah dipakai → 409 CONFLICT, bukan 500",
        status == 409,
        f"HTTP {status}, code {(order_taken.get('error') or {}).get('code')}",
    )

    status, reordered = call(
        "GET", f"/workflows/definitions/{extra_definition_id}", admin_token
    )
    check(
        "menambah step tidak menomori ulang step yang sudah ada",
        status == 200
        and [step["order"] for step in reordered["data"]["steps"]] == [1, 2],
        f"urutan {[step['order'] for step in (reordered.get('data') or {}).get('steps', [])]}",
    )

    # --- Project dan anggota ---

    status, project = call(
        "POST", "/projects", tokens["manager"],
        {"code": PROJECT_CODE, "name": "Project Probe Workflow",
         "owner_id": users["manager"]["id"]},
    )
    if status != 201:
        raise RuntimeError(f"buat project probe gagal: {status} {project}")
    project_id = project["data"]["project"]["id"]

    for member in ("contributor", "viewer"):
        call(
            "POST", f"/projects/{project_id}/members", tokens["manager"],
            {"user_id": users[member]["id"], "role": member},
        )

    # --- Siklus penuh: submit → approve → selesai ---

    status, document = call(
        "POST", "/documents", tokens["contributor"],
        {"project_id": project_id, "title": "BRD Probe Workflow"},
    )
    document_id = document["data"]["document"]["id"]
    document_number = document["data"]["document"]["document_number"]

    status, instance = call(
        "POST", "/workflows/submit", tokens["contributor"],
        {"document_id": document_id, "workflow_definition_id": definition_id},
    )
    data = instance.get("data") or {}
    check(
        "submit membuat instance running di step pertama, version 0",
        status == 201 and data.get("status") == "running"
        and data.get("current_step") == 1 and data.get("version") == 0,
        f"HTTP {status}, status {data.get('status')}, step {data.get('current_step')}, version {data.get('version')}",
    )
    check(
        "dokumen berpindah ke in_review dan terikat ke instance-nya",
        data.get("document_status") == "in_review",
        f"document_status {data.get('document_status')}",
    )
    check(
        "current_step_deadline terisi (NOW() + deadline_days step)",
        data.get("current_step_deadline") is not None,
        f"deadline {data.get('current_step_deadline')}",
    )
    # Penanggung jawab step bersifat **role-based** (`43-WORKFLOW.md` §5): yang
    # berhak adalah setiap pemegang role itu, bukan satu orang. Karena itu
    # keduanya harus ada, dan tidak ada role lain yang ikut.
    responsible = set(data.get("responsible_user_ids") or [])
    check(
        "penanggung jawab step pertama adalah **semua** pemegang role manager",
        responsible == {users["manager"]["id"], users["manager2"]["id"]},
        f"responsible_user_ids {sorted(responsible)}",
    )

    instance_id = data.get("id")

    status, again = call(
        "POST", "/workflows/submit", tokens["contributor"],
        {"document_id": document_id, "workflow_definition_id": definition_id},
    )
    check(
        "submit kedua atas dokumen yang sama → 409 CONFLICT",
        status == 409,
        f"HTTP {status}, code {(again.get('error') or {}).get('code')}",
    )

    notifications = psql(
        f"SELECT count(*) FROM notifications WHERE user_id = '{users['manager']['id']}' "
        "AND type = 'APPROVAL_REQUIRED'"
    )
    check(
        "penanggung jawab step menerima satu notifikasi APPROVAL_REQUIRED",
        notifications == "1",
        f"baris notifikasi {notifications}",
    )

    audit = psql(
        f"SELECT count(*) FROM audit_logs WHERE action = 'DOCUMENT_SUBMITTED' "
        f"AND entity_id = '{document_number}'"
    )
    check(
        "jejak audit DOCUMENT_SUBMITTED menunjuk dokumennya",
        audit == "1",
        f"baris audit {audit}",
    )

    # --- Daftar Approvals dan cakupannya ---

    status, queue = call("GET", "/workflows/instances?scope=assigned_to_me", tokens["manager"])
    rows = queue.get("data") or []
    check(
        "antrean assigned_to_me manager memuat tepat instance yang step aktifnya miliknya",
        status == 200 and (queue.get("meta") or {}).get("total") == 1
        and len(rows) == 1 and rows[0]["current_step_name"] == "Technical Review",
        f"HTTP {status}, total {(queue.get('meta') or {}).get('total')}, step "
        f"{rows[0]['current_step_name'] if rows else '-'}",
    )
    check(
        "keterlambatan step tampil sebagai turunan `is_overdue`, bukan status",
        rows and rows[0]["is_overdue"] is False and rows[0]["status"] == "running",
        f"is_overdue {rows[0]['is_overdue'] if rows else '-'}, status {rows[0]['status'] if rows else '-'}",
    )

    status, stranger_queue = call("GET", "/workflows/instances", tokens["stranger"])
    check(
        "non-anggota melihat 0 baris dengan total 0 (bukan 404 yang membocorkan)",
        status == 200 and (stranger_queue.get("meta") or {}).get("total") == 0,
        f"HTTP {status}, total {(stranger_queue.get('meta') or {}).get('total')}",
    )

    status, stranger_detail = call(
        "GET", f"/workflows/instances/{instance_id}", tokens["stranger"]
    )
    check(
        "detail instance di luar cakupan → 404 NOT_FOUND",
        status == 404,
        f"HTTP {status}",
    )

    # --- Izin aksi datang dari isi body ---

    status, contributor_action = call(
        "POST", f"/workflows/instances/{instance_id}/actions", tokens["contributor"],
        {"action": "approve"},
    )
    check(
        "contributor ditolak 403 meski route-nya hanya menuntut workflow_instance:read",
        status == 403,
        f"HTTP {status}, code {(contributor_action.get('error') or {}).get('code')}",
    )

    status, admin_action = call(
        "POST", f"/workflows/instances/{instance_id}/actions", admin_token,
        {"action": "approve"},
    )
    check(
        "administrator lolos izin tetapi bukan penanggung jawab step → 403",
        status == 403,
        f"HTTP {status}, code {(admin_action.get('error') or {}).get('code')}",
    )

    status, bad_action = call(
        "POST", f"/workflows/instances/{instance_id}/actions", tokens["manager"],
        {"action": "escalate"},
    )
    check(
        "aksi di luar kosakata ditolak 422 pada field action",
        status == 422 and "action" in (field_of(bad_action) or []),
        f"HTTP {status}, details {field_of(bad_action)}",
    )

    status, conflict = call(
        "POST", f"/workflows/instances/{instance_id}/actions", tokens["manager"],
        {"action": "approve", "version": 7},
    )
    details = field_of(conflict)
    check(
        "version basi → 409 WORKFLOW_CONFLICT dengan details objek keadaan terkini",
        status == 409
        and (conflict.get("error") or {}).get("code") == "WORKFLOW_CONFLICT"
        and isinstance(details, dict)
        and details.get("current_step") == 1
        and details.get("current_status") == "running"
        and details.get("current_version") == 0,
        f"HTTP {status}, details {details}",
    )

    status, advanced = call(
        "POST", f"/workflows/instances/{instance_id}/actions", tokens["manager"],
        {"action": "approve", "comment": "ok teknis", "version": 0},
    )
    data = advanced.get("data") or {}
    check(
        "approve step tengah memindahkan instance ke step 2 dan menaikkan version",
        status == 200 and data.get("current_step") == 2 and data.get("version") == 1
        and data.get("status") == "running",
        f"HTTP {status}, step {data.get('current_step')}, version {data.get('version')}",
    )
    check(
        "dokumen tetap in_review di step tengah",
        data.get("document_status") == "in_review",
        f"document_status {data.get('document_status')}",
    )
    check(
        "riwayat aksi ikut pada response aksi, lengkap dengan aktornya",
        len(data.get("actions") or []) == 1
        and data["actions"][0]["actor_username"] == users["manager"]["username"],
        f"{len(data.get('actions') or [])} baris riwayat",
    )

    audit = psql(
        "SELECT action FROM audit_logs WHERE action IN ('WORKFLOW_STEP_ADVANCED','DOCUMENT_APPROVED') "
        f"AND entity_id = '{document_number}' ORDER BY created_at"
    )
    check(
        "approve step tengah tercatat WORKFLOW_STEP_ADVANCED, **bukan** DOCUMENT_APPROVED",
        audit.splitlines() == ["WORKFLOW_STEP_ADVANCED"],
        f"aksi audit {audit.splitlines()}",
    )

    status, completed = call(
        "POST", f"/workflows/instances/{instance_id}/actions", admin_token,
        {"action": "approve"},
    )
    data = completed.get("data") or {}
    check(
        "approve step terakhir menyelesaikan instance dan menyetujui dokumen",
        status == 200 and data.get("status") == "completed"
        and data.get("document_status") == "approved" and data.get("completed_at") is not None,
        f"HTTP {status}, status {data.get('status')}, dokumen {data.get('document_status')}",
    )
    check(
        "deadline dikosongkan setelah selesai (tidak ada step yang menunggu)",
        data.get("current_step_deadline") is None,
        f"deadline {data.get('current_step_deadline')}",
    )

    status, after_done = call(
        "POST", f"/workflows/instances/{instance_id}/actions", admin_token,
        {"action": "approve"},
    )
    check(
        "aksi atas instance yang sudah selesai → 409 CONFLICT (bukan 403)",
        status == 409,
        f"HTTP {status}, code {(after_done.get('error') or {}).get('code')}",
    )

    # --- Siklus revisi penuh ---

    status, revision_definition = call(
        "POST", "/workflows/definitions", admin_token,
        {"name": "Probe WF Tiga Step", "steps": [
            {"name": "Langkah Satu", "order": 1, "responsible_role": "manager", "deadline_days": 3},
            {"name": "Langkah Dua", "order": 2, "responsible_role": "manager", "deadline_days": 3},
            {"name": "Langkah Tiga", "order": 3, "responsible_role": "manager", "deadline_days": 3},
        ]},
    )
    revision_definition_id = revision_definition["data"]["id"]

    status, revision_document = call(
        "POST", "/documents", tokens["contributor"],
        {"project_id": project_id, "title": "BRD Probe Revisi"},
    )
    revision_document_id = revision_document["data"]["document"]["id"]

    status, revision_instance = call(
        "POST", "/workflows/submit", tokens["contributor"],
        {"document_id": revision_document_id, "workflow_definition_id": revision_definition_id},
    )
    revision_instance_id = revision_instance["data"]["id"]

    # Dua approve, sehingga step aktifnya menjadi step 3 — satu-satunya keadaan
    # yang dapat membedakan "mundur satu step" dari "reset ke step 1" (C-022).
    for expected_step in (2, 3):
        status, step_data = call(
            "POST", f"/workflows/instances/{revision_instance_id}/actions", tokens["manager"],
            {"action": "approve"},
        )
        if status != 200 or step_data["data"]["current_step"] != expected_step:
            raise RuntimeError(f"approve menuju step {expected_step} gagal: {status} {step_data}")

    status, revised = call(
        "POST", f"/workflows/instances/{revision_instance_id}/actions", tokens["manager"],
        {"action": "request_revision", "comment": "perbaiki bab 3"},
    )
    data = revised.get("data") or {}
    check(
        "request_revision dari step 3 mundur ke step 2 — step sebelumnya, bukan step 1",
        status == 200 and data.get("current_step") == 2
        and data.get("current_step_name") == "Langkah Dua",
        f"HTTP {status}, step {data.get('current_step')} ({data.get('current_step_name')})",
    )
    check(
        "instance tetap running; revision_required hanya milik dokumen (ADR-0016)",
        data.get("status") == "running"
        and data.get("document_status") == "revision_required",
        f"status {data.get('status')}, dokumen {data.get('document_status')}",
    )

    status, paused = call(
        "POST", f"/workflows/instances/{revision_instance_id}/actions", tokens["manager"],
        {"action": "approve"},
    )
    check(
        "selama jeda revisi seluruh aksi ditolak 409 walau instance masih running",
        status == 409,
        f"HTTP {status}, code {(paused.get('error') or {}).get('code')}",
    )

    status, early_resubmit = call(
        "POST", f"/workflows/instances/{revision_instance_id}/resubmit", tokens["contributor"],
        {},
    )
    check(
        "resubmit tanpa versi baru ditolak 409 (review tidak diulang atas berkas lama)",
        status == 409,
        f"HTTP {status}, code {(early_resubmit.get('error') or {}).get('code')}",
    )

    upload_status = psql(
        "SELECT count(*) FROM document_versions WHERE document_id = "
        f"'{revision_document_id}'"
    )
    check(
        "dokumen revisi belum punya satu pun versi berkas",
        upload_status == "0",
        f"baris document_versions {upload_status}",
    )

    # Unggahan versi baru lewat route sungguhan (multipart), lalu re-submit.
    status, uploaded = upload_version(tokens["contributor"], revision_document_id, "revisi.pdf")
    if status != 201:
        raise RuntimeError(f"unggah versi revisi gagal: {status} {uploaded}")

    version_before = data.get("version")
    status, resumed = call(
        "POST", f"/workflows/instances/{revision_instance_id}/resubmit", tokens["contributor"],
        {"version": version_before},
    )
    data = resumed.get("data") or {}
    check(
        "resubmit melanjutkan instance yang sama dengan version naik satu",
        status == 200 and data.get("id") == revision_instance_id
        and data.get("version") == version_before + 1,
        f"HTTP {status}, id {data.get('id')}, version {data.get('version')}",
    )
    check(
        "resubmit mengembalikan dokumen ke in_review tanpa memindahkan step",
        data.get("document_status") == "in_review" and data.get("current_step") == 2,
        f"dokumen {data.get('document_status')}, step {data.get('current_step')}",
    )
    check(
        "resubmit **tidak** menambah baris workflow_actions (bukan keputusan reviewer)",
        len(data.get("actions") or []) == 3,
        f"{len(data.get('actions') or [])} baris riwayat",
    )

    actions = psql(
        f"SELECT count(*) FROM workflow_actions WHERE instance_id = '{revision_instance_id}'"
    )
    check(
        "tabel workflow_actions tetap memuat tiga keputusan reviewer",
        actions == "3",
        f"baris aksi {actions}",
    )

    audit = psql(
        "SELECT action FROM audit_logs WHERE action IN "
        "('DOCUMENT_SUBMITTED','DOCUMENT_RESUBMITTED') "
        f"AND entity_id = (SELECT document_number FROM documents WHERE id = '{revision_document_id}') "
        "ORDER BY created_at"
    )
    check(
        "riwayat audit membedakan DOCUMENT_SUBMITTED dari DOCUMENT_RESUBMITTED",
        audit.splitlines() == ["DOCUMENT_SUBMITTED", "DOCUMENT_RESUBMITTED"],
        f"aksi audit {audit.splitlines()}",
    )

    # Siklus terbuka: reviewer yang tadi meminta revisi dapat memutuskan lagi.
    status, reopened = call(
        "POST", f"/workflows/instances/{revision_instance_id}/actions", tokens["manager"],
        {"action": "approve", "comment": "revisi ok"},
    )
    check(
        "siklus aksi terbuka: reviewer yang sama memutuskan lagi setelah re-submit (C-025)",
        status == 200 and reopened["data"]["current_step"] == 3,
        f"HTTP {status}, step {(reopened.get('data') or {}).get('current_step')}",
    )

    # --- Keterlambatan sebagai turunan ---

    psql(
        "UPDATE workflow_instances SET current_step_deadline = NOW() - INTERVAL '1 hour' "
        f"WHERE id = '{revision_instance_id}'"
    )
    status, overdue_list = call("GET", "/workflows/instances?status=running", tokens["manager"])
    overdue_row = next(
        (row for row in (overdue_list.get("data") or []) if row["id"] == revision_instance_id),
        None,
    )
    check(
        "deadline lewat → is_overdue true dengan status tetap running (ADR-0012)",
        overdue_row is not None and overdue_row["is_overdue"] is True
        and overdue_row["status"] == "running",
        f"is_overdue {overdue_row['is_overdue'] if overdue_row else '-'}, "
        f"status {overdue_row['status'] if overdue_row else '-'}",
    )

    status, bogus_status = call("GET", "/workflows/instances?status=pending", tokens["manager"])
    check(
        "status di luar kosakata ditolak 422 pada field status",
        status == 422 and "status" in (field_of(bogus_status) or []),
        f"HTTP {status}, details {field_of(bogus_status)}",
    )

    status, bogus_scope = call("GET", "/workflows/instances?scope=mine", tokens["manager"])
    check(
        "scope tidak dikenal ditolak 422, bukan diabaikan",
        status == 422 and "scope" in (field_of(bogus_scope) or []),
        f"HTTP {status}, details {field_of(bogus_scope)}",
    )

    # --- Bersih-bersih ---

    cleanup()
    remaining = psql(
        "SELECT count(*) FROM workflow_instances wi JOIN documents d ON d.id = wi.document_id "
        f"JOIN projects p ON p.id = d.project_id WHERE p.code = '{PROJECT_CODE}'"
    )
    trailing = psql(
        "SELECT count(*) FROM workflow_definitions WHERE name LIKE 'Probe WF%'"
    )
    audit_after = int(psql("SELECT count(*) FROM audit_logs"))
    check(
        "bersih-bersih: tidak ada instance/definisi tersisa dan audit kembali ke baseline",
        remaining == "0" and trailing == "0" and audit_after == BASELINE_AUDIT_ROWS,
        f"instance {remaining}, definisi {trailing}, audit {audit_after}/{BASELINE_AUDIT_ROWS}",
    )

    passed = sum(1 for line in RESULTS if line.startswith("PASS"))
    failed = len(RESULTS) - passed
    print(f"\n{passed}/{len(RESULTS)} asersi PASS, {failed} FAIL", flush=True)
    return 1 if failed else 0


def upload_version(token: str, document_id: str, filename: str):
    """Mengunggah versi berkas lewat `POST /documents/:id/upload` (multipart)."""
    boundary = "----probe-workflow-boundary"
    content = b"%PDF-1.4 berkas uji modul workflow\n"
    body = b"".join([
        f"--{boundary}\r\n".encode(),
        f'Content-Disposition: form-data; name="file"; filename="{filename}"\r\n'.encode(),
        b"Content-Type: application/pdf\r\n\r\n",
        content,
        f"\r\n--{boundary}--\r\n".encode(),
    ])
    request = urllib.request.Request(
        BASE + f"/documents/{document_id}/upload",
        data=body,
        headers={
            "Content-Type": f"multipart/form-data; boundary={boundary}",
            "Authorization": f"Bearer {token}",
            "Accept": "application/json",
        },
        method="POST",
    )
    try:
        with urllib.request.urlopen(request) as response:
            return response.status, json.loads(response.read().decode() or "{}")
    except urllib.error.HTTPError as error:
        raw = error.read().decode()
        try:
            return error.code, json.loads(raw or "{}")
        except json.JSONDecodeError:
            return error.code, {"raw": raw}


if __name__ == "__main__":
    sys.exit(main())
