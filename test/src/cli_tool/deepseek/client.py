from __future__ import annotations

import os
from typing import Any, Dict, Generator, List, Optional

import httpx

from .exceptions import (
    DeepseekAPIError,
    DeepseekAuthError,
    DeepseekConnectionError,
    DeepseekRateLimitError,
    DeepseekTimeoutError,
)
from .types import (
    ChatCompletion,
    ChatCompletionChoice,
    ChatCompletionMessage,
    ChatCompletionUsage,
)


class DeepseekClient:
    """Client for the Deepseek Chat Completion API.

    The API is compatible with the OpenAI SDK format. You can pass this client
    as the ``http_client`` to the official ``openai`` package for seamless
    integration.

    Usage with the OpenAI SDK::

        from openai import OpenAI
        from cli_tool.deepseek import DeepseekClient

        client = DeepseekClient(api_key="sk-...")
        openai_client = OpenAI(
            base_url=client.base_url,
            api_key=client.api_key,
            http_client=client._http_client,
        )
    """

    BASE_URL = "https://api.deepseek.com"
    CHAT_COMPLETIONS_PATH = "/chat/completions"

    def __init__(
        self,
        *,
        api_key: Optional[str] = None,
        base_url: Optional[str] = None,
        timeout: float = 600.0,
        max_retries: int = 3,
    ):
        self._api_key = api_key or os.environ.get("DEEPSEEK_API_KEY")
        if not self._api_key:
            raise DeepseekAuthError(
                "API key is required. Set DEEPSEEK_API_KEY environment variable "
                "or pass api_key= to DeepseekClient()."
            )

        self.base_url = (base_url or self.BASE_URL).rstrip("/")
        self.timeout = timeout
        self.max_retries = max_retries

        self._http_client = httpx.Client(
            base_url=self.base_url,
            timeout=httpx.Timeout(timeout, connect=10.0),
            headers={
                "Authorization": f"Bearer {self._api_key}",
                "Content-Type": "application/json",
            },
        )

    @property
    def api_key(self) -> str:
        return self._api_key

    def close(self) -> None:
        self._http_client.close()

    def __enter__(self) -> "DeepseekClient":
        return self

    def __exit__(self, *args: Any) -> None:
        self.close()

    # ------------------------------------------------------------------
    # Chat Completions
    # ------------------------------------------------------------------

    def chat_completion(
        self,
        *,
        model: str,
        messages: List[Dict[str, Any]],
        stream: bool = False,
        temperature: Optional[float] = None,
        top_p: Optional[float] = None,
        max_tokens: Optional[int] = None,
        stop: Optional[List[str]] = None,
        frequency_penalty: Optional[float] = None,
        presence_penalty: Optional[float] = None,
        tools: Optional[List[Dict[str, Any]]] = None,
        tool_choice: Optional[str] = None,
        **kwargs: Any,
    ) -> ChatCompletion:
        """Send a chat completion request (non-streaming)."""
        payload = _build_payload(
            model=model,
            messages=messages,
            stream=False,
            temperature=temperature,
            top_p=top_p,
            max_tokens=max_tokens,
            stop=stop,
            frequency_penalty=frequency_penalty,
            presence_penalty=presence_penalty,
            tools=tools,
            tool_choice=tool_choice,
            extra=kwargs,
        )
        response_data = self._request("POST", self.CHAT_COMPLETIONS_PATH, json=payload)
        return _parse_chat_completion(response_data)

    def chat_completion_stream(
        self,
        *,
        model: str,
        messages: List[Dict[str, Any]],
        temperature: Optional[float] = None,
        top_p: Optional[float] = None,
        max_tokens: Optional[int] = None,
        stop: Optional[List[str]] = None,
        frequency_penalty: Optional[float] = None,
        presence_penalty: Optional[float] = None,
        tools: Optional[List[Dict[str, Any]]] = None,
        tool_choice: Optional[str] = None,
        **kwargs: Any,
    ) -> Generator[ChatCompletion, None, None]:
        """Send a chat completion request with streaming enabled.

        Yields ``ChatCompletion`` objects for each streamed chunk.
        """
        payload = _build_payload(
            model=model,
            messages=messages,
            stream=True,
            temperature=temperature,
            top_p=top_p,
            max_tokens=max_tokens,
            stop=stop,
            frequency_penalty=frequency_penalty,
            presence_penalty=presence_penalty,
            tools=tools,
            tool_choice=tool_choice,
            extra=kwargs,
        )
        with self._http_client.stream(
            "POST",
            self.CHAT_COMPLETIONS_PATH,
            json=payload,
            timeout=self.timeout,
        ) as response:
            _raise_on_error(response)
            for line in response.iter_lines():
                line = line.strip()
                if not line or not line.startswith("data: "):
                    continue
                data = line[len("data: "):]
                if data == "[DONE]":
                    return
                try:
                    import json
                    yield _parse_chat_completion(json.loads(data))
                except (ValueError, KeyError, TypeError):
                    continue

    # ------------------------------------------------------------------
    # Internal
    # ------------------------------------------------------------------

    def _request(
        self,
        method: str,
        path: str,
        *,
        json: Optional[Dict[str, Any]] = None,
    ) -> Dict[str, Any]:
        attempts = self.max_retries + 1
        last_exc: Optional[Exception] = None

        for attempt in range(attempts):
            try:
                response = self._http_client.request(method, path, json=json)
                _raise_on_error(response)
                return response.json()
            except httpx.TimeoutException as exc:
                last_exc = DeepseekTimeoutError(
                    f"Request timed out after {self.timeout}s: {exc}"
                )
                if attempt == self.max_retries:
                    raise last_exc
            except httpx.ConnectError as exc:
                raise DeepseekConnectionError(
                    f"Failed to connect to {self.base_url}: {exc}"
                ) from exc
            except (DeepseekAPIError, DeepseekRateLimitError) as exc:
                if not _is_retryable(exc.status_code) or attempt == self.max_retries:
                    raise
                last_exc = exc

        raise last_exc  # type: ignore[misc]


