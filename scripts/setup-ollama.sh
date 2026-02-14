#!/bin/bash
# Nephoran Intent Operator - Ollama Setup Script
# This script helps set up Ollama with recommended models

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

echo "╔════════════════════════════════════════════════════════════╗"
echo "║   Nephoran Intent Operator - Ollama Setup                 ║"
echo "╚════════════════════════════════════════════════════════════╝"
echo ""

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Functions
print_info() {
    echo -e "${BLUE}ℹ${NC} $1"
}

print_success() {
    echo -e "${GREEN}✓${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}⚠${NC} $1"
}

print_error() {
    echo -e "${RED}✗${NC} $1"
}

# Check if Ollama is installed
check_ollama() {
    if command -v ollama &> /dev/null; then
        print_success "Ollama is installed ($(ollama --version))"
        return 0
    else
        print_error "Ollama is not installed"
        return 1
    fi
}

# Install Ollama
install_ollama() {
    print_info "Installing Ollama..."

    if [[ "$OSTYPE" == "linux-gnu"* ]]; then
        curl -fsSL https://ollama.com/install.sh | sh
    elif [[ "$OSTYPE" == "darwin"* ]]; then
        print_info "Please download Ollama from: https://ollama.com/download/mac"
        print_info "Or run: brew install ollama"
        exit 1
    else
        print_error "Unsupported OS: $OSTYPE"
        exit 1
    fi

    print_success "Ollama installed successfully"
}

# Start Ollama service
start_ollama() {
    print_info "Starting Ollama service..."

    # Check if already running
    if curl -s http://localhost:11434/api/tags &> /dev/null; then
        print_success "Ollama is already running"
        return 0
    fi

    # Start Ollama in background
    ollama serve &> /tmp/ollama.log &
    OLLAMA_PID=$!

    # Wait for Ollama to be ready
    print_info "Waiting for Ollama to start..."
    for i in {1..30}; do
        if curl -s http://localhost:11434/api/tags &> /dev/null; then
            print_success "Ollama is running (PID: $OLLAMA_PID)"
            return 0
        fi
        sleep 1
    done

    print_error "Ollama failed to start"
    return 1
}

# Download recommended model
download_model() {
    local model=$1

    print_info "Checking if model '$model' exists..."

    if ollama list | grep -q "$model"; then
        print_success "Model '$model' is already downloaded"
        return 0
    fi

    print_info "Downloading model '$model'..."
    print_warning "This may take a few minutes depending on your internet speed"

    ollama pull "$model"

    if [ $? -eq 0 ]; then
        print_success "Model '$model' downloaded successfully"
        return 0
    else
        print_error "Failed to download model '$model'"
        return 1
    fi
}

# Test model
test_model() {
    local model=$1

    print_info "Testing model '$model'..."

    local test_prompt="Deploy AMF with 3 replicas in namespace 5g-core. Output as JSON."

    echo ""
    print_info "Prompt: $test_prompt"
    echo ""

    ollama run "$model" "$test_prompt"

    echo ""
    print_success "Model test completed"
}

# Create .env file
create_env_file() {
    local model=$1
    local env_file="$PROJECT_ROOT/.env"

    if [ -f "$env_file" ]; then
        print_warning ".env file already exists"
        read -p "Do you want to overwrite it? (y/N): " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            print_info "Skipping .env file creation"
            return 0
        fi
    fi

    print_info "Creating .env file..."

    cp "$PROJECT_ROOT/.env.ollama.example" "$env_file"

    # Update model in .env
    sed -i.bak "s/OLLAMA_MODEL=.*/OLLAMA_MODEL=$model/" "$env_file"
    rm -f "$env_file.bak"

    print_success ".env file created at $env_file"
    print_info "You can edit this file to customize your configuration"
}

# Main menu
show_menu() {
    echo ""
    echo "Select a model to download:"
    echo ""
    echo "1) llama2:7b       - Llama 2 7B (Recommended for development, ~4GB)"
    echo "2) mistral:7b      - Mistral 7B (Better quality, ~4GB)"
    echo "3) llama2:13b      - Llama 2 13B (Production quality, ~8GB)"
    echo "4) codellama:7b    - Code Llama 7B (Code-focused, ~4GB)"
    echo "5) Custom model    - Enter custom model name"
    echo "6) Skip            - Skip model download"
    echo ""
    read -p "Enter your choice (1-6): " choice

    case $choice in
        1) echo "llama2:7b" ;;
        2) echo "mistral:7b" ;;
        3) echo "llama2:13b" ;;
        4) echo "codellama:7b" ;;
        5)
            read -p "Enter custom model name: " custom_model
            echo "$custom_model"
            ;;
        6) echo "" ;;
        *)
            print_error "Invalid choice"
            exit 1
            ;;
    esac
}

# Main script
main() {
    echo ""
    print_info "Step 1: Checking Ollama installation"

    if ! check_ollama; then
        read -p "Do you want to install Ollama now? (Y/n): " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Nn]$ ]]; then
            install_ollama
        else
            print_error "Ollama is required. Exiting."
            exit 1
        fi
    fi

    echo ""
    print_info "Step 2: Starting Ollama service"

    if ! start_ollama; then
        print_error "Failed to start Ollama. Check logs at /tmp/ollama.log"
        exit 1
    fi

    echo ""
    print_info "Step 3: Model selection"

    MODEL=$(show_menu)

    if [ -n "$MODEL" ]; then
        echo ""
        print_info "Step 4: Downloading model"

        if download_model "$MODEL"; then
            echo ""
            print_info "Step 5: Testing model"
            read -p "Do you want to test the model now? (Y/n): " -n 1 -r
            echo
            if [[ ! $REPLY =~ ^[Nn]$ ]]; then
                test_model "$MODEL"
            fi

            echo ""
            print_info "Step 6: Creating configuration"
            create_env_file "$MODEL"
        fi
    else
        print_info "Skipping model download"
    fi

    echo ""
    print_success "Setup completed!"
    echo ""
    print_info "Next steps:"
    echo "  1. Review and adjust .env file if needed"
    echo "  2. Start services: docker-compose -f docker-compose.ollama.yml up -d"
    echo "  3. Or run locally: cd rag-python && uvicorn api:app --reload"
    echo "  4. Access Swagger UI: http://localhost:8000/docs"
    echo ""
    print_info "For more information, see: docs/OLLAMA_INTEGRATION.md"
    echo ""
}

# Run main
main
