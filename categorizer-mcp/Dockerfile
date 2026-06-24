# Portable image for the categorizer MCP service.
# Stateless: it talks to an external Postgres and Ollama, so the image carries no
# data — configure everything through env vars (see .env.example).
FROM python:3.12-slim

ENV PYTHONUNBUFFERED=1 \
    PIP_NO_CACHE_DIR=1 \
    PIP_DISABLE_PIP_VERSION_CHECK=1 \
    # Listen on all interfaces by default so the container is reachable; override if needed.
    MCP_HOST=0.0.0.0 \
    MCP_PORT=8090

WORKDIR /app

# Install deps via the project metadata so the dependency set stays single-sourced.
COPY pyproject.toml README.md ./
COPY src ./src
RUN pip install .

# Drop privileges.
RUN useradd --create-home --uid 10001 app && chown -R app /app
USER app

EXPOSE 8090

# Lightweight liveness probe against the REST health route.
HEALTHCHECK --interval=30s --timeout=5s --start-period=20s --retries=3 \
    CMD python -c "import os,urllib.request,sys; \
url=f'http://127.0.0.1:{os.getenv(\"MCP_PORT\",\"8090\")}/health'; \
sys.exit(0 if urllib.request.urlopen(url, timeout=4).status==200 else 1)" || exit 1

CMD ["categorizer-mcp"]
