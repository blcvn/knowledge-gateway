#!/bin/bash
for file in $(find ../services -name "main.go" | grep "cmd/server"); do
  if ! grep -q "RegisterForwardService" "$file"; then
    echo "Patching $file"
    
    # Add import if not exists
    if ! grep -q "\"vnp-memory/pkg/forward\"" "$file"; then
      awk '/import \(/ { print; print "\t\"vnp-memory/pkg/forward\""; next }1' "$file" > tmp.go && mv tmp.go "$file"
    fi
    
    # Add router registration before healthCheck or before net.Listen
    awk '/healthCheck := health.NewServer\(\)/ {
      print "\t// Setup ForwardService Router"
      print "\trouter := forward.NewRouter()"
      print "\tforward.RegisterForwardService(grpcServer, router)"
      print ""
      print $0
      next
    }1' "$file" > tmp.go && mv tmp.go "$file"
  fi
done
