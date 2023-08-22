#!/bin/bash

GOLINES_OUT="$(golines -l .)"
if [ -n "$GOLINES_OUT" ]; then
  echo "golines is failing"
  $GOLINES_OUT
  exit 1
fi