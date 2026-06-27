from pathlib import Path

from cli_tool.db.connection import get_db


def _load_schema() -> str:
    schema_path = Path(__file__).parent / "schema.sql"
    return schema_path.read_text()


def _get_current_version(conn) -> int:
    table_exists = conn.execute(
        "SELECT name FROM sqlite_master WHERE type='table' AND name='user_setting'"
    ).fetchone()
    if not table_exists:
        return 0
    row = conn.execute(
        "SELECT value FROM user_setting WHERE key = 'db_version'"
    ).fetchone()
    return int(row["value"]) if row else 0


def run_migrations() -> None:
    conn = get_db()
    schema_sql = _load_schema()
    _get_current_version(conn)
    conn.executescript(schema_sql)
    conn.commit()
