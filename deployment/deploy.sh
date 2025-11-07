#!/bin/bash
set -e

echo "=== Genred English Application Deployment Script ==="

# 変数設定
APP_DIR="/opt/genred_english"
SERVICE_NAME="genred_english"

# Dockerイメージをビルド
echo "Building Docker image..."
docker build -t genred_english:latest .

# アプリケーションディレクトリを作成
echo "Setting up application directory..."
sudo mkdir -p ${APP_DIR}

# 環境変数ファイルをコピー（存在しない場合）
if [ ! -f ${APP_DIR}/.env ]; then
    echo "Creating .env file from example..."
    sudo cp deployment/.env.example ${APP_DIR}/.env
    echo "IMPORTANT: Edit ${APP_DIR}/.env with your actual DATABASE_URL"
fi

# systemdサービスファイルをコピー
echo "Installing systemd service..."
sudo cp deployment/genred_english.service /etc/systemd/system/

# Nginx設定をコピー
echo "Installing Nginx configuration..."
sudo cp nginx/prod/nginx.conf /etc/nginx/sites-available/genred_english
sudo ln -sf /etc/nginx/sites-available/genred_english /etc/nginx/sites-enabled/genred_english

# デフォルトのNginx設定を無効化（必要に応じて）
if [ -L /etc/nginx/sites-enabled/default ]; then
    echo "Disabling default Nginx site..."
    sudo rm /etc/nginx/sites-enabled/default
fi

# Nginx設定をテスト
echo "Testing Nginx configuration..."
sudo nginx -t

# systemdをリロード
echo "Reloading systemd..."
sudo systemctl daemon-reload

# サービスを有効化して起動
echo "Enabling and starting FastAPI service..."
sudo systemctl enable ${SERVICE_NAME}
sudo systemctl restart ${SERVICE_NAME}

# Nginxを再起動
echo "Restarting Nginx..."
sudo systemctl restart nginx

# ステータス確認
echo ""
echo "=== Service Status ==="
sudo systemctl status ${SERVICE_NAME} --no-pager
echo ""
echo "=== Nginx Status ==="
sudo systemctl status nginx --no-pager

echo ""
echo "=== Deployment Complete ==="
echo "To view logs: sudo journalctl -u ${SERVICE_NAME} -f"
echo "To edit environment: sudo nano ${APP_DIR}/.env"
echo "Don't forget to edit ${APP_DIR}/.env with your actual DATABASE_URL!"
