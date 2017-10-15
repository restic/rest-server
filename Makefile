.PHONY: default rest-server install uninstall clean

default: rest-server

rest-server:
	@go run build.go

install: rest-server
	sudo /usr/bin/install -m 755 rest-server /usr/local/bin/rest-server

uninstall:
	sudo rm -f /usr/local/bin/rest-server

clean:
	rm -f rest-server
