FROM ubuntu:22.04

# タイムゾーンの設定
ENV DEBIAN_FRONTEND=noninteractive
ENV TZ=Asia/Tokyo

# 必要なパッケージのインストール
RUN apt-get update && apt-get install -y \
    python3.11 \
    python3-pip \
    curl \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

# uvのインストール
RUN curl -LsSf https://astral.sh/uv/install.sh | sh && \
    ln -s /root/.local/bin/uv /usr/local/bin/uv
ENV PATH="/root/.local/bin:$PATH"

# 作業ディレクトリの設定
WORKDIR /app

# pyproject.tomlとuv.lockをコピー（依存関係のキャッシュ用）
COPY pyproject.toml README.md ./
COPY uv.lock ./

# 依存関係のインストール
RUN uv sync

# アプリケーションのコピー
COPY . .

# ポート8000を公開
EXPOSE 8000

# アプリケーションを起動
CMD ["uv", "run", "uvicorn", "app.main:app", "--host", "0.0.0.0", "--port", "8000", "--reload"]
