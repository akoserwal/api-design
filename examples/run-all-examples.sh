#!/bin/bash

# REST API Course - Run All Examples Script
# This script helps you run and test all the validation examples

set -e

echo "🚀 REST API Course - Code Examples Runner"
echo "========================================"
echo ""

# Function to check if a port is in use
check_port() {
    local port=$1
    if lsof -Pi :$port -sTCP:LISTEN -t >/dev/null 2>&1; then
        echo "⚠️  Port $port is already in use. Please stop the service or use a different port."
        return 1
    fi
    return 0
}

# Function to run example and wait for user input
run_example() {
    local lesson_dir=$1
    local port=$2
    local description=$3
    
    echo "📚 $description"
    echo "Directory: $lesson_dir"
    echo "Port: $port"
    echo ""
    
    if [ ! -d "$lesson_dir" ]; then
        echo "❌ Directory $lesson_dir not found!"
        return 1
    fi
    
    cd "$lesson_dir"
    
    if [ ! -f "go.mod" ]; then
        echo "❌ No go.mod found in $lesson_dir!"
        cd ..
        return 1
    fi
    
    echo "📦 Installing dependencies..."
    go mod download
    
    echo "🔨 Building..."
    if ! go build .; then
        echo "❌ Build failed!"
        cd ..
        return 1
    fi
    
    echo "✅ Build successful!"
    echo ""
    echo "🌐 Starting server on port $port..."
    echo "   Press Ctrl+C to stop the server"
    echo "   Then press Enter to continue to next example"
    echo ""
    
    # Start the server in the background
    go run . &
    SERVER_PID=$!
    
    # Wait for user to press Enter
    read -p "Press Enter when you're done testing this example..."
    
    # Kill the server
    kill $SERVER_PID 2>/dev/null || true
    wait $SERVER_PID 2>/dev/null || true
    
    cd ..
    echo ""
    echo "✅ Example completed!"
    echo ""
}

# Function to run tests
run_tests() {
    local lesson_dir=$1
    local description=$2
    
    echo "🧪 Running tests for: $description"
    echo "Directory: $lesson_dir"
    echo ""
    
    if [ ! -d "$lesson_dir" ]; then
        echo "❌ Directory $lesson_dir not found!"
        return 1
    fi
    
    cd "$lesson_dir"
    
    if [ ! -f "go.mod" ]; then
        echo "❌ No go.mod found in $lesson_dir!"
        cd ..
        return 1
    fi
    
    echo "🧪 Running tests..."
    if go test -v; then
        echo "✅ All tests passed!"
    else
        echo "❌ Some tests failed!"
    fi
    
    cd ..
    echo ""
}

# Main menu
show_menu() {
    echo "Select an option:"
    echo "1. Run Lesson 1: REST Concepts (Port 8080-8082)"
    echo "2. Run Lesson 2: HTTP Fundamentals (Port 8083-8086)"
    echo "3. Run Lesson 5: First REST API (Port 8087)"
    echo "4. Run All Examples Sequentially"
    echo "5. Test All Examples (Run Unit Tests)"
    echo "6. Quick Setup (Install all dependencies)"
    echo "7. Exit"
    echo ""
}

# Quick setup function
quick_setup() {
    echo "🔧 Quick Setup - Installing all dependencies..."
    echo ""
    
    for dir in lesson-*/; do
        if [ -d "$dir" ] && [ -f "$dir/go.mod" ]; then
            echo "📦 Setting up $dir..."
            cd "$dir"
            go mod download
            go mod tidy
            cd ..
            echo "✅ $dir setup complete!"
        fi
    done
    
    echo ""
    echo "✅ All dependencies installed!"
    echo ""
}

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo "❌ Go is not installed or not in PATH!"
    echo "Please install Go from https://golang.org/dl/"
    exit 1
fi

echo "✅ Go version: $(go version)"
echo ""

