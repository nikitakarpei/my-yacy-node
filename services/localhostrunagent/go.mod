module github.com/nikitakarpei/yacy-rwi-node/localhostrunagent

go 1.27

require (
	github.com/nikitakarpei/yacy-rwi-node/processenvironmentlease v0.0.0
	github.com/nikitakarpei/yacy-rwi-node/serviceruntime v0.0.0
	github.com/pires/go-proxyproto v0.15.0
)

replace github.com/nikitakarpei/yacy-rwi-node/processenvironmentlease => ../../libraries/processenvironmentlease

replace github.com/nikitakarpei/yacy-rwi-node/serviceruntime => ../../libraries/serviceruntime
