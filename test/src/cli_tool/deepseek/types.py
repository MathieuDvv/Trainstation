from dataclasses import dataclass
from typing import Any, Dict, List, Optional


@dataclass
class ChatCompletionMessage:
    role: str
    content: str
    tool_calls: Optional[List[Dict[str, Any]]] = None
    tool_call_id: Optional[str] = None
    name: Optional[str] = None


@dataclass
class ChatCompletionChoice:
    index: int
    message: ChatCompletionMessage
    finish_reason: Optional[str] = None
    logprobs: Optional[Any] = None


@dataclass
class ChatCompletionUsage:
    prompt_tokens: int
    completion_tokens: int
    total_tokens: int


@dataclass
class ChatCompletion:
    id: str
    object: str
    created: int
    model: str
    choices: List[ChatCompletionChoice]
    usage: Optional[ChatCompletionUsage] = None
    system_fingerprint: Optional[str] = None
