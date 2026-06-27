from unittest import mock

import httpx
import pytest

from cli_tool.deepseek.client import (
    DeepseekClient,
    _build_payload,
    _is_retryable,
    _parse_chat_completion,
    _raise_on_error,
)
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
    ChatCompletionUsage,
)

# ---------------------------------------------------------------------------
# Helpers for building mock responses
# ---------------------------------------------------------------------------


def _make_completion_response(
    content="Hello!", role="assistant", model="deepseek-chat"
):
    return {
        "id": "cmpl-abc123",
        "object": "chat.completion",
        "created": 1700000000,
        "model": model,
        "choices": [
            {
                "index": 0,
                "message": {"role": role, "content": content},
                "finish_reason": "stop",
            }
        ],
        "usage": {
            "prompt_tokens": 10,
            "completion_tokens": 20,
            "total_tokens": 30,
        },
    }


def _make_stream_chunk(content="Hello", index=0, finish_reason=None):
    return {
        "id": "cmpl-abc123",
        "object": "chat.completion.chunk",
        "created": 1700000000,
        "model": "deepseek-chat",
        "choices": [
            {
                "index": index,
                "delta": {"role": "assistant", "content": content},
                "finish_reason": finish_reason,
            }
        ],
    }


def _mock_response(status_code=200, json_data=None, text=""):
    resp = mock.MagicMock(spec=httpx.Response)
    resp.status_code = status_code
    resp.is_success = 200 <= status_code < 300
    resp.json.return_value = json_data or {}
    resp.text = text
    resp.reason_phrase = "OK"
    return resp


def _mock_stream_context(status_code=200, lines=None):
    """Return a mock that supports context manager protocol for streaming."""
    resp = mock.MagicMock(spec=httpx.Response)
    resp.status_code = status_code
    resp.is_success = 200 <= status_code < 300
    resp.iter_lines.return_value = lines or []
    ctx = mock.MagicMock()
    ctx.__enter__.return_value = resp
    ctx.__exit__.return_value = False
    return ctx


# ---------------------------------------------------------------------------
# Client initialisation
# ---------------------------------------------------------------------------


class TestClientInit:
    def test_init_with_api_key_param(self):
        client = DeepseekClient(api_key="sk-test")
        assert client.api_key == "sk-test"

    def test_init_with_env_var(self):
        with mock.patch.dict("os.environ", {"DEEPSEEK_API_KEY": "sk-env"}, clear=True):
            client = DeepseekClient()
            assert client.api_key == "sk-env"

    def test_init_missing_api_key_raises(self):
        with mock.patch.dict("os.environ", {}, clear=True):
            with pytest.raises(DeepseekAuthError, match="API key is required"):
                DeepseekClient()

    def test_init_with_custom_base_url(self):
        with mock.patch.dict("os.environ", {"DEEPSEEK_API_KEY": "sk"}, clear=True):
            client = DeepseekClient(api_key="sk", base_url="https://custom.api.com/v1")
            assert client.base_url == "https://custom.api.com/v1"

    def test_init_with_custom_timeout(self):
        with mock.patch.dict("os.environ", {"DEEPSEEK_API_KEY": "sk"}, clear=True):
            client = DeepseekClient(api_key="sk", timeout=30.0)
            assert client.timeout == 30.0

    def test_init_with_custom_max_retries(self):
        with mock.patch.dict("os.environ", {"DEEPSEEK_API_KEY": "sk"}, clear=True):
            client = DeepseekClient(api_key="sk", max_retries=5)
            assert client.max_retries == 5

    def test_close(self):
        with mock.patch.dict("os.environ", {"DEEPSEEK_API_KEY": "sk"}, clear=True):
            client = DeepseekClient(api_key="sk")
            client.close()  # should not raise

    def test_context_manager(self):
        with mock.patch.dict("os.environ", {"DEEPSEEK_API_KEY": "sk"}, clear=True):
            with DeepseekClient(api_key="sk") as client:
                assert client.api_key == "sk"
            # Exiting should close the underlying client


# ---------------------------------------------------------------------------
# Non-streaming chat_completion
# ---------------------------------------------------------------------------


