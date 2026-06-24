"""Categorizer MCP: an Ollama-backed product categorization microservice.

Mirrors the Go backend's categorizer (a fixed category set, a product-name reuse
cache, and Ollama structured outputs) but exposes it as MCP tools over HTTP so an
agent or the backend can categorize receipt products with a few small messages.
"""

__version__ = "0.1.0"
