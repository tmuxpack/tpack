FROM archlinux:latest

ARG RUNNER_VERSION=2.333.1
ARG RUNNER_ARCH=x64

RUN pacman -Syu --noconfirm \
    && pacman -S --noconfirm \
        base-devel \
        git \
        openssh \
        curl \
        jq \
        sudo \
        ca-certificates \
        icu \
        krb5 \
    && pacman -Scc --noconfirm

RUN useradd --create-home --shell /bin/bash --uid 1001 runner \
    && mkdir -p /home/runner/.ssh /home/runner/_work \
    && chown -R runner:runner /home/runner

USER runner
WORKDIR /home/runner

RUN curl -fsSL -o actions-runner.tar.gz \
        "https://github.com/actions/runner/releases/download/v${RUNNER_VERSION}/actions-runner-linux-${RUNNER_ARCH}-${RUNNER_VERSION}.tar.gz" \
    && tar xzf actions-runner.tar.gz \
    && rm actions-runner.tar.gz

COPY --chown=runner:runner entrypoint.sh /usr/local/bin/entrypoint.sh

ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]
