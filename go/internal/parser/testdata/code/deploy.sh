#!/bin/bash

# Deployment script for web application
# Author: DevOps Team
# Usage: ./deploy.sh <environment> <version>

set -euo pipefail

# Global variables
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
LOG_FILE="/var/log/deploy.log"
DEPLOY_USER="deploy"
APP_NAME="web-app"

# Configuration
ENVIRONMENTS=("dev" "staging" "production")
DEFAULT_TIMEOUT=300
MAX_RETRIES=3

# Function to log messages
log_message() {
    local level="$1"
    local message="$2"
    local timestamp=$(date '+%Y-%m-%d %H:%M:%S')
    echo "[${timestamp}] [${level}] ${message}" | tee -a "${LOG_FILE}"
}

# Function to check prerequisites
check_prerequisites() {
    log_message "INFO" "Checking prerequisites..."

    local required_commands=("docker" "kubectl" "aws" "jq")

    for cmd in "${required_commands[@]}"; do
        if ! command -v "${cmd}" &> /dev/null; then
            log_message "ERROR" "Required command not found: ${cmd}"
            return 1
        fi
    done

    log_message "INFO" "All prerequisites satisfied"
    return 0
}

# Function to validate environment
validate_environment() {
    local env="$1"

    for valid_env in "${ENVIRONMENTS[@]}"; do
        if [[ "${env}" == "${valid_env}" ]]; then
            return 0
        fi
    done

    log_message "ERROR" "Invalid environment: ${env}"
    log_message "INFO" "Valid environments: ${ENVIRONMENTS[*]}"
    return 1
}

# Function to build Docker image
build_image() {
    local version="$1"
    local image_tag="${APP_NAME}:${version}"

    log_message "INFO" "Building Docker image: ${image_tag}"

    if docker build -t "${image_tag}" .; then
        log_message "INFO" "Successfully built image: ${image_tag}"
        return 0
    else
        log_message "ERROR" "Failed to build image: ${image_tag}"
        return 1
    fi
}

# Function to push image to registry
push_image() {
    local version="$1"
    local registry="${ECR_REGISTRY:-registry.example.com}"
    local image_tag="${registry}/${APP_NAME}:${version}"

    log_message "INFO" "Pushing image to registry: ${image_tag}"

    # Tag image
    docker tag "${APP_NAME}:${version}" "${image_tag}"

    # Push with retries
    local retries=0
    while [[ ${retries} -lt ${MAX_RETRIES} ]]; do
        if docker push "${image_tag}"; then
            log_message "INFO" "Successfully pushed image: ${image_tag}"
            return 0
        fi

        retries=$((retries + 1))
        log_message "WARN" "Push failed, retry ${retries}/${MAX_RETRIES}"
        sleep 5
    done

    log_message "ERROR" "Failed to push image after ${MAX_RETRIES} retries"
    return 1
}

# Function to update Kubernetes deployment
update_deployment() {
    local environment="$1"
    local version="$2"
    local namespace="${environment}"

    log_message "INFO" "Updating deployment in ${environment}"

    # Update deployment image
    kubectl set image deployment/${APP_NAME} \
        ${APP_NAME}=${APP_NAME}:${version} \
        -n "${namespace}"

    # Wait for rollout
    if kubectl rollout status deployment/${APP_NAME} \
        -n "${namespace}" \
        --timeout=${DEFAULT_TIMEOUT}s; then
        log_message "INFO" "Deployment successful in ${environment}"
        return 0
    else
        log_message "ERROR" "Deployment failed in ${environment}"
        return 1
    fi
}

# Function to run health check
health_check() {
    local environment="$1"
    local url="https://${environment}.example.com/health"

    log_message "INFO" "Running health check: ${url}"

    local response=$(curl -s -o /dev/null -w "%{http_code}" "${url}")

    if [[ "${response}" == "200" ]]; then
        log_message "INFO" "Health check passed"
        return 0
    else
        log_message "ERROR" "Health check failed: HTTP ${response}"
        return 1
    fi
}

# Function to rollback deployment
rollback_deployment() {
    local environment="$1"
    local namespace="${environment}"

    log_message "WARN" "Rolling back deployment in ${environment}"

    kubectl rollout undo deployment/${APP_NAME} -n "${namespace}"

    if kubectl rollout status deployment/${APP_NAME} -n "${namespace}"; then
        log_message "INFO" "Rollback successful"
        return 0
    else
        log_message "ERROR" "Rollback failed"
        return 1
    fi
}

# Function to send notification
send_notification() {
    local status="$1"
    local message="$2"

    case "${status}" in
        success)
            log_message "INFO" "Deployment succeeded: ${message}"
            ;;
        failure)
            log_message "ERROR" "Deployment failed: ${message}"
            ;;
        *)
            log_message "WARN" "Unknown status: ${status}"
            ;;
    esac
}

# Main deployment function
deploy() {
    local environment="$1"
    local version="$2"

    log_message "INFO" "Starting deployment: ${APP_NAME} v${version} to ${environment}"

    # Validate inputs
    if ! validate_environment "${environment}"; then
        return 1
    fi

    # Check prerequisites
    if ! check_prerequisites; then
        send_notification "failure" "Prerequisites check failed"
        return 1
    fi

    # Build image
    if ! build_image "${version}"; then
        send_notification "failure" "Image build failed"
        return 1
    fi

    # Push image
    if ! push_image "${version}"; then
        send_notification "failure" "Image push failed"
        return 1
    fi

    # Update deployment
    if ! update_deployment "${environment}" "${version}"; then
        log_message "ERROR" "Deployment update failed, initiating rollback"
        rollback_deployment "${environment}"
        send_notification "failure" "Deployment failed and rolled back"
        return 1
    fi

    # Health check
    sleep 10
    if ! health_check "${environment}"; then
        log_message "ERROR" "Health check failed, initiating rollback"
        rollback_deployment "${environment}"
        send_notification "failure" "Health check failed and rolled back"
        return 1
    fi

    send_notification "success" "Deployment completed successfully"
    log_message "INFO" "Deployment completed: ${APP_NAME} v${version} to ${environment}"
    return 0
}

# Parse command line arguments
if [[ $# -lt 2 ]]; then
    echo "Usage: $0 <environment> <version>"
    echo "Environments: ${ENVIRONMENTS[*]}"
    exit 1
fi

ENVIRONMENT="$1"
VERSION="$2"

# Run deployment
deploy "${ENVIRONMENT}" "${VERSION}"
exit $?
