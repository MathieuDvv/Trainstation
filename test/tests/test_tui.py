import os
from unittest import mock

import pytest

from cli_tool.config import Config
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


# ---------------------------------------------------------------------------
# ChatApp instantiation
# ---------------------------------------------------------------------------


class TestChatAppInit:
    def test_init_without_api_key_shows_error(self):
        with mock.patch.dict(os.environ, {}, clear=True):
            with mock.patch("cli_tool.tui.get_config") as mock_get_config:
                cfg = Config()
                mock_get_config.return_value = cfg

                from cli_tool.tui import ChatApp

                app = ChatApp()
                assert app.client is None
                assert app.messages == []

    def test_init_with_api_key_creates_client(self):
        with mock.patch.dict(
            os.environ, {"DEEPSEEK_API_KEY": "sk-test"}, clear=True
        ):
            with mock.patch("cli_tool.tui.get_config") as mock_get_config:
                cfg = Config(api_key="sk-test")
                cfg._loaded = True
                mock_get_config.return_value = cfg

                from cli_tool.tui import ChatApp

                with mock.patch(
                    "cli_tool.tui.DeepseekClient"
                ) as MockClient:
                    mock_client = mock.MagicMock()
                    MockClient.return_value = mock_client

                    app = ChatApp()
                    assert app.client is not None
                    MockClient.assert_called_once_with(api_key="sk-test")

    def test_config_read_correctly(self):
        with mock.patch.dict(os.environ, {}, clear=True):
            with mock.patch("cli_tool.tui.get_config") as mock_get_config:
                cfg = Config(
                    api_key="sk-cfg-test",
                    model="deepseek-v4-pro",
                    temperature=0.3,
                    max_tokens=2048,
                    _loaded=True,
                )
                mock_get_config.return_value = cfg

                from cli_tool.tui import ChatApp

                with mock.patch(
                    "cli_tool.tui.DeepseekClient"
                ) as MockClient:
                    MockClient.return_value = mock.MagicMock()

                    app = ChatApp()
                    assert app.config.model == "deepseek-v4-pro"
                    assert app.config.temperature == 0.3
                    assert app.config.max_tokens == 2048
                    assert app.config.api_key == "sk-cfg-test"


# ---------------------------------------------------------------------------
# ChatApp input handling
# ---------------------------------------------------------------------------


class TestChatAppInput:
    def test_input_submitted_no_client_does_nothing(self):
        with mock.patch.dict(os.environ, {}, clear=True):
            with mock.patch("cli_tool.tui.get_config") as mock_get_config:
                cfg = Config()
                mock_get_config.return_value = cfg

                from cli_tool.tui import ChatApp

                app = ChatApp()
                with mock.patch.object(app, "query_one") as mock_query:
                    event = mock.MagicMock()
                    event.value = "Hello"
                    app.on_input_submitted(event)
                    mock_query.assert_not_called()

    def test_input_submitted_empty_message(self):
        with mock.patch.dict(
            os.environ, {"DEEPSEEK_API_KEY": "sk-test"}, clear=True
        ):
            with mock.patch("cli_tool.tui.get_config") as mock_get_config:
                cfg = Config(api_key="sk-test", _loaded=True)
                mock_get_config.return_value = cfg

                from cli_tool.tui import ChatApp

                with mock.patch(
                    "cli_tool.tui.DeepseekClient"
                ) as MockClient:
                    MockClient.return_value = mock.MagicMock()

                    app = ChatApp()
                    with mock.patch.object(app, "query_one") as mock_query:
                        event = mock.MagicMock()
                        event.value = "   "
                        app.on_input_submitted(event)
                        mock_query.assert_not_called()

    def test_input_submitted_adds_user_message(self):
        with mock.patch.dict(
            os.environ, {"DEEPSEEK_API_KEY": "sk-test"}, clear=True
        ):
            with mock.patch("cli_tool.tui.get_config") as mock_get_config:
                cfg = Config(api_key="sk-test", _loaded=True)
                mock_get_config.return_value = cfg

                from cli_tool.tui import ChatApp

                with mock.patch(
                    "cli_tool.tui.DeepseekClient"
                ) as MockClient:
                    mock_client = mock.MagicMock()
                    MockClient.return_value = mock_client

                    app = ChatApp()
                    app.call_api = mock.MagicMock()  # bypass @work decorator

                    # Mock compose-time widgets
                    mock_history = mock.MagicMock()
                    mock_input = mock.MagicMock()

                    def fake_query_one(selector, *extra):
                        if "chat-history" in str(selector):
                            return mock_history
                        if "Input" in str(selector):
                            return mock_input
                        return mock.MagicMock()

                    with mock.patch.object(
                        app, "query_one", side_effect=fake_query_one
                    ):
                        event = mock.MagicMock()
                        event.value = "Hello world"
                        app.on_input_submitted(event)

                        # User message was stored
                        assert app.messages[0] == {
                            "role": "user",
                            "content": "Hello world",
                        }
                        # User message widget was mounted
                        mock_history.mount.assert_called()
                        # Input was cleared
                        assert mock_input.value == ""