class TestChatCompletion:
    def test_non_streaming_returns_completion(self):
        with mock.patch.dict("os.environ", {"DEEPSEEK_API_KEY": "sk"}, clear=True):
            client = DeepseekClient(api_key="sk")
            mock_resp = _mock_response(
                status_code=200,
                json_data=_make_completion_response(content="Hi there!"),
            )
            client._http_client.request = mock.MagicMock(return_value=mock_resp)

            result = client.chat_completion(
                model="deepseek-chat",
                messages=[{"role": "user", "content": "Hello"}],
            )

            assert isinstance(result, ChatCompletion)
            assert result.id == "cmpl-abc123"
            assert result.choices[0].message.content == "Hi there!"
            assert result.usage.prompt_tokens == 10
            assert result.usage.completion_tokens == 20
            assert result.usage.total_tokens == 30

    def test_passes_all_parameters(self):
        with mock.patch.dict("os.environ", {"DEEPSEEK_API_KEY": "sk"}, clear=True):
            client = DeepseekClient(api_key="sk")
            mock_resp = _mock_response(
                status_code=200,
                json_data=_make_completion_response(content="ok"),
            )
            client._http_client.request = mock.MagicMock(return_value=mock_resp)

            client.chat_completion(
                model="deepseek-chat",
                messages=[{"role": "user", "content": "Hi"}],
                temperature=0.5,
                top_p=0.9,
                max_tokens=1024,
                stop=["\n"],
                frequency_penalty=0.1,
                presence_penalty=0.2,
                tools=[{"type": "function", "function": {"name": "test"}}],
                tool_choice="auto",
            )

            call_args = client._http_client.request.call_args
            payload = call_args.kwargs["json"]
            assert payload["temperature"] == 0.5
            assert payload["top_p"] == 0.9
            assert payload["max_tokens"] == 1024
            assert payload["stop"] == ["\n"]
            assert payload["frequency_penalty"] == 0.1
            assert payload["presence_penalty"] == 0.2
            assert payload["tools"] == [
                {"type": "function", "function": {"name": "test"}}
            ]
            assert payload["tool_choice"] == "auto"
            assert payload["stream"] is False

    def test_none_parameters_are_omitted(self):
        with mock.patch.dict("os.environ", {"DEEPSEEK_API_KEY": "sk"}, clear=True):
            client = DeepseekClient(api_key="sk")
            mock_resp = _mock_response(
                status_code=200,
                json_data=_make_completion_response(content="ok"),
            )
            client._http_client.request = mock.MagicMock(return_value=mock_resp)

            client.chat_completion(
                model="deepseek-chat",
                messages=[{"role": "user", "content": "Hi"}],
            )

            call_args = client._http_client.request.call_args
            payload = call_args.kwargs["json"]
            assert "temperature" not in payload
            assert "top_p" not in payload
            assert "max_tokens" not in payload

    def test_defaults_to_https_api_deepseek_com(self):
        with mock.patch.dict("os.environ", {"DEEPSEEK_API_KEY": "sk"}, clear=True):
            client = DeepseekClient(api_key="sk")
            assert client.base_url == "https://api.deepseek.com"

    def test_empty_choices(self):
        with mock.patch.dict("os.environ", {"DEEPSEEK_API_KEY": "sk"}, clear=True):
            client = DeepseekClient(api_key="sk")
            json_data = {
                "id": "cmpl-abc",
                "object": "chat.completion",
                "created": 1,
                "model": "deepseek-chat",
                "choices": [],
            }
            mock_resp = _mock_response(status_code=200, json_data=json_data)
            client._http_client.request = mock.MagicMock(return_value=mock_resp)

            result = client.chat_completion(
                model="deepseek-chat",
                messages=[{"role": "user", "content": "Hi"}],
            )
            assert result.choices == []


# ---------------------------------------------------------------------------
# Streaming chat_completion_stream
# ---------------------------------------------------------------------------


