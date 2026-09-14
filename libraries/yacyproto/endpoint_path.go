package yacyproto

type EndpointMethodSet uint8

const (
	EndpointMethodGet EndpointMethodSet = 1 << iota
	EndpointMethodPost
)

const (
	PathHello       = "/yacy/hello.html"
	PathTransferRWI = "/yacy/transferRWI.html"
	PathTransferURL = "/yacy/transferURL.html"
	PathSearch      = "/yacy/search.html"
	PathURLMetadata = "/yacy/urls.xml"
	PathQuery       = "/yacy/query.html"
)

const (
	EndpointMethodsGetPost = EndpointMethodGet | EndpointMethodPost
	EndpointMethodsPost    = EndpointMethodPost
)

const (
	HelloEndpointMethods       = EndpointMethodsGetPost
	TransferRWIEndpointMethods = EndpointMethodsPost
	TransferURLEndpointMethods = EndpointMethodsPost
	SearchEndpointMethods      = EndpointMethodsGetPost
	URLMetadataEndpointMethods = EndpointMethodsGetPost
	QueryEndpointMethods       = EndpointMethodsGetPost
)
