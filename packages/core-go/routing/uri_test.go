package routing

import (
	"strings"
	"testing"
)

func TestExtractRoutingFromURI_SchemeValidation(t *testing.T) {
	t.Run("rejects URIs without web+stellar scheme", func(t *testing.T) {
		result := ExtractRoutingFromURI("https://example.com/pay?destination=G...")
		if result.Success {
			t.Fatalf("expected failure, got success")
		}
		if result.Code != ErrCodeInvalidURI {
			t.Errorf("expected code %s, got %s", ErrCodeInvalidURI, result.Code)
		}
	})

	t.Run("rejects empty string", func(t *testing.T) {
		result := ExtractRoutingFromURI("")
		if result.Success {
			t.Fatalf("expected failure, got success")
		}
		if result.Code != ErrCodeInvalidURI {
			t.Errorf("expected code %s, got %s", ErrCodeInvalidURI, result.Code)
		}
	})
}

func TestExtractRoutingFromURI_OperationValidation(t *testing.T) {
	t.Run("rejects unsupported 'tx' operation", func(t *testing.T) {
		result := ExtractRoutingFromURI("web+stellar:tx?xdr=AAAA...")
		if result.Success {
			t.Fatalf("expected failure, got success")
		}
		if result.Code != ErrCodeUnsupportedOperation {
			t.Errorf("expected code %s, got %s", ErrCodeUnsupportedOperation, result.Code)
		}
	})

	t.Run("rejects unknown operations", func(t *testing.T) {
		result := ExtractRoutingFromURI("web+stellar:unknown?destination=G...")
		if result.Success {
			t.Fatalf("expected failure, got success")
		}
		if result.Code != ErrCodeUnsupportedOperation {
			t.Errorf("expected code %s, got %s", ErrCodeUnsupportedOperation, result.Code)
		}
	})
}

func TestExtractRoutingFromURI_DestinationValidation(t *testing.T) {
	t.Run("rejects missing destination", func(t *testing.T) {
		result := ExtractRoutingFromURI("web+stellar:pay?amount=100")
		if result.Success {
			t.Fatalf("expected failure, got success")
		}
		if result.Code != ErrCodeMissingDestination {
			t.Errorf("expected code %s, got %s", ErrCodeMissingDestination, result.Code)
		}
	})

	t.Run("rejects empty destination", func(t *testing.T) {
		result := ExtractRoutingFromURI("web+stellar:pay?destination=")
		if result.Success {
			t.Fatalf("expected failure, got success")
		}
		if result.Code != ErrCodeMissingDestination {
			t.Errorf("expected code %s, got %s", ErrCodeMissingDestination, result.Code)
		}
	})
}