class TestChatCompletionStream:
    def test_streaming_yields_chunks(self):
        with mock.patch.dict("os.environ", {"DEEPSEEK_API_KEY": "sk"}, clear=True):
            client = DeepseekClient(api_key="sk")
            lines = [
                'data: {"id":"1","object":"chat.completion.chunk","created":1,"model":"m","choices":[{"index":0,"delta":{"content":"Hello"},"finish_reason":null}]}',
                'data: {"id":"2","object":"chat.completion.chunk","created":1,"model":"m","choices":[{"index":0,"delta":{"content":" world"},"finish_reason":null}]}',
                "data: [DONE]",
            ]
            mock_stream = _mock_stream_context(status_code=200, lines=lines)

            client._http_client.stream = mock.MagicMock(return_value=mock_stream)

            chunks = list(
                client.chat_completion_stream(
                    model="deepseek-chat",
                    messages=[{"role": "user", "content": "Hi"}],
                )
            )

            assert len(chunks) == 2
            assert chunks[0].choices[0].message.content == "Hello"
            assert chunks[1].choices[0].message.content == " world"

    def test_streaming_skips_non_data_lines(self):
        with mock.patch.dict("os.environ", {"DEEPSEEK_API_KEY": "sk"}, clear=True):
            client = DeepseekClient(api_key="sk")
            lines = [
                "",
                'data: {"id":"1","object":"chat.completion.chunk","created":1,"model":"m","choices":[{"index":0,"delta":{"content":"Hi"},"finish_reason":null}]}',
                "data: [DONE]",
            ]
            mock_stream = _mock_stream_context(status_code=200, lines=lines)

            client._http_client.stream = mock.MagicMock(return_value=mock_stream)

            chunks = list(
                client.chat_completion_stream(
                    model="deepseek-chat",
                    messages=[{"role": "user", "content": "Hi"}],
                )
            )

            assert len(chunks) == 1
            assert chunks[0].choices[0].message.content == "Hi"

    def test_streaming_handles_invalid_json_gracefully(self):
        with mock.patch.dict("os.environ", {"DEEPSEEK_API_KEY": "sk"}, clear=True):
            client = DeepseekClient(api_key="sk")
            lines = [
                "data: not-valid-json",
                'data: {"id":"1","object":"chat.completion.chunk","created":1,"model":"m","choices":[{"index":0,"delta":{"content":"ok"},"finish_reason":"stop"}]}',
                "data: [DONE]",
            ]
            mock_stream = _mock_stream_context(status_code=200, lines=lines)

            client._http_client.stream = mock.MagicMock(return_value=mock_stream)

            chunks = list(
                client.chat_completion_stream(
                    model="deepseek-chat",
                    messages=[{"role": "user", "content": "Hi"}],
                )
            )

            assert len(chunks) == 1
            assert chunks[0].choices[0].message.content == "ok"

    def test_streaming_with_tool_calls_and_finish_reason(self):
        with mock.patch.dict("os.environ", {"DEEPSEEK_API_KEY": "sk"}, clear=True):
            client = DeepseekClient(api_key="sk")
            lines = [
                'data: {"id":"1","object":"chat.completion.chunk","created":1,"model":"m","choices":[{"index":0,"delta":{"content":""},"finish_reason":null}]}',
                'data: {"id":"1","object":"chat.completion.chunk","created":1,"model":"m","choices":[{"index":0,"delta":{"content":""},"finish_reason":"stop"}]}',
                "data: [DONE]",
            ]
            mock_stream = _mock_stream_context(status_code=200, lines=lines)

            client._http_client.stream = mock.MagicMock(return_value=mock_stream)

            chunks = list(
                client.chat_completion_stream(
                    model="deepseek-chat",
                    messages=[{"role": "user", "content": "Hi"}],
                )
            )

            assert len(chunks) == 2
            assert chunks[1].choices[0].finish_reason == "stop"


# ---------------------------------------------------------------------------
# Error handling
# ---------------------------------------------------------------------------


