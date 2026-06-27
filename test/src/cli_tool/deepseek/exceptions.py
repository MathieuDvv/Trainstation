class DeepseekError(Exception):
    """Base exception for all Deepseek client errors."""


class DeepseekAPIError(DeepseekError):
    """Raised when the API returns an error response."""

    def __init__(self, message, status_code=None, body=None):
        super().__init__(message)
        self.status_code = status_code
        self.body = body


class DeepseekAuthError(DeepseekAPIError):
    """Raised on authentication failures (401)."""


class DeepseekRateLimitError(DeepseekAPIError):
    """Raised when rate-limited (429)."""


class DeepseekConnectionError(DeepseekError):
    """Raised when a network connection to the API fails."""


class DeepseekTimeoutError(DeepseekError):
    """Raised when a request times out."""