func TestExtractRoutingFromURI_GAddress(t *testing.T) {
	const validG = "GCALNQQBXAPZ2WIRSDDBMSTAKCUH5SG6U76YBFLQLIXJTF7FE5AX7AOO"

	t.Run("parses basic pay URI with destination only", func(t *testing.T) {
		result := ExtractRoutingFromURI("web+stellar:pay?destination=" + validG)
		if !result.Success {
			t.Fatalf("expected success, got error: %s", result.Error)
		}
		if result.RawParams.Destination != validG {
			t.Errorf("expected destination %s, got %s", validG, result.RawParams.Destination)
		}
		if result.RawParams.Memo != "" {
			t.Errorf("expected empty memo, got %s", result.RawParams.Memo)
		}
		if result.RawParams.MemoType != "" {
			t.Errorf("expected empty memoType, got %s", result.RawParams.MemoType)
		}
		if result.Routing.DestinationBaseAccount != validG {
			t.Errorf("expected routing destinationBaseAccount %s, got %s", validG, result.Routing.DestinationBaseAccount)
		}
	})

	t.Run("parses URI with memo and memo_type", func(t *testing.T) {
		uri := "web+stellar:pay?destination=" + validG + "&memo=123&memo_type=MEMO_ID"
		result := ExtractRoutingFromURI(uri)
		if !result.Success {
			t.Fatalf("expected success, got error: %s", result.Error)
		}
		if result.RawParams.Memo != "123" {
			t.Errorf("expected memo 123, got %s", result.RawParams.Memo)
		}
		if result.RawParams.MemoType != "MEMO_ID" {
			t.Errorf("expected memoType MEMO_ID, got %s", result.RawParams.MemoType)
		}
		if result.Routing.RoutingID == nil || result.Routing.RoutingID.String() != "123" {
			t.Errorf("expected routingId 123, got %v", result.Routing.RoutingID)
		}
	})

	t.Run("parses URI with URL-encoded memo text", func(t *testing.T) {
		uri := "web+stellar:pay?destination=" + validG + "&memo=hello%20world&memo_type=MEMO_TEXT"
		result := ExtractRoutingFromURI(uri)
		if !result.Success {
			t.Fatalf("expected success, got error: %s", result.Error)
		}
		if result.RawParams.Memo != "hello world" {
			t.Errorf("expected memo 'hello world', got '%s'", result.RawParams.Memo)
		}
		if result.RawParams.MemoType != "MEMO_TEXT" {
			t.Errorf("expected memoType MEMO_TEXT, got '%s'", result.RawParams.MemoType)
		}
	})

	t.Run("parses URI with all optional parameters", func(t *testing.T) {
		const issuer = "GAP5LETOV6YIE62YAM56STDANPRDO7ZFDBGSNHJQIYGGKSMOZAHOOS2S"
		uri := "web+stellar:pay?destination=" + validG +
			"&amount=100.1234567" +
			"&asset_code=USDC" +
			"&asset_issuer=" + issuer +
			"&memo=invoice%23123" +
			"&memo_type=MEMO_TEXT" +
			"&callback=url%3Ahttps%3A%2F%2Fexample.com%2Fcallback" +
			"&msg=Pay%20me%20with%20lumens" +
			"&origin_domain=example.com"

		result := ExtractRoutingFromURI(uri)
		if !result.Success {
			t.Fatalf("expected success, got error: %s", result.Error)
		}
		if result.RawParams.Destination != validG {
			t.Errorf("destination mismatch: %s", result.RawParams.Destination)
		}
		if result.RawParams.Amount != "100.1234567" {
			t.Errorf("amount mismatch: %s", result.RawParams.Amount)
		}
		if result.RawParams.AssetCode != "USDC" {
			t.Errorf("assetCode mismatch: %s", result.RawParams.AssetCode)
		}
		if result.RawParams.AssetIssuer != issuer {
			t.Errorf("assetIssuer mismatch: %s", result.RawParams.AssetIssuer)
		}
		if result.RawParams.Memo != "invoice#123" {
			t.Errorf("memo mismatch: %s", result.RawParams.Memo)
		}
		if result.RawParams.MemoType != "MEMO_TEXT" {
			t.Errorf("memoType mismatch: %s", result.RawParams.MemoType)
		}
		if result.RawParams.Callback != "url:https://example.com/callback" {
			t.Errorf("callback mismatch: %s", result.RawParams.Callback)
		}
		if result.RawParams.Msg != "Pay me with lumens" {
			t.Errorf("msg mismatch: %s", result.RawParams.Msg)
		}
		if result.RawParams.OriginDomain != "example.com" {
			t.Errorf("originDomain mismatch: %s", result.RawParams.OriginDomain)
		}
	})
}

func TestExtractRoutingFromURI_MAddress(t *testing.T) {
	const mAddress = "MAYCUYT553C5LHVE2XPW5GMEJT4BXGM7AHMJWLAPZP53KJO7EIQACAAAAAAAAAAAAD672"

	t.Run("passes M-address to extractRouting for canonical expansion", func(t *testing.T) {
		uri := "web+stellar:pay?destination=" + mAddress
		result := ExtractRoutingFromURI(uri)
		if !result.Success {
			t.Fatalf("expected success, got error: %s", result.Error)
		}
		if !strings.HasPrefix(result.Routing.DestinationBaseAccount, "G") {
			t.Errorf("expected base account starting with G, got %s", result.Routing.DestinationBaseAccount)
		}
		if result.Routing.RoutingID == nil {
			t.Fatalf("expected non-nil RoutingID for M-address")
		}
	})
}

func TestExtractRoutingFromURI_EdgeCases(t *testing.T) {
	const validG = "GCALNQQBXAPZ2WIRSDDBMSTAKCUH5SG6U76YBFLQLIXJTF7FE5AX7AOO"

	t.Run("handles URI with no query parameters", func(t *testing.T) {
		result := ExtractRoutingFromURI("web+stellar:pay")
		if result.Success {
			t.Fatalf("expected failure, got success")
		}
		if result.Code != ErrCodeMissingDestination {
			t.Errorf("expected code %s, got %s", ErrCodeMissingDestination, result.Code)
		}
	})

	t.Run("handles malformed URL encoding gracefully", func(t *testing.T) {
		uri := "web+stellar:pay?destination=" + validG + "&memo=%ZZ"
		result := ExtractRoutingFromURI(uri)
		if !result.Success {
			t.Fatalf("expected success, got error: %s", result.Error)
		}
		if result.RawParams.Memo != "%ZZ" {
			t.Errorf("expected fallback raw memo '%%ZZ', got '%s'", result.RawParams.Memo)
		}
	})

	t.Run("trims whitespace from destination", func(t *testing.T) {
		uri := "web+stellar:pay?destination=%20" + validG + "%20"
		result := ExtractRoutingFromURI(uri)
		if !result.Success {
			t.Fatalf("expected success, got error: %s", result.Error)
		}
		if result.RawParams.Destination != validG {
			t.Errorf("expected destination %s, got %s", validG, result.RawParams.Destination)
		}
	})
}
