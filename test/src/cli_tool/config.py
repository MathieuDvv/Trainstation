import os
from dataclasses import dataclass, field
from pathlib import Path

try:
    import yaml
except ImportError:
    yaml = None


DEFAULT_MODEL = "deepseek-chat"
DEFAULT_TEMPERATURE = 0.7
DEFAULT_MAX_TOKENS = 4096

ENV_API_KEY = "DEEPSEEK_API_KEY"


def _config_paths():
    return [
        Path("~/.deepseek.yaml").expanduser().resolve(),
        Path("~/.config/deepseek/config.yaml").expanduser().resolve(),
    ]


@dataclass
class Config:
    api_key: str | None = None
    model: str = DEFAULT_MODEL
    temperature: float = DEFAULT_TEMPERATURE
    max_tokens: int = DEFAULT_MAX_TOKENS

    _loaded: bool = field(default=False, repr=False)

    def _load_from_file(self, path: Path) -> dict | None:
        if not path.is_file():
            return None
        if yaml is None:
            return None
        try:
            with open(path) as f:
                data = yaml.safe_load(f)
            return data if isinstance(data, dict) else None
        except Exception:
            return None

    def load(self) -> None:
        if self._loaded:
            return

        env_key = os.environ.get(ENV_API_KEY)
        if env_key is not None:
            self.api_key = env_key

        for cfg_path in _config_paths():
            data = self._load_from_file(cfg_path)
            if data is None:
                continue
            if self.api_key is None:
                self.api_key = data.get("api_key")
            if "model" in data:
                self.model = data["model"]
            if "temperature" in data:
                self.temperature = float(data["temperature"])
            if "max_tokens" in data:
                self.max_tokens = int(data["max_tokens"])
            break

        self._loaded = True

    @property
    def is_configured(self) -> bool:
        return self.api_key is not None


_config: Config | None = None


def get_config() -> Config:
    global _config
    if _config is None:
        _config = Config()
        _config.load()
    return _config
