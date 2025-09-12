"""
Chat completion providers for LLM interactions.
"""

import logging
import os
from abc import ABC, abstractmethod
from typing import List, Dict, Any, Optional

logger = logging.getLogger(__name__)


class ChatProvider(ABC):
    """Abstract base class for chat completion providers."""
    
    @abstractmethod
    def chat_completion(
        self, 
        messages: List[Dict[str, str]], 
        temperature: float = 0.7,
        max_tokens: Optional[int] = None
    ) -> str:
        """
        Generate a chat completion.
        
        Args:
            messages: List of message dicts with 'role' and 'content'
            temperature: Sampling temperature (0-1)
            max_tokens: Maximum tokens to generate
            
        Returns:
            Generated response text
        """
        pass
    
    @abstractmethod
    def is_available(self) -> bool:
        """Check if the provider is available and configured."""
        pass


class OpenAIChatProvider(ChatProvider):
    """OpenAI chat completion provider."""
    
    def __init__(self, api_key: Optional[str] = None, model: str = "gpt-4"):
        """
        Initialize OpenAI chat provider.
        
        Args:
            api_key: OpenAI API key (optional, can use env var)
            model: Model to use (default: gpt-4)
        """
        self.api_key = api_key or os.environ.get("OPENAI_API_KEY")
        self.model = model
        self._client = None
        
        if not self.api_key:
            logger.warning("OpenAI API key not found")
    
    def _get_client(self):
        """Get or create OpenAI client."""
        if self._client is None:
            try:
                from openai import OpenAI
                self._client = OpenAI(api_key=self.api_key)
            except ImportError:
                raise ImportError("OpenAI library required: pip install openai")
        return self._client
    
    def chat_completion(
        self, 
        messages: List[Dict[str, str]], 
        temperature: float = 0.7,
        max_tokens: Optional[int] = None
    ) -> str:
        """Generate chat completion using OpenAI."""
        client = self._get_client()
        
        try:
            response = client.chat.completions.create(
                model=self.model,
                messages=messages,
                temperature=temperature,
                max_tokens=max_tokens
            )
            return response.choices[0].message.content
        except Exception as e:
            logger.error(f"OpenAI chat completion failed: {e}")
            raise
    
    def is_available(self) -> bool:
        """Check if OpenAI is available."""
        if not self.api_key:
            return False
        
        try:
            from openai import OpenAI
            return True
        except ImportError:
            return False


class AnthropicChatProvider(ChatProvider):
    """Anthropic Claude chat completion provider."""
    
    def __init__(self, api_key: Optional[str] = None, model: str = "claude-3-opus-20240229"):
        """
        Initialize Anthropic chat provider.
        
        Args:
            api_key: Anthropic API key (optional, can use env var)
            model: Model to use (default: claude-3-opus)
        """
        self.api_key = api_key or os.environ.get("ANTHROPIC_API_KEY")
        self.model = model
        self._client = None
        
        if not self.api_key:
            logger.warning("Anthropic API key not found")
    
    def _get_client(self):
        """Get or create Anthropic client."""
        if self._client is None:
            try:
                from anthropic import Anthropic
                self._client = Anthropic(api_key=self.api_key)
            except ImportError:
                raise ImportError("Anthropic library required: pip install anthropic")
        return self._client
    
    def chat_completion(
        self, 
        messages: List[Dict[str, str]], 
        temperature: float = 0.7,
        max_tokens: Optional[int] = None
    ) -> str:
        """Generate chat completion using Anthropic."""
        client = self._get_client()
        
        # Convert messages to Anthropic format
        system_message = None
        user_messages = []
        
        for msg in messages:
            if msg["role"] == "system":
                # Ensure system message is a string
                content = msg["content"]
                system_message = content if isinstance(content, str) else str(content)
            else:
                # Ensure user/assistant messages have proper format
                user_messages.append({
                    "role": msg["role"],
                    "content": msg["content"] if isinstance(msg["content"], str) else str(msg["content"])
                })
        
        try:
            response = client.messages.create(
                model=self.model,
                messages=user_messages,
                system=system_message,
                temperature=temperature,
                max_tokens=max_tokens or 4096
            )
            return response.content[0].text
        except Exception as e:
            logger.error(f"Anthropic chat completion failed: {e}")
            raise
    
    def is_available(self) -> bool:
        """Check if Anthropic is available."""
        if not self.api_key:
            return False
        
        try:
            from anthropic import Anthropic
            return True
        except ImportError:
            return False