# ---------------------------------------------------------------------------
# ChatApp error display
# ---------------------------------------------------------------------------


class TestChatAppShowError:
    def test_show_error_mounts_message(self):
        with mock.patch.dict(
            os.environ, {"DEEPSEEK_API_KEY": "sk-test"}, clear=True
        ):
            with mock.patch("cli_tool.tui.get_config") as mock_get_config:
                cfg = Config(api_key="sk-test", _loaded=True)
                mock_get_config.return_value = cfg

                from cli_tool.tui import ChatApp, ChatMessage

                with mock.patch(
                    "cli_tool.tui.DeepseekClient"
                ) as MockClient:
                    MockClient.return_value = mock.MagicMock()

                    app = ChatApp()

                    mock_history = mock.MagicMock()
                    mock_history.mount = mock.MagicMock()
                    mock_history.scroll_end = mock.MagicMock()

                    with mock.patch.object(
                        app, "query_one", return_value=mock_history
                    ):
                        app._show_error("Test error message")

                        mock_history.mount.assert_called_once()
                        args, _ = mock_history.mount.call_args
                        widget = args[0]
                        assert isinstance(widget, ChatMessage)
                        assert "error-message" in widget.classes
                        mock_history.scroll_end.assert_called_once_with(
                            animate=False
                        )


# ---------------------------------------------------------------------------
# Conversation history tracking
# ---------------------------------------------------------------------------


class TestConversationHistory:
    def test_messages_tracked_correctly(self):
        with mock.patch.dict(
            os.environ, {"DEEPSEEK_API_KEY": "sk-test"}, clear=True
        ):
            with mock.patch("cli_tool.tui.get_config") as mock_get_config:
                cfg = Config(api_key="sk-test", _loaded=True)
                mock_get_config.return_value = cfg

                from cli_tool.tui import ChatApp

                with mock.patch(
                    "cli_tool.tui.DeepseekClient"
                ) as MockClient:
                    mock_client = mock.MagicMock()
                    MockClient.return_value = mock_client

                    app = ChatApp()
                    app.call_api = mock.MagicMock()  # bypass @work decorator

                    # Simulate a conversation turn
                    app.messages = [
                        {"role": "user", "content": "Hi"},
                        {"role": "assistant", "content": "Hello!"},
                    ]

                    # Mock the query for input submission
                    mock_history = mock.MagicMock()
                    mock_input = mock.MagicMock()

                    def fake_query_one(selector, *extra):
                        if "chat-history" in str(selector):
                            return mock_history
                        if "Input" in str(selector):
                            return mock_input
                        return mock.MagicMock()

                    with mock.patch.object(
                        app, "query_one", side_effect=fake_query_one
                    ):
                        event = mock.MagicMock()
                        event.value = "How are you?"
                        app.on_input_submitted(event)

                        assert len(app.messages) == 3
                        assert app.messages[2] == {
                            "role": "user",
                            "content": "How are you?",
                        }

    def test_multiple_turns_build_history(self):
        with mock.patch.dict(
            os.environ, {"DEEPSEEK_API_KEY": "sk-test"}, clear=True
        ):
            with mock.patch("cli_tool.tui.get_config") as mock_get_config:
                cfg = Config(api_key="sk-test", _loaded=True)
                mock_get_config.return_value = cfg

                from cli_tool.tui import ChatApp

                with mock.patch(
                    "cli_tool.tui.DeepseekClient"
                ) as MockClient:
                    mock_client = mock.MagicMock()
                    MockClient.return_value = mock_client

                    app = ChatApp()
                    app.call_api = mock.MagicMock()  # bypass @work decorator

                    # Simulate multiple turns
                    app.messages = [
                        {"role": "user", "content": "Q1"},
                        {"role": "assistant", "content": "A1"},
                        {"role": "user", "content": "Q2"},
                        {"role": "assistant", "content": "A2"},
                    ]

                    mock_history = mock.MagicMock()
                    mock_input = mock.MagicMock()

                    def fake_query_one(selector, *extra):
                        if "chat-history" in str(selector):
                            return mock_history
                        if "Input" in str(selector):
                            return mock_input
                        return mock.MagicMock()

                    with mock.patch.object(
                        app, "query_one", side_effect=fake_query_one
                    ):
                        event = mock.MagicMock()
                        event.value = "Q3"
                        app.on_input_submitted(event)

                        assert len(app.messages) == 5
                        assert app.messages[4] == {
                            "role": "user",
                            "content": "Q3",
                        }
