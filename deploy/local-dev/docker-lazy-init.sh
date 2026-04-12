#!/bin/bash
# Lazy-init wrapper for Docker CLI.
# Starts dockerd automatically on first docker command.
# Works with KubeOpenCode's command override pattern since it wraps
# the docker binary itself, not the container entrypoint.

DOCKER_REAL="/usr/bin/docker.real"
DOCKERD_LOG="/tmp/dockerd.log"
DOCKERD_PIDFILE="/tmp/dockerd.pid"

# Check if dockerd is already running
if ! ${DOCKER_REAL} info >/dev/null 2>&1; then
    # Acquire lock to prevent concurrent dockerd starts
    exec 200>/tmp/dockerd.lock
    flock -n 200 || {
        # Another process is starting dockerd, wait for it
        flock 200
        exec ${DOCKER_REAL} "$@"
    }

    # Double-check after acquiring lock
    if ! ${DOCKER_REAL} info >/dev/null 2>&1; then
        # Mount tmpfs at /var/lib/docker to avoid overlay-on-overlay issues.
        # Container filesystems use overlayfs (from containerd), and nested
        # overlayfs is not supported. tmpfs provides a clean mount point.
        if ! mountpoint -q /var/lib/docker 2>/dev/null; then
            mkdir -p /var/lib/docker
            mount -t tmpfs tmpfs /var/lib/docker
        fi

        echo "Starting Docker daemon..." >&2
        dockerd &>"${DOCKERD_LOG}" &
        echo $! > "${DOCKERD_PIDFILE}"

        # Wait for Docker daemon to be ready
        timeout=30
        while ! ${DOCKER_REAL} info >/dev/null 2>&1; do
            timeout=$((timeout - 1))
            if [ $timeout -le 0 ]; then
                echo "ERROR: Docker daemon failed to start. Check ${DOCKERD_LOG}" >&2
                exit 1
            fi
            sleep 1
        done
        echo "Docker daemon ready" >&2
    fi
fi

exec ${DOCKER_REAL} "$@"
