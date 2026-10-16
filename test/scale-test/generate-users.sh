#!/bin/bash
# Script to generate KrknUser resources with temporary passwords for scale testing
# Generates 15 test users with associated password secrets

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
MANIFESTS_DIR="${SCRIPT_DIR}/manifests"
NUM_USERS=15
NAMESPACE="${NAMESPACE:-krkn-operator-system}"
PASSWORD="TempPass123!"  # Default temporary password for all test users
KUBECONFIG_PATH="${KUBECONFIG_PATH:-}"  # Optional custom kubeconfig path

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

print_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if htpasswd is available for bcrypt hashing
check_dependencies() {
    if ! command -v htpasswd &> /dev/null; then
        print_error "htpasswd command not found. Please install apache2-utils:"
        print_info "  On macOS: brew install httpd"
        print_info "  On Ubuntu/Debian: sudo apt-get install apache2-utils"
        print_info "  On RHEL/Fedora: sudo dnf install httpd-tools"
        exit 1
    fi
}

# Generate bcrypt hash for password
generate_password_hash() {
    local password="$1"
    # Use htpasswd to generate bcrypt hash (algorithm 2y, cost 10)
    htpasswd -nbBC 10 "" "$password" | sed 's/^://g'
}

# Generate a user manifest from template
generate_user_manifest() {
    local user_num="$1"
    local user_id="test-user-${user_num}@krkn-chaos.dev"
    local name="Test"
    local surname="User${user_num}"
    local organization="Krkn Chaos Engineering"
    local role="user"
    local group_labels=""

    # Make first 2 users admins for testing
    if [ "$user_num" -le 2 ]; then
        role="admin"
        # Admins don't need group assignments
    # Users 3-7: qa-team
    elif [ "$user_num" -le 7 ]; then
        group_labels="    group.krkn.krkn-chaos.dev/qa-team: \"true\""
    # Users 8-12: developers
    elif [ "$user_num" -le 12 ]; then
        group_labels="    group.krkn.krkn-chaos.dev/developers: \"true\""
    # Users 13-15: sre-team
    else
        group_labels="    group.krkn.krkn-chaos.dev/sre-team: \"true\""
    fi

    local secret_name="krkn-user-${user_num}-password"

    # Read template and substitute variables
    if [ ! -f "${SCRIPT_DIR}/user-template.yaml" ]; then
        print_error "Template file not found: ${SCRIPT_DIR}/user-template.yaml"
        exit 1
    fi

    sed -e "s|{{USER_NUM}}|${user_num}|g" \
        -e "s|{{USER_ID}}|${user_id}|g" \
        -e "s|{{NAME}}|${name}|g" \
        -e "s|{{SURNAME}}|${surname}|g" \
        -e "s|{{ORGANIZATION}}|${organization}|g" \
        -e "s|{{ROLE}}|${role}|g" \
        -e "s|{{SECRET_NAME}}|${secret_name}|g" \
        -e "s|{{PASSWORD_HASH}}|${PASSWORD_HASH}|g" \
        -e "s|{{NAMESPACE}}|${NAMESPACE}|g" \
        -e "s|{{GROUP_LABELS}}|${group_labels}|g" \
        "${SCRIPT_DIR}/user-template.yaml" | \
        grep -v "^#" | \
        grep -v "^[[:space:]]*$" > "${MANIFESTS_DIR}/user-${user_num}.yaml"
}

