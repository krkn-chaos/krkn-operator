# Use Python 3.11 slim image
FROM docker.io/library/python:3.11-slim

WORKDIR /app

# Install system dependencies
RUN apt-get update && apt-get install -y --no-install-recommends \
    gcc \
    git \
    && rm -rf /var/lib/apt/lists/*

# Copy requirements and install dependencies globally
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# Copy application code
COPY server.py .
COPY generated/ ./generated/

# Create non-root user for security
RUN useradd -r -u 1001 -m -s /sbin/nologin nonroot && \
    chown -R nonroot:nonroot /app

# Set environment variables
ENV PYTHONUNBUFFERED=1

# Switch to non-root user
USER 1001

# Expose gRPC port
EXPOSE 50051

# Run the server
CMD ["python", "server.py"]
