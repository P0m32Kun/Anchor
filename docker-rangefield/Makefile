.PHONY: up down restart logs ps status health \
        test-httpx test-naabu test-nuclei test-all \
        seed-anchor clean

# --- 生命周期 ---

up:
	@echo "[+] 启动靶场..."
	docker compose up -d
	@echo "[+] 等待服务就绪..."
	@sleep 5
	@docker compose ps

down:
	@echo "[-] 停止并清理靶场..."
	docker compose down -v

restart: down up

logs:
	docker compose logs -f

ps:
	docker compose ps

status:
	@docker compose ps --format "table {{.Name}}\t{{.Status}}\t{{.Ports}}"

health:
	@for svc in nginx tomcat grafana redis mysql; do \
		status=$$(docker inspect --format='{{.State.Health.Status}}' rf-$$svc 2>/dev/null || echo "N/A"); \
		echo "rf-$$svc: $$status"; \
	done

# --- CLI 工具测试 ---

TESTDATA_DIR ?= ./testdata

test-httpx:
	@mkdir -p $(TESTDATA_DIR)
	@echo "=== httpx Web 探测 ==="
	cat targets-httpx.txt | httpx -json -o $(TESTDATA_DIR)/httpx.jsonl
	@echo "结果: $(TESTDATA_DIR)/httpx.jsonl"

test-naabu:
	@mkdir -p $(TESTDATA_DIR)
	@echo "=== Naabu 端口扫描 ==="
	cat targets-naabu.txt | naabu -json -o $(TESTDATA_DIR)/naabu.jsonl
	@echo "结果: $(TESTDATA_DIR)/naabu.jsonl"

test-nuclei:
	@mkdir -p $(TESTDATA_DIR)
	@echo "=== Nuclei 漏洞扫描 (标准配置) ==="
	nuclei -jsonl -l targets-httpx.txt -o $(TESTDATA_DIR)/nuclei.jsonl
	@echo "结果: $(TESTDATA_DIR)/nuclei.jsonl"

test-nuclei-tags:
	@mkdir -p $(TESTDATA_DIR)
	@echo "=== Nuclei 按 Tag 扫描 (模拟 Anchor 行为) ==="
	nuclei -jsonl -l targets-httpx.txt -tags nginx,tomcat,grafana,redis,mysql -o $(TESTDATA_DIR)/nuclei-tags.jsonl
	@echo "结果: $(TESTDATA_DIR)/nuclei-tags.jsonl"

test-all: test-httpx test-naabu test-nuclei
	@echo ""
	@echo "=== 全部测试完成 ==="
	@ls -la $(TESTDATA_DIR)/

# --- Anchor 集成 ---

seed-anchor:
	@echo "[+] 向 Anchor 注入靶场 WebEndpoint..."
	@bash scripts/seed-anchor.sh

# --- 清理 ---

clean:
	rm -rf $(TESTDATA_DIR)
