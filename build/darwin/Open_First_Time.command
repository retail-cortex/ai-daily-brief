#!/bin/bash
# Copyright 2026 Retail Cortex
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

echo "===================================================="
echo "⚡ AI Daily Brief - macOS Gatekeeper Helper"
echo "===================================================="
echo ""
echo "Unlocking quarantine permissions for AI Daily Brief..."

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"

# 1. Clear quarantine on /Applications if copied
if [ -d "/Applications/AI-Daily-Brief.app" ]; then
    xattr -cr "/Applications/AI-Daily-Brief.app" 2>/dev/null
    echo "✓ Cleared quarantine on /Applications/AI-Daily-Brief.app"
fi

# 2. Clear quarantine on ~/Applications if copied there
if [ -d "$HOME/Applications/AI-Daily-Brief.app" ]; then
    xattr -cr "$HOME/Applications/AI-Daily-Brief.app" 2>/dev/null
    echo "✓ Cleared quarantine on $HOME/Applications/AI-Daily-Brief.app"
fi

# 3. Clear quarantine in current directory
if [ -d "$DIR/AI-Daily-Brief.app" ]; then
    xattr -cr "$DIR/AI-Daily-Brief.app" 2>/dev/null
    echo "✓ Cleared quarantine on $DIR/AI-Daily-Brief.app"
fi

echo ""
echo "🚀 Launching AI Daily Brief..."
if [ -d "/Applications/AI-Daily-Brief.app" ]; then
    open "/Applications/AI-Daily-Brief.app"
elif [ -d "$HOME/Applications/AI-Daily-Brief.app" ]; then
    open "$HOME/Applications/AI-Daily-Brief.app"
elif [ -d "$DIR/AI-Daily-Brief.app" ]; then
    open "$DIR/AI-Daily-Brief.app"
fi

echo "Done! You can close this terminal window."