class TestErrorHandling:
    def test_401_raises_auth_error(self):
        with mock.patch.dict("os.environ", {"DEEPSEEK_API_KEY": "sk"}, clear=True):
            client = DeepseekClient(api_key="sk")
            mock_resp = _mock_response(
                status_code=401,
                json_data={"error": {"message": "Invalid API key"}},
            )
            client._http_client.request = mock.MagicMock(return_value=mock_resp)

            with pytest.raises(DeepseekAuthError, match="Invalid API key"):
                client.chat_completion(
                    model="deepseek-chat",
                    messages=[{"role": "user", "content": "Hi"}],
                )

    def test_429_raises_rate_limit_error(self):
        with mock.patch.dict("os.environ", {"DEEPSEEK_API_KEY": "sk"}, clear=True):
            client = DeepseekClient(api_key="sk")
            mock_resp = _mock_response(
                status_code=429,
                json_data={"error": {"message": "Too many requests"}},
            )
            client._http_client.request = mock.MagicMock(return_value=mock_resp)

            with pytest.raises(DeepseekRateLimitError, match="Too many requests"):
                client.chat_completion(
                    model="deepseek-chat",
                    messages=[{"role": "user", "content": "Hi"}],
                )

    def test_500_raises_api_error(self):
        with mock.patch.dict("os.environ", {"DEEPSEEK_API_KEY": "sk"}, clear=True):
            client = DeepseekClient(api_key="sk")
            mock_resp = _mock_response(
                status_code=500,
                json_data={"error": {"message": "Internal server error"}},
            )
            client._http_client.request = mock.MagicMock(return_value=mock_resp)

            with pytest.raises(DeepseekAPIError, match="Internal server error"):
                client.chat_completion(
                    model="deepseek-chat",
                    messages=[{"role": "user", "content": "Hi"}],
                )

    def test_400_raises_api_error(self):
        with mock.patch.dict("os.environ", {"DEEPSEEK_API_KEY": "sk"}, clear=True):
            client = DeepseekClient(api_key="sk")
            mock_resp = _mock_response(
                status_code=400,
                json_data={"error": {"message": "Bad request"}},
            )
            client._http_client.request = mock.MagicMock(return_value=mock_resp)

            with pytest.raises(DeepseekAPIError, match="Bad request"):
                client.chat_completion(
                    model="deepseek-chat",
                    messages=[{"role": "user", "content": "Hi"}],
                )

    def test_error_with_non_json_body(self):
        with mock.patch.dict("os.environ", {"DEEPSEEK_API_KEY": "sk"}, clear=True):
            client = DeepseekClient(api_key="sk")
            mock_resp = _mock_response(status_code=503, text="Service Unavailable")
            mock_resp.json.side_effect = ValueError("Not JSON")
            client._http_client.request = mock.MagicMock(return_value=mock_resp)

            with pytest.raises(DeepseekAPIError, match="Service Unavailable"):
                client.chat_completion(
                    model="deepseek-chat",
                    messages=[{"role": "user", "content": "Hi"}],
                )

    def test_connection_error(self):
        with mock.patch.dict("os.environ", {"DEEPSEEK_API_KEY": "sk"}, clear=True):
            client = DeepseekClient(api_key="sk")
            client._http_client.request = mock.MagicMock(
                side_effect=httpx.ConnectError("Connection refused")
            )

            with pytest.raises(
                DeepseekConnectionError, match="Failed to connect to"
            ):
                client.chat_completion(
                    model="deepseek-chat",
                    messages=[{"role": "user", "content": "Hi"}],
                )

    def test_timeout_error(self):
        with mock.patch.dict("os.environ", {"DEEPSEEK_API_KEY": "sk"}, clear=True):
            client = DeepseekClient(api_key="sk", max_retries=0)
            client._http_client.request = mock.MagicMock(
                side_effect=httpx.TimeoutException("timed out")
            )

            with pytest.raises(DeepseekTimeoutError, match="timed out"):
                client.chat_completion(
                    model="deepseek-chat",
                    messages=[{"role": "user", "content": "Hi"}],
                )

    def test_api_error_stores_status_code_and_body(self):
        with mock.patch.dict("os.environ", {"DEEPSEEK_API_KEY": "sk"}, clear=True):
            client = DeepseekClient(api_key="sk")
            body = {"error": {"message": "Bad request"}}
            mock_resp = _mock_response(status_code=400, json_data=body)
            client._http_client.request = mock.MagicMock(return_value=mock_resp)

            try:
                client.chat_completion(
                    model="deepseek-chat",
                    messages=[{"role": "user", "content": "Hi"}],
                )
            except DeepseekAPIError as exc:
                assert exc.status_code == 400
                assert exc.body == body


# ---------------------------------------------------------------------------
# Retry logic
# ---------------------------------------------------------------------------


