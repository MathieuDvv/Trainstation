import sqlite3
from pathlib import Path

_db_path: Path | None = None
_connection: sqlite3.Connection | None = None


def set_db_path(path: Path) -> None:
    global _db_path
    _db_path = path


def get_db_path() -> Path:
    global _db_path
    if _db_path is None:
        _db_path = Path.home() / ".trainstation" / "trainstation.db"
    return _db_path


def get_db() -> sqlite3.Connection:
    global _connection
    if _connection is None:
        db_path = get_db_path()
        db_path.parent.mkdir(parents=True, exist_ok=True)
        _connection = sqlite3.connect(str(db_path))
        _connection.row_factory = sqlite3.Row
        _connection.execute("PRAGMA journal_mode = WAL")
        _connection.execute("PRAGMA foreign_keys = ON")
    return _connection


def close_db() -> None:
    global _connection
    if _connection is not None:
        _connection.close()
        _connection = None


def reset_db() -> None:
    global _connection
    close_db()
    db_path = get_db_path()
    if db_path.exists():
        db_path.unlink()
