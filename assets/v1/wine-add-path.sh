#!/bin/bash
set -e
NEW_ENTRY="$1"
KEY="HKEY_CURRENT_USER\Environment"
CURRENT=$(wine reg query "$KEY" /v PATH 2>/dev/null | tr -d "\r" | grep "REG_EXPAND_SZ" | sed "s/^.*REG_EXPAND_SZ\s*//")

if [ -z "$CURRENT" ]; then
  FINAL="$NEW_ENTRY"
else
  FINAL="$CURRENT;$NEW_ENTRY"
fi

wine reg add "$KEY" /v PATH /t REG_EXPAND_SZ /d "$FINAL" /f
wineserver -w