def _build_payload(
    *,
    model: str,
    messages: List[Dict[str, Any]],
    stream: bool,
    temperature: Optional[float],
    top_p: Optional[float],
    max_tokens: Optional[int],
    stop: Optional[List[str]],
    frequency_penalty: Optional[float],
    presence_penalty: Optional[float],
    tools: Optional[List[Dict[str, Any]]],
    tool_choice: Optional[str],
    extra: Dict[str, Any],
) -> Dict[str, Any]:
    payload: Dict[str, Any] = {
        "model": model,
        "messages": messages,
        "stream": stream,
    }
    if temperature is not None:
        payload["temperature"] = temperature
    if top_p is not None:
        payload["top_p"] = top_p
    if max_tokens is not None:
        payload["max_tokens"] = max_tokens
    if stop is not None:
        payload["stop"] = stop
    if frequency_penalty is not None:
        payload["frequency_penalty"] = frequency_penalty
    if presence_penalty is not None:
        payload["presence_penalty"] = presence_penalty
    if tools is not None:
        payload["tools"] = tools
    if tool_choice is not None:
        payload["tool_choice"] = tool_choice
    payload.update(extra)
    return payload


def _raise_on_error(response: httpx.Response) -> None:
    if response.is_success:
        return

    status = response.status_code
    try:
        body = response.json()
    except Exception:
        body = {"error": {"message": response.text or ""}}

    error_data = body.get("error", {})
    message = error_data.get("message", response.reason_phrase or "Unknown error")

    if status == 401:
        raise DeepseekAuthError(message, status_code=status, body=body)
    if status == 429:
        raise DeepseekRateLimitError(message, status_code=status, body=body)
    raise DeepseekAPIError(message, status_code=status, body=body)


def _is_retryable(status_code: Optional[int]) -> bool:
    if status_code is None:
        return True
    return status_code >= 500 or status_code == 429


def _parse_chat_completion(data: Dict[str, Any]) -> ChatCompletion:
    choices = []
    for choice_data in data.get("choices", []):
        msg_data = choice_data.get("delta") or choice_data.get("message") or {}
        message = ChatCompletionMessage(
            role=msg_data.get("role", ""),
            content=msg_data.get("content", "") or "",
            tool_calls=msg_data.get("tool_calls"),
            tool_call_id=msg_data.get("tool_call_id"),
            name=msg_data.get("name"),
        )
        choices.append(
            ChatCompletionChoice(
                index=choice_data.get("index", 0),
                message=message,
                finish_reason=choice_data.get("finish_reason"),
                logprobs=choice_data.get("logprobs"),
            )
        )

    usage = None
    usage_data = data.get("usage")
    if usage_data:
        usage = ChatCompletionUsage(
            prompt_tokens=usage_data.get("prompt_tokens", 0),
            completion_tokens=usage_data.get("completion_tokens", 0),
            total_tokens=usage_data.get("total_tokens", 0),
        )

    return ChatCompletion(
        id=data.get("id", ""),
        object=data.get("object", ""),
        created=data.get("created", 0),
        model=data.get("model", ""),
        choices=choices,
        usage=usage,
        system_fingerprint=data.get("system_fingerprint"),
    )
