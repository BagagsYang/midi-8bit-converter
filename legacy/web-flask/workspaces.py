import hashlib
import json
import os
import secrets
import shutil
import sqlite3
import time
import uuid
from contextlib import contextmanager
from dataclasses import dataclass

import synthesis_jobs


WORKSPACE_COOKIE_MAX_LENGTH = 256


@dataclass(frozen=True)
class WorkspaceContext:
    id: int
    token: str
    created: bool = False


class WorkspaceError(Exception):
    def __init__(self, code, message, status_code=400):
        super().__init__(message)
        self.code = code
        self.message = message
        self.status_code = status_code


class WorkspaceService:
    def __init__(
        self,
        job_root,
        workspace_ttl_seconds,
        max_queued_files,
        max_upload_bytes,
        max_converted_files,
        default_config,
        run_inline=False,
        logger=None,
        max_workers=synthesis_jobs.DEFAULT_RENDER_WORKERS,
        max_queue_size=synthesis_jobs.DEFAULT_RENDER_QUEUE_SIZE,
    ):
        self.job_root = job_root
        self.workspace_ttl_seconds = workspace_ttl_seconds
        self.max_queued_files = max_queued_files
        self.max_upload_bytes = max_upload_bytes
        self.max_converted_files = max_converted_files
        self.default_config = default_config
        self.run_inline = run_inline
        self.logger = logger
        self.max_workers = max_workers
        self.max_queue_size = max_queue_size
        self.db_path = os.path.join(self.job_root, "workspaces.sqlite3")
        self.workspace_root = os.path.join(self.job_root, "workspaces")
        self._ensure_schema()

    @contextmanager
    def connect(self):
        os.makedirs(self.job_root, exist_ok=True)
        connection = sqlite3.connect(self.db_path, timeout=5.0)
        connection.row_factory = sqlite3.Row
        connection.execute("PRAGMA foreign_keys=ON")
        connection.execute("PRAGMA busy_timeout=5000")
        connection.execute("PRAGMA journal_mode=WAL")
        try:
            yield connection
            connection.commit()
        except Exception:
            connection.rollback()
            raise
        finally:
            connection.close()

    def _ensure_schema(self):
        with self.connect() as connection:
            connection.executescript(
                """
                CREATE TABLE IF NOT EXISTS workspaces (
                    id INTEGER PRIMARY KEY AUTOINCREMENT,
                    token_hash TEXT NOT NULL UNIQUE,
                    created_at REAL NOT NULL,
                    updated_at REAL NOT NULL,
                    expires_at REAL NOT NULL,
                    config_json TEXT NOT NULL
                );

                CREATE TABLE IF NOT EXISTS uploads (
                    id INTEGER PRIMARY KEY AUTOINCREMENT,
                    workspace_id INTEGER NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
                    file_id TEXT NOT NULL UNIQUE,
                    original_name TEXT NOT NULL,
                    size_bytes INTEGER NOT NULL,
                    queue_position INTEGER NOT NULL,
                    created_at REAL NOT NULL
                );

                CREATE INDEX IF NOT EXISTS idx_uploads_workspace_position
                    ON uploads(workspace_id, queue_position);

                CREATE TABLE IF NOT EXISTS jobs (
                    id INTEGER PRIMARY KEY AUTOINCREMENT,
                    workspace_id INTEGER NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
                    upload_id INTEGER REFERENCES uploads(id) ON DELETE SET NULL,
                    job_id TEXT NOT NULL UNIQUE,
                    status TEXT NOT NULL,
                    source_name TEXT NOT NULL,
                    download_name TEXT,
                    size_bytes INTEGER,
                    error TEXT,
                    created_at REAL NOT NULL,
                    updated_at REAL NOT NULL,
                    expires_at REAL NOT NULL,
                    config_json TEXT NOT NULL
                );

                CREATE INDEX IF NOT EXISTS idx_jobs_workspace_created
                    ON jobs(workspace_id, created_at);
                """
            )

    def token_hash(self, token):
        return hashlib.sha256(token.encode("utf-8")).hexdigest()

    def is_plausible_token(self, token):
        return (
            isinstance(token, str)
            and 16 <= len(token) <= WORKSPACE_COOKIE_MAX_LENGTH
        )

    def workspace_dir(self, workspace_id):
        return os.path.join(self.workspace_root, str(workspace_id))

    def upload_path(self, workspace_id, file_id):
        return os.path.join(self.workspace_dir(workspace_id), "uploads", f"{file_id}.mid")

    def job_dir(self, workspace_id, job_id):
        return os.path.join(self.workspace_dir(workspace_id), "jobs", job_id)

    def job_input_path(self, workspace_id, job_id):
        return os.path.join(self.job_dir(workspace_id, job_id), "input.mid")

    def job_output_path(self, workspace_id, job_id):
        return os.path.join(self.job_dir(workspace_id, job_id), "output.wav")

    def cleanup_expired_workspaces(self):
        now = time.time()
        expired_ids = []
        with self.connect() as connection:
            rows = connection.execute(
                "SELECT id FROM workspaces WHERE expires_at <= ?",
                (now,),
            ).fetchall()
            expired_ids = [row["id"] for row in rows]
            for workspace_id in expired_ids:
                connection.execute("DELETE FROM workspaces WHERE id = ?", (workspace_id,))

        for workspace_id in expired_ids:
            shutil.rmtree(self.workspace_dir(workspace_id), ignore_errors=True)

    def cleanup_expired_jobs(self, workspace):
        now = time.time()
        expired_job_ids = []
        with self.connect() as connection:
            rows = connection.execute(
                "SELECT job_id FROM jobs WHERE workspace_id = ? AND expires_at <= ?",
                (workspace.id, now),
            ).fetchall()
            expired_job_ids = [row["job_id"] for row in rows]
            connection.execute(
                "DELETE FROM jobs WHERE workspace_id = ? AND expires_at <= ?",
                (workspace.id, now),
            )

        for job_id in expired_job_ids:
            shutil.rmtree(self.job_dir(workspace.id, job_id), ignore_errors=True)

    def get_or_create_workspace(self, token):
        self.cleanup_expired_workspaces()
        workspace = self.get_active_workspace(token)
        if workspace is not None:
            return workspace
        return self.create_workspace()

    def get_active_workspace(self, token):
        if not self.is_plausible_token(token):
            return None

        now = time.time()
        token_hash = self.token_hash(token)
        with self.connect() as connection:
            row = connection.execute(
                "SELECT * FROM workspaces WHERE token_hash = ?",
                (token_hash,),
            ).fetchone()
            if row is None:
                return None
            if row["expires_at"] <= now:
                connection.execute("DELETE FROM workspaces WHERE id = ?", (row["id"],))
                workspace_id = row["id"]
            else:
                connection.execute(
                    """
                    UPDATE workspaces
                    SET updated_at = ?, expires_at = ?
                    WHERE id = ?
                    """,
                    (now, now + self.workspace_ttl_seconds, row["id"]),
                )
                return WorkspaceContext(id=row["id"], token=token, created=False)

        shutil.rmtree(self.workspace_dir(workspace_id), ignore_errors=True)
        return None

    def create_workspace(self):
        now = time.time()
        token = secrets.token_urlsafe(32)
        config_json = self._canonical_json(self.default_config)
        with self.connect() as connection:
            cursor = connection.execute(
                """
                INSERT INTO workspaces (
                    token_hash,
                    created_at,
                    updated_at,
                    expires_at,
                    config_json
                )
                VALUES (?, ?, ?, ?, ?)
                """,
                (
                    self.token_hash(token),
                    now,
                    now,
                    now + self.workspace_ttl_seconds,
                    config_json,
                ),
            )
            workspace_id = cursor.lastrowid
        os.makedirs(self.workspace_dir(workspace_id), exist_ok=True)
        return WorkspaceContext(id=workspace_id, token=token, created=True)

    def state_payload(self, workspace):
        self.cleanup_expired_jobs(workspace)
        with self.connect() as connection:
            workspace_row = connection.execute(
                "SELECT * FROM workspaces WHERE id = ?",
                (workspace.id,),
            ).fetchone()
            upload_rows = connection.execute(
                """
                SELECT file_id, original_name, size_bytes, queue_position, created_at
                FROM uploads
                WHERE workspace_id = ?
                ORDER BY queue_position ASC, created_at ASC
                """,
                (workspace.id,),
            ).fetchall()
            job_rows = connection.execute(
                """
                SELECT *
                FROM jobs
                WHERE workspace_id = ? AND status = 'ready' AND expires_at > ?
                ORDER BY updated_at DESC
                """,
                (workspace.id, time.time()),
            ).fetchall()

        return {
            "workspace": {
                "expires_at": workspace_row["expires_at"],
            },
            "limits": self.limits_payload(),
            "config": json.loads(workspace_row["config_json"]),
            "uploads": [self.upload_payload(row) for row in upload_rows],
            "converted_files": [self.converted_payload(row) for row in job_rows],
        }

    def limits_payload(self):
        return {
            "max_queued_files": self.max_queued_files,
            "max_upload_bytes": self.max_upload_bytes,
            "max_converted_files": self.max_converted_files,
        }

    def upload_payload(self, row):
        return {
            "file_id": row["file_id"],
            "name": row["original_name"],
            "size": row["size_bytes"],
            "created_at": row["created_at"],
        }

    def converted_payload(self, row):
        return {
            "job_id": row["job_id"],
            "name": row["download_name"] or "output.wav",
            "source_name": row["source_name"],
            "size": row["size_bytes"] or 0,
            "download_url": f"/api/synthesis-jobs/{row['job_id']}/download",
            "delete_url": f"/api/synthesis-jobs/{row['job_id']}",
            "created_at": row["created_at"],
            "updated_at": row["updated_at"],
            "expires_at": row["expires_at"],
        }

    def create_upload_from_temp(self, workspace, source_path, original_name):
        size_bytes = os.path.getsize(source_path)
        file_id = uuid.uuid4().hex
        now = time.time()
        destination = self.upload_path(workspace.id, file_id)

        with self.connect() as connection:
            count_row = connection.execute(
                "SELECT COUNT(*) AS count, COALESCE(SUM(size_bytes), 0) AS bytes FROM uploads WHERE workspace_id = ?",
                (workspace.id,),
            ).fetchone()
            if count_row["count"] >= self.max_queued_files:
                raise WorkspaceError(
                    "workspace_queue_limit",
                    "This temporary workspace has reached the queued file limit.",
                    409,
                )
            if count_row["bytes"] + size_bytes > self.max_upload_bytes:
                raise WorkspaceError(
                    "workspace_upload_bytes_limit",
                    "This temporary workspace has reached the upload storage limit.",
                    413,
                )
            position_row = connection.execute(
                "SELECT COALESCE(MAX(queue_position), -1) + 1 AS position FROM uploads WHERE workspace_id = ?",
                (workspace.id,),
            ).fetchone()
            connection.execute(
                """
                INSERT INTO uploads (
                    workspace_id,
                    file_id,
                    original_name,
                    size_bytes,
                    queue_position,
                    created_at
                )
                VALUES (?, ?, ?, ?, ?, ?)
                """,
                (
                    workspace.id,
                    file_id,
                    os.path.basename(original_name or "upload.mid"),
                    size_bytes,
                    position_row["position"],
                    now,
                ),
            )

        try:
            os.makedirs(os.path.dirname(destination), exist_ok=True)
            shutil.move(source_path, destination)
        except Exception:
            self.delete_upload(workspace, file_id)
            raise

        row = self.get_upload(workspace, file_id)
        return self.upload_payload(row)

    def get_upload(self, workspace, file_id):
        with self.connect() as connection:
            return connection.execute(
                """
                SELECT *
                FROM uploads
                WHERE workspace_id = ? AND file_id = ?
                """,
                (workspace.id, file_id),
            ).fetchone()

    def delete_upload(self, workspace, file_id):
        with self.connect() as connection:
            row = connection.execute(
                """
                SELECT *
                FROM uploads
                WHERE workspace_id = ? AND file_id = ?
                """,
                (workspace.id, file_id),
            ).fetchone()
            if row is None:
                return False
            connection.execute("DELETE FROM uploads WHERE id = ?", (row["id"],))
            connection.execute(
                """
                UPDATE uploads
                SET queue_position = queue_position - 1
                WHERE workspace_id = ? AND queue_position > ?
                """,
                (workspace.id, row["queue_position"]),
            )

        try:
            os.unlink(self.upload_path(workspace.id, file_id))
        except FileNotFoundError:
            pass
        return True

    def replace_queue(self, workspace, file_ids):
        with self.connect() as connection:
            rows = connection.execute(
                """
                SELECT file_id
                FROM uploads
                WHERE workspace_id = ?
                """,
                (workspace.id,),
            ).fetchall()
            current_ids = [row["file_id"] for row in rows]
            if sorted(current_ids) != sorted(file_ids) or len(set(file_ids)) != len(file_ids):
                raise WorkspaceError(
                    "invalid_queue",
                    "Queue order must contain each active file id exactly once.",
                    422,
                )
            for position, file_id in enumerate(file_ids):
                connection.execute(
                    """
                    UPDATE uploads
                    SET queue_position = ?
                    WHERE workspace_id = ? AND file_id = ?
                    """,
                    (position, workspace.id, file_id),
                )

        return self.state_payload(workspace)["uploads"]

    def save_config(self, workspace, config):
        config_json = self._canonical_json(config)
        now = time.time()
        with self.connect() as connection:
            connection.execute(
                """
                UPDATE workspaces
                SET config_json = ?, updated_at = ?, expires_at = ?
                WHERE id = ?
                """,
                (config_json, now, now + self.workspace_ttl_seconds, workspace.id),
            )
        return config

    def prepare_job(self, workspace, source_path, source_name, config, upload_row=None):
        self.cleanup_expired_jobs(workspace)
        job_id = uuid.uuid4().hex
        now = time.time()
        config_json = self._canonical_json(config)
        input_path = self.job_input_path(workspace.id, job_id)

        with self.connect() as connection:
            count_row = connection.execute(
                """
                SELECT COUNT(*) AS count
                FROM jobs
                WHERE workspace_id = ? AND status IN ('queued', 'rendering', 'ready')
                """,
                (workspace.id,),
            ).fetchone()
            if count_row["count"] >= self.max_converted_files:
                raise WorkspaceError(
                    "workspace_converted_limit",
                    "This temporary workspace has reached the converted file limit.",
                    409,
                )
            connection.execute(
                """
                INSERT INTO jobs (
                    workspace_id,
                    upload_id,
                    job_id,
                    status,
                    source_name,
                    created_at,
                    updated_at,
                    expires_at,
                    config_json
                )
                VALUES (?, ?, ?, 'queued', ?, ?, ?, ?, ?)
                """,
                (
                    workspace.id,
                    upload_row["id"] if upload_row is not None else None,
                    job_id,
                    os.path.basename(source_name or "upload.mid"),
                    now,
                    now,
                    now + self.workspace_ttl_seconds,
                    config_json,
                ),
            )

        try:
            os.makedirs(os.path.dirname(input_path), exist_ok=True)
            shutil.copyfile(source_path, input_path)
        except Exception:
            self.delete_job(workspace, job_id)
            raise

        return job_id, input_path

    def start_job(self, workspace, job_id, input_path, form_payload, source_name, render_uploaded_wav):
        if self.run_inline:
            self.run_job(workspace.id, job_id, input_path, form_payload, source_name, render_uploaded_wav)
            return

        executor = synthesis_jobs.get_render_executor(self.max_workers, self.max_queue_size)
        executor.submit(
            self.run_job,
            workspace.id,
            job_id,
            input_path,
            form_payload,
            source_name,
            render_uploaded_wav,
        )

    def run_job(self, workspace_id, job_id, input_path, form_payload, source_name, render_uploaded_wav):
        output_path = self.job_output_path(workspace_id, job_id)
        self.update_job(job_id, status="rendering")
        try:
            class SavedUpload:
                filename = source_name

                def save(self, destination):
                    shutil.copyfile(input_path, destination)

            download_name = render_uploaded_wav(SavedUpload(), form_payload, output_path)
            self.update_job(
                job_id,
                status="ready",
                download_name=download_name,
                size_bytes=os.path.getsize(output_path),
                expires_at=time.time() + self.workspace_ttl_seconds,
            )
        except Exception as exc:
            if self.logger is not None:
                if isinstance(exc, (EOFError, ValueError)):
                    self.logger.warning("Workspace synthesis job %s failed: %s", job_id, exc)
                else:
                    self.logger.exception("Workspace synthesis job %s failed", job_id)
            self.update_job(
                job_id,
                status="failed",
                error=synthesis_jobs.job_error_message(exc),
                expires_at=time.time() + self.workspace_ttl_seconds,
            )
        finally:
            try:
                os.unlink(input_path)
            except FileNotFoundError:
                pass

    def update_job(self, job_id, **updates):
        if not updates:
            return
        updates["updated_at"] = time.time()
        assignments = ", ".join(f"{key} = ?" for key in updates)
        values = list(updates.values())
        values.append(job_id)
        with self.connect() as connection:
            connection.execute(
                f"UPDATE jobs SET {assignments} WHERE job_id = ?",
                values,
            )

    def get_job(self, workspace, job_id):
        with self.connect() as connection:
            row = connection.execute(
                """
                SELECT *
                FROM jobs
                WHERE workspace_id = ? AND job_id = ?
                """,
                (workspace.id, job_id),
            ).fetchone()

        if row is None:
            return None, False
        if row["expires_at"] <= time.time():
            self.delete_job(workspace, job_id)
            return None, True
        return row, False

    def delete_job(self, workspace, job_id):
        with self.connect() as connection:
            row = connection.execute(
                """
                SELECT *
                FROM jobs
                WHERE workspace_id = ? AND job_id = ?
                """,
                (workspace.id, job_id),
            ).fetchone()
            if row is None:
                return False
            connection.execute("DELETE FROM jobs WHERE id = ?", (row["id"],))

        shutil.rmtree(self.job_dir(workspace.id, job_id), ignore_errors=True)
        return True

    def job_payload(self, row):
        payload = {
            "job_id": row["job_id"],
            "status": row["status"],
            "source_name": row["source_name"],
            "created_at": row["created_at"],
            "updated_at": row["updated_at"],
            "expires_at": row["expires_at"],
        }
        for key in ("download_name", "size_bytes", "error"):
            if row[key] is not None:
                payload[key] = row[key]
        if row["status"] == "ready":
            payload["download_url"] = f"/api/synthesis-jobs/{row['job_id']}/download"
            payload["delete_url"] = f"/api/synthesis-jobs/{row['job_id']}"
        return payload

    def _canonical_json(self, payload):
        return json.dumps(payload, sort_keys=True, separators=(",", ":"))
