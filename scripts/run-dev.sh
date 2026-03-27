#!/bin/bash
cd "$(dirname "$0")/.."
GMESSAGE_LOG_LEVEL=debug go run .