# Main loop
while true; do
    show_menu
    read -p "Enter your choice (1-7): " choice
    echo ""
    
    case $choice in
        1)
            echo "🎯 Running Lesson 1: REST Concepts Examples"
            echo ""
            
            echo "This lesson includes multiple servers:"
            echo "1. Richardson Maturity Model Demo (Port 8080)"
            echo "2. HATEOAS Example (Port 8081)"
            echo "3. REST Principles Demo (Port 8082)"
            echo ""
            
            read -p "Run Richardson Maturity Model Demo? (y/n): " run_levels
            if [[ $run_levels == "y" || $run_levels == "Y" ]]; then
                cd lesson-01-rest-concepts
                echo "🌐 Starting Richardson Maturity Model Demo on port 8080..."
                echo "Test commands:"
                echo "curl -X POST http://localhost:8080/level0 -d '{\"action\":\"getUsers\"}' -H 'Content-Type: application/json'"
                echo "curl http://localhost:8080/level2/users"
                echo ""
                go run rest-levels.go &
                SERVER_PID=$!
                read -p "Press Enter when done testing..."
                kill $SERVER_PID 2>/dev/null || true
                wait $SERVER_PID 2>/dev/null || true
                cd ..
                echo ""
            fi
            
            read -p "Run HATEOAS Example? (y/n): " run_hateoas
            if [[ $run_hateoas == "y" || $run_hateoas == "Y" ]]; then
                cd lesson-01-rest-concepts
                echo "🌐 Starting HATEOAS Example on port 8081..."
                echo "Start at: http://localhost:8081/"
                echo "Follow the _links to navigate the API"
                echo ""
                go run hateoas-example.go &
                SERVER_PID=$!
                read -p "Press Enter when done testing..."
                kill $SERVER_PID 2>/dev/null || true
                wait $SERVER_PID 2>/dev/null || true
                cd ..
                echo ""
            fi
            
            read -p "Run REST Principles Demo? (y/n): " run_principles
            if [[ $run_principles == "y" || $run_principles == "Y" ]]; then
                cd lesson-01-rest-concepts
                echo "🌐 Starting REST Principles Demo on port 8082..."
                echo "Try: curl http://localhost:8082/products"
                echo "Try: curl http://localhost:8082/products?category=Electronics"
                echo ""
                go run rest-principles.go &
                SERVER_PID=$!
                read -p "Press Enter when done testing..."
                kill $SERVER_PID 2>/dev/null || true
                wait $SERVER_PID 2>/dev/null || true
                cd ..
                echo ""
            fi
            ;;
            
        2)
            echo "🎯 Running Lesson 2: HTTP Fundamentals Examples"
            echo ""
            
            echo "This lesson includes multiple servers:"
            echo "1. HTTP Methods Demo (Port 8083)"
            echo "2. Status Codes Demo (Port 8084)"
            echo ""
            
            read -p "Run HTTP Methods Demo? (y/n): " run_methods
            if [[ $run_methods == "y" || $run_methods == "Y" ]]; then
                cd lesson-02-http-fundamentals
                echo "🌐 Starting HTTP Methods Demo on port 8083..."
                echo "Visit: http://localhost:8083/ for test commands"
                echo ""
                go run http-methods.go &
                SERVER_PID=$!
                read -p "Press Enter when done testing..."
                kill $SERVER_PID 2>/dev/null || true
                wait $SERVER_PID 2>/dev/null || true
                cd ..
                echo ""
            fi
            
            read -p "Run Status Codes Demo? (y/n): " run_status
            if [[ $run_status == "y" || $run_status == "Y" ]]; then
                cd lesson-02-http-fundamentals
                echo "🌐 Starting Status Codes Demo on port 8084..."
                echo "Try: curl http://localhost:8084/api/test/200"
                echo "Try: curl http://localhost:8084/api/test/404"
                echo ""
                go run status-codes.go &
                SERVER_PID=$!
                read -p "Press Enter when done testing..."
                kill $SERVER_PID 2>/dev/null || true
                wait $SERVER_PID 2>/dev/null || true
                cd ..
                echo ""
            fi
            ;;
            
        3)
            run_example "lesson-05-first-api" "8087" "Lesson 5: Complete Task Management REST API"
            ;;
            
        4)
            echo "🎯 Running All Examples Sequentially"
            echo ""
            echo "This will run each example one by one."
            echo "You can test each one and press Enter to continue to the next."
            echo ""
            read -p "Continue? (y/n): " continue_all
            if [[ $continue_all == "y" || $continue_all == "Y" ]]; then
                # Run all examples in sequence
                run_example "lesson-01-rest-concepts" "8080" "Lesson 1: REST Concepts - Richardson Maturity Model"
                run_example "lesson-02-http-fundamentals" "8083" "Lesson 2: HTTP Fundamentals - Methods Demo"
                run_example "lesson-05-first-api" "8087" "Lesson 5: Complete Task Management API"
            fi
            ;;
            
        5)
            echo "🧪 Running All Tests"
            echo ""
            run_tests "lesson-05-first-api" "Lesson 5: First REST API"
            echo "✅ All tests completed!"
            ;;
            
        6)
            quick_setup
            ;;
            
        7)
            echo "👋 Goodbye! Happy coding!"
            exit 0
            ;;
            
        *)
            echo "❌ Invalid choice. Please select 1-7."
            echo ""
            ;;
    esac
done