GJSON_VER := $(shell grep 'github.com/tidwall/gjson' go.mod | awk '{print $$2}')
SJSON_VER := $(shell grep 'github.com/tidwall/sjson' go.mod | awk '{print $$2}')
PRETY_VER := $(shell grep 'github.com/tidwall/pretty' go.mod | awk '{print $$2}')
GOBOLT       := bin/gobolt
GOBOLT_ARM64 := bin/gobolt-linux-arm64
GOBOLT_AMD64 := bin/gobolt-linux-amd64
ASCIIPLAY 	:= bin/AsciiMotionPlayer
ASCIIPLAY_GUI := bin/AsciiMotionPlayer.exe

.PHONY: help tidy all asciiplay gobolt clean

# HIGHLIGHT_LAST = awk '{for(i=1;i<NF;i++) printf "%s ", $$i; printf "\033[1;36m%s\033[0m\n", $$NF}'
# Highlight the last column (Filenames and relative directories) output by ls
define HIGHLIGHT_LAST
awk '{ \
    for(i=1;i<NF;i++) printf "%s ", $$i; \
    printf "\033[1;36m%s\033[0m\n", $$NF \
}'
endef

help:
	@echo "Available commands:"
	@echo "  tidy             - Clean up Go dependencies"
	@echo "  all              - Build all targets（Contains ARM64/AMD64 cross-compilation）"
	@echo "  asciiplay        - Build asciiplay"
	@echo "  gobolt           - Build gobolt"
	@echo "  clean            - Clean all generated files"

tidy:
	@echo "Clean up Go dependencies..."
	go mod tidy

all: gobolt asciiplay

icon_png:
	@echo ""
	@echo "----------generate 32x32 PNG icon..."
	@echo ""
	magick ./cmd/asciiplay/assets/icon.svg -background none -resize 32x32 ./cmd/asciiplay/assets/icon_32.png

icon_ico:
	@echo ""
	@echo "----------Generate multi-sized ICO icons..."
	@echo ""
	rm -f ./cmd/asciiplay/icon.ico
	magick ./cmd/asciiplay/assets/icon.svg -background none -define icon:auto-resize=16,32,48,64,128,256 ./cmd/asciiplay/icon.ico
	rsrc -ico ./cmd/asciiplay/icon.ico -o ./cmd/asciiplay/icon.syso

# asciiplay
$(ASCIIPLAY): tidy icon_png
	@echo ""
	@echo "----------Build asciiplay..."
	@echo ""
	go build -ldflags="-s -w" -o $(ASCIIPLAY) ./cmd/asciiplay

$(ASCIIPLAY_GUI): tidy icon_png icon_ico
	@echo ""
	@echo "----------Build asciiplay gui ..."
	@echo ""
	go build -ldflags="-s -w -H windowsgui" -o $(ASCIIPLAY_GUI) ./cmd/asciiplay

asciiplay: $(ASCIIPLAY) $(ASCIIPLAY_GUI)
	@echo ""
	@echo "----------asciiplay build completed:"
	@echo ""
	@ls -lh $(ASCIIPLAY) $(ASCIIPLAY_GUI) | $(HIGHLIGHT_LAST)

# gobolt (Windows)
$(GOBOLT): tidy
	@echo ""
	@echo "----------Build gobolt (Windows)..."
	@echo ""
	go build -ldflags="-s -w \
		-X 'common_tool/pkg/qqjson.GjsonVersion=$(GJSON_VER)' \
		-X 'common_tool/pkg/qqjson.PrettyVersion=$(PRETY_VER)' \
		-X 'common_tool/pkg/qqjson.SjsonVersion=$(SJSON_VER)'" \
		-o $(GOBOLT) ./cmd/gobolt

# gobolt Linux/ARM64 cross compile (Pure Go, no CGO enabled)
$(GOBOLT_ARM64): tidy
	@echo ""
	@echo "----------cross compile gobolt to Linux/ARM64..."
	@echo ""
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags="-s -w \
				-X 'common_tool/pkg/qqjson.GjsonVersion=$(GJSON_VER)' \
				-X 'common_tool/pkg/qqjson.PrettyVersion=$(PRETY_VER)' \
				-X 'common_tool/pkg/qqjson.SjsonVersion=$(SJSON_VER)'" \
				-o $(GOBOLT_ARM64) ./cmd/gobolt

# gobolt Linux/x86_64(amd64) cross compile (Pure Go, no CGO enabled)
$(GOBOLT_AMD64): tidy
	@echo ""
	@echo "----------cross compile gobolt to Linux/x86_64(amd64)..."
	@echo ""
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w \
			-X 'common_tool/pkg/qqjson.GjsonVersion=$(GJSON_VER)' \
			-X 'common_tool/pkg/qqjson.PrettyVersion=$(PRETY_VER)' \
			-X 'common_tool/pkg/qqjson.SjsonVersion=$(SJSON_VER)'" \
			-o $(GOBOLT_AMD64) ./cmd/gobolt

gobolt: $(GOBOLT) $(GOBOLT_AMD64) $(GOBOLT_ARM64)
	@echo ""
	@echo "----------gobolt build completed:"
	@echo ""
	@ls -lh $(GOBOLT) $(GOBOLT_AMD64) $(GOBOLT_ARM64) | $(HIGHLIGHT_LAST)

# Clean all generated files
clean:
	@echo ""
	@echo "----------Clean generated files..."
	@echo ""
	rm -rf bin/*
	rm -rf ./cmd/asciiplay/icon.*

