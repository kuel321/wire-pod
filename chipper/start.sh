#!/bin/bash

UNAME=$(uname -a)

if [[ $EUID -ne 0 ]]; then
  echo "This script must be run as root. Run: sudo ./start.sh"
  exit 1
fi

# Enter chipper directory
if [[ -d ./chipper ]]; then
  cd chipper
fi

# Ensure source.sh exists
if [[ ! -f ./source.sh ]]; then
  echo "You need to create a source.sh file. Run setup.sh first."
  exit 1
fi

source source.sh

echo "Deleting old chipper binary..."
rm -f ./chipper

echo "Building new chipper binary..."
BUILD_CMD="go build -o chipper"

###############################################
# SELECT ENTRYPOINT BASED ON STT_SERVICE
###############################################

case "${STT_SERVICE}" in

  "leopard")
    MAIN_FILE="cmd/leopard/main.go"
    ;;

  "rhino")
    MAIN_FILE="cmd/experimental/rhino/main.go"
    ;;

  "houndify")
    MAIN_FILE="cmd/experimental/houndify/main.go"
    ;;

  "whisper")
    MAIN_FILE="cmd/experimental/whisper/main.go"
    ;;

  "whisper.cpp")
    export C_INCLUDE_PATH="../whisper.cpp"
    export LIBRARY_PATH="../whisper.cpp"

    MAIN_FILE="cmd/experimental/whisper.cpp/main.go"

    # macOS Metal flags
    if [[ ${UNAME} == *"Darwin"* ]]; then
      export GGML_METAL_PATH_RESOURCES="../whisper.cpp"
      BUILD_CMD="go build -ldflags \"-extldflags '-framework Foundation -framework Metal -framework MetalKit'\" -o chipper"
    fi
    ;;

  "vosk")
    export CGO_ENABLED=1
    export CGO_CFLAGS="-I$HOME/.vosk/libvosk"
    export CGO_LDFLAGS="-L$HOME/.vosk/libvosk -lvosk -ldl -lpthread"
    export LD_LIBRARY_PATH="$HOME/.vosk/libvosk:$LD_LIBRARY_PATH"

    MAIN_FILE="cmd/vosk/main.go"
    ;;

  *)
    # Default = coqui
    export CGO_LDFLAGS="-L$HOME/.coqui/"
    export CGO_CXXFLAGS="-I$HOME/.coqui/"
    export LD_LIBRARY_PATH="$HOME/.coqui/:$LD_LIBRARY_PATH"

    MAIN_FILE="cmd/coqui/main.go"
    ;;
esac

###############################################
# BUILD AND RUN
###############################################

echo "Building using: $MAIN_FILE"
eval $BUILD_CMD "$MAIN_FILE"

if [[ $? -ne 0 ]]; then
  echo "❌ Build failed."
  exit 1
fi

echo "✅ Build successful. Starting chipper..."
./chipper
