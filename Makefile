COM_MES=bin/com_mes
COM_TSD=bin/com_tsd

# 统一执行 `go mod tidy`
tidy:
	echo "清理Go依赖..."
	go mod tidy

# 默认执行构建所有
all: $(COM_MES) $(COM_TSD)


# 编译 com_mes
$(COM_MES): tidy
	go build -ldflags="-s -w" -o $(COM_MES) ./cmd/com_mes

# 编译 tool2
$(COM_TSD): tidy
	go build -ldflags="-s -w" -o $(COM_TSD) ./cmd/com_tsd

# 构建并执行 com_mes
test-com_mes: $(COM_MES)
	rm -rf bin/com_mes
	make $(COM_MES)
	cp -f bin/com_mes test/TestPlat/bin/

# 构建并执行 com_tsd
test-com_tsd: $(COM_TSD)
	rm -rf bin/com_tsd
	make $(COM_TSD)

# 清理编译文件
clean:
	rm -rf bin/*
