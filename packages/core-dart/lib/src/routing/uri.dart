import 'extract.dart';
import 'routing_result.dart';

/// Parameters supported by SEP-0007 pay operations.
class Sep7PayParams {
  final String destination;
  final String? amount;
  final String? assetCode;
  final String? assetIssuer;
  final String? memo;
  final String? memoType;
  final String? callback;
  final String? msg;
  final String? networkPassphrase;
  final String? originDomain;
  final String? signature;

  const Sep7PayParams({
    required this.destination,
    this.amount,
    this.assetCode,
    this.assetIssuer,
    this.memo,
    this.memoType,
    this.callback,
    this.msg,
    this.networkPassphrase,
    this.originDomain,
    this.signature,
  });

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is Sep7PayParams &&
          runtimeType == other.runtimeType &&
          destination == other.destination &&
          amount == other.amount &&
          assetCode == other.assetCode &&
          assetIssuer == other.assetIssuer &&
          memo == other.memo &&
          memoType == other.memoType &&
          callback == other.callback &&
          msg == other.msg &&
          networkPassphrase == other.networkPassphrase &&
          originDomain == other.originDomain &&
          signature == other.signature;

  @override
  int get hashCode => Object.hash(
        destination,
        amount,
        assetCode,
        assetIssuer,
        memo,
        memoType,
        callback,
        msg,
        networkPassphrase,
        originDomain,
        signature,
      );
}

/// Standard error codes for SEP-0007 URI parsing.
abstract final class Sep7UriErrorCode {
  static const String invalidUri = 'INVALID_URI';
  static const String unsupportedOperation = 'UNSUPPORTED_OPERATION';
  static const String missingDestination = 'MISSING_DESTINATION';
  static const String invalidEncoding = 'INVALID_ENCODING';
}

/// Result of parsing a SEP-0007 payment URI.
sealed class ExtractRoutingFromUriResult {
  final bool success;

  const ExtractRoutingFromUriResult._({required this.success});

  const factory ExtractRoutingFromUriResult.success({
    required RoutingResult routing,
    required Sep7PayParams rawParams,
  }) = ExtractRoutingFromUriSuccess;

  const factory ExtractRoutingFromUriResult.failure({
    required String error,
    required String code,
  }) = ExtractRoutingFromUriFailure;
}

/// Successful result of SEP-0007 URI parsing.
final class ExtractRoutingFromUriSuccess extends ExtractRoutingFromUriResult {
  final RoutingResult routing;
  final Sep7PayParams rawParams;

  const ExtractRoutingFromUriSuccess({
    required this.routing,
    required this.rawParams,
  }) : super._(success: true);

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is ExtractRoutingFromUriSuccess &&
          routing == other.routing &&
          rawParams == other.rawParams;

  @override
  int get hashCode => Object.hash(routing, rawParams);
}

/// Failed result of SEP-0007 URI parsing.
final class ExtractRoutingFromUriFailure extends ExtractRoutingFromUriResult {
  final String error;
  final String code;

  const ExtractRoutingFromUriFailure({
    required this.error,
    required this.code,
  }) : super._(success: false);

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is ExtractRoutingFromUriFailure &&
          error == other.error &&
          code == other.code;

  @override
  int get hashCode => Object.hash(error, code);
}

/// Map SEP-0007 memo_type values to internal KnownMemoType.
String _mapMemoType(String? sep7MemoType) {
  if (sep7MemoType == null || sep7MemoType.isEmpty) return 'none';

  final upper = sep7MemoType.toUpperCase();
  switch (upper) {
    case 'MEMO_ID':
      return 'id';
    case 'MEMO_TEXT':
      return 'text';
    case 'MEMO_HASH':
      return 'hash';
    case 'MEMO_RETURN':
      return 'return';
    default:
      return 'none';
  }
}

