package routing

import (
	"net/url"
	"strings"
)

// SEP7PayParams represents parsed parameters from a SEP-0007 pay URI.
type SEP7PayParams struct {
	Destination       string `json:"destination"`
	Amount            string `json:"amount,omitempty"`
	AssetCode         string `json:"assetCode,omitempty"`
	AssetIssuer       string `json:"assetIssuer,omitempty"`
	Memo              string `json:"memo,omitempty"`
	MemoType          string `json:"memoType,omitempty"`
	Callback          string `json:"callback,omitempty"`
	Msg               string `json:"msg,omitempty"`
	NetworkPassphrase string `json:"networkPassphrase,omitempty"`
	OriginDomain      string `json:"originDomain,omitempty"`
	Signature         string `json:"signature,omitempty"`
}

// Error codes for SEP-0007 URI parsing.
const (
	ErrCodeInvalidURI            = "INVALID_URI"
	ErrCodeUnsupportedOperation = "UNSUPPORTED_OPERATION"
	ErrCodeMissingDestination    = "MISSING_DESTINATION"
	ErrCodeInvalidEncoding       = "INVALID_ENCODING"
)

// ExtractRoutingFromURIResult represents the result of parsing a SEP-0007 URI.
type ExtractRoutingFromURIResult struct {
	Success   bool          `json:"success"`
	Routing   RoutingResult `json:"routing,omitempty"`
	RawParams SEP7PayParams `json:"rawParams,omitempty"`
	Error     string        `json:"error,omitempty"`
	Code      string        `json:"code,omitempty"`
}

// mapMemoType maps SEP-0007 memo_type values to internal KnownMemoType ("none", "id", "text", "hash", "return").
func mapMemoType(sep7MemoType string) string {
	if sep7MemoType == "" {
		return "none"
	}

	upper := strings.ToUpper(sep7MemoType)
	switch upper {
	case "MEMO_ID":
		return "id"
	case "MEMO_TEXT":
		return "text"
	case "MEMO_HASH":
		return "hash"
	case "MEMO_RETURN":
		return "return"
	default:
		return "none"
	}
}

// safelyDecode decodes a URL percent-encoded string.
// If decoding fails, it falls back to the raw value.
func safelyDecode(val string) string {
	if val == "" {
		return ""
	}
	decoded, err := url.QueryUnescape(val)
	if err != nil {
		return val
	}
	return decoded
}

// ExtractRoutingFromURI parses a SEP-0007 URI and extracts canonical routing information.
// Supported format:
//
//	web+stellar:pay?destination=<G...>&memo=<value>&memo_type=<MEMO_TEXT|MEMO_ID|MEMO_HASH|MEMO_RETURN>
func ExtractRoutingFromURI(uriString string) ExtractRoutingFromURIResult {
	// 1. Validate scheme
	const schemePrefix = "web+stellar:"
	if !strings.HasPrefix(uriString, schemePrefix) {
		return ExtractRoutingFromURIResult{
			Success: false,
			Error:   "URI must use 'web+stellar:' scheme",
			Code:    ErrCodeInvalidURI,
		}
	}

	// 2. Split operation and query string
	withoutScheme := uriString[len(schemePrefix):]
	var operation, queryString string
	if qIdx := strings.IndexByte(withoutScheme, '?'); qIdx != -1 {
		operation = withoutScheme[:qIdx]
		queryString = withoutScheme[qIdx+1:]
	} else {
		operation = withoutScheme
		queryString = ""
	}

	// 3. Only 'pay' operation is supported for routing extraction
	if operation != "pay" {
		return ExtractRoutingFromURIResult{
			Success: false,
			Error:   "Unsupported operation: '" + operation + "'. Only 'pay' is supported for routing extraction.",
			Code:    ErrCodeUnsupportedOperation,
		}
	}

	// 4. Parse query parameters safely
	// Note: Parse query string manually or via url.ParseQuery.
	// In Go, url.ParseQuery returns an error if any parameter contains invalid percent encoding (%ZZ).
	// But SEP-0007 requirements state we should handle malformed encoding gracefully without failing the entire parse
	// when possible, or safely parse parameters.
	// Let's parse query parameters splitting on '&', then '='.
	paramMap := make(map[string]string)
	if queryString != "" {
		pairs := strings.Split(queryString, "&")
		for _, pair := range pairs {
			if pair == "" {
				continue
			}
			var key, val string
			if eqIdx := strings.IndexByte(pair, '='); eqIdx != -1 {
				key = pair[:eqIdx]
				val = pair[eqIdx+1:]
			} else {
				key = pair
				val = ""
			}
			// Only take the first occurrence if multiple, or set if not present
			if _, exists := paramMap[key]; !exists {
				paramMap[key] = val
			}
		}
	}

	// 5. Extract required destination
	rawDest, destExists := paramMap["destination"]
	decodedDest := safelyDecode(rawDest)
	trimmedDest := strings.TrimSpace(decodedDest)
	if !destExists || trimmedDest == "" {
		return ExtractRoutingFromURIResult{
			Success: false,
			Error:   "Missing required 'destination' parameter",
			Code:    ErrCodeMissingDestination,
		}
	}

	// 6. Extract optional parameters with safe decoding
	rawParams := SEP7PayParams{
		Destination: trimmedDest,
	}

	if v, ok := paramMap["amount"]; ok && v != "" {
		rawParams.Amount = safelyDecode(v)
	}
	if v, ok := paramMap["asset_code"]; ok && v != "" {
		rawParams.AssetCode = safelyDecode(v)
	}
	if v, ok := paramMap["asset_issuer"]; ok && v != "" {
		rawParams.AssetIssuer = safelyDecode(v)
	}
	if v, ok := paramMap["memo"]; ok && v != "" {
		rawParams.Memo = safelyDecode(v)
	}
	if v, ok := paramMap["memo_type"]; ok && v != "" {
		rawParams.MemoType = safelyDecode(v)
	}
	if v, ok := paramMap["callback"]; ok && v != "" {
		rawParams.Callback = safelyDecode(v)
	}
	if v, ok := paramMap["msg"]; ok && v != "" {
		rawParams.Msg = safelyDecode(v)
	}
	if v, ok := paramMap["network_passphrase"]; ok && v != "" {
		rawParams.NetworkPassphrase = safelyDecode(v)
	}
	if v, ok := paramMap["origin_domain"]; ok && v != "" {
		rawParams.OriginDomain = safelyDecode(v)
	}
	if v, ok := paramMap["signature"]; ok && v != "" {
		rawParams.Signature = safelyDecode(v)
	}

	// 7. Build RoutingInput for ExtractRouting
	routingInput := RoutingInput{
		Destination:   rawParams.Destination,
		MemoType:      mapMemoType(rawParams.MemoType),
		MemoValue:     rawParams.Memo,
		SourceAccount: "",
	}

	// 8. Delegate to core ExtractRouting logic
	routingResult := ExtractRouting(routingInput)

	// 9. Return combined result
	return ExtractRoutingFromURIResult{
		Success:   true,
		Routing:   routingResult,
		RawParams: rawParams,
	}
}
