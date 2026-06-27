import os
from unittest import mock

import pytest
from click.testing import CliRunner

from cli_tool.cli import _get_client, main
from cli_tool.deepseek.exceptions import (
    DeepseekAPIError,
    DeepseekAuthError,
    DeepseekConnectionError,
    DeepseekRateLimitError,
    DeepseekTimeoutError,
)
from cli_tool.deepseek.types import (
    ChatCompletion,
    ChatCompletionChoice,
    ChatCompletionMessage,
)


def _make_completion(content="Hello from Deepseek!", model="deepseek-chat"):
    return ChatCompletion(
        id="cmpl-abc",
        object="chat.completion",
        created=1,
        model=model,
        choices=[
            ChatCompletionChoice(
                index=0,
                message=ChatCompletionMessage(role="assistant", content=content),
                finish_reason="stop",
            )
        ],
    )


def _make_stream_chunks(*contents):
    for i, content in enumerate(contents):
        yield ChatCompletion(
            id="cmpl-abc",
            object="chat.completion.chunk",
            created=1,
            model="deepseek-chat",
            choices=[
                ChatCompletionChoice(
                    index=0,
                    message=ChatCompletionMessage(role="assistant", content=content),
                    finish_reason="stop" if i == len(contents) - 1 else None,
                )
            ],
        )


# ---------------------------------------------------------------------------
# Basic CLI
# ---------------------------------------------------------------------------


class TestBasicCLI:
    def test_hello_default(self):
        runner = CliRunner()
        result = runner.invoke(main, ["hello"])
        assert result.exit_code == 0
        assert "Hello, World!" in result.output

    def test_hello_with_name(self):
        runner = CliRunner()
        result = runner.invoke(main, ["hello", "Alice"])
        assert result.exit_code == 0
        assert "Hello, Alice!" in result.output

    def test_version(self):
        runner = CliRunner()
        result = runner.invoke(main, ["--version"])
        assert result.exit_code == 0
        assert "version" in result.output


# ---------------------------------------------------------------------------
# Deepseek chat command
# ---------------------------------------------------------------------------