/// Safely decode a URL component with graceful fallback to raw string.
String? _safelyDecode(String? value) {
  if (value == null || value.isEmpty) {
    return null;
  }
  try {
    return Uri.decodeQueryComponent(value);
  } catch (_) {
    return value;
  }
}

/// Parses a SEP-0007 URI and extracts canonical routing information.
///
/// Supported format:
///   `web+stellar:pay?destination=<G...>&memo=<value>&memo_type=<MEMO_TEXT|MEMO_ID|MEMO_HASH|MEMO_RETURN>`
///
/// Returns [ExtractRoutingFromUriResult].
ExtractRoutingFromUriResult extractRoutingFromURI(String uriString) {
  const schemePrefix = 'web+stellar:';
  // 1. Validate scheme
  if (!uriString.startsWith(schemePrefix)) {
    return const ExtractRoutingFromUriResult.failure(
      error: "URI must use 'web+stellar:' scheme",
      code: Sep7UriErrorCode.invalidUri,
    );
  }

  // 2. Split operation and query string
  final withoutScheme = uriString.substring(schemePrefix.length);
  final qIndex = withoutScheme.indexOf('?');
  final String operation;
  final String queryString;
  if (qIndex != -1) {
    operation = withoutScheme.substring(0, qIndex);
    queryString = withoutScheme.substring(qIndex + 1);
  } else {
    operation = withoutScheme;
    queryString = '';
  }

  // 3. Only 'pay' operation is supported for routing extraction
  if (operation != 'pay') {
    return ExtractRoutingFromUriResult.failure(
      error:
          "Unsupported operation: '$operation'. Only 'pay' is supported for routing extraction.",
      code: Sep7UriErrorCode.unsupportedOperation,
    );
  }

  // 4. Parse query parameters safely
  final Map<String, String> paramMap = {};
  if (queryString.isNotEmpty) {
    final pairs = queryString.split('&');
    for (final pair in pairs) {
      if (pair.isEmpty) continue;
      final eqIndex = pair.indexOf('=');
      final String key;
      final String val;
      if (eqIndex != -1) {
        key = pair.substring(0, eqIndex);
        val = pair.substring(eqIndex + 1);
      } else {
        key = pair;
        val = '';
      }
      if (!paramMap.containsKey(key)) {
        paramMap[key] = val;
      }
    }
  }

  // 5. Extract required destination
  final rawDestination = paramMap['destination'];
  final decodedDestination = _safelyDecode(rawDestination);
  final destination = decodedDestination?.trim();
  if (destination == null || destination.isEmpty) {
    return const ExtractRoutingFromUriResult.failure(
      error: "Missing required 'destination' parameter",
      code: Sep7UriErrorCode.missingDestination,
    );
  }

  // 6. Extract optional parameters with safe decoding
  final rawParams = Sep7PayParams(
    destination: destination,
    amount: _safelyDecode(paramMap['amount']),
    assetCode: _safelyDecode(paramMap['asset_code']),
    assetIssuer: _safelyDecode(paramMap['asset_issuer']),
    memo: _safelyDecode(paramMap['memo']),
    memoType: _safelyDecode(paramMap['memo_type']),
    callback: _safelyDecode(paramMap['callback']),
    msg: _safelyDecode(paramMap['msg']),
    networkPassphrase: _safelyDecode(paramMap['network_passphrase']),
    originDomain: _safelyDecode(paramMap['origin_domain']),
    signature: _safelyDecode(paramMap['signature']),
  );

  // 7. Build RoutingInput for extractRoutingSync
  final routingInput = RoutingInput(
    destination: rawParams.destination,
    memoType: _mapMemoType(rawParams.memoType),
    memoValue: rawParams.memo,
    sourceAccount: null,
  );

  // 8. Delegate to core extractRoutingSync
  final routingResult = extractRoutingSync(routingInput);

  // 9. Return combined success result
  return ExtractRoutingFromUriResult.success(
    routing: routingResult,
    rawParams: rawParams,
  );
}
