import StellarSdk from "@stellar/stellar-sdk";
import { RoutingResult } from "./types";
import { extractRouting } from "./extract";

const { Transaction } = StellarSdk;

/**
 * Operation types that carry a deposit destination.
 * `createAccount` and other operations are deliberately excluded.
 */
const PAYMENT_LIKE_OP_TYPES = new Set([
  "payment",
  "pathPaymentStrictReceive",
  "pathPaymentStrictSend",
]);

const HEX_DIGITS = "0123456789abcdef";
const utf8Decoder = new TextDecoder();

function isPaymentLikeOperation(op: any): boolean {
  return !!op && PAYMENT_LIKE_OP_TYPES.has(op.type);
}

function bytesToHex(bytes: Uint8Array): string {
  let hex = "";
  for (let i = 0; i < bytes.length; i++) {
    hex += HEX_DIGITS[bytes[i] >> 4] + HEX_DIGITS[bytes[i] & 0x0f];
  }
  return hex;
}

/**
 * Converts the various `tx.memo.value` shapes emitted by
 * @stellar/stellar-sdk into the string form expected by `extractRouting`.
 * Buffers are decoded per memo type: MEMO_HASH / MEMO_RETURN hold opaque
 * bytes (hex-encoded), MEMO_TEXT / MEMO_ID hold UTF-8 text.
 */
function memoValueToString(value: unknown, memoType: string): string | null {
  if (value === null || value === undefined) return null;
  if (typeof value === "string") return value;
  if (typeof value === "bigint" || typeof value === "number") {
    return value.toString();
  }
  if (value instanceof Uint8Array) {
    return memoType === "hash" || memoType === "return"
      ? bytesToHex(value)
      : utf8Decoder.decode(value);
  }
  return null;
}

/**
 * Extracts routing information from a Stellar transaction.
 *
 * Considers the first operation only, but accepts every payment-like
 * operation type (payment, pathPaymentStrictReceive, pathPaymentStrictSend).
 * Use `extractAllRoutingFromTx` to route batched operations.
 *
 * @param {Transaction} tx - The transaction to extract routing info from.
 * @returns {RoutingResult | null} The routing result or null if the first operation is not a payment.
 */
export function extractRoutingFromTx(tx: any): RoutingResult | null {
  const op = tx.operations[0];
  if (!isPaymentLikeOperation(op)) return null;

  return extractRouting({
    destination: op.destination,
    memoType: tx.memo.type,
    memoValue: memoValueToString(tx.memo.value, tx.memo.type),
    sourceAccount: tx.source ?? null,
  });
}

/**
 * A routing result paired with the index of the operation it was derived from.
 */
export type OperationRoutingResult = {
  operationIndex: number;
  result: RoutingResult;
};

/**
 * Extracts routing information from every payment-like operation in a
 * transaction, so batched deposits are not silently dropped.
 *
 * @param {Transaction} tx - The transaction to inspect.
 * @returns One entry per payment-like operation, tagged with its index in `tx.operations`.
 */
export function extractAllRoutingFromTx(tx: any): OperationRoutingResult[] {
  const memoType = tx.memo.type;
  const memoValue = memoValueToString(tx.memo.value, memoType);
  const sourceAccount = tx.source ?? null;
  const results: OperationRoutingResult[] = [];

  for (let operationIndex = 0; operationIndex < tx.operations.length; operationIndex++) {
    const op = tx.operations[operationIndex];
    if (!isPaymentLikeOperation(op)) continue;

    results.push({
      operationIndex,
      result: extractRouting({
        destination: op.destination,
        memoType,
        memoValue,
        sourceAccount,
      }),
    });
  }

  return results;
}