class OllamaChatProvider(ChatProvider):
    """Ollama local model chat completion provider."""
    
    def __init__(self, model: str = "llama2", base_url: str = "http://localhost:11434"):
        """
        Initialize Ollama chat provider.
        
        Args:
            model: Model to use (default: llama2)
            base_url: Ollama API URL (default: http://localhost:11434)
        """
        self.model = model
        self.base_url = base_url
    
    def chat_completion(
        self, 
        messages: List[Dict[str, str]], 
        temperature: float = 0.7,
        max_tokens: Optional[int] = None
    ) -> str:
        """Generate chat completion using Ollama."""
        import requests
        
        # Convert messages to Ollama format
        prompt = ""
        for msg in messages:
            role = msg["role"]
            content = msg["content"]
            if role == "system":
                prompt += f"System: {content}\n\n"
            elif role == "user":
                prompt += f"User: {content}\n\n"
            elif role == "assistant":
                prompt += f"Assistant: {content}\n\n"
        
        prompt += "Assistant: "
        
        try:
            response = requests.post(
                f"{self.base_url}/api/generate",
                json={
                    "model": self.model,
                    "prompt": prompt,
                    "temperature": temperature,
                    "stream": False
                }
            )
            response.raise_for_status()
            return response.json()["response"]
        except Exception as e:
            logger.error(f"Ollama chat completion failed: {e}")
            raise
    
    def is_available(self) -> bool:
        """Check if Ollama is available."""
        import requests
        try:
            response = requests.get(f"{self.base_url}/api/tags")
            return response.status_code == 200
        except:
            return False


class MockChatProvider(ChatProvider):
    """Mock chat provider for testing."""
    
    def __init__(self, responses: Optional[List[str]] = None):
        """Initialize mock provider with predefined responses."""
        self.responses = responses or [
            "I understand you want to create an ontology. Let's start with the domain.",
            "Financial documents often contain entities like companies, people, and metrics.",
            "Based on your description, I suggest extracting revenue, profit, and growth metrics."
        ]
        self.response_index = 0
    
    def chat_completion(
        self, 
        messages: List[Dict[str, str]], 
        temperature: float = 0.7,
        max_tokens: Optional[int] = None
    ) -> str:
        """Return mock response."""
        if self.response_index < len(self.responses):
            response = self.responses[self.response_index]
            self.response_index += 1
            return response
        return "I can help you refine the ontology further. What specific aspects would you like to focus on?"
    
    def is_available(self) -> bool:
        """Mock provider is always available."""
        return True


def create_chat_provider(
    provider: str = "auto", 
    model: Optional[str] = None,
    **kwargs
) -> ChatProvider:
    """
    Create a chat provider instance.
    
    Args:
        provider: Provider name (openai, anthropic, ollama, mock, auto)
        model: Specific model to use
        **kwargs: Additional provider-specific arguments
        
    Returns:
        ChatProvider instance
    """
    if provider == "auto":
        # Auto-detect available provider - prioritize Anthropic if ANTHROPIC_API_KEY exists
        providers = [
            ("anthropic", AnthropicChatProvider),
            ("openai", OpenAIChatProvider),
            ("ollama", OllamaChatProvider)
        ]
        
        for name, provider_class in providers:
            try:
                instance = provider_class(model=model) if model else provider_class()
                if instance.is_available():
                    logger.info(f"Auto-selected {name} as chat provider")
                    return instance
            except Exception as e:
                logger.debug(f"Provider {name} not available: {e}")
                continue
        
        # Fall back to mock if no real provider available
        logger.warning("No LLM provider available, using mock provider")
        return MockChatProvider()
    
    elif provider == "openai":
        return OpenAIChatProvider(model=model or "gpt-4", **kwargs)
    
    elif provider == "anthropic":
        return AnthropicChatProvider(model=model or "claude-3-opus-20240229", **kwargs)
    
    elif provider == "ollama":
        return OllamaChatProvider(model=model or "llama2", **kwargs)
    
    elif provider == "mock":
        return MockChatProvider(**kwargs)
    
    else:
        raise ValueError(f"Unknown provider: {provider}")