# /bin/bash


CMD=$1

case $CMD in
    "lint")
        golangci-lint run
    ;;
    "test")
        go test ./manager/*.go
    ;;
    *)
        echo "commend $CMD not supported"
    ;;
esac