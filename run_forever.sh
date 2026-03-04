#!/bin/bash
set -x
main() {
  logfile="output_$(date +%Y%m%d_%H%M%S).log"
  ./autozap 2>&1 | tee "$logfile"
  if [ $? -eq 0 ]; then
    rm -f "$logfile"
  fi
  sleep 60
  main
}

main
