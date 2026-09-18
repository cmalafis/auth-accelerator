# UBI-minimal image wrapping the prebuilt static binary. goreleaser builds the
# binary first and drops it into the build context, so there's no Go toolchain
# in the image. This is the optional container path; the static binary is the
# primary way to run the tool.
FROM registry.access.redhat.com/ubi9/ubi-minimal:latest

LABEL org.opencontainers.image.title="auth-accelerator" \
      org.opencontainers.image.description="Generate OpenShift 4.20+ smart-card (PIV/CAC) external-OIDC config + runbook" \
      org.opencontainers.image.source="https://github.com/cmalafis/auth-accelerator" \
      org.opencontainers.image.licenses="GPL-3.0-or-later"

COPY auth-accelerator /usr/local/bin/auth-accelerator

# Ship the license texts with the binary — the MIT/BSD/Apache deps it statically
# links require their notices to accompany binary distributions.
COPY LICENSE THIRD-PARTY-NOTICES.md /usr/share/licenses/auth-accelerator/

# Run as a non-root user; the tool only writes to the output dir the user picks.
USER 1001
WORKDIR /work

ENTRYPOINT ["/usr/local/bin/auth-accelerator"]
CMD ["--help"]
