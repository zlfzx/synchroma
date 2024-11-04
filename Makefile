APP_NAME := synchroma
VERSION := 1.0.0
BUILD_DIR := bin

# List of supported architectures
ARCHITECTURES := linux/amd64 linux/arm64 darwin/amd64 darwin/arm64

# Default target
all: clean build

# Clean up build directory
clean:
	rm -rf $(BUILD_DIR)

# Build the application
build:
	@mkdir -p $(BUILD_DIR)
	@for arch in $(ARCHITECTURES); do \
		GOOS=$$(echo $$arch | cut -d'/' -f1) ; \
		GOARCH=$$(echo $$arch | cut -d'/' -f2) ; \
		OUTPUT=$(BUILD_DIR)/$(APP_NAME)-$$GOOS-$$GOARCH ; \
		if [ $$GOOS = "windows" ]; then OUTPUT=$$OUTPUT.exe; fi ; \
		echo "Building for $$GOOS/$$GOARCH..." ; \
		GOOS=$$GOOS GOARCH=$$GOARCH go build -ldflags="-s -w" -o $$OUTPUT . ; \
	done

# Package the application
package: build
	@cd $(BUILD_DIR) && for f in *; do \
		tar -czvf "$$f-$(VERSION).tar.gz" "$$f" && rm "$$f" ; \
	done
