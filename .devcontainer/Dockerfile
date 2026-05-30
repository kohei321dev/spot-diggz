# spot-diggz統合開発環境
# Rust + TypeScript/React + Terraform + GCP環境

FROM mcr.microsoft.com/vscode/devcontainers/base:ubuntu-22.04

# 環境変数設定
ENV DEBIAN_FRONTEND=noninteractive
ENV RUST_VERSION=1.90.0
ENV NODE_VERSION=22.14.0
ENV TERRAFORM_VERSION=1.6.6
ENV GCLOUD_VERSION=456.0.0

# パッケージアップデート・基本ツールインストール
RUN apt-get update && apt-get install -y \
    curl \
    wget \
    git \
    build-essential \
    pkg-config \
    libssl-dev \
    libsqlite3-dev \
    postgresql-client \
    redis-tools \
    jq \
    unzip \
    vim \
    htop \
    tree \
    ca-certificates \
    gnupg \
    lsb-release \
    && rm -rf /var/lib/apt/lists/*

# Docker CLI インストール
RUN curl -fsSL https://download.docker.com/linux/ubuntu/gpg | gpg --dearmor -o /usr/share/keyrings/docker-archive-keyring.gpg \
    && echo "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/docker-archive-keyring.gpg] https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable" | tee /etc/apt/sources.list.d/docker.list > /dev/null \
    && apt-get update \
    && apt-get install -y docker-ce-cli \
    && rm -rf /var/lib/apt/lists/*

# Rust toolchain インストール
RUN curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -y --default-toolchain ${RUST_VERSION} \
    && . $HOME/.cargo/env \
    && rustup component add rustfmt clippy rust-analyzer

# Node.js & npm インストール (nvmを使用)
RUN curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.39.0/install.sh | bash \
    && . ~/.nvm/nvm.sh \
    && nvm install ${NODE_VERSION} \
    && nvm use ${NODE_VERSION} \
    && nvm alias default ${NODE_VERSION} \
    && npm install -g yarn pnpm @angular/cli create-react-app

# Terraform インストール
RUN wget https://releases.hashicorp.com/terraform/${TERRAFORM_VERSION}/terraform_${TERRAFORM_VERSION}_linux_amd64.zip \
    && unzip terraform_${TERRAFORM_VERSION}_linux_amd64.zip \
    && mv terraform /usr/local/bin/ \
    && rm terraform_${TERRAFORM_VERSION}_linux_amd64.zip

# Google Cloud CLI インストール
RUN curl https://dl.google.com/dl/cloudsdk/channels/rapid/downloads/google-cloud-cli-${GCLOUD_VERSION}-linux-x86_64.tar.gz | tar -xz -C /opt \
    && /opt/google-cloud-sdk/install.sh --quiet --usage-reporting=false --path-update=true \
    && ln -s /opt/google-cloud-sdk/bin/* /usr/local/bin/

# VSCode Serverインストール（Codespacesで自動的に処理されるが、念のため）
RUN curl -fsSL https://code-server.dev/install.sh | sh -s -- --dry-run

# 開発用ツールインストール
RUN . ~/.nvm/nvm.sh \
    && npm install -g \
    eslint \
    prettier \
    typescript \
    ts-node \
    @types/node \
    concurrently \
    nodemon

# Rust開発ツール追加インストール
# cargo-binstallで事前ビルド済みバイナリを優先し、CIのメモリ負荷を軽減する
RUN . $HOME/.cargo/env \
    && curl -fsSL https://raw.githubusercontent.com/cargo-bins/cargo-binstall/main/install-from-binstall-release.sh | bash \
    && cargo binstall -y \
    cargo-watch \
    cargo-edit \
    cargo-audit \
    cargo-outdated \
    wasm-pack

# 作業ディレクトリ設定
WORKDIR /workspace

# 開発用スクリプトコピー
COPY .devcontainer/setup.sh /workspace/.devcontainer/setup.sh
RUN chmod +x /workspace/.devcontainer/setup.sh

# ユーザー設定（Codespacesではvscodeユーザーがデフォルト）
RUN groupadd -r docker || true \
    && usermod -aG docker vscode \
    && echo 'export PATH="$HOME/.cargo/bin:$PATH"' >> /home/vscode/.bashrc \
    && echo 'export PATH="$HOME/.nvm/versions/node/v${NODE_VERSION}/bin:$PATH"' >> /home/vscode/.bashrc \
    && echo 'source ~/.nvm/nvm.sh' >> /home/vscode/.bashrc \
    && echo 'source $HOME/.cargo/env' >> /home/vscode/.bashrc

# zsh設定（oh-my-zshはdevcontainer featuresで処理）
RUN echo 'export PATH="$HOME/.cargo/bin:$PATH"' >> /home/vscode/.zshrc \
    && echo 'export PATH="$HOME/.nvm/versions/node/v${NODE_VERSION}/bin:$PATH"' >> /home/vscode/.zshrc \
    && echo 'source ~/.nvm/nvm.sh' >> /home/vscode/.zshrc \
    && echo 'source $HOME/.cargo/env' >> /home/vscode/.zshrc

# 権限設定
RUN chown -R vscode:vscode /home/vscode \
    && chmod 755 /home/vscode

USER vscode

# 環境変数の最終設定
ENV PATH="/home/vscode/.cargo/bin:/home/vscode/.nvm/versions/node/v${NODE_VERSION}/bin:/opt/google-cloud-sdk/bin:${PATH}"

# ヘルスチェック
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:3000/ || exit 1
