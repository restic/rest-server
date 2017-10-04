.PHONY: default rest-server clean

default: rest-server

rest-server:
	go run build.go

clean:
	rm -f rest-server
