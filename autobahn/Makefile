
.PHONY: test
.DEFAULT_GOAL:=test

name?=local

log?=0

autobahn_path=.
cfg_path=$(autobahn_path)/config
reports_path=$(autobahn_path)/reports

cfg_base=fuzzingclient.base.json
cfg_fast=fuzzingclient.fast.json
cfg_temp=fuzzingclient.temp.json

suite?=fast

ifeq ($(suite),fast)
cfg=$(cfg_fast)
endif
ifeq ($(suite),full)
cfg=$(cfg_base)
endif
ifndef cfg
cfg=$(cfg_temp)
$(shell jq '.cases = ["$(suite)"]' $(cfg_path)/$(cfg_base) > $(cfg_path)/$(cfg)) 
endif

test:
	docker run -it --rm -v ${PWD}/$(cfg_path):/config \
		--add-host=host.docker.internal:host-gateway \
		-v ${PWD}/$(reports_path):/reports crossbario/autobahn-testsuite \
		wstest -m fuzzingclient -s /config/$(cfg)
	@rm -f $(cfg_path)/$(cfg_temp)
