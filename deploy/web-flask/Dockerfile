FROM python:3.12-slim@sha256:401f6e1a67dad31a1bd78e9ad22d0ee0a3b52154e6bd30e90be696bb6a3d7461

ENV PYTHONDONTWRITEBYTECODE=1 \
    PYTHONUNBUFFERED=1 \
    PORT=8000 \
    WEB_CONCURRENCY=2 \
    GUNICORN_TIMEOUT=600 \
    WEB_RENDER_WORKERS=2 \
    WEB_RENDER_QUEUE_SIZE=8 \
    WEB_DOWNLOAD_TTL_SECONDS=1800 \
    WEB_WORKSPACE_TTL_SECONDS=86400 \
    WEB_WORKSPACE_MAX_QUEUED_FILES=20 \
    WEB_WORKSPACE_MAX_UPLOAD_BYTES=104857600 \
    WEB_WORKSPACE_MAX_CONVERTED_FILES=20 \
    WEB_SYNTHESISE_JOB_ROOT=/tmp/octabit-jobs

WORKDIR /app

RUN groupadd --system app \
    && useradd --system --gid app --home-dir /app --shell /usr/sbin/nologin app

COPY deploy/web-flask/build-requirements.lock ./deploy/web-flask/build-requirements.lock
COPY deploy/web-flask/requirements.lock ./deploy/web-flask/requirements.lock

RUN python -m pip install --no-cache-dir --require-hashes -r deploy/web-flask/build-requirements.lock \
    && python -m pip install --no-cache-dir --no-build-isolation --require-hashes -r deploy/web-flask/requirements.lock

COPY legacy/web-flask ./legacy/web-flask
COPY legacy/python-renderer ./legacy/python-renderer
COPY assets/previews ./assets/previews
COPY LICENSE.md ./LICENSE.md

RUN chown -R app:app /app

USER app

EXPOSE 8000

HEALTHCHECK --interval=30s --timeout=5s --start-period=20s --retries=3 CMD python -c "import os, urllib.request; urllib.request.urlopen('http://127.0.0.1:%s/' % os.environ.get('PORT', '8000'), timeout=5).read(1)"

CMD ["sh", "-c", "gunicorn --chdir /app/legacy/web-flask --bind 0.0.0.0:${PORT:-8000} --workers ${WEB_CONCURRENCY:-2} --timeout ${GUNICORN_TIMEOUT:-600} --access-logfile - --error-logfile - app:app"]
