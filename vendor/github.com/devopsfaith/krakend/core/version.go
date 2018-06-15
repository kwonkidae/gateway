package core

import "fmt"

// KrakendHeaderName is the name of the custom KrakenD header
const KrakendHeaderName = "Lawtalk-Gateway"

// KrakendVersion is the version of the build
var KrakendVersion = "1.0"

// KrakendHeaderValue is the value of the custom KrakenD header
var KrakendHeaderValue = fmt.Sprintf("Version %s", KrakendVersion)

// KrakendUserAgent is the value of the user agent header sent to the backends
var KrakendUserAgent = fmt.Sprintf("%s Version %s", KrakendHeaderName, KrakendVersion)