class TestRetryLogic:
    def test_retries_on_429(self):
        with mock.patch.dict("os.environ", {"DEEPSEEK_API_KEY": "sk"}, clear=True):
            client = DeepseekClient(api_key="sk", max_retries=2)
            error_resp = _mock_response(
                status_code=429,
                json_data={"error": {"message": "Rate limited"}},
            )
            success_resp = _mock_response(
                status_code=200,
                json_data=_make_completion_response(content="after retry"),
            )
            client._http_client.request = mock.MagicMock(
                side_effect=[error_resp, error_resp, success_resp]
            )

            result = client.chat_completion(
                model="deepseek-chat",
                messages=[{"role": "user", "content": "Hi"}],
            )

            assert client._http_client.request.call_count == 3
            assert result.choices[0].message.content == "after retry"

    def test_retries_on_5xx(self):
        with mock.patch.dict("os.environ", {"DEEPSEEK_API_KEY": "sk"}, clear=True):
            client = DeepseekClient(api_key="sk", max_retries=2)
            error_resp = _mock_response(
                status_code=500,
                json_data={"error": {"message": "Server error"}},
            )
            success_resp = _mock_response(
                status_code=200,
                json_data=_make_completion_response(content="ok"),
            )
            client._http_client.request = mock.MagicMock(
                side_effect=[error_resp, error_resp, success_resp]
            )

            result = client.chat_completion(
                model="deepseek-chat",
                messages=[{"role": "user", "content": "Hi"}],
            )

            assert client._http_client.request.call_count == 3
            assert result.choices[0].message.content == "ok"

    def test_no_retry_on_401(self):
        with mock.patch.dict("os.environ", {"DEEPSEEK_API_KEY": "sk"}, clear=True):
            client = DeepseekClient(api_key="sk", max_retries=2)
            error_resp = _mock_response(
                status_code=401,
                json_data={"error": {"message": "Unauthorized"}},
            )
            client._http_client.request = mock.MagicMock(
                side_effect=[error_resp, error_resp]
            )

            with pytest.raises(DeepseekAuthError):
                client.chat_completion(
                    model="deepseek-chat",
                    messages=[{"role": "user", "content": "Hi"}],
                )

            assert client._http_client.request.call_count == 1

    def test_no_retry_on_400(self):
        with mock.patch.dict("os.environ", {"DEEPSEEK_API_KEY": "sk"}, clear=True):
            client = DeepseekClient(api_key="sk", max_retries=2)
            error_resp = _mock_response(
                status_code=400,
                json_data={"error": {"message": "Bad request"}},
            )
            client._http_client.request = mock.MagicMock(
                side_effect=[error_resp, error_resp]
            )

            with pytest.raises(DeepseekAPIError):
                client.chat_completion(
                    model="deepseek-chat",
                    messages=[{"role": "user", "content": "Hi"}],
                )

            assert client._http_client.request.call_count == 1

    def test_retry_exhausted_raises_last_error(self):
        with mock.patch.dict("os.environ", {"DEEPSEEK_API_KEY": "sk"}, clear=True):
            client = DeepseekClient(api_key="sk", max_retries=1)
            error_resp = _mock_response(
                status_code=500,
                json_data={"error": {"message": "Still broken"}},
            )
            client._http_client.request = mock.MagicMock(
                return_value=error_resp
            )

            with pytest.raises(DeepseekAPIError, match="Still broken"):
                client.chat_completion(
                    model="deepseek-chat",
                    messages=[{"role": "user", "content": "Hi"}],
                )

            assert client._http_client.request.call_count == 2


# ---------------------------------------------------------------------------
# Helper functions
# ---------------------------------------------------------------------------


class TestBuildPayload:
    def test_minimal_payload(self):
        payload = _build_payload(
            model="deepseek-chat",
            messages=[{"role": "user", "content": "Hi"}],
            stream=False,
            temperature=None,
            top_p=None,
            max_tokens=None,
            stop=None,
            frequency_penalty=None,
            presence_penalty=None,
            tools=None,
            tool_choice=None,
            extra={},
        )
        assert payload == {
            "model": "deepseek-chat",
            "messages": [{"role": "user", "content": "Hi"}],
            "stream": False,
        }

    def test_full_payload(self):
        payload = _build_payload(
            model="deepseek-chat",
            messages=[{"role": "user", "content": "Hi"}],
            stream=True,
            temperature=0.5,
            top_p=0.9,
            max_tokens=1024,
            stop=["\n"],
            frequency_penalty=0.1,
            presence_penalty=0.2,
            tools=[{"type": "function", "function": {"name": "test"}}],
            tool_choice="auto",
            extra={"custom": "value"},
        )
        assert payload["temperature"] == 0.5
        assert payload["top_p"] == 0.9
        assert payload["max_tokens"] == 1024
        assert payload["stop"] == ["\n"]
        assert payload["frequency_penalty"] == 0.1
        assert payload["presence_penalty"] == 0.2
        assert payload["tools"] == [{"type": "function", "function": {"name": "test"}}]
        assert payload["tool_choice"] == "auto"
        assert payload["custom"] == "value"

    def test_extra_params_merged(self):
        payload = _build_payload(
            model="deepseek-chat",
            messages=[{"role": "user", "content": "Hi"}],
            stream=False,
            temperature=None,
            top_p=None,
            max_tokens=None,
            stop=None,
            frequency_penalty=None,
            presence_penalty=None,
            tools=None,
            tool_choice=None,
            extra={"response_format": {"type": "json_object"}},
        )
        assert payload["response_format"] == {"type": "json_object"}


