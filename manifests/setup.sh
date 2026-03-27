#!/bin/bash

# Script to setup Kubernetes manifests from examples

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo "🚀 Setting up Kubernetes manifests..."

# Check if files already exist
if [ -f "$SCRIPT_DIR/configmap.yaml" ]; then
    read -p "⚠️  configmap.yaml already exists. Overwrite? (y/N): " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        echo "Skipping configmap.yaml"
    else
        cp "$SCRIPT_DIR/configmap.yaml.example" "$SCRIPT_DIR/configmap.yaml"
        echo "✅ Created configmap.yaml from example"
    fi
else
    cp "$SCRIPT_DIR/configmap.yaml.example" "$SCRIPT_DIR/configmap.yaml"
    echo "✅ Created configmap.yaml from example"
fi

if [ -f "$SCRIPT_DIR/secret.yaml" ]; then
    read -p "⚠️  secret.yaml already exists. Overwrite? (y/N): " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        echo "Skipping secret.yaml"
    else
        cp "$SCRIPT_DIR/secret.yaml.example" "$SCRIPT_DIR/secret.yaml"
        echo "✅ Created secret.yaml from example"
    fi
else
    cp "$SCRIPT_DIR/secret.yaml.example" "$SCRIPT_DIR/secret.yaml"
    echo "✅ Created secret.yaml from example"
fi

echo ""
echo "📝 Next steps:"
echo "1. Edit manifests/configmap.yaml and replace CHANGE_ME values"
echo "2. Edit manifests/secret.yaml and replace CHANGE_ME values"
echo "3. Run: kubectl apply -f manifests/configmap.yaml"
echo "4. Run: kubectl apply -f manifests/secret.yaml"
echo ""
echo "⚠️  Remember: Never commit configmap.yaml or secret.yaml to Git!"
