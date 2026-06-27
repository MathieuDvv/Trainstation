import os
from unittest import mock

from cli_tool.config import (
    DEFAULT_MAX_TOKENS,
    DEFAULT_MODEL,
    DEFAULT_TEMPERATURE,
    ENV_API_KEY,
    Config,
    get_config,
)


class TestConfigDefaults:
    def test_defaults(self):
        with mock.patch.dict(os.environ, {}, clear=True):
            cfg = Config()
            cfg.load()
            assert cfg.api_key is None
            assert cfg.model == DEFAULT_MODEL
            assert cfg.temperature == DEFAULT_TEMPERATURE
            assert cfg.max_tokens == DEFAULT_MAX_TOKENS
            assert not cfg.is_configured

    def test_load_is_idempotent(self):
        with mock.patch.dict(os.environ, {}, clear=True):
            cfg = Config()
            cfg.load()
            assert cfg._loaded
            cfg.load()  # should not re-read


class TestEnvVar:
    def test_api_key_from_env(self):
        with mock.patch.dict(os.environ, {ENV_API_KEY: "sk-env-key"}, clear=True):
            cfg = Config()
            cfg.load()
            assert cfg.api_key == "sk-env-key"
            assert cfg.is_configured

    def test_other_settings_from_env(self):
        with mock.patch.dict(os.environ, {ENV_API_KEY: "sk-foo"}, clear=True):
            cfg = Config()
            cfg.load()
            assert cfg.model == DEFAULT_MODEL
            assert cfg.temperature == DEFAULT_TEMPERATURE


class TestConfigFile:
    def test_load_from_home_yaml(self, tmp_path):
        config_file = tmp_path / ".deepseek.yaml"
        config_file.write_text(
            "api_key: sk-file-key\n"
            "model: deepseek-v4-pro\n"
            "temperature: 0.3\n"
            "max_tokens: 2048\n"
        )
        with mock.patch.dict(os.environ, {}, clear=True):
            cfg = Config()
            with mock.patch(
                "cli_tool.config._config_paths",
                return_value=[config_file],
            ):
                cfg.load()
                assert cfg.api_key == "sk-file-key"
                assert cfg.model == "deepseek-v4-pro"
                assert cfg.temperature == 0.3
                assert cfg.max_tokens == 2048

    def test_env_overrides_file(self, tmp_path):
        config_file = tmp_path / ".deepseek.yaml"
        config_file.write_text("api_key: sk-file-key\n")
        with mock.patch.dict(os.environ, {ENV_API_KEY: "sk-env-key"}, clear=True):
            cfg = Config()
            with mock.patch(
                "cli_tool.config._config_paths",
                return_value=[config_file],
            ):
                cfg.load()
                assert cfg.api_key == "sk-env-key"

    def test_file_fills_missing_env(self, tmp_path):
        config_file = tmp_path / ".deepseek.yaml"
        config_file.write_text(
            "model: deepseek-v4-pro\ntemperature: 0.3\n"
        )
        with mock.patch.dict(os.environ, {}, clear=True):
            cfg = Config()
            with mock.patch(
                "cli_tool.config._config_paths",
                return_value=[config_file],
            ):
                cfg.load()
                assert cfg.api_key is None
                assert cfg.model == "deepseek-v4-pro"
                assert cfg.temperature == 0.3


class TestNosetup:
    def test_yaml_not_present(self):
        with mock.patch.dict(os.environ, {}, clear=True):
            with mock.patch("cli_tool.config.yaml", None):
                cfg = Config(api_key="sk-manual")
                cfg.load()
                assert cfg.api_key == "sk-manual"
                assert cfg.model == DEFAULT_MODEL


class TestGetConfigSingleton:
    def test_returns_same_instance(self):
        import cli_tool.config

        cli_tool.config._config = None
        with mock.patch.dict(os.environ, {}, clear=True):
            cfg1 = get_config()
            cfg2 = get_config()
            assert cfg1 is cfg2
