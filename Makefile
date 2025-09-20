# Makefile for cygwin 环境
# 支持本机 Windows 可执行和交叉编译 Linux/ARM64

COM_MES      := bin/com_mes
COM_TSD      := bin/com_tsd
GOBOLT       := bin/gobolt
GOBOLT_ARM64 := bin/gobolt-linux-arm64
ASCIIPLAY 	:= bin/AsciiMotionPlayer_console
ASCIIPLAY_GUI := bin/AsciiMotionPlayer.exe

.PHONY: help tidy all test test-com_mes test-com_tsd test-gobolt build-arm64 clean

help:
	@echo "可用命令:"
	@echo "  tidy             - 清理 Go 依赖"
	@echo "  all              - 构建所有目标（含 ARM64 交叉编译）"
	@echo "  test             - 构建并测试所有项目"
	@echo "  test-com_mes     - 构建并测试 com_mes"
	@echo "  test-com_tsd     - 构建并测试 com_tsd"
	@echo "  test-asciiplay   - 构建并测试 asciiplay"
	@echo "  test-gobolt      - 构建并测试 gobolt (本地 Windows 可执行)"
	@echo "  build-arm64      - 只交叉编译 gobolt 为 Linux/ARM64"
	@echo "  clean            - 清理所有生成文件"

# 统一执行 `go mod tidy`
tidy:
	@echo "清理 Go 依赖..."
	go mod tidy

# 默认构建：com_mes, com_tsd, gobolt(Win), gobolt-Linux-ARM64
all: $(COM_MES) $(COM_TSD) $(GOBOLT) $(GOBOLT_ARM64) $(ASCIIPLAY) $(ASCIIPLAY_GUI)

# 构建并测试
test: test-com_mes test-com_tsd test-gobolt test-asciiplay

# com_mes
$(COM_MES): tidy
	@echo "构建 com_mes..."
	go build -ldflags="-s -w" -o $(COM_MES) ./cmd/com_mes

icon_png:
	@echo "生成 32x32 PNG 图标..."
	magick ./cmd/asciiplay/assets/icon.svg -background none -resize 32x32 ./cmd/asciiplay/assets/icon_32.png

icon_ico:
	@echo "生成多尺寸 ICO 图标..."
	rm -f ./cmd/asciiplay/icon.ico
	magick ./cmd/asciiplay/assets/icon.svg -background none -define icon:auto-resize=16,32,48,64,128,256 ./cmd/asciiplay/icon.ico
	rsrc -ico ./cmd/asciiplay/icon.ico -o ./cmd/asciiplay/icon.syso

# asciiplay
$(ASCIIPLAY): tidy icon_png
	@echo "构建 asciiplay..."
	go build -ldflags="-s -w" -o $(ASCIIPLAY) ./cmd/asciiplay

$(ASCIIPLAY_GUI): tidy icon_png icon_ico
	@echo "构建 asciiplay gui ..."
	go build -ldflags="-s -w -H windowsgui" -o $(ASCIIPLAY_GUI) ./cmd/asciiplay

# com_tsd
$(COM_TSD): tidy
	@echo "构建 com_tsd..."
	go build -ldflags="-s -w" -o $(COM_TSD) ./cmd/com_tsd

# gobolt (本机 Windows 可执行)
$(GOBOLT): tidy
	@echo "构建 gobolt (本机 Windows 可执行)..."
	go build -ldflags="-s -w" -o $(GOBOLT) ./cmd/gobolt

# gobolt Linux/ARM64 交叉编译 (纯 Go，不启用 CGO)
$(GOBOLT_ARM64): tidy
	@echo "交叉编译 gobolt 为 Linux/ARM64..."
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o $(GOBOLT_ARM64) ./cmd/gobolt

# 专门做 ARM64 构建
build-arm64: $(GOBOLT_ARM64)

# 构建并复制到测试目录（可根据实际需求开启/注释）
test-asciiplay: $(ASCIIPLAY) $(ASCIIPLAY_GUI)
	@echo "测试 asciiplay..."
	rm -rf $(ASCIIPLAY)
	rm -rf $(ASCIIPLAY_GUI)
	make $(ASCIIPLAY)
	cp -f $(ASCIIPLAY) test/TestPlat/bin/
	make $(ASCIIPLAY_GUI)
	cp -f $(ASCIIPLAY_GUI) test/TestPlat/bin/

# 构建并复制到测试目录（可根据实际需求开启/注释）
test-com_mes: $(COM_MES)
	@echo "测试 com_mes..."
	rm -rf bin/com_mes
	make $(COM_MES)
	cp -f bin/com_mes test/TestPlat/bin/

test-com_tsd: $(COM_TSD)
	@echo "测试 com_tsd..."
	rm -rf bin/com_tsd
	make $(COM_TSD)
	cp -f bin/com_tsd test/TestPlat/bin/

test-gobolt: $(GOBOLT)
	@echo "测试 gobolt..."
	rm -rf bin/gobolt
	make $(GOBOLT)
	cp -f bin/gobolt test/TestPlat/bin/

# 清理所有生成文件
clean:
	@echo "清理生成文件..."
	rm -rf bin/*
	rm -rf ./cmd/asciiplay/icon.*

