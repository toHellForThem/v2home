# OpenWrt 24.10 userland used for smoke-testing the static binary.
# The binary is produced by `make build-openwrt` / `scripts/openwrt-test.sh`
# and copied into deploy/openwrt/bin/v2home before this image is built.
ARG OPENWRT_IMAGE=openwrt/rootfs
ARG OPENWRT_TAG=x86-64-24.10.0

FROM ${OPENWRT_IMAGE}:${OPENWRT_TAG}

COPY bin/v2home /usr/sbin/v2home
COPY files/etc/init.d/v2home /etc/init.d/v2home
COPY files/etc/config/v2home /etc/config/v2home

RUN chmod 0755 /usr/sbin/v2home /etc/init.d/v2home

EXPOSE 8080

# Run the binary directly for the smoke test. To exercise procd instead, start
# the container with --privileged and override the entrypoint with /sbin/init.
ENTRYPOINT ["/usr/sbin/v2home"]
CMD ["--listen", "0.0.0.0:8080"]
