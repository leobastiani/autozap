#!/bin/bash

main() {
  logfile="output_$(date +%Y%m%d_%H%M%S).log"
  autozap >"$logfile" 2>&1
  if [ $? -eq 0 ]; then
    rm -f "$logfile"
  fi
  sleep 600
  main
}

main
