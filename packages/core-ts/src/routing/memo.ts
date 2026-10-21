import { Warning } from "../address/types";
import { KnownMemoType } from "./types";

export type NormalizeResult = {
  normalized: string | null;
  warnings: Warning[];
};

/**
 * Maximum value for a 64-bit unsigned integer (uint64).
 */
const UINT64_MAX = BigInt("18446744073709551615");

/**
 * Maximum UTF-8 byte length of a MEMO_TEXT value (Stellar protocol limit).
 */
export const MEMO_TEXT_MAX_BYTES = 28;

/**
 * Byte length of a MEMO_HASH / MEMO_RETURN value.
 */
export const MEMO_HASH_BYTES = 32;

/**
 * Number of hex characters encoding a MEMO_HASH / MEMO_RETURN value.
 */
export const MEMO_HASH_HEX_LENGTH = MEMO_HASH_BYTES * 2;

export type MemoValidationErrorCode =
  | "MEMO_TEXT_TOO_LONG"
  | "MEMO_ID_INVALID"
  | "MEMO_HASH_INVALID";

export type MemoValidationError = {
  code: MemoValidationErrorCode;
  message: string;
};

export type MemoValidationResult = {
  valid: boolean;
  /**
   * Length of the memo value in UTF-8 bytes (never UTF-16 code units, so
   * multi-byte characters are counted correctly).
   */
  byteLength: number;
  /**
   * Protocol limit for the memo type, expressed in `lengthUnit`, or null when
   * the memo type has no length limit (e.g. MEMO_ID).
   */
  maxLength: number | null;
  /**
   * Unit `maxLength` is expressed in: UTF-8 bytes for MEMO_TEXT, hex
   * characters for MEMO_HASH / MEMO_RETURN, null when unbounded.
   */
  lengthUnit: "bytes" | "characters" | null;
  error: MemoValidationError | null;
};

const utf8Encoder = new TextEncoder();

/**
 * Validates a memo value against the Stellar protocol limits for its type:
 * MEMO_TEXT must fit in 28 UTF-8 bytes, MEMO_ID must be an unsigned 64-bit
 * integer, and MEMO_HASH / MEMO_RETURN must be 32 bytes (64 hex characters).
 *
 * @param value - The raw memo value as entered by the user.
 * @param type - The memo type the value will be submitted as.
 * @returns The UTF-8 byte length, the applicable protocol limit, and the first
 * violation found (or null when the value is valid).
 */
export function validateMemo(
  value: string,
  type: KnownMemoType
): MemoValidationResult {
  const byteLength = utf8Encoder.encode(value).length;

  switch (type) {
    case "text": {
      if (byteLength > MEMO_TEXT_MAX_BYTES) {
        return {
          valid: false,
          byteLength,
          maxLength: MEMO_TEXT_MAX_BYTES,
          lengthUnit: "bytes",
          error: {
            code: "MEMO_TEXT_TOO_LONG",
            message: `MEMO_TEXT cannot exceed ${MEMO_TEXT_MAX_BYTES} UTF-8 bytes (got ${byteLength}).`,
          },
        };
      }

      return {
        valid: true,
        byteLength,
        maxLength: MEMO_TEXT_MAX_BYTES,
        lengthUnit: "bytes",
        error: null,
      };
    }

    case "id": {
      const { normalized } = normalizeMemoTextId(value);

      if (normalized === null) {
        return {
          valid: false,
          byteLength,
          maxLength: null,
          lengthUnit: null,
          error: {
            code: "MEMO_ID_INVALID",
            message: "MEMO_ID must be an unsigned 64-bit integer.",
          },
        };
      }

      return {
        valid: true,
        byteLength,
        maxLength: null,
        lengthUnit: null,
        error: null,
      };
    }

    case "hash":
    case "return": {
      const label = type === "hash" ? "MEMO_HASH" : "MEMO_RETURN";
      const result = {
        byteLength,
        maxLength: MEMO_HASH_HEX_LENGTH,
        lengthUnit: "characters" as const,
      };

      if (value.length !== MEMO_HASH_HEX_LENGTH) {
        return {
          ...result,
          valid: false,
          error: {
            code: "MEMO_HASH_INVALID",
            message: `${label} must be exactly ${MEMO_HASH_BYTES} bytes (${MEMO_HASH_HEX_LENGTH} hex characters).`,
          },
        };
      }

      if (!/^[0-9a-fA-F]+$/.test(value)) {
        return {
          ...result,
          valid: false,
          error: {
            code: "MEMO_HASH_INVALID",
            message: `${label} must be a hexadecimal string.`,
          },
        };
      }

      return { ...result, valid: true, error: null };
    }

    case "none": {
      return {
        valid: true,
        byteLength,
        maxLength: null,
        lengthUnit: null,
        error: null,
      };
    }
  }
}

/**
 * Normalizes a numeric string into a canonical uint64 representation.
 * Strips leading zeros and validates that the value is within uint64 boundaries.
 * 
 * @param s - The numeric string to normalize.
 * @returns Result containing the normalized string (or null if invalid) and any warnings.
 */
export function normalizeMemoTextId(s: string): NormalizeResult {
  const warnings: Warning[] = [];

  if (s.length === 0 || !/^\d+$/.test(s)) {
    return { normalized: null, warnings };
  }

  let normalized = s.replace(/^0+/, "");
  if (normalized === "") {
    normalized = "0";
  }

  if (normalized !== s) {
    warnings.push({
      code: "NON_CANONICAL_ROUTING_ID",
      severity: "warn",
      message:
        "Memo routing ID had leading zeros. Normalized to canonical decimal.",
      normalization: { original: s, normalized },
    });
  }

  try {
    const val = BigInt(normalized);
    if (val > UINT64_MAX) {
      return { normalized: null, warnings };
    }
  } catch {
    return { normalized: null, warnings };
  }

  return { normalized, warnings };
}