class TestParseChatCompletion:
    def test_parse_basic(self):
        data = {
            "id": "cmpl-1",
            "object": "chat.completion",
            "created": 1700000000,
            "model": "deepseek-chat",
            "choices": [
                {
                    "index": 0,
                    "message": {"role": "assistant", "content": "Hello!"},
                    "finish_reason": "stop",
                }
            ],
        }
        result = _parse_chat_completion(data)
        assert result.id == "cmpl-1"
        assert result.object == "chat.completion"
        assert result.model == "deepseek-chat"
        assert result.choices[0].index == 0
        assert result.choices[0].message.role == "assistant"
        assert result.choices[0].message.content == "Hello!"
        assert result.choices[0].finish_reason == "stop"
        assert result.usage is None

    def test_parse_with_usage(self):
        data = {
            "id": "cmpl-1",
            "object": "chat.completion",
            "created": 1,
            "model": "m",
            "choices": [],
            "usage": {
                "prompt_tokens": 10,
                "completion_tokens": 20,
                "total_tokens": 30,
            },
        }
        result = _parse_chat_completion(data)
        assert result.usage.prompt_tokens == 10
        assert result.usage.completion_tokens == 20
        assert result.usage.total_tokens == 30

    def test_parse_streaming_chunk_with_delta(self):
        data = {
            "id": "cmpl-1",
            "object": "chat.completion.chunk",
            "created": 1,
            "model": "m",
            "choices": [
                {
                    "index": 0,
                    "delta": {"content": "Hi"},
                    "finish_reason": None,
                }
            ],
        }
        result = _parse_chat_completion(data)
        assert result.choices[0].message.content == "Hi"

    def test_parse_with_system_fingerprint(self):
        data = {
            "id": "cmpl-1",
            "object": "chat.completion",
            "created": 1,
            "model": "m",
            "choices": [],
            "system_fingerprint": "fp_abc123",
        }
        result = _parse_chat_completion(data)
        assert result.system_fingerprint == "fp_abc123"

    def test_parse_empty_content_is_empty_string(self):
        data = {
            "id": "cmpl-1",
            "object": "chat.completion",
            "created": 1,
            "model": "m",
            "choices": [
                {
                    "index": 0,
                    "message": {"role": "assistant"},
                    "finish_reason": "stop",
                }
            ],
        }
        result = _parse_chat_completion(data)
        assert result.choices[0].message.content == ""


class TestRaiseOnError:
    def test_success_status_no_raise(self):
        resp = _mock_response(status_code=200, json_data={})
        _raise_on_error(resp)  # should not raise

    def test_401_raises_auth_error(self):
        resp = _mock_response(
            status_code=401, json_data={"error": {"message": "Unauthorized"}}
        )
        with pytest.raises(DeepseekAuthError):
            _raise_on_error(resp)

    def test_429_raises_rate_limit_error(self):
        resp = _mock_response(
            status_code=429, json_data={"error": {"message": "Too many"}}
        )
        with pytest.raises(DeepseekRateLimitError):
            _raise_on_error(resp)

    def test_unknown_error_raises_api_error(self):
        resp = _mock_response(
            status_code=418, json_data={"error": {"message": "I'm a teapot"}}
        )
        with pytest.raises(DeepseekAPIError):
            _raise_on_error(resp)


class TestIsRetryable:
    def test_500_is_retryable(self):
        assert _is_retryable(500) is True

    def test_502_is_retryable(self):
        assert _is_retryable(502) is True

    def test_503_is_retryable(self):
        assert _is_retryable(503) is True

    def test_429_is_retryable(self):
        assert _is_retryable(429) is True

    def test_401_not_retryable(self):
        assert _is_retryable(401) is False

    def test_400_not_retryable(self):
        assert _is_retryable(400) is False

    def test_none_is_retryable(self):
        assert _is_retryable(None) is True
