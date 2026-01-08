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

# Set environment variables
ENV PYTHONUNBUFFERED=1

# Expose gRPC port
EXPOSE 50051

# Run the server
CMD ["python", "server.py"]
