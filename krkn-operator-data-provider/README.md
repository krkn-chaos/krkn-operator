# krkn-operator-data-provider

gRPC Python service that provides data from Kubernetes clusters using krkn-lib.

## Overview

This service receives kubeconfig information and uses krkn-lib to interact with Kubernetes clusters, providing node lists and other cluster data to the krkn-operator.

## Setup

### Prerequisites

- Python 3.8+
- pip

### Installation

```bash
# Install dependencies
pip install -r requirements.txt

# Generate gRPC Python code from protobuf
./generate_proto.sh
```

## Running the Server

```bash
python3 server.py
```

The server will start on port `50051` by default.

## API

### GetNodes

Retrieves the list of nodes from a Kubernetes cluster.

**Request:**
- `kubeconfig_base64` (string): Kubeconfig in base64 format

**Response:**
- `nodes` (repeated string): List of node names

## Development

### Regenerating gRPC Code

If you modify the protobuf definition in `../proto/dataprovider.proto`, regenerate the Python code:

```bash
./generate_proto.sh
```

## Dependencies

- **grpcio**: gRPC Python library
- **grpcio-tools**: gRPC Python tools for code generation
- **krkn-lib**: Krkn library for Kubernetes interactions (from git branch init_from_string)