class TestDeepseekChat:
    def test_get_client_with_key(self):
        with mock.patch.dict(os.environ, {"DEEPSEEK_API_KEY": "sk-test"}, clear=True):
            client = _get_client()
            assert client.api_key == "sk-test"

    def test_get_client_missing_key(self):
        with mock.patch.dict(os.environ, {}, clear=True):
            with pytest.raises(Exception):  # click.UsageError
                _get_client()

    def test_chat_non_streaming(self):
        completion = _make_completion(content="Hello, World!")
        with mock.patch.dict(os.environ, {"DEEPSEEK_API_KEY": "sk-test"}, clear=True):
            runner = CliRunner()
            with mock.patch(
                "cli_tool.cli.DeepseekClient.chat_completion",
                return_value=completion,
            ):
                result = runner.invoke(
                    main, ["deepseek", "chat", "--no-stream", "Say hi"]
                )
                assert result.exit_code == 0
                assert "Hello, World!" in result.output

    def test_chat_streaming(self):
        chunks = list(_make_stream_chunks("Hello", " world", "!"))
        with mock.patch.dict(os.environ, {"DEEPSEEK_API_KEY": "sk-test"}, clear=True):
            runner = CliRunner()
            with mock.patch(
                "cli_tool.cli.DeepseekClient.chat_completion_stream",
                return_value=chunks,
            ):
                result = runner.invoke(main, ["deepseek", "chat", "Say hi"])
                assert result.exit_code == 0
                assert "Hello world!" in result.output

    def test_chat_streaming_empty_deltas_filtered(self):
        chunks = list(_make_stream_chunks("", "Hello", ""))
        with mock.patch.dict(os.environ, {"DEEPSEEK_API_KEY": "sk-test"}, clear=True):
            runner = CliRunner()
            with mock.patch(
                "cli_tool.cli.DeepseekClient.chat_completion_stream",
                return_value=chunks,
            ):
                result = runner.invoke(main, ["deepseek", "chat", "test"])
                assert result.exit_code == 0
                assert "Hello" in result.output

    def test_chat_missing_api_key(self):
        with mock.patch.dict(os.environ, {}, clear=True):
            runner = CliRunner()
            result = runner.invoke(main, ["deepseek", "chat", "Hello"])
            assert result.exit_code != 0
            assert "DEEPSEEK_API_KEY" in result.output

    def test_chat_with_system_prompt(self):
        completion = _make_completion(content="ok")
        with mock.patch.dict(os.environ, {"DEEPSEEK_API_KEY": "sk-test"}, clear=True):
            runner = CliRunner()
            with mock.patch(
                "cli_tool.cli.DeepseekClient.chat_completion",
                return_value=completion,
            ) as mock_chat:
                runner.invoke(
                    main,
                    [
                        "deepseek",
                        "chat",
                        "--no-stream",
                        "-s",
                        "You are helpful",
                        "Hello",
                    ],
                )
                messages = mock_chat.call_args.kwargs["messages"]
                assert messages[0]["role"] == "system"
                assert messages[0]["content"] == "You are helpful"

    def test_chat_with_temperature(self):
        completion = _make_completion(content="ok")
        with mock.patch.dict(os.environ, {"DEEPSEEK_API_KEY": "sk-test"}, clear=True):
            runner = CliRunner()
            with mock.patch(
                "cli_tool.cli.DeepseekClient.chat_completion",
                return_value=completion,
            ) as mock_chat:
                runner.invoke(
                    main,
                    [
                        "deepseek",
                        "chat",
                        "--no-stream",
                        "-t",
                        "0.5",
                        "Hello",
                    ],
                )
                assert mock_chat.call_args.kwargs["temperature"] == 0.5

    def test_chat_default_model(self):
        completion = _make_completion(content="ok")
        with mock.patch.dict(os.environ, {"DEEPSEEK_API_KEY": "sk-test"}, clear=True):
            runner = CliRunner()
            with mock.patch(
                "cli_tool.cli.DeepseekClient.chat_completion",
                return_value=completion,
            ) as mock_chat:
                runner.invoke(main, ["deepseek", "chat", "--no-stream", "Hello"])
                assert mock_chat.call_args.kwargs["model"] == "deepseek-chat"

    def test_chat_custom_model(self):
        completion = _make_completion(content="ok")
        with mock.patch.dict(os.environ, {"DEEPSEEK_API_KEY": "sk-test"}, clear=True):
            runner = CliRunner()
            with mock.patch(
                "cli_tool.cli.DeepseekClient.chat_completion",
                return_value=completion,
            ) as mock_chat:
                runner.invoke(
                    main,
                    ["deepseek", "chat", "--no-stream", "-m", "deepseek-coder", "Hi"],
                )
                assert mock_chat.call_args.kwargs["model"] == "deepseek-coder"


# ---------------------------------------------------------------------------
# Error handling in deepseek chat command
# ---------------------------------------------------------------------------


class TestDeepseekChatErrors:
    def _invoke_chat(self, exc_to_raise, expected_output):
        with mock.patch.dict(os.environ, {"DEEPSEEK_API_KEY": "sk-test"}, clear=True):
            runner = CliRunner()
            with mock.patch(
                "cli_tool.cli.DeepseekClient.chat_completion",
                side_effect=exc_to_raise,
            ):
                result = runner.invoke(
                    main, ["deepseek", "chat", "--no-stream", "Hello"]
                )
                assert result.exit_code == 1
                assert expected_output in result.output

    def test_auth_error(self):
        self._invoke_chat(
            DeepseekAuthError("Invalid key", status_code=401, body={}),
            "Invalid API key",
        )

    def test_rate_limit_error(self):
        self._invoke_chat(
            DeepseekRateLimitError("Too many", status_code=429, body={}),
            "Rate limit exceeded",
        )

    def test_timeout_error(self):
        self._invoke_chat(
            DeepseekTimeoutError("timed out"),
            "timed out",
        )

    def test_connection_error(self):
        self._invoke_chat(
            DeepseekConnectionError("Failed"),
            "Could not connect",
        )

    def test_generic_api_error(self):
        self._invoke_chat(
            DeepseekAPIError("Bad request", status_code=400, body={}),
            "Error (400)",
        )


# ---------------------------------------------------------------------------
# Chat TUI command (launches Textual app)
# ---------------------------------------------------------------------------


class TestChatTUICommand:
    def test_chat_launches_tui(self):
        with mock.patch.dict(os.environ, {"DEEPSEEK_API_KEY": "sk"}, clear=True):
            runner = CliRunner()
            with mock.patch("cli_tool.tui.ChatApp") as MockApp:
                mock_instance = mock.MagicMock()
                MockApp.return_value = mock_instance

                result = runner.invoke(main, ["chat"])
                MockApp.assert_called_once()
                mock_instance.run.assert_called_once()
