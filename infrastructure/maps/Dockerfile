FROM python:3.12.10-alpine3.21@sha256:9c51ecce261773a684c8345b2d4673700055c513b4d54bc0719337d3e4ee552e
RUN pip install --no-cache-dir gunicorn==23.0.0 packaging==24.2
WORKDIR /app
COPY --chmod=644 server.py /app/server.py
USER 65534:65534
EXPOSE 8000
CMD ["gunicorn", "--bind", "0.0.0.0:8000", "--workers", "2", "--threads", "4", "--timeout", "20", "--access-logfile", "-", "server:application"]
