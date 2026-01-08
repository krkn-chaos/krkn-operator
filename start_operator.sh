#!/bin/bash
# start_operator.sh
# Script to build and run the krkn-operator locally

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Default values
API_PORT=${API_PORT:-8080}
METRICS_ADDR=${METRICS_ADDR:-":8443"}
HEALTH_PROBE_ADDR=${HEALTH_PROBE_ADDR:-":8083"}
KRKN_NAMESPACE=${KRKN_NAMESPACE:-"default"}
INSTALL_CRDS=${INSTALL_CRDS:-true}
BUILD=${BUILD:-true}

# Function to print colored messages
print_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Function to check prerequisites
check_prerequisites() {
    print_info "Checking prerequisites..."

    # Check if kubectl is installed
    if ! command -v kubectl &> /dev/null; then
        print_error "kubectl not found. Please install kubectl first."
        exit 1
    fi

    # Check if kubectl can connect to a cluster
    if ! kubectl cluster-info &> /dev/null; then
        print_error "Cannot connect to Kubernetes cluster. Please check your kubeconfig."
        exit 1
    fi

    print_info "Cluster connection: OK"
    kubectl cluster-info | head -n 1
}

# Function to install CRDs
install_crds() {
    if [ "$INSTALL_CRDS" = "true" ]; then
        print_info "Installing CRDs..."
        make install
        print_info "CRDs installed successfully"
    else
        print_warning "Skipping CRD installation (INSTALL_CRDS=false)"
    fi
}

# Function to build the operator
build_operator() {
    if [ "$BUILD" = "true" ]; then
        print_info "Building operator..."
        make build
        print_info "Build completed successfully"
    else
        print_warning "Skipping build (BUILD=false)"
    fi
}

# Function to run the operator
run_operator() {
    print_info "Starting krkn-operator..."
    print_info "Configuration:"
    print_info "  API Port: $API_PORT"
    print_info "  Metrics Address: $METRICS_ADDR"
    print_info "  Health Probe Address: $HEALTH_PROBE_ADDR"
    print_info "  KrknTargetRequest Namespace: $KRKN_NAMESPACE"
    echo ""
    print_info "Press Ctrl+C to stop the operator"
    echo ""

    # Run the operator with custom flags and export KRKN_NAMESPACE
    export KRKN_NAMESPACE
    ./bin/manager \
        --api-port="$API_PORT" \
        --metrics-bind-address="$METRICS_ADDR" \
        --health-probe-bind-address="$HEALTH_PROBE_ADDR" \
        --metrics-secure=false
}

# Function to cleanup on exit
cleanup() {
    print_info "Shutting down operator..."
    exit 0
}

# Trap SIGINT and SIGTERM for cleanup
trap cleanup SIGINT SIGTERM

# Main execution
main() {
    echo ""
    print_info "=== krkn-operator Launcher ==="
    echo ""

    check_prerequisites
    install_crds
    build_operator
    run_operator
}

# Show usage
usage() {
    cat << EOF
Usage: $0 [options]

Options:
    -h, --help              Show this help message
    -p, --api-port PORT     Set REST API port (default: 8080)
    -m, --metrics ADDR      Set metrics address (default: :8443)
    -H, --health ADDR       Set health probe address (default: :8083)
    --skip-crds             Skip CRD installation
    --skip-build            Skip operator build

Environment variables:
    API_PORT                REST API port (default: 8080)
    METRICS_ADDR            Metrics endpoint address (default: :8443)
    HEALTH_PROBE_ADDR       Health probe address (default: :8083)
    KRKN_NAMESPACE          Namespace for KrknTargetRequest CRs (default: default)
    INSTALL_CRDS            Install CRDs (default: true)
    BUILD                   Build operator (default: true)

Examples:
    # Run with defaults
    $0

    # Run with custom API port
    $0 --api-port 9090

    # Skip CRD installation and build
    $0 --skip-crds --skip-build

    # Using environment variables
    API_PORT=9090 $0
EOF
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -h|--help)
            usage
            exit 0
            ;;
        -p|--api-port)
            API_PORT="$2"
            shift 2
            ;;
        -m|--metrics)
            METRICS_ADDR="$2"
            shift 2
            ;;
        -H|--health)
            HEALTH_PROBE_ADDR="$2"
            shift 2
            ;;
        --skip-crds)
            INSTALL_CRDS=false
            shift
            ;;
        --skip-build)
            BUILD=false
            shift
            ;;
        *)
            print_error "Unknown option: $1"
            usage
            exit 1
            ;;
    esac
done

# Run main
main