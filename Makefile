.PHONY: secrets secrets-api secrets-ui

# 全環境変数ファイルを 1Password から生成
secrets:
	@bash web/scripts/secrets.sh

# API の .env のみ生成
secrets-api:
	@op inject -i web/api/.env.tpl -o web/api/.env --force
	@echo "[INFO] web/api/.env を生成しました"

# UI の .env.local のみ生成
secrets-ui:
	@op inject -i web/ui/.env.tpl -o web/ui/.env.local --force
	@echo "[INFO] web/ui/.env.local を生成しました"
