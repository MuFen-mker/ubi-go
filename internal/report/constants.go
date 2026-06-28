package report

const (
	ubiServicesUrl = "https://public-ubiservices.ubi.com"
	reportUrl      = "v1/applications/global/cshelp/cases/api/player-report-cases"

	// Ubisoft player-report-case constants.
	// NOTE: productInstallmentId is game-specific and is supplied per request
	// by the caller (see ReportPayload), so it is intentionally not constant here.
	ubiCategoryId  = "334"
	requestType    = "334"
	caseLocale     = "en-gb"
	contactChannel = "Email"
	caseOrigin     = "API"
)

// platformIDMapping maps a platform slug to its Ubisoft platform id.
var platformIDMapping = map[string]string{
	"uplay": "9",
	"psn":   "47",
	"xbl":   "43",
}
