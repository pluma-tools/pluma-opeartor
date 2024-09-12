#!/bin/bash

set -eu

buf generate --timeout 10m -v \
  --path operator