# Generate all manifests
generate_all_manifests() {
    print_info "Generating manifests for ${NUM_USERS} users..."

    # Clean up old manifests
    rm -f "${MANIFESTS_DIR}"/*.yaml

    # Generate password hash once (same for all users for simplicity)
    print_info "Generating bcrypt password hash..."
    PASSWORD_HASH=$(generate_password_hash "$PASSWORD")

    # Copy group definitions first
    print_info "Adding group definitions..."
    cp "${SCRIPT_DIR}/sample-groups.yaml" "${MANIFESTS_DIR}/00-groups.yaml"

    # Generate individual user manifests
    for i in $(seq 1 $NUM_USERS); do
        generate_user_manifest "$i"
        print_info "Generated user-${i}.yaml"
    done

    # Create a combined manifest for easy application (groups first, then users)
    cat "${MANIFESTS_DIR}/00-groups.yaml" "${MANIFESTS_DIR}"/user-*.yaml > "${MANIFESTS_DIR}/all-users.yaml"
    print_info "Combined groups and users into all-users.yaml"
}

# Apply manifests to cluster
apply_manifests() {
    print_info "Applying user manifests to cluster..."

    # Set KUBECONFIG if custom path provided
    local kubectl_cmd="kubectl"
    if [ -n "$KUBECONFIG_PATH" ]; then
        export KUBECONFIG="$KUBECONFIG_PATH"
        print_info "Using kubeconfig: ${KUBECONFIG_PATH}"
    fi

    if ! kubectl get namespace "${NAMESPACE}" &> /dev/null; then
        print_warn "Namespace ${NAMESPACE} does not exist. Creating it..."
        kubectl create namespace "${NAMESPACE}"
    fi

    kubectl apply -f "${MANIFESTS_DIR}/all-users.yaml"
    print_info "Successfully applied ${NUM_USERS} users to namespace ${NAMESPACE}"

    # Enable all users (set status.active=true)
    print_info "Enabling all users..."
    for i in $(seq 1 $NUM_USERS); do
        kubectl patch krknuser krkn-user-${i} \
            -n "${NAMESPACE}" \
            --type=merge \
            --subresource=status \
            -p '{"status":{"active":true}}' > /dev/null 2>&1 || true
    done
    print_info "All users enabled"
}

# Delete all generated users
cleanup_users() {
    print_info "Deleting all generated users from cluster..."

    # Set KUBECONFIG if custom path provided
    if [ -n "$KUBECONFIG_PATH" ]; then
        export KUBECONFIG="$KUBECONFIG_PATH"
        print_info "Using kubeconfig: ${KUBECONFIG_PATH}"
    fi

    kubectl delete -f "${MANIFESTS_DIR}/all-users.yaml" --ignore-not-found=true
    print_info "Successfully deleted all test users"
}

# Display credentials
show_credentials() {
    print_info "Test User Credentials:"
    echo "======================"
    echo "Password for all users: ${PASSWORD}"
    echo ""
    echo "User Assignments:"
    echo ""
    echo "Admins (no group):"
    for i in $(seq 1 2); do
        printf "  %2d. test-user-%d@krkn-chaos.dev\n" "$i" "$i"
    done
    echo ""
    echo "QA Team (users 3-7):"
    for i in $(seq 3 7); do
        printf "  %2d. test-user-%d@krkn-chaos.dev\n" "$i" "$i"
    done
    echo ""
    echo "Developers (users 8-12):"
    for i in $(seq 8 12); do
        printf "  %2d. test-user-%d@krkn-chaos.dev\n" "$i" "$i"
    done
    echo ""
    echo "SRE Team (users 13-15):"
    for i in $(seq 13 15); do
        printf "  %2d. test-user-%d@krkn-chaos.dev\n" "$i" "$i"
    done
    echo ""
    print_warn "These are TEMPORARY passwords for testing only!"
}

# Main function
main() {
    local command="${1:-help}"

    check_dependencies

    case "$command" in
        generate)
            generate_all_manifests
            show_credentials
            ;;
        apply)
            if [ ! -f "${MANIFESTS_DIR}/all-users.yaml" ]; then
                print_warn "Manifests not found. Generating first..."
                generate_all_manifests
            fi
            apply_manifests
            show_credentials
            ;;
        delete)
            cleanup_users
            ;;
        show)
            show_credentials
            ;;
        help|*)
            echo "Usage: $0 {generate|apply|delete|show}"
            echo ""
            echo "Commands:"
            echo "  generate  - Generate user manifests (YAML files)"
            echo "  apply     - Apply user manifests to cluster"
            echo "  delete    - Delete all generated users from cluster"
            echo "  show      - Display user credentials"
            echo ""
            echo "Environment variables:"
            echo "  NAMESPACE       - Target namespace (default: krkn-operator-system)"
            echo "  KUBECONFIG_PATH - Path to kubeconfig file (default: uses current context)"
            echo ""
            echo "Examples:"
            echo "  $0 generate                                    # Generate manifests only"
            echo "  $0 apply                                       # Generate and apply to default cluster"
            echo "  NAMESPACE=test $0 apply                        # Apply to 'test' namespace"
            echo "  KUBECONFIG_PATH=~/.kube/my-cluster $0 apply    # Use specific kubeconfig"
            echo "  $0 delete                                      # Remove all test users"
            exit 0
            ;;
    esac
}

main "$@"
