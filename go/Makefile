
.PHONY: autobahn

autobahn:
	docker run -it --rm -v ${PWD}/config:/config \
		-v ${PWD}/reports:/reports crossbario/autobahn-testsuite \
		wstest -m fuzzingclient -s /config/fuzzingclient.json
