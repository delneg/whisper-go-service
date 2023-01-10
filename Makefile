BUILD_DIR := build
MODELS_DIR := models
EXAMPLES_DIR := $(wildcard examples/*)
INCLUDE_PATH := $(abspath ./libs)
LIBRARY_PATH := $(abspath ./libs)

app: mkdir modtidy
	@echo Build $(notdir $@)
	@C_INCLUDE_PATH=${INCLUDE_PATH} LIBRARY_PATH=${LIBRARY_PATH} go build ${BUILD_FLAGS} -o ${BUILD_DIR}/app .

mkdir:
	@echo Mkdir ${BUILD_DIR}
	@install -d ${BUILD_DIR}
	@echo Mkdir ${MODELS_DIR}
	@install -d ${MODELS_DIR}

modtidy:
	@go mod tidy

clean: 
	@echo Clean
	@rm -fr $(BUILD_DIR)
	@go clean
