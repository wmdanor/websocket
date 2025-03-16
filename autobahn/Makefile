
.PHONY: test
.DEFAULT_GOAL:=test

name?=local

log?=0

autobahn_path=.
cfg_path=$(autobahn_path)/config
reports_path=$(autobahn_path)/reports

mode?=client
fullmode=fuzzing$(mode)

ifeq ($(mode),server)
portarg=-p 9001:9001
endif

cfg_base=$(fullmode).base.json
cfg_fast=$(fullmode).fast.json
cfg_temp=$(fullmode).temp.json

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
	docker run -it --rm \
		--add-host=host.docker.internal:host-gateway \
		-v ${PWD}/$(cfg_path):/config \
		-v ${PWD}/$(reports_path):/reports \
		$(portarg) \
		crossbario/autobahn-testsuite \
		wstest -m $(fullmode) -s /config/$(cfg)
	@rm -f $(cfg_path)/$(cfg_temp)
