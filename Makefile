APP_NAME := synchroma
VERSION := 0.4.0
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
		GOOS=$$GOOS GOARCH=$$GOARCH go build -ldflags="-s -w -X synchroma/cmd.Version=$(VERSION)" -o $$OUTPUT . ; \
	done

# Package the application
package: clean build
	@cd $(BUILD_DIR) && for f in *; do \
		mv "$$f" "$(APP_NAME)" ; \
		tar -czvf "$$f-$(VERSION).tar.gz" "$(APP_NAME)" && rm "$(APP_NAME)" ; \
	done
