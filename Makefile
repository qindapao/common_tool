COM_MES=bin/com_mes
COM_TSD=bin/com_tsd
GOBOLT=bin/gobolt

help:
	@echo "可用命令:"
	@echo "  tidy          - 清理 Go 依赖"
	@echo "  all           - 构建所有目标"
	@echo "  test          - 执行所有测试"
	@echo "  test-com_mes  - 构建并测试 com_mes"
	@echo "  test-com_tsd  - 构建并测试 com_tsd"
	@echo "  test-gobolt   - 构建并测试 gobolt"
	@echo "  clean         - 清理编译文件"


# 统一执行 `go mod tidy`
tidy:
	@echo "清理Go依赖..."
	go mod tidy

# 默认执行构建所有
all: $(COM_MES) $(COM_TSD) $(GOBOLT)
# 执行所有测试构建
test: test-com_tsd test-com_mes test-gobolt
	# 注释掉的原因是在cygwin的环境下无法执行
	# go test ./... -v

# 编译 com_mes
$(COM_MES): tidy
	go build -ldflags="-s -w" -o $(COM_MES) ./cmd/com_mes

# 编译 tool2
$(COM_TSD): tidy
	go build -ldflags="-s -w" -o $(COM_TSD) ./cmd/com_tsd

$(GOBOLT): tidy
	go build -ldflags="-s -w" -o $(GOBOLT) ./cmd/gobolt

# 构建并执行 com_mes
test-com_mes: $(COM_MES)
	rm -rf bin/com_mes
	make $(COM_MES)
	cp -f bin/com_mes test/TestPlat/bin/

# 构建并执行 com_tsd
test-com_tsd: $(COM_TSD)
	rm -rf bin/com_tsd
	make $(COM_TSD)
	cp -f bin/com_tsd test/TestPlat/bin/

# 构建并执行 gobolt
test-gobolt: $(GOBOLT)
	rm -rf bin/gobolt
	make $(GOBOLT)
	cp -f bin/gobolt test/TestPlat/bin/

# 清理编译文件
clean:
	rm -rf bin/*